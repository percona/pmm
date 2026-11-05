---
name: mysql-availability
description: pmm_mysql_down, pmm_mysql_restarted, pmm_mysql_too_many_connections, and pmm_agent_down on a node running MySQL. MySQL down, crashed or restarted, exporter cannot connect, connections near max_connections.
---

# MySQL availability, restarts and connection saturation

## When to use
The alerts above, or "MySQL is down / restarted / refuses connections". For pmm_agent_down, first run `pmm_list_services(node_id=<node_id>)`: no MySQL on that node → **exporter-scrape-failures**. Replication thread alerts → **mysql-replication-topology**; PXC non-Primary or desync → **mysql-pxc-galera**.

## Steps
1. **Scope.** `pmm_list_firing_alerts()` (note other alerts on the same node). `pmm_list_services(service_type=mysql)` → service_id, node_id, address, port. `pmm_list_agents(service_id=<id>)` → mysqld_exporter `status`; `pmm_list_agents(node_id=<node_id>)` → pmm-agent `connected`.
2. **Classify the outage** (`execute_prometheus_instant_query`):
   - `mysql_up{service_name="<svc>"}`: 1 up; 0 = exporter runs but cannot connect to mysqld; no series = nothing scraped.
   - `up{service_name="<svc>"}`: 0 = exporter not answering scrapes; no series at all = pmm-agent/vmagent or the host is gone (expect pmm_agent_down).
   - pmm-agent `connected=false` → `tcp_check` the MySQL address:port. Port open = MySQL alive, only monitoring is broken (→ **exporter-scrape-failures**); closed = host or mysqld down.
3. **Timeline.** `pmm_observability_map(engine=mysql, intent=availability, service_id=<id>)`, then `pmm_metrics_snapshot` over the alert window ±1h on:
   - `mysql_global_status_uptime{service_name="<svc>"}`: a drop to ~0 marks each restart (count them).
   - `node_memory_MemAvailable_bytes{node_name="<node>"}`: collapse right before the restart → OOM kill.
   - `node_filesystem_avail_bytes{node_name="<node>"}`: datadir/binlog mount near 0 → disk full.
   - `node_time_seconds{node_name="<node>"} - node_boot_time_seconds{node_name="<node>"}`: small = host reboot, not a mysqld crash.
4. **Error log** (`pmm_clickhouse_query`, database=otel; widen/narrow to the outage):
   ```sql
   SELECT Timestamp, SeverityText, Body FROM logs
   WHERE ResourceAttributes['node_name'] = '<node_name>'
     AND (LogAttributes['log.file.name'] LIKE '%error%' OR LogAttributes['log.file.name'] LIKE '%mysqld%'
          OR LogAttributes['log.file.name'] LIKE '%.err')
     AND Timestamp >= now() - INTERVAL 6 HOUR
   ORDER BY Timestamp DESC LIMIT 100
   ```
   0 rows → list the node's file names (`GROUP BY LogAttributes['log.file.name']`). Kernel OOM, if syslog is collected: `Body ILIKE '%out of memory%' OR Body ILIKE '%killed process%'`. Report every ERROR line around the stop **and** the next start attempt (a clean shutdown can precede a failed start).
5. **Connections branch** (pmm_mysql_too_many_connections, or mysql_up=0 with "Too many connections" in logs): `pmm_observability_map(engine=mysql, intent=connections, service_id=<id>)`; snapshot `mysql_global_status_threads_connected`, `mysql_global_status_threads_running`, `mysql_global_status_max_used_connections`, `mysql_global_variables_max_connections`, `rate(mysql_global_status_aborted_connects[5m])` (all `{service_name="<svc>"}`). If threads_running climbs too, pull top QAN queries for the window (**mysql-query-analysis**) and check `intent=locks`.

