---
name: postgresql-incidents
description: Percona_MS_PostgresqlLockConflicts, Percona_MS_PostgresqlExporterError,
  Percona_MS_PostgresqlReplicationLag, Percona_MS_PostgresqlTooManyConnections, Percona_MS_PostgresqlDeadLocks,
  Percona_MS_PostgresqlSlowQueries, Percona_MS_PostgresqlCommitRateLow, Percona_MS_PostgresqlTooManyLocksAcquired,
  Percona_MS_PostgresqlArchiveFailed, Percona_MS_PostgresqlIdleInTransaction, Percona_MS_PostgresqlUpTime,
  Percona_MS_PostgresqlTransactionDuration, Percona_MS_PostgresqlIsDown, Percona_MS_PostgresqlWraparound
  — PostgreSQL availability, locks, replication, WAL archive, wraparound, connections
---

# PostgreSQL incidents (managed service alerts)

## Goal

Map the firing PostgreSQL alert to the right diagnostic path, use PMM (inventory, metrics, Grafana, logs) first, then prescribe exact **psql** / operator SQL and shell where Holmes has no database shell.

## Workflow

### Alert routing

| Alert | Focus |
|--------|--------|
| **Percona_MS_PostgresqlLockConflicts** | Blocked vs blocking sessions; optional `pg_terminate_backend` with approval |
| **Percona_MS_PostgresqlExporterError** | PMM postgres_exporter or RDS remote exporter broken — metrics gap |
| **Percona_MS_PostgresqlReplicationLag** | WAL send/write/flush/replay; disk/network/long xact on primary or standby |
| **Percona_MS_PostgresqlTooManyConnections** | Near `max_connections`; idle / idle-in-transaction |
| **Percona_MS_PostgresqlDeadLocks** | Frequent deadlocks — app pattern, logs |
| **Percona_MS_PostgresqlSlowQueries** | Top statements — correlate with QAN (`pmm.metrics`); workload-style analysis |
| **Percona_MS_PostgresqlCommitRateLow** | High rollback ratio — logs + `pg_stat_database` |
| **Percona_MS_PostgresqlTooManyLocksAcquired** | Lock volume / contention |
| **Percona_MS_PostgresqlArchiveFailed** | WAL archiving to backup destination failing |
| **Percona_MS_PostgresqlIdleInTransaction** | Long idle-in-transaction holding locks |
| **Percona_MS_PostgresqlUpTime** / **Percona_MS_PostgresqlTransactionDuration** | Abnormally long transactions |
| **Percona_MS_PostgresqlIsDown** | Instance not accepting connections / exporter `up==0` |
| **Percona_MS_PostgresqlWraparound** | XID/multixact wraparound risk — vacuum health |

1. **Scope**
   * **pmm-inventory** → **service_id**, **node_id**, **agent_id** for the PostgreSQL service from alert labels.

2. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=postgresql`, `service_id` from step 1. Pick **intent** by alert:
     | Alert cluster | intent |
     |---|---|
     | IsDown, UpTime, ExporterError | availability |
     | TooManyConnections, IdleInTransaction, TransactionDuration | connections |
     | ReplicationLag, ArchiveFailed, Wraparound | replication or wal |
     | LockConflicts, DeadLocks, TooManyLocksAcquired | locks |
     | SlowQueries, CommitRateLow | workload or slow_queries |
   * Render **2–4** panels from map over alert window; **embed every `image_url`**. **`pmm_metrics_snapshot`** on panel `expr`.
   * **`PostgresqlSlowQueries`**: **`pmm.metrics`** top patterns by `service_id`; use **`general`** skill for full period workload if needed.

3. **Logs**
   * **otel.logs** for PostgreSQL pod/host logs in window — errors, archive failures, deadlocks (ClickHouse-safe time filters).

4. **Branch: locks / lock conflicts** (`PostgresqlLockConflicts`, blocking with replication lag)
   * Operator (superuser or `pg_terminate_backend` grant) — blocking pairs:
   ```sql
   SELECT activity.pid,
     activity.usename,
     left(activity.query, 200) AS query,
     blocking.pid AS blocking_pid,
     left(blocking.query, 200) AS blocking_query
   FROM pg_stat_activity AS activity
   JOIN pg_stat_activity AS blocking ON blocking.pid = ANY (pg_blocking_pids(activity.pid));
   ```
   * For a full wait-for graph, use the recursive `pg_locks` tree query from the managed service Node alert runbook (PostgreSQL lock conflicts section).
   * Terminate only with customer approval:
   ```sql
   SELECT pg_terminate_backend(<pid>);
   ```

5. **Branch: replication lag** (`PostgresqlReplicationLag`)
   ```sql
   SELECT * FROM pg_stat_replication;
   ```
   Interpret **sent_lsn**, **write_lsn**, **flush_lsn**, **replay_lsn** (network vs replica disk vs replay blocked).
   * With slots:
   ```sql
   SELECT a.client_addr, b.slot_name, a.state,
     pg_current_wal_lsn() AS current_wal, a.replay_lsn, a.replay_lag AS lag_in_time,
     pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), a.replay_lsn)) AS lag_in_size
   FROM pg_stat_replication a
   JOIN pg_replication_slots b ON a.pid = b.active_pid
   ORDER BY pg_wal_lsn_diff(pg_current_wal_lsn(), a.replay_lsn) DESC;
   ```
   * Blocking on primary:
   ```sql
   SELECT activity.pid, activity.usename, activity.query,
     blocking.pid AS blocking_id, blocking.query AS blocking_query
   FROM pg_stat_activity activity
   JOIN pg_stat_activity blocking ON blocking.pid = ANY (pg_blocking_pids(activity.pid));
   ```
   * Long queries:
   ```sql
   SELECT now() - query_start AS age, *
   FROM pg_stat_activity
   WHERE now() - query_start > interval '1 minute'
     AND state IN ('idle in transaction', 'active');
   ```

6. **Branch: too many connections** (`PostgresqlTooManyConnections`)
   ```sql
   SHOW max_connections;
   SELECT count(*) AS total_connections FROM pg_stat_activity;
   SELECT usename, application_name, client_addr, count(*)
   FROM pg_stat_activity
   GROUP BY 1, 2, 3 ORDER BY 4 DESC;
   ```
   Idle / idle-in-transaction:
   ```sql
   SELECT pid, usename, state, query FROM pg_stat_activity
   WHERE state IN ('idle', 'idle in transaction') ORDER BY state, pid;
   ```
   Terminate idle (example policy — adjust with customer):
   ```sql
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE state = 'idle' AND query_start < now() - interval '5 minutes';
   ```
   Temporary guardrails (requires reload/restart per deployment):
   ```sql
   ALTER SYSTEM SET idle_in_transaction_session_timeout = '5min';
   ALTER SYSTEM SET statement_timeout = '30s';
   ```

7. **Branch: deadlocks** (`PostgresqlDeadLocks`)
   * Confirm rate in PMM; grep logs on host:
   ```bash
   grep -i deadlock /var/log/postgresql/postgresql*.log
   ```
   * Customer: application transaction ordering / lock escalation.

8. **Branch: commit rate low** (`PostgresqlCommitRateLow`)
   ```sql
   SELECT datname, xact_commit, xact_rollback,
     round((xact_rollback::numeric * 100) / NULLIF(xact_commit + xact_rollback, 0), 4) AS rollback_percent
   FROM pg_stat_database
   WHERE datname NOT IN ('template0','template1')
   ORDER BY rollback_percent DESC NULLS LAST;
   ```
   ```sql
   SELECT * FROM pg_stat_activity WHERE query ILIKE '%rollback%';
   ```

9. **Branch: exporter error** (`PostgresqlExporterError`)
   * Treat like monitoring path: PMM server container `/srv/logs/pmm-agent.log` or DB node `journalctl -u pmm-agent` — **operator only**; Holmes uses inventory + PMM logs tool if available.
   * After upgrades or RDS/Aurora: unsupported functions in exporter — escalate per platform runbook.

10. **Branch: archive failed** (`PostgresqlArchiveFailed`)
    * Check `archive_command` / `archive_library` status in logs; disk on archive destination; permissions.

11. **Branch: instance down** (`PostgresqlIsDown`)
    * Confirm `up` metric and process; **node-capacity-saturation** and **pmm-monitoring-health** may overlap; check **PostgresqlExporterError** vs real outage.

12. **Branch: wraparound** (`PostgresqlWraparound`)
    * Vacuum progress, oldest xmin, long transactions blocking vacuum — operator SQL from internal wraparound playbook; escalate urgently.

13. **Slow queries**
    * For workload/spike-dominated investigations use the **general** skill (workload / time-window analysis). For per-query plan, index, and rewrite analysis use the **postgresql-query-analysis** skill — QAN top statements from **pmm.metrics** (filter by **service_id**) plus EXPLAIN via `psql`.

## Synthesize Findings

Tie alert to: resource, replication stage, lock graph, connection pool leak, archive path, or monitoring false positive. Note customer-visible risk (stale reads, rejects, data loss on failover).

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

* Locks: end blocker or tune app; add `lock_timeout`, `idle_in_transaction_session_timeout` where appropriate.
* Replication: fix network/disk; end long transactions; reschedule heavy jobs; tune `max_wal_size` / parallel apply only with analysis.
* Connections: pooler (PgBouncer), raise `max_connections` only with sizing review, kill idle sessions with approval.
* Archive: repair `archive_command`, destination space, credentials.
* Exporter: fix agent/config/version; exclude unsupported metrics on RDS if documented.
* Wraparound: emergency vacuum / kill blockers per PostgreSQL procedures — include exact `VACUUM (FREEZE, ANALYZE, VERBOSE)` only when appropriate to environment.
