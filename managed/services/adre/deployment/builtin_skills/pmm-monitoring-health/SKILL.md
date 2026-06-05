---
name: pmm-monitoring-health
description: Percona_MS_DeadManSnitch, PMM_dead_man_snitch, Percona_MS_NodeAgentDown,
  Percona_MS_TimeDriftPmmAgents — PMM server heartbeat, monitor host reachability,
  pmm-agent down, NTP/clock drift between agents and PMM
---

# PMM monitoring and agent health

## Goal

Establish whether the incident is loss of PMM visibility (server, heartbeat, agent, or clock skew) versus a real database outage. Use PMM-native tools first; shell steps are for operators when Holmes has no SSH.

## Workflow

### Alert routing (use labels / alertname)

- **Percona_MS_DeadManSnitch / PMM_dead_man_snitch**: Heartbeat from PMM to Dead Man’s Snitch failed — PMM server down, network path broken, or rule misconfigured.
- **Percona_MS_NodeAgentDown**: `pmm-agent` on a monitored host is not reporting — no metrics/QAN from that node until restored.
- **Percona_MS_TimeDriftPmmAgents**: Clock skew between agent and PMM server risks incorrect graphs, QAN windows, and TLS validation issues.

1. **Resolve scope**
   * Call **pmm-inventory** and map alert labels to **node_id**, **service_id**, and **agent_id** when the alert names a specific node or service.
   * For DeadManSnitch with no node label, treat scope as **PMM server / monitoring plane** (still use inventory to confirm which PMM instance customers use).

2. **Firing alerts context**
   * If **pmm_list_firing_alerts** (or equivalent PMM alerts API) is available, list active alerts around the same time to see correlated `PMM_dead_man_snitch`, agent scrape, or multiple nodes down.

3. **PMM metrics and panels**
   * **No PMM-health observability-map route** — use **`pmm_list_dashboard_panels`** on PMM home/infra dashboard UIDs, then **`pmm_render_grafana_panel`**. **Embed every successful `image_url`**.
   * Scoped metrics: **`up`**, agent scrape series with tight matchers — **`pmm_list_metric_names`** with known prefix only (max 50); never unfiltered `__name__` browse.

4. **Logs**
   * Query **otel.logs** for PMM server or agent errors in the same window. Use ClickHouse-native time predicates for log timestamps — not RFC3339 strings in WHERE clauses.

5. **Human / operator steps (no PMM tool substitute)**
   * **PMM server on monitor host**: verify process (e.g. `podman ps`, `systemctl --user status pmm-server`) — Holmes cannot run these; output exact commands for the operator.
   * **pmm-agent on DB node**: `systemctl status pmm-agent` (or user unit), `journalctl -u pmm-agent`, restart: `sudo systemctl restart pmm-agent` after checking logs.
   * **Dead Man’s Snitch**: manual heartbeat test: `curl -d "m=just checking in" "https://nosnch.in/<SNITCH_ID>"` — replace `<SNITCH_ID>` from operator config; verify token in PMM alerting rule matches DMS.
   * **Time drift**: on node and PMM host, compare `timedatectl` / `chronyc tracking` / NTP — operator fixes NTP/chrony; include exact commands only when the environment is known.

## Synthesize Findings

Separate: (a) PMM or agent unavailable, (b) misconfiguration (tokens, rules), (c) network partition, (d) clock skew only. State impact: which nodes lost monitoring and for how long.

### Environment availability matrix (shared)

Before prescribing a command, decide **which surfaces this deployment exposes**. PMM tools (inventory, observability map, metrics snapshot, QAN on `pmm.metrics`, `otel.logs`, Grafana render) are available in **every** environment — always try those first. Host shell, DB superuser, and config-file edits are **not** universal.

| Surface | Self-managed (VM / bare-metal / Docker) | Kubernetes / Operator (PSMDB, PXC, PG operator) | Managed cloud (RDS / Aurora / Cloud SQL / Atlas) |
|---|---|---|---|
| PMM metrics / QAN / observability map | ✅ | ✅ | ✅ (exporter may be **remote/cloud-mode**) |
| `otel.logs` (PMM-collected) | ✅ | ✅ | ⚠️ partial — DB logs often only in the cloud console |
| Host shell (`journalctl`, `grep /var/log/...`, `openssl`, `df`) | ✅ operator | ⚠️ via `kubectl exec` into the pod, not the node | ❌ none |
| DB shell (`psql`, `mongosh`, `mysql`) | ✅ operator | ✅ operator via `kubectl exec` | ✅ but **no superuser** (rds_superuser / atlasAdmin only) |
| `ALTER SYSTEM` / `mongod.conf` / `my.cnf` edits | ✅ | ❌ — change the **Custom Resource**, not the file (operator reverts file edits) | ❌ — change a **parameter group / cluster config**, then reboot/apply |
| `pg_terminate_backend`, `KILL`, force-primary, oplog resize | ✅ with approval | ✅ with approval | ⚠️ often restricted or wrapped by a cloud API |

