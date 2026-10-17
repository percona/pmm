---
name: postgresql-incidents
description: pmm_postgresql_down, pmm_postgresql_restarted, pmm_postgresql_too_many_connections, pmm_postgresql_high_statement_timeouts, pmm_postgresql_high_transaction_rollbacks; PostgreSQL outage or restart, connection exhaustion, blocking locks, deadlocks, replica lag, idle in transaction, WAL archiving.
---

# PostgreSQL incidents

## When to use
The alerts above, or symptoms: connection errors, sessions piling up behind locks, deadlocks, replica lag, WAL or archive growth.
Vacuum, bloat, dead tuples, wraparound, unused slots: use **postgresql-maintenance**. Slow statements and plans: use **postgresql-query-analysis**.

## Steps
1. Scope: `pmm_list_firing_alerts()` (labels, start time); `pmm_list_services(service_type=postgresql)` for `service_id`, `node_id`, address (tells self-managed / operator / cloud); `pmm_list_agents(service_id=<id>)` for `postgres_exporter` status.
2. Metrics: `pmm_observability_map(engine=postgresql, intent=<intent>, service_id=<id>)`, then `pmm_metrics_snapshot` on the panel exprs over the alert window. Add these exprs when the map does not cover them (scope each with `{service_id="<id>"}`):

| Alert / symptom | intent | Extra PromQL |
|---|---|---|
| down, restarted | availability | `pg_up`, `pg_postmaster_uptime_seconds`, `pg_replication_is_replica` (role flip = failover) |
| too_many_connections, idle in transaction | connections | `sum by (state) (pg_stat_activity_count)`, `pg_settings_max_connections`, `pg_stat_activity_max_state_duration{state="idle in transaction"}`, `pg_stat_activity_max_tx_duration` |
| rollbacks, statement timeouts | workload | `sum by (datname) (rate(pg_stat_database_xact_rollback[5m]))` vs `rate(pg_stat_database_xact_commit[5m])` |
| locks, deadlocks | locks | `pg_lock_conflicts_count` (by `blocking_pid`), `sum by (mode) (pg_locks_count)`, `rate(pg_stat_database_deadlocks[5m])` |
| replica lag | none: `pmm_list_dashboard_panels("postgresql-replication-overview")` | replica: `pg_replication_lag_seconds`, `pg_custom_stat_wal_receiver_lag_bytes`; primary: `pg_custom_stat_replication_replay_lag_seconds` (by `client_addr`); `rate(pg_stat_database_conflicts[5m])` |
| WAL, archiving | wal | `rate(pg_stat_archiver_failed_count[5m])`, `pg_stat_archiver_last_archive_age`, `pg_replication_slots_pg_wal_lsn_diff` (by `slot_name`), `rate(pg_stat_replication_pg_current_wal_lsn_bytes[5m])` |

3. Logs: `pmm_clickhouse_query(database=otel)`:
   ```sql
   SELECT Timestamp, SeverityText, Body FROM logs
   WHERE ResourceAttributes['node_name'] = '<node_name>' AND LogAttributes['log.file.name'] ILIKE '%postg%'
     AND Timestamp >= '<window start>' AND (SeverityText IN ('FATAL','PANIC','ERROR') OR Body ILIKE '%deadlock%' OR Body ILIKE '%timeout%' OR Body ILIKE '%archive%' OR Body ILIKE '%signal%')
   ORDER BY Timestamp DESC LIMIT 50
   ```
   No rows means PMM does not collect this node's PostgreSQL log; say so (managed cloud: provider log console).

