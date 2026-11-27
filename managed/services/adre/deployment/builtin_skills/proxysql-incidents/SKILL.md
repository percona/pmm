---
name: proxysql-incidents
description: pmm_proxysql_server_status. ProxySQL backend OFFLINE_SOFT/OFFLINE_HARD or shunned in a hostgroup, hostgroup with no ONLINE server ("Max connect timeout reached while reaching hostgroup"), ProxySQL down, backend connection errors.
---

# ProxySQL incidents

## When to use
The alert above, ProxySQL clients failing or hanging, ProxySQL service down in PMM. HAProxy → **haproxy-incidents**. The root cause is often the backend: hand off to **mysql-availability**, **mysql-replication-topology** or **mysql-pxc-galera** once the backend is identified.

## Steps
1. **Scope.** Alert labels: service_name (the ProxySQL service), node_name, hostgroup, endpoint (backend host:port). `pmm_list_services(service_type=proxysql)` → service_id. Map the endpoint to a MySQL service by address and port in `pmm_list_services(service_type=mysql)`.
2. **ProxySQL itself.** `execute_prometheus_instant_query("proxysql_up{service_name=\"<svc>\"}")`: 0 = exporter cannot reach the ProxySQL admin interface (ProxySQL down or admin credentials); no series = pmm-agent/host problem (**exporter-scrape-failures**).
3. **Dashboard** (no observability-map route for ProxySQL): `pmm_search_dashboards('ProxySQL')` → `proxysql-instance-summary`; `pmm_list_dashboard_panels(dashboard_uid="proxysql-instance-summary")`. Useful panels: 1029 Endpoint Statuses (timeline), 57 Endpoint Status, 52 Failed Backend Connections, 51 Active Backend Connections, 50 Endpoints Latency, 55 Client Connections. Render 1029 when the user should see when endpoints changed state.
4. **Snapshot** (`pmm_metrics_snapshot`, alert window ±1h, `{service_name="<svc>"}`):
   - `proxysql_runtime_servers_status`: per endpoint/hostgroup; when each left 1 and whether any endpoint in the hostgroup stayed ONLINE.
   - `rate(proxysql_connection_pool_conn_err[5m])`: backend connect errors, usually rising **before** a shun.
   - `proxysql_connection_pool_latency_us`: monitor ping latency to each backend.
   - `proxysql_connection_pool_conn_used` and `proxysql_connection_pool_conn_free`: pool exhaustion.
   - `proxysql_mysql_status_client_connections_connected`: frontend connection spike.
5. **Backend health** for the endpoint's MySQL service: `mysql_up`, `mysql_slave_status_seconds_behind_master`, `mysql_global_variables_read_only`; for PXC backends `mysql_global_status_wsrep_local_state` and `mysql_global_status_wsrep_ready` (all `{service_name="<backend>"}`).
6. **ProxySQL log** (`pmm_clickhouse_query`, database=otel): `ResourceAttributes['node_name'] = '<proxysql_node>'` AND `LogAttributes['log.file.name'] LIKE '%proxysql%'`, ORDER BY Timestamp DESC LIMIT 50. Look for "Shunning server", MySQL_Monitor errors, read_only / replication lag / Galera state changes.
7. **For the user** (ProxySQL admin, port 6032):
   ```sql
   SELECT hostgroup_id, hostname, port, status, weight, max_connections, max_replication_lag FROM runtime_mysql_servers ORDER BY hostgroup_id;
   SELECT hostgroup, srv_host, srv_port, status, ConnUsed, ConnFree, ConnERR, Latency_us FROM stats_mysql_connection_pool;
   SELECT * FROM monitor.mysql_server_connect_log ORDER BY time_start_us DESC LIMIT 10;
   SELECT * FROM monitor.mysql_server_ping_log ORDER BY time_start_us DESC LIMIT 10;
   SELECT * FROM monitor.mysql_server_read_only_log ORDER BY time_start_us DESC LIMIT 10;
   SELECT * FROM monitor.mysql_server_replication_lag_log ORDER BY time_start_us DESC LIMIT 10;
   SELECT * FROM runtime_mysql_replication_hostgroups;  -- or runtime_mysql_galera_hostgroups / runtime_mysql_group_replication_hostgroups
   ```

## Interpretation
- `proxysql_runtime_servers_status`: 1 ONLINE, 2 SHUNNED, 3 OFFLINE_SOFT, 4 OFFLINE_HARD, 5 SHUNNED_REPLICATION_LAG. The alert fires at status ≥ threshold (default 4, so OFFLINE_HARD and SHUNNED_REPLICATION_LAG; 3 adds OFFLINE_SOFT) for 5m. Plain SHUNNED never fires it.
- OFFLINE_HARD: set by an admin, or by Galera / Group Replication hostgroup monitoring when the node is not Synced/ONLINE, wsrep_ready=0 or non-Primary.
- OFFLINE_SOFT: graceful drain (manual or maintenance script); existing connections finish, no new ones.
- SHUNNED_REPLICATION_LAG: replica lag above `max_replication_lag` for that server.
- SHUNNED: connect/ping errors exceeded `mysql-shun_on_failures`: backend unreachable, out of max_connections, or monitor/app credentials mismatch.
- Every endpoint of a hostgroup non-ONLINE = that hostgroup is dead; clients get "Max connect timeout reached while reaching hostgroup N".
- Healthy backend metrics + growing conn_err = credentials or network between ProxySQL and the backend.

## Remediation
Run with approval in the ProxySQL admin interface. On PXC Operator (k8s) deployments ProxySQL config comes from the CR; change it there.
- **Fix the backend first** (down, lagging, desynced) with the matching skill; ProxySQL brings it back automatically once monitor checks pass.
- **Manually offlined server, backend healthy**:
  ```sql
  UPDATE mysql_servers SET status='ONLINE' WHERE hostgroup_id=<hg> AND hostname='<host>' AND port=<port>;
  LOAD MYSQL SERVERS TO RUNTIME; SAVE MYSQL SERVERS TO DISK;
  ```
- **Lag shunning**: fix replica lag, or raise the tolerance: `UPDATE mysql_servers SET max_replication_lag=<s> WHERE hostname='<host>' AND port=<port>;` then LOAD/SAVE as above.
- **Monitor credentials**: create/align the monitor user on backends, then
  ```sql
  UPDATE global_variables SET variable_value='<pwd>' WHERE variable_name='mysql-monitor_password';
  LOAD MYSQL VARIABLES TO RUNTIME; SAVE MYSQL VARIABLES TO DISK;
  ```
- **ProxySQL down** — self-managed: `sudo systemctl status proxysql`, `sudo journalctl -u proxysql --since "-1h"`, `sudo systemctl start proxysql`; k8s: `kubectl logs <proxysql-pod> -c proxysql`.
- **Resilience**: keep at least two servers per reader hostgroup and a backup writer, so one backend change cannot empty a hostgroup.

## Related skills
mysql-availability, mysql-replication-topology, mysql-pxc-galera.
