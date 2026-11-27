---
name: mysql-replication-topology
description: pmm_mysql_replication_io_running, pmm_mysql_replication_sql_running. MySQL replica IO/SQL thread stopped, replication errors, replica lag, writable replica (read_only), errant GTID, Group Replication member not ONLINE.
---

# MySQL replication (async and Group Replication)

## When to use
The alerts above, or replica lag / broken replication / failover readiness questions. PXC/Galera → **mysql-pxc-galera**; replica or source host down → **mysql-availability**.

## Steps
1. **Topology.** `pmm_list_firing_alerts()`; map the alert's service_name/instance to a service with `pmm_list_services(service_type=mysql)`. Peers share the `replication_set` or `cluster` label. `pmm_discover_series_labels(service_id=<replica>, metric_prefix="mysql_slave_status_", start=<t>)` shows the channel and source labels.
2. **Replica metrics.** `pmm_observability_map(engine=mysql, intent=replication, service_id=<replica>)` (IO/SQL thread, lag, binlog panels). `pmm_metrics_snapshot` over the alert window ±1h, all `{service_name="<replica>"}`:
   - `mysql_slave_status_replica_io_running` / `mysql_slave_status_replica_sql_running` (older servers: `..._slave_io_running` / `..._slave_sql_running`): first 0 = when it broke.
   - `mysql_slave_status_last_errno`: error code of the stopped thread.
   - `mysql_slave_status_seconds_behind_master - mysql_slave_status_sql_delay` (or `..._seconds_behind_source`): real lag, excluding intentional delay.
   - `mysql_slave_status_relay_log_space`: grows while the SQL thread lags or is stopped.
   - `mysql_global_variables_read_only`: 1 expected on every replica, 0 on the source.
   On the source: `increase(mysql_binlog_size_bytes{service_name="<source>"}[1h])`: write bursts that line up with lag.
3. **Group Replication** (when `mysql_perf_schema_replication_group_member_info` exists for the service): `pmm_observability_map(engine=mysql, intent=group_replication, service_id=<id>)`; snapshot `mysql_perf_schema_replication_group_member_info` (labels member_state, member_role) and `mysql_perf_schema_replication_group_worker_lag_in_seconds`.
4. **Error log** on the replica node (`pmm_clickhouse_query`, database=otel): `ResourceAttributes['node_name'] = '<node>'` AND `(Body ILIKE '%replica%' OR Body ILIKE '%slave%' OR Body ILIKE '%relay%')`, ORDER BY Timestamp DESC LIMIT 50. Quote the exact error text.
5. **What PMM cannot see** (give the user to run): `SHOW REPLICA STATUS\G` (`SHOW SLAVE STATUS\G` before 8.0.22): Last_IO_Error, Last_SQL_Error, Retrieved/Executed_Gtid_Set; multi-threaded failures: `SELECT * FROM performance_schema.replication_applier_status_by_worker\G`. Errant GTIDs: `SELECT @@GLOBAL.gtid_executed;` on source and replica, then `SELECT GTID_SUBTRACT('<replica_set>', '<source_set>');`.

## Interpretation
- Alerts fire when a source is configured (`mysql_slave_status_source_server_id > 0` or `..._master_server_id > 0`) and the IO or SQL thread metric is 0 for 1m.
- IO thread errors: 1045 wrong replication credentials; 2003/2005 network, DNS or source down; 2061 caching_sha2_password needs SSL or GET_SOURCE_PUBLIC_KEY; 1236/13114 source already purged needed binlogs/GTIDs → reseed.
- SQL thread errors: 1062 duplicate key / 1032 row not found = data drift (writes on the replica, skipped events, unsafe statements); 1146 missing table; 1205 lock wait timeout = a long query or backup holding locks on the replica.
- Lag: rises steadily with threads running = apply too slow (single-threaded apply, huge transactions, ROW events on tables **without a primary key**, replica CPU/IO saturated); tracks source binlog bursts = batch jobs on source; lag is meaningless while the IO thread is stopped.
- read_only=0 on a replica explains drift; read_only=1 on the source means writes fail (failover tool or manual change).
- Non-empty GTID_SUBTRACT = errant transactions; promoting that replica (or repointing others to it) breaks replication.
- Group Replication: any member_state other than ONLINE needs action; losing majority blocks writes on the whole group.

## Remediation
Run with approval. RDS/Aurora: use `CALL mysql.rds_skip_repl_error;`, `CALL mysql.rds_start_replication;` and the parameter group instead of SET GLOBAL. On Operator-managed clusters let the operator own read_only and topology.
- **IO thread**: fix network/credentials, then `CHANGE REPLICATION SOURCE TO SOURCE_PASSWORD='<pwd>', GET_SOURCE_PUBLIC_KEY=1;` (`CHANGE MASTER TO` before 8.0.23) and `START REPLICA;`. Purged binlogs: rebuild the replica (xtrabackup or `CLONE INSTANCE FROM ...` on 8.0.17+).
- **SQL thread 1062/1032**: fix the row on the replica to match the source (find it from Last_SQL_Error), `START REPLICA;`, then verify with `pt-table-checksum`. Skipping an event hides drift; with GTID:
  ```sql
  STOP REPLICA;
  SET GTID_NEXT = '<source_uuid>:<failing_seq>'; BEGIN; COMMIT; SET GTID_NEXT = 'AUTOMATIC';
  START REPLICA;
  ```
  Without GTID: `SET GLOBAL sql_replica_skip_counter = 1;` (`sql_slave_skip_counter` before 8.0.26).
- **Lag**:
  ```sql
  STOP REPLICA SQL_THREAD;
  SET GLOBAL replica_parallel_workers = 8;          -- 5.7: slave_parallel_workers + slave_parallel_type='LOGICAL_CLOCK'
  SET GLOBAL replica_preserve_commit_order = ON;
  START REPLICA SQL_THREAD;
  ```
  Add primary keys to tables that lack them (`ALTER TABLE <db>.<t> ADD PRIMARY KEY (<col>);`); split large DML on the source into batches (`pt-archiver` or `DELETE ... LIMIT 10000` loops); kill long queries blocking the applier. Catch-up only, on the replica, revert afterwards: `SET GLOBAL innodb_flush_log_at_trx_commit = 2; SET GLOBAL sync_binlog = 0;`.
- **Writable replica**: `SET PERSIST read_only = ON; SET PERSIST super_read_only = ON;` (my.cnf on 5.7); find the writer in the general/audit log or binlog of the replica.
- **Errant GTID**: if the transactions are not needed, inject one empty transaction per errant GTID on the **source** (same GTID_NEXT block) so every server agrees; if they changed data, decide which side is correct; many errant GTIDs → rebuild the replica.
- **Group Replication member in ERROR**: read its error log, then `STOP GROUP_REPLICATION; START GROUP_REPLICATION;` on that member. Majority lost: `SET GLOBAL group_replication_force_members = '<host:port>,...';` only per the documented procedure, DBA-led.

## Related skills
mysql-availability, mysql-query-analysis (slow statements on the source), node-capacity-saturation (saturated replica).