## Interpretation
- **down**: exporter enabled but `pg_up` is not 1 for 1m; it also fires when the exporter stops reporting. Exporter `RUNNING` with `pg_up=0` means PostgreSQL is unreachable or rejects the exporter (pg_hba, password, SSL, `too many clients`). Exporter not running is a monitoring fault: use **exporter-scrape-failures**.
- **restarted**: `pg_postmaster_uptime_seconds` below 300 s (default). Ranked causes: OOM kill (`terminated by signal 9` in log, node memory drop), crash (`PANIC`), failover (`pg_replication_is_replica` changed), planned restart or upgrade.
- **too_many_connections**: sessions outside `postgres`/template DBs above 80% of `max_connections` for 5m. Mostly `idle`: oversized or leaking app pools. Mostly `idle in transaction`: the app does not commit (these also hold locks and block vacuum). Mostly `active` plus lock conflicts: a blocker or slow query causes a pile-up.
- **high_statement_timeouts**: the template reads `postgresql_errors_total{type="statement_timeout"}` (> 3/s). PMM's postgres_exporter does not produce this metric. Check `pmm_list_metric_names(prefix="postgresql_errors")`; if absent, count `canceling statement due to statement timeout` in the logs and take the statements to postgresql-query-analysis.
- **high_transaction_rollbacks**: rollbacks above 10% of commit+rollback per database over 3m. Causes: application errors (constraint violations, serialization failures, deadlocks, lock or statement timeouts) or frameworks that roll back read-only transactions. A long-standing ratio with no change point is usually harmless; a sudden jump points to a deploy or an error storm in the logs.
- **locks**: a `blocking_pid` blocking many sessions is the root, not its victims. A deadlock rate above 0 that persists means the application takes locks in inconsistent order. `pg_stat_database_conflicts` on a standby means recovery cancelled queries.
- **replica lag**: write lag means network; flush lag means replica disk; replay-only lag means replay is blocked by standby queries or is CPU-bound. Long transactions or bulk writes on the primary inflate all three.
- **archiving**: `failed_count` rising while `last_archive_age` grows means `archive_command` is failing. WAL then accumulates in `pg_wal` until the disk fills and the server stops.

## Remediation
Blocking tree (cancel first; terminate only with approval):
```sql
SELECT pid, usename, state, now() - xact_start AS xact_age, pg_blocking_pids(pid) AS blocked_by, left(query, 120) AS query
FROM pg_stat_activity WHERE cardinality(pg_blocking_pids(pid)) > 0 OR state = 'idle in transaction' ORDER BY xact_age DESC NULLS LAST;
SELECT pg_cancel_backend(<pid>);
SELECT pg_terminate_backend(<pid>);
```
Connections:
```sql
SELECT datname, usename, application_name, state, count(*) FROM pg_stat_activity GROUP BY 1,2,3,4 ORDER BY 5 DESC LIMIT 20;
```
Fix with a pooler (PgBouncer, transaction mode), smaller app pools, `idle_in_transaction_session_timeout = '5min'`, `idle_session_timeout` (PG14+), and per-role `ALTER ROLE <app> SET statement_timeout = '30s'` / `lock_timeout = '5s'`. Raising `max_connections` needs a restart and memory sizing (`work_mem` x sessions).
Settings: self-managed `ALTER SYSTEM SET ...; SELECT pg_reload_conf();`; Kubernetes: the Operator CR's PostgreSQL parameters; managed cloud: the parameter group.

Rollbacks: set `log_min_error_statement = error` to log the failing statements, then fix the application error path.

Replication (primary, then standby):
```sql
SELECT application_name, client_addr, state, sync_state, write_lag, flush_lag, replay_lag,
       pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS replay_gap FROM pg_stat_replication;
SELECT now() - pg_last_xact_replay_timestamp() AS replay_delay, pg_is_wal_replay_paused();
```
End long primary transactions, move bulk jobs off-peak, fix replica disk or network. For standby query cancellations, use `hot_standby_feedback = on` (adds primary bloat) or a larger `max_standby_streaming_delay`.

Archiving:
```sql
SELECT archived_count, last_archived_wal, last_archived_time, failed_count, last_failed_wal, last_failed_time FROM pg_stat_archiver;
```
Fix the archive destination (credentials, space, network) or the backup tool's config. Self-managed only: `df -h $PGDATA/pg_wal`. Never delete files from `pg_wal` by hand.

Down or restarted:
- Self-managed: `systemctl status 'postgresql*'`, `journalctl -u 'postgresql*' --since '-2h'`, `dmesg -T | grep -i -E 'oom|killed process'`.
- Kubernetes: `kubectl describe pod <pod>` and `kubectl logs <pod> --previous`.
- Managed cloud: the provider's event log.

## Related skills
postgresql-maintenance, postgresql-query-analysis, node-capacity-saturation (OOM, disk), exporter-scrape-failures (exporter or agent down).
