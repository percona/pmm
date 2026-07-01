---
name: mysql-replication-topology
description: Percona_MS_MySQLReplicaLag, Percona_MS_MySQLReplication, Percona_MS_MySQLReplicaReadOnly,
  Percona_MS_ErrantGTID — MySQL/MariaDB replica lag, replication IO/SQL threads, unexpected
  read_only, errant GTID
---

# MySQL replication topology (replica lag, threads, read_only, errant GTID)

## Goal

Determine why replication is slow or stopped, whether a replica is writable by mistake, and whether errant GTIDs block failover—using PMM replication metrics and `SHOW REPLICA STATUS`, then exact remediation SQL.

## Workflow

### Alert routing

| Alert | Focus |
|--------|--------|
| **Percona_MS_MySQLReplicaLag** | `Seconds_Behind_Source` / `Seconds_Behind_Master` vs `sql_delay` |
| **Percona_MS_MySQLReplication** | IO or SQL thread not running |
| **Percona_MS_MySQLReplicaReadOnly** | Replica with `read_only=OFF` |
| **Percona_MS_ErrantGTID** | GTID on replica not on primary |

1. **Scope**
   * **pmm-inventory** → **service_id** for primary and replica from topology labels.

2. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=mysql`, `intent=replication`, `service_id` from step 1 (replica or primary as scoped).
   * Render lag / IO/SQL thread panels from map; **embed every `image_url`**. **`pmm_metrics_snapshot`** on panel `expr`.

3. **Logs**
   * **otel.logs** for replication errors, duplicate key, network to primary.

4. **Branch: lag** (`MySQLReplicaLag`)
   ```sql
   SHOW REPLICA STATUS\G
   ```
   * Check `Replica_IO_Running`, `Replica_SQL_Running`, `Seconds_Behind_Source` (or `Seconds_Behind_Master` on older versions).
   * On replica:
   ```sql
   SHOW PROCESSLIST;
   ```
   * Find `system user` SQL thread state; kill long blocking **SELECT** on replica only with customer approval.
   * Parallel apply (example — validate version and policy):
   ```sql
   STOP REPLICA;
   SET GLOBAL replica_parallel_type = 'LOGICAL_CLOCK';   -- or slave_parallel_type on 5.7
   SET GLOBAL replica_parallel_workers = 8;
   START REPLICA;
   ```
   * Replica IO tuning (customer approval, understand durability tradeoffs):
   ```sql
   SET GLOBAL innodb_flush_log_at_trx_commit = 2;
   SET GLOBAL sync_binlog = 0;
   ```
   * Large DML on source: recommend batched deletes/updates or `pt-archiver`.

5. **Branch: replication stopped** (`MySQLReplication`)
   ```sql
   SHOW REPLICA STATUS\G
   ```
   * **IO thread**: network, port 3306, replication user, missing binlog on source — fix connectivity/creds; `CHANGE REPLICATION SOURCE TO` only per procedure.
   * **SQL thread**: read `Last_SQL_Error` (e.g. 1062 duplicate). Resolve duplicate row vs `SKIP` — prefer consistent data fix:
   ```sql
   STOP REPLICA;
   SET GLOBAL sql_replica_skip_counter = 1;   -- LAST RESORT; deprecated paths exist in 8.0+ — use GTID-aware procedures from internal docs
   START REPLICA;
   ```
   * Prefer reseed from backup if errors repeat.
   * Enforce on replicas:
   ```sql
   SET GLOBAL read_only = 1;
   SET GLOBAL super_read_only = 1;
   ```

6. **Branch: replica not read-only** (`MySQLReplicaReadOnly`)
   ```sql
   SELECT @@global.read_only, @@global.super_read_only;
   ```
   ```sql
   SET GLOBAL read_only = 1;
   SET GLOBAL super_read_only = 1;
   ```
   * Persist in `my.cnf`:
   ```ini
   [mysqld]
   read_only = 1
   super_read_only = 1
   ```
   * Audit users with `SUPER` / `SYSTEM_VARIABLES_ADMIN`.

7. **Branch: errant GTID** (`ErrantGTID`)
   ```sql
   SHOW REPLICA STATUS\G
   ```
   * Compare `Executed_Gtid_Set` replica vs primary.
   ```sql
   SELECT GTID_SUBTRACT('<replica_executed_gtid_set>', '<primary_executed_gtid_set>');
   ```
   * Remove errant data on replica if any; inject empty transaction on **primary** for each errant GTID (CHG-controlled):
   ```sql
   SET GTID_NEXT = '<UUID>:<seq>';
   BEGIN; COMMIT;
   SET GTID_NEXT = AUTOMATIC;
   ```
   * Many errant GTIDs: reclone replica from consistent source.

## Synthesize Findings

State IO vs SQL bottleneck, whether lag is catch-up vs stuck thread, whether replica accepted writes, and exact GTID gap if errant. Note failover tool impact (Orchestrator).

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

* Lag: capacity, parallel workers, batch writes on primary, end replica blockers.
* Stopped SQL: fix data drift or reseed; avoid blind skip counter.
* read_only: enable super_read_only on all replicas; fix app double-writes.
* Errant GTID: empty trx on primary or full resync per severity.
