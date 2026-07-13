---
name: mysql-availability
description: Percona_MS_MySQLInstanceNotAvailableOrRestarted, Percona_MS_MySQLTooManyConnections,
  Percona_MS_MySQLThreadCreate, Percona_MS_MySQLTooManyThreadsRunning, Percona_MS_MySQLDeadlock,
  Percona_MS_MySQLAutoIncrementExhaustion, Percona_MS_MySQLPrimaryReadOnly, Percona_MS_MySQLHistoryLength,
  Percona_MS_MySQLInnoDBCheckpointAge, Percona_MS_MySQLScrapingIssues, Percona_MS_IPController
  — mysqld up/down, connections, threads, deadlocks, InnoDB, autoincrement, VIP (non-ProxySQL)
---

# MySQL availability, connections, and InnoDB (non-replication, non-PXC)

## Goal

Diagnose MySQL instance down/restart, connection/thread pressure, deadlocks, InnoDB history/checkpoint pressure, autoincrement limits, unexpected primary read-only, exporter scrape failures, and MySQL-side IP controller signals—using PMM first, then exact SQL and shell for operators.

## Workflow

### Alert routing

| Alert | Focus |
|--------|--------|
| **Percona_MS_MySQLInstanceNotAvailableOrRestarted** | `mysql_up`, low uptime — crash vs maintenance |
| **Percona_MS_MySQLTooManyConnections** | Near `max_connections` |
| **Percona_MS_MySQLThreadCreate** | High `Threads_created` rate vs `thread_cache_size` |
| **Percona_MS_MySQLTooManyThreadsRunning** | High `Threads_running` — locks, slow IO, hot queries |
| **Percona_MS_MySQLDeadlock** | InnoDB deadlock rate |
| **Percona_MS_MySQLAutoIncrementExhaustion** | AUTO_INCREMENT nearing type max |
| **Percona_MS_MySQLPrimaryReadOnly** | Primary should be writable |
| **Percona_MS_MySQLHistoryLength** | InnoDB undo history length — long transactions / purge |
| **Percona_MS_MySQLInnoDBCheckpointAge** | Checkpoint age vs max — IO/redo pressure |
| **Percona_MS_MySQLScrapingIssues** | Exporter cannot scrape mysqld |
| **Percona_MS_IPController** | GAS ip_controller / VIP failover script errors (MySQL cluster VIP) |

1. **Scope**
   * **pmm-inventory** → **service_id**, **node_id**, **agent_id** from alert labels.

2. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=mysql`, `service_id` from step 1. Pick **intent** by alert:
     | Alert cluster | intent |
     |---|---|
     | InstanceNotAvailableOrRestarted, ScrapingIssues | availability |
     | TooManyConnections, ThreadCreate, TooManyThreadsRunning | connections |
     | Deadlock, HistoryLength, InnoDBCheckpointAge | innodb |
     | PrimaryReadOnly | availability or workload |
   * Render **2–4** panels from map; **embed every `image_url`**. **`pmm_metrics_snapshot`** on panel `expr`.
   * For connection/thread pressure with slow queries: **`pmm.metrics`** top patterns + **`general`** skill Step 3b if MySQL.

3. **Logs**
   * **otel.logs** for mysqld error log lines in window (crash, OOM, corruption).

4. **Branch: instance down / restarted**
   * Operator:
   ```bash
   sudo systemctl status mysqld || sudo systemctl status mysql
   sudo systemctl start mysqld   # only if logs permit
   ```
   * Error log (path from `my.cnf` `log_error`):
   ```bash
   sudo less <error_log_path>
   sudo dmesg -T
   sudo less /var/log/syslog
   ```
   * If primary and HA exists, coordinate failover with customer before long restarts.

5. **Branch: too many connections**
   ```sql
   SELECT * FROM information_schema.processlist WHERE command != 'Sleep' ORDER BY time DESC;
   SHOW ENGINE INNODB STATUS\G
   ```
   * Kill after approval:
   ```sql
   KILL <thread_id>;
   ```
   * Temporary relief:
   ```sql
   SET GLOBAL max_connections = <n>;
   ```
   * Check ProxySQL multiplexing if used; the **general** skill (workload / time-window analysis) for sustained slow-query-driven pileups.

6. **Branch: thread create / thread cache**
   ```sql
   SHOW GLOBAL STATUS LIKE 'Threads_created';
   SHOW GLOBAL STATUS LIKE 'Connections';
   SHOW GLOBAL VARIABLES LIKE 'thread_cache_size';
   SHOW GLOBAL STATUS LIKE 'Threads_cached';
   SHOW GLOBAL STATUS LIKE 'Aborted_clients';
   ```
   ```sql
   SET GLOBAL thread_cache_size = 128;
   ```

7. **Branch: too many threads running**
   ```sql
   SHOW GLOBAL STATUS LIKE 'Threads_%';
   SELECT * FROM information_schema.processlist WHERE command != 'Sleep' ORDER BY time DESC;
   SHOW ENGINE INNODB STATUS\G
   ```
   * `KILL` hot sessions with approval; correlate CPU/disk via **node-capacity-saturation** skill.
   * Last resort (customer approval):
   ```bash
   sudo systemctl restart mysql
   ```

8. **Branch: deadlocks**
   ```sql
   SHOW ENGINE INNODB STATUS\G
   ```
   * Read `LATEST DETECTED DEADLOCK`; PMM deadlocks graph; app retries for 1213; indexes + shorter transactions.

9. **Branch: autoincrement exhaustion**
   * PMM / `information_schema` for offending table; plan `ALTER` to BIGINT — example:
   ```sql
   ALTER TABLE db1.t MODIFY id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT;
   ```
   * Run only after sizing and maintenance window plan.

10. **Branch: primary read_only**
    ```sql
    SELECT @@global.read_only, @@global.super_read_only;
    ```
    ```sql
    SET GLOBAL read_only = 0;
    SET GLOBAL super_read_only = 0;
    ```
    * Persist in `my.cnf` only per change process; investigate why it flipped (failover tool, maintenance).

11. **Branch: history length (undo)**
    * `SHOW ENGINE INNODB STATUS` — History list length; find long transactions in `information_schema.processlist`; end or kill with approval; tune purge only per analysis.

12. **Branch: InnoDB checkpoint age**
    * Correlate redo log usage, disk IO; may need IO capacity, `innodb_log_file_size` / throughput tuning — include exact `SET GLOBAL` only when variables are confirmed for version.

13. **Branch: scraping issues**
    * Same spine as exporter: **pmm-agent** logs on DB node (`journalctl -u pmm-agent`); credentials/exporter user; **pmm-monitoring-health** overlap.

14. **Branch: IP controller (MySQL VIP)**
    * Operator on monitor host:
    ```bash
    ps -ef | grep -i ip_controller
    less ~/.local/percona/ip_controller_<cluster_name>.log
    ```
    * If DB unreachable, recover MySQL first; verify GAS/cron and tool versions per internal docs.

## Synthesize Findings

Separate crash/OOM vs config vs workload vs monitoring gap. For connections/threads, name lead blockers from InnoDB status. For IP controller, separate script stall vs DB down.

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

* Start mysqld only when error log allows; fix `my.cnf` errors, disk full, corruption per Oracle/Percona procedures.
* **max_connections** / **thread_cache_size**: temporary `SET GLOBAL`; permanent my.cnf + restart policy.
* **read_only** on primary: clear only when topology role is confirmed.
* **Deadlocks / slow queries**: indexes, chunk DML, retries — link QAN via workload runbook.
* **Autoincrement**: ALTER column type; archive old rows if business allows.
* **IP controller**: fix cron, upgrade gas-tools, restore DB connectivity.
