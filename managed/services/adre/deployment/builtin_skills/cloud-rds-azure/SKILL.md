---
name: cloud-rds-azure
description: "Amazon RDS / Aurora and Azure Database services monitored remotely (rds_exporter, azure_database_exporter): which host metrics exist and which do not, Enhanced Monitoring vs CloudWatch, QAN via performance_schema / pg_stat_statements, fixes via parameter groups and the provider console."
---

# Cloud databases: Amazon RDS / Aurora, Azure

## When to use
The node type is `remote_rds` or `remote_azure_database` (`pmm_list_nodes` groups), or the user mentions RDS, Aurora or Azure. Use it next to the engine skill: this one says what data exists and where fixes go. Other DBaaS (Cloud SQL and similar) are plain remote services: database metrics and QAN only, no host metrics at all.

## What PMM has
- Exporters run on a pmm-agent elsewhere, normally PMM Server's own (`pmm_agent_id` `pmm-server`), always in pull mode. Database metrics (mysqld_exporter / postgres_exporter) are the same as self-managed: engine skills apply unchanged.
- No node_exporter: `node_cpu_seconds_total`, `node_memory_*`, `node_filesystem_*`, `node_disk_*`, `node_load1` do not exist for these nodes, and pmm_node_* alerts never fire for them. Host signals instead (names from PMM dashboards):

| Source | Metrics |
|---|---|
| RDS CloudWatch ("basic", `/basic`, low resolution) | `aws_rds_read_latency_average`, `aws_rds_write_latency_average` (ms), `aws_rds_cpu_credit_usage_average` (burstable classes), `aws_rds_bin_log_disk_usage_average` |
| RDS Enhanced Monitoring (`/enhanced`, medium resolution) | `rdsosmetrics_cpuUtilization_wait`, `rdsosmetrics_memory_total/free/cached/buffers`, `rdsosmetrics_diskIO_readIOsPS/writeIOsPS/readLatency/writeLatency/diskQueueDepth/util/readThroughput/writeThroughput`, `rdsosmetrics_network_rx/tx`, `rdsosmetrics_tasks_running/blocked`, `rdsosmetrics_General_numVCPUs` |
| RDS CPU | `node_cpu_average` by `mode` (`total` = overall, plus user, system, wait, steal...) |
| Azure Monitor | `node_cpu_average{node_type="remote_azure_database"}`, `azure_memory_percent_average`, `azure_storage_percent_average`, `azure_storage_used_bytes_average`, `azure_resource_info` |

- RDS free storage has no dashboard metric: try `pmm_list_metric_names(prefix="aws_rds_")` and `(prefix="rdsosmetrics_fileSys")`; if absent, the user reads CloudWatch FreeStorageSpace.
- Aurora MySQL counters: dashboard `mysql-amazonaurora` (`mysql_global_status_auroradb_*`).
- QAN: MySQL from performance_schema (the slow log is not readable remotely), PostgreSQL from pg_stat_statements. `pmm.metrics` queries and `pmm_db_action` MySQL explain/show actions work as for any service.
- Logs: database error and slow logs stay at the provider; `otel.logs` has none for these nodes.

## Steps
1. `pmm_get_node(node_id)`: `node_type`, `region`, `az`, `node_model` (instance class). `pmm_list_agents(service_id)` and `(node_id)`: exporter `status`, rds_exporter `basic_metrics_disabled` / `enhanced_metrics_disabled`, and whether a QAN agent exists.
2. Database-level analysis: the engine skill (connections, QAN, replication, locks).
3. Host-level: `pmm_observability_map(engine="node", intent="cpu_memory" or "disk_io", ...)`; PMM node panels fall back to the cloud metrics above. `pmm_metrics_snapshot` on `node_cpu_average{node_name="N",mode="total"}`, disk latency and, for t-classes, credit usage.
4. Is Enhanced Monitoring flowing? `pmm_list_metric_names(prefix="rdsosmetrics_")` empty: off on the instance, disabled in PMM, or IAM cannot read the RDSOSMetrics log group. Exporter health: `up{job=~"rds_exporter.*"}`, `rds_exporter_scrape_duration_seconds`, `rds_exporter_requests_total` (dashboard `prometheus-status`, RDS rows); Azure `up{agent_type="azure_database_exporter"}`.

## Interpretation
- Burstable classes (db.t3, db.t4g): credit usage climbing while CPU sits flat at a plateau suggests exhausted credits and throttling; the user confirms CPUCreditBalance in CloudWatch.
- Sustained disk latency above ~10 ms with a deep queue (`rdsosmetrics_diskIO_diskQueueDepth`) and IOPS flat at one level: storage IOPS/throughput limit (gp2 burst, gp3/io1 provisioned value). High `rdsosmetrics_cpuUtilization_wait`: I/O-bound.
- Azure storage percent > 90: close to full; Azure servers can go read-only when storage runs out.
- CloudWatch data arrives minutes late and at 1-minute or coarser steps: empty recent minutes are not an outage.
- All cloud metrics missing while DB metrics flow: exporter credentials (AWS keys, Azure client secret expiry) or permissions, not the database.

## Remediation (managed cloud: no host shell, no superuser, no config files)
- Settings: RDS parameter group / Azure server parameters, then apply (static ones need a reboot). QAN prerequisites: `performance_schema = 1` (static); PostgreSQL `pg_stat_statements` in `shared_preload_libraries` then `CREATE EXTENSION pg_stat_statements;` (Azure also `pg_stat_statements.track = all`).
- Capacity: change instance class, storage type, IOPS or enable storage autoscaling in the console, or `aws rds modify-db-instance --db-instance-identifier <id> --db-instance-class <class> --apply-immediately`.
- Enhanced Monitoring: enable it on the instance (Monitoring > Enhanced Monitoring, granularity) and give PMM's IAM user `rds:DescribeDBInstances`, `cloudwatch:GetMetricStatistics`, `cloudwatch:ListMetrics`, `logs:DescribeLogStreams`, `logs:GetLogEvents`, `logs:FilterLogEvents` on `RDSOSMetrics`. Azure: the app registration needs the Monitoring Reader role on the subscription.
- Ending sessions: RDS MySQL `CALL mysql.rds_kill(<processlist_id>);`; PostgreSQL `SELECT pg_terminate_backend(<pid>);` as a member of rds_superuser / azure_pg_admin (not for superuser-owned sessions).
- Logs: RDS console > Logs & events or CloudWatch Logs; Azure portal > Server logs. Never suggest SSH, journalctl, my.cnf or postgresql.conf edits.
