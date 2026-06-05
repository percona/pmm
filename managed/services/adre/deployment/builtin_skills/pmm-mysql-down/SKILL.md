---
name: pmm_mysql_down
description: 'Investigate MySQL service down (pmm_mysql_down alert): check mysql_up,
  logs (otel.logs), permissions, and PMM agent'
---

# MySQL service down (pmm_mysql_down)

## Purpose

Use this runbook when investigating the **pmm_mysql_down** alert or any incident where a MySQL service monitored by PMM is reported as down. It uses Prometheus metrics (mysql_up), ClickHouse logs (otel.logs), and PMM inventory to determine why the service is down and what to do next.


## When NOT to use this runbook

**Do not fetch this runbook** when:

- When no one requested a service/node down investigation.
- When answering a general question.
- The message is **casual** (hi, thanks, ping, ok) or **off-topic**.
- The user asked for **only one specific instant metric** and no interpretation across time or systems (e.g. “current QPS” only) — unless they also asked to explain workload or a period.
- The question is a **simple factual lookup** answerable in **one short sentence** without a multi-step investigation (e.g. “How many MySQL services are there?”, “What’s the replication lag right now?”, “Is node X up?”) — answer with the minimal tool calls instead.

If unsure: prefer **not** fetching this runbook; answer directly with tools.

## Prerequisites

- PMM with Prometheus/VictoriaMetrics and a MySQL agent (or node exporter for the node).
- For logs: ClickHouse OTel toolset (otel.logs). Filter by `ResourceAttributes['node_name']` or `ResourceAttributes['pmm_source']`; never use bare columns like node_name.
- Use **mysql_up** metric only for MySQL service availability. Do not use other metrics for service up/down.

## Steps

1. **Confirm the service is down**
   - Query Prometheus: `mysql_up` for the affected instance/labels. A value of 0 means down.
   - Use **pmm-inventory** to get **service_id**, **node_id**, **agent_id** for the affected MySQL service.
   - Optional context panels: **pmm_observability_map** `engine=mysql`, `intent=availability` — render 1–2 panels and **embed `image_url`** if useful for timeline around the outage.

2. **Check MySQL and PMM agent logs**
   - Query ClickHouse `otel.logs` filtered by the node or source:
     `WHERE ResourceAttributes['node_name'] = '<node_name>'` (or `ResourceAttributes['pmm_source'] = 'mysql'`).
     Order by `Timestamp DESC`, limit 50–100.
   - Look for: MySQL startup/shutdown messages, permission errors (e.g. binlog.index, data directory), OOM, crash, or "Aborting". Also check for PMM agent errors (connectivity, auth) on that node.

3. **Check Every Related Log Entry**
   - Even if MySQL had a clean shutdown, check what could be the reason why is it not starting. Check and analyse and report every errors in the log.

4. **Identify root cause**
   - **Permission errors** (e.g. `File './binlog.index' not found (OS errno 13 - Permission denied)`): MySQL data directory or binlog files not owned/readable by the mysql user. Suggest checking ownership and permissions.
   - **Repeated restarts / crash**: Check logs for stack traces, OOM, or config errors.
   - **PMM agent unreachable**: If logs show agent or connectivity issues, note that the problem may be agent-side or network; still report MySQL-level findings if present.

5. **Recommend actions**
   - Fix file permissions (e.g. `chown mysql:mysql` for data dir and binlog files, `chmod` as appropriate).
   - Ensure MySQL process runs as the correct user and that the data directory path is correct.
   - If the cause is unclear from logs, recommend starting MySQL manually and monitoring logs for the next failure.

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
## Notes

- Always use **mysql_up** for MySQL service availability. For node availability use the appropriate *_up metric.
- In otel.logs, filter only with `ResourceAttributes['node_name']` and/or `ResourceAttributes['pmm_source']`; do not use non-existent columns like node_name or pmm_source as bare column names.
- Include relevant log lines (Timestamp and Body) in the investigation report so the user sees the evidence.