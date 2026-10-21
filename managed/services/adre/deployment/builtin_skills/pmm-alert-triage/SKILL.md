---
name: pmm-alert-triage
description: "Entry point for firing alerts (pmm_agent_down, pmm_backup_error, pmm_mysql_*, pmm_postgresql_*, pmm_mongodb_*, pmm_node_*, pmm_proxysql_server_status, pmm_valkey_down, pmm_redis_down) and 'what is wrong': list, group, confirm still true, route each alert to its skill."
---

# Alert triage and routing

## When to use
The user asks which alerts fire, what is wrong, or names an alert without more context. Once the alert is known, fetch the skill from the table and continue there.

## Steps
1. `pmm_list_firing_alerts()`. Per alert read `labels.template_name` (the pmm_* template; `alertname` is the rule title the user typed and may differ), `severity`, `service_name`, `node_name`, plus `annotations.summary`/`description` and `startsAt`. An `Error` annotation means the rule failed to evaluate (query or datasource error), not that the condition is true: treat it as a PMM problem (pmm-server-health).
2. **Group before routing.** Several alerts on one node: node or pmm-agent first (pmm_agent_down, pmm_node_*). Several `*_down` on services served by one pmm-agent (same `node_name`, or remote services on `pmm-server`): one agent/network problem, not N outages. Order: critical first, then earliest `startsAt` (likely root cause).
3. **Confirm it is still true** with `execute_prometheus_instant_query`, scoped by `service_name` / `node_name`, using the check in the table. Not confirmed: resolved or flapping; say so with `startsAt` and the rule's `for`.
4. `fetch_skill` for the 1-2 alerts that matter; do not fetch one skill per alert when they share a root cause.
5. No `template_name` label: a custom rule. Route by the metric names in its annotations. `list_prometheus_rules` shows only vmalert rule files (`/srv/prometheus/rules`), not the Grafana-managed rules PMM creates from templates.
6. Nothing firing: say so. PMM ships templates but creates no rules by default; rules exist only if someone created them (Alerting > Alert rule templates).

## Routing table
| template_name | Check (instant query) | Skill |
|---|---|---|
| pmm_agent_down | `pmm_managed_inventory_agents{agent_type="pmm-agent",node_name="N"}` 1 connected, 0 not | exporter-scrape-failures |
| pmm_mysql_down | `mysql_up{service_name="S"}`; series missing and `up{service_name="S"}` 0/missing = not scraped | mysql-availability (not scraped: exporter-scrape-failures) |
| pmm_mysql_restarted | `mysql_global_status_uptime` < 300 | mysql-availability |
| pmm_mysql_too_many_connections | `mysql_global_status_threads_connected / mysql_global_variables_max_connections * 100` > 80 | mysql-availability |
| pmm_mysql_replication_io_running, pmm_mysql_replication_sql_running | `mysql_slave_status_replica_io_running` / `mysql_slave_status_replica_sql_running` (`slave_*` on old versions) == 0 | mysql-replication-topology |
| pmm_postgresql_down | `pg_up` (same not-scraped rule) | postgresql-incidents |
| pmm_postgresql_restarted, _too_many_connections, _high_statement_timeouts, _high_transaction_rollbacks, _unused_replication_slots | `pg_postmaster_uptime_seconds`; `sum(pg_stat_activity_count)/pg_settings_max_connections`; `rate(postgresql_errors_total{type="statement_timeout"}[5m])`; `pg_stat_database_xact_rollback` vs `_xact_commit` rate; `pg_replication_slots_active == 0` | postgresql-incidents |
| pmm_postgresql_high_table_bloat, _high_index_bloat, _high_number_dead_tuples, _not_autovacuumed, _not_autoanalyzed | `pg_bloat_table_bloat_pct`; `pg_bloat_btree_bloat_pct`; `pg_stat_user_tables_n_dead_tup`; `time() - pg_stat_user_tables_last_autovacuum` / `_last_autoanalyze` | postgresql-maintenance |
| pmm_mongodb_down, _restarted | `mongodb_up`; `mongodb_instance_uptime_seconds` | mongodb-incidents |
| pmm_mongodb_replset_no_primary, _replset_primary_changed, _unusual_state, _replication_lag, _oplog_window | `mongodb_ss_repl_isWritablePrimary`; `changes(mongodb_mongod_replset_my_state[10m])`; `mongodb_rs_members_state`; `mongodb_mongod_replset_member_replication_lag`; `mongodb_mongod_replset_oplog_head_timestamp - mongodb_mongod_replset_oplog_tail_timestamp` | mongodb-incidents |
| pmm_mongodb_read_tickets, _write_tickets | `mongodb_mongod_wiredtiger_concurrent_transactions_available_tickets{txn_rw="read"\|"write"}` < 50 | mongodb-incidents, then mongodb-query-analysis |
| pmm_mongodb_dbpath_disk_space, _host_ssl_cert_expiry, _too_many_chunk_migrations, _cve_2025_14847_zlib | `mongodb_dbstats_fsUsedSize / mongodb_dbstats_fsTotalSize`; `mongodb_ss_security_SSLServerCertificateExpirationDate`; `mongodb_mongos_sharding_changelog_10min_total{event="moveChunk.start"}`; `mongodb_version_info` | mongodb-incidents |
| pmm_mongodb_pbm_agent_down, _pbm_backup_failed, _pbm_backup_stale, _pbm_backup_duration, _pbm_backup_size | `mongodb_pbm_agent_status{self="1"}` > 0; `mongodb_pbm_backup_size_bytes{status="error"}`; `mongodb_pbm_backup_last_transition_ts{status="done"}` | pmm-backup-management |
| pmm_backup_error | `pmm_managed_backups_artifacts{status="error"}` | pmm-backup-management |
| pmm_node_high_cpu_load, _low_free_memory, _swap_filled_up | `rate(node_cpu_seconds_total{mode="idle"}[5m])`; `node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes`; `node_memory_SwapFree_bytes / node_memory_SwapTotal_bytes` | node-capacity-saturation |
| pmm_proxysql_server_status | `proxysql_runtime_servers_status` >= 3 (3 OFFLINE_SOFT, 4 OFFLINE_HARD) | proxysql-incidents |
| pmm_valkey_down, pmm_redis_down | `redis_up` (same not-scraped rule) | valkey-redis-incidents |

## Interpretation
- `*_down` templates fire when the exporter is enabled in inventory but `*_up` is 0 or missing, so "down" can mean "not scraped". `up{service_name="S"}` 1 with `*_up` 0: the database refuses the exporter (really down, auth, max connections). `up` 0 or missing: exporter or agent path (exporter-scrape-failures).
- `pmm_backup_error` and `pmm_mongodb_pbm_backup_failed` keep firing while the failed artifact exists, even after the next backup succeeded.
- `*_restarted` clears by itself after 5 min of uptime; the question is why it restarted (logs, OOM, failover).
- Services on RDS/Azure have no node_exporter, so pmm_node_* never fires for them; see cloud-rds-azure.

## Remediation
None here: each routed skill gives exact fixes. To mute noise during work, the user can add a silence in Alerting; changing a threshold means editing the rule's parameters, not the template.