## Interpretation
- pmm_mysql_down fires when an enabled mysqld_exporter has no `mysql_up == 1` for 1m: mysqld down **or** monitoring broken. pmm_mysql_restarted: uptime < 300 s (default threshold). pmm_mysql_too_many_connections: 5-min max of Threads_connected > 80 % of max_connections for 5m. pmm_agent_down: pmm-agent not connected to PMM Server for 1m.
- `mysql_up=0` with the port open: exporter credentials (password rotated, account locked, the `pmm` user's MAX_USER_CONNECTIONS), or max_connections exhausted so the exporter gets no slot.
- Error log evidence → cause:
  | Evidence | Cause |
  |---|---|
  | log stops with no "Shutdown complete"; next start says "not shut down normally"/crash recovery; MemAvailable ~0 before | OOM kill |
  | `errno: 28`, "No space left on device", "Disk is full" | disk full |
  | `OS errno 13 - Permission denied`, "Can't create/write to file", binlog.index not found | datadir/binlog ownership, SELinux/AppArmor |
  | "unknown variable", "[ERROR] Aborting" right after start | bad my.cnf / startup option |
  | "Starting crash recovery", "Apply batch completed" | recovery in progress, wait |
  | "signal 11", "Assertion failure", stack trace, page corrupted | mysqld bug or corruption |
  | "Normal shutdown" … "Shutdown complete" | planned stop; find who/what restarted it |
- Connections: high connected + low running = idle or leaked connections, oversized pools, no wait_timeout; high connected + high running = pileup behind slow queries or locks; a step change = deploy, failover moving traffic, proxy multiplexing lost. Rising aborted_connects = auth failures or client timeouts.

## Remediation
- **OOM** — self-managed: confirm with `sudo dmesg -T | grep -iE 'out of memory|killed process'`; lower `innodb_buffer_pool_size` (`SET GLOBAL innodb_buffer_pool_size = <bytes>;` resizes online, persist in my.cnf), cut max_connections or per-thread buffers, move co-located processes. k8s: `kubectl describe pod <pod>` shows `OOMKilled` → raise the memory limit / lower buffer pool in the Operator CR. RDS/Aurora: parameter group or larger instance class.
- **Disk full** — self-managed: `df -h <datadir>`, `sudo du -xh <datadir> | sort -h | tail -20`; grow the volume or remove non-MySQL files; never delete ibdata*, redo/undo or .ibd files. Once up: `PURGE BINARY LOGS BEFORE NOW() - INTERVAL 3 DAY;` (only after replicas have read them) and set `binlog_expire_logs_seconds`. RDS: increase allocated storage / enable storage autoscaling.
- **Permissions** — self-managed: `sudo chown -R mysql:mysql <datadir> <binlog_dir>`; SELinux: `ls -lZ <datadir>`, `sudo restorecon -Rv <datadir>`; then `sudo systemctl start mysqld`.
- **Bad config** — self-managed: `mysqld --validate-config` (8.0.16+), fix my.cnf, start. k8s: fix the CR; RDS: parameter group.
- **Crash recovery / corruption** — do not kill mysqld during recovery. Corruption: restore from backup or rebuild the replica; `innodb_force_recovery` only to dump data, DBA-led.
- **Monitoring only** (port open, mysql_up=0): reset the exporter account, e.g. `ALTER USER 'pmm'@'<host>' IDENTIFIED BY '<password PMM uses>' ACCOUNT UNLOCK;`; pmm-agent problems → **exporter-scrape-failures**.
- **Too many connections** (with approval):
  ```sql
  SELECT user, host, db, command, COUNT(*) AS n, MAX(time) AS max_s
  FROM information_schema.processlist GROUP BY user, host, db, command ORDER BY n DESC LIMIT 20;
  KILL <id>;                          -- RDS: CALL mysql.rds_kill(<id>);
  SET GLOBAL max_connections = <n>;   -- check node memory first; persist via SET PERSIST / my.cnf / CR / parameter group
  SET GLOBAL wait_timeout = 600;      -- closes idle sessions
  ALTER USER '<app>'@'<host>' WITH MAX_USER_CONNECTIONS <n>;
  ```
  Long-term: right-size application pools or put ProxySQL in front; fix the slow queries behind a pileup.

## Related skills
mysql-query-analysis, node-capacity-saturation, exporter-scrape-failures.
