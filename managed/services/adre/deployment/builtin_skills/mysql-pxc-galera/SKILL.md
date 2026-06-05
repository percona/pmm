---
name: mysql-pxc-galera
description: Percona_MS_PXCWsrepDesync, Percona_MS_PXCFlowControl, Percona_MS_PXCClusterStatus
  — Percona XtraDB Cluster wsrep desync, Galera flow control paused, cluster non-Primary
  / split-brain risk
---

# Percona XtraDB Cluster / Galera (wsrep)

## Goal

Diagnose wsrep desync, flow control, and non-Primary cluster state using PMM wsrep metrics and `SHOW GLOBAL STATUS`, then apply exact SQL and shell for PXC—without guessing bootstrap in production.

## Workflow

### Alert routing

| Alert | Focus |
|--------|--------|
| **Percona_MS_PXCWsrepDesync** | `wsrep_local_state` / comment Desynced |
| **Percona_MS_PXCFlowControl** | `wsrep_flow_control_paused` / paused time |
| **Percona_MS_PXCClusterStatus** | Non-Primary / split / quorum loss |

1. **Scope**
   * **pmm-inventory** → **service_id** for each PXC node; note cluster name from labels.

2. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=mysql`, `intent=group_replication` (or `replication` if map warns), `service_id` from step 1.
   * Render wsrep / cluster panels from map; **embed every `image_url`**. **`pmm_metrics_snapshot`** on panel `expr` for flow control, recv queue, cluster status.

3. **Logs**
   * **otel.logs** + operator error log: SST, IST, partition, flow control messages.

4. **Branch: desync** (`PXCWsrepDesync`)
   ```sql
   SHOW GLOBAL STATUS LIKE 'wsrep_local_state_comment';
   SELECT @@global.wsrep_desync;
   ```
   * If SST/IST in progress: wait; monitor datadir growth:
   ```bash
   sudo du -sch /var/lib/mysql
   ```
   * If manual desync should end:
   ```sql
   SET GLOBAL wsrep_desync = OFF;
   SHOW GLOBAL STATUS LIKE 'wsrep_local_state_comment';
   ```
   * Frequent desync under load: PRB for `wsrep_slave_threads`, disk IO, network.

5. **Branch: flow control** (`PXCFlowControl`)
   ```sql
   SHOW GLOBAL STATUS LIKE 'wsrep_local_recv_queue%';
   SHOW GLOBAL STATUS LIKE 'wsrep_flow_control_paused';
   ```
   * Operator: `top`, `iostat -xz 1` on slow node.
   * Increase appliers with customer approval:
   ```sql
   SET GLOBAL wsrep_slave_threads = <N>;
   ```
   * Break large transactions into chunks; storage upgrade if one node is persistently slow.

6. **Branch: cluster non-Primary / quorum** (`PXCClusterStatus`)
   ```sql
   SHOW GLOBAL STATUS LIKE 'wsrep_cluster_status';
   SHOW GLOBAL STATUS LIKE 'wsrep_connected';
   SHOW GLOBAL STATUS LIKE 'wsrep_cluster_size';
   ```
   * Check inter-node connectivity (Galera ports, e.g. 4567–4568) with `ping`/`nc` from operator.
   * Single node disconnected: restart MySQL on that node after log review:
   ```bash
   sudo systemctl restart mysql
   ```
   * **Quorum lost / full cluster down**: bootstrap is destructive — follow official PXC bootstrap procedure only with customer sign-off. Outline only (operator):
   ```bash
   sudo systemctl stop mysql
   ```
   * On chosen donor node with most recent data, set `safe_to_bootstrap: 1` in `grastate.dat` per Percona docs, then start with bootstrap service name your OS uses, e.g.:
   ```bash
   sudo systemctl start mysql@bootstrap
   ```
   * Then start other nodes normally and verify `wsrep_cluster_size` and Primary.

## Synthesize Findings

State whether desync is maintenance-induced, catch-up, or capacity; whether flow control is cluster-wide stall; whether partition or single-node failure; **never** recommend bootstrap without explicit quorum-loss procedure.

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

* End `wsrep_desync` when safe; wait for SST/IST to complete.
* Flow control: more `wsrep_slave_threads`, smaller transactions, faster disk, network fix.
* Non-Primary: restore connectivity; restart non-primary member; full bootstrap only per runbook and leadership approval.
* Escalate recurring flow control or desync to PRB for sizing and Galera tuning.
