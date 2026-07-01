---
name: node-capacity-saturation
description: Percona_MS_DiskSpaceLow, Percona_MS_NodeLowFreeMemory, Percona_MS_NodeLowFreeMemoryTrend,
  Percona_MS_NodeCpuUtilization, Percona_MS_NodeIOUtilization, Percona_MS_NodeLoad,
  Percona_MS_NodeNetworkErrors, Percona_MS_CPUSaturationRDS — Linux/EC2 node disk,
  RAM, CPU, IO, load average, network errors
---

# Node capacity and saturation

## Goal

Determine whether the alert reflects real resource exhaustion versus noise; identify the dominant resource (disk, memory, CPU, IO, network); correlate with database and co-tenant processes on the PMM-monitored host. Prefer completing the critical path in one turn.

## Workflow

### Alert routing (use labels / alertname)

- **Percona_MS_DiskSpaceLow**: Filesystem low free space or forecast to full — use mount/device from labels; check growth rate.
- **Percona_MS_NodeLowFreeMemory / Percona_MS_NodeLowFreeMemoryTrend**: Low available RAM or worsening trend — distinguish cache vs genuine pressure; correlate with OOM or DB buffer usage in metrics.
- **Percona_MS_NodeCpuUtilization, Percona_MS_NodeLoad, Percona_MS_NodeIOUtilization, Percona_MS_CPUSaturationRDS**: Correlate CPU, load, and IO wait before attributing to a single subsystem (RDS-specific rule still maps to CPU saturation semantics).
- **Percona_MS_NodeNetworkErrors**: Interface errors/drops; correlate with app timeouts in logs if present.

1. **Resolve scope in PMM**
   * Call **pmm-inventory** and map alert labels (instance, node, service_name) to **service_id**, **node_id**, and **agent_id**.
   * If the alert is node-wide, list every **service_id** on that **node_id** for correlation.

2. **Time window**
   * Align with alert firing or user window. For **pmm_render_grafana_panel**, use Grafana-style `from`/`to` (e.g. `now-3h`, `now`).
   * For **pmm.metrics** (ClickHouse QAN), use ClickHouse-native time literals or `parseDateTimeBestEffort` — do not use RFC3339 strings in WHERE.

3. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=node`, `service_id` / node from step 1. Pick **intent** by alert:
     | Alert cluster | intent |
     |---|---|
     | DiskSpaceLow | disk_io |
     | NodeLowFreeMemory, NodeLowFreeMemoryTrend | cpu_memory |
     | NodeCpuUtilization, NodeLoad, NodeIOUtilization, CPUSaturationRDS | cpu_memory |
     | NodeNetworkErrors | network |
   * Render **2–4** panels from map; **embed every `image_url`**. **`pmm_metrics_snapshot`** on panel `expr`.

4. **Logs (optional)**
   * **otel.logs** for OOM killer, disk full, I/O errors, ext4/xfs messages in the same window (ClickHouse-appropriate time filters).

5. **Workload-driven saturation**
   * If evidence points to query load rather than raw OS cap, for **MySQL** the **general** skill (workload / time-window analysis) or the **general** skill (if present) for QAN + **pmm_mysql_explain** depth — do not duplicate that full sequence here.

6. **Human-only steps**
   * Filesystem cleanup, LVM/cloud volume grow, or `du` investigation require shell access. State that Holmes cannot run them; give operators exact commands when paths are known, e.g. `df -h`, `sudo du -xh /var/lib/mysql | sort -h | tail -20`.

## Synthesize Findings

Correlate inode vs space exhaustion; memory vs swap vs reclaimable cache; CPU vs `iowait` vs load. Say whether the database service is the primary consumer or another process on the node. Tie timestamps to alert onset.

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

* **Disk low**: rotate logs, purge old backups, expand volume — e.g. after confirming mount: `sudo truncate -s 0 /var/log/huge.log` (only if safe), or cloud CLI resize per provider docs; include full command with device/volume id from evidence.
* **Memory pressure**: tune `innodb_buffer_pool_size` / app heap only when config is confirmed; add RAM or reduce workload; avoid generic advice without metric backing.
* **CPU/IO saturation**: reduce hot queries (workload runbook), scale instance class, or fix storage throughput limits — pair with panel/Prometheus evidence.
* **Network errors**: check cable/driver/MTU/SG; replace faulty NIC or escalate cloud network — document what metrics showed which interface.