**Routing rules:**

- **Always lead with PMM/QAN/`otel.logs`.** They work everywhere and need no host/DB access.
- **Gate host-shell and superuser steps.** Label them **"operator, self-managed / k8s only"**. On managed cloud, **skip** them and use the cloud-native equivalent (console logs, Performance Insights / Cloud Monitoring, parameter groups, provider CLI).
- **Config changes:** on **operator** deployments edit the **CR** (e.g. `PerconaServerMongoDB`, `PerconaXtraDBCluster`, `PerconaPGCluster`) and let the operator roll it out; on **managed cloud** edit the **parameter group / flag** and apply per the provider; only on **self-managed** edit the file directly and reload/restart.
- **Detect the environment** from inventory (`service_type`, `node_model`/cloud labels, address like `*.rds.amazonaws.com`, `*.mongodb.net`) before recommending a step. If unknown, state the assumption and give the **PMM-only** path plus the operator path as an option.
- **Never tell the user to run a command their environment can't execute** (e.g. `journalctl` on RDS). Offer the reachable alternative instead.
### PMM observability and panel embed rules (shared)

**Evidence hierarchy:** inventory → **`pmm_observability_map`** → **`pmm_metrics_snapshot`** on panel `expr` → **`pmm_render_grafana_panel`** (embed PNGs, best-effort) → QAN + MySQL EXPLAIN → scoped fallbacks (`pmm_discover_series_labels`, `pmm_list_metric_names` prefix max 50) → **`execute_prometheus_range_query`** at most one per turn, last resort.

**Render failures are non-blocking:** 502, timeout, blank PNG, or curl error on render must **never** skip snapshot, QAN, EXPLAIN, or final synthesis. State **`rendered M/N`**; deliver analysis from snapshot + QAN even when **`rendered 0/N`**.

**FORBIDDEN:** unfiltered `GET /api/v1/label/__name__/values`, full dashboard JSON to the LLM, guessing panel IDs or metric names.

**Observability map:** `GET /v1/grafana/observability-map` via **`pmm_observability_map`**. Pass **`engine`** (`mysql`, `postgresql`, `mongodb`, `valkey`, `node`), **`intent`**, **`service_id`**. Use returned **`primary.dashboard_uid`**, **`panels[].id`**, **`panels[].expr`**. Fall back to **`pmm_list_dashboard_panels`** only when the map warns or intent is unknown (e.g. ProxySQL — no map route yet).

**Metrics snapshot:** **`pmm_metrics_snapshot`** on panel `expr` — returns server-computed **stats** (min/max/mean/median/p25/p75/p95/p99), **change_points**, **anomalies**; not raw matrices. Run **immediately after map** — does not require render. `start`/`end`: RFC3339 or Unix (not `now-24h`). Requires ADRE enabled.

**Render panels (best-effort when this skill uses Grafana):**
1. Inventory first: **`service_id`**, **`node_id`**, **`agent_id`**, **`service_name`**, **`node_name`**, **`version`** → pass all as `var_*` overrides to **`pmm_render_grafana_panel`**. Missing **`agent_id`** often causes blank/timeout renders.
2. Tool: **`POST /v1/grafana/render/resolve`** only (via **`pmm_render_grafana_panel`**).
3. Render **sequentially**; `from`/`to` without quotes (e.g. `now-6h`, `now`). Retry once with shorter window or fixed vars on failure — then **continue snapshot/QAN/EXPLAIN**.
4. **Embed every successful render** in the user-visible answer:
   ```markdown
   ![Panel title](/v1/grafana/render/blob/{hash}.png)
   [Open in Grafana](dashboard_url)
   ```
   Use **`image_url`** and **`dashboard_url` exactly** from tool JSON. Do not skip images for successful renders; do not use `dashboard_url` as img src, do not rebuild URLs.
5. State **`rendered M/N`** if any panel failed; **do not abort** the investigation.

**Scoped metric fallback (when map warns):** **`pmm_discover_series_labels`** with `service_id` + **`metric_prefix`** from map fallback, or **`pmm_list_metric_names`** with prefix only (max 50).

**QAN (database services):** For workload-correlated incidents, query **`pmm.metrics`** top slow patterns by **`service_id`** — see **`general`** skill for query shape and MySQL EXPLAIN Step 3b when applicable. **Always run** even when render fails.
## Recommended Remediation Steps

* **Agent down after crash**: fix config in `/path/to/pmm-agent.yaml` if logs show errors, then `sudo systemctl restart pmm-agent` and `sudo systemctl status pmm-agent`.
* **PMM server down**: start stack per install docs (e.g. `systemctl --user start pmm-server` or container start); verify UI loads and heartbeat alert clears.
* **Time sync**: `sudo chronyc makestep` or restart `chronyd`/`systemd-timesyncd` per distro after root cause (hypervisor drift, manual clock change).
* **DMS false positive**: correct webhook URL/snitch ID in Grafana alert rule; re-test with `curl` as above.
