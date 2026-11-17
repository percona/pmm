---
name: haproxy-incidents
description: HAProxy services monitored by PMM (no alert template). Backend or server DOWN, failing or flapping health checks, 5xx / connection / response errors, requests queueing, sessions at maxconn limits, HAProxy restarts or scrape failures.
---

# HAProxy incidents

## When to use
An HAProxy service (service_type haproxy) misbehaves: clients get 503 / connection resets, a backend shows DOWN, latency added by queueing. ProxySQL → **proxysql-incidents**. Once a backend database is identified as the cause, hand off to its skill (**mysql-availability**, **mysql-pxc-galera**, …).

## Steps
1. **Scope.** `pmm_list_services(service_type=haproxy)` → service_id, service_name, node_id. `pmm_list_agents(service_id=<id>)`: PMM scrapes HAProxy's built-in Prometheus endpoint through an external-exporter agent; check its status.
2. **Reachability and restarts** (`execute_prometheus_instant_query`): `up{service_name="<svc>"}` 0 = HAProxy or its metrics frontend is unreachable; `time() - haproxy_process_start_time_seconds{service_name="<svc>"}` = seconds since the last start or reload.
3. **Dashboard** (no observability-map route): `pmm_search_dashboards('HAProxy')` → `haproxy-instance-summary`; `pmm_list_dashboard_panels(dashboard_uid="haproxy-instance-summary")`. Useful panels: 1008 Backend Sections Status, 1400 Servers in Section (state timeline), 1017 Active Servers, 1378 Checks Statistic, 1080 Queued Requests, 1265 Queue Time, 1206 Session Limit, 1275 Connection Errors, 1092 Response Errors, 1044 HTTP Responses by Code. Render 1400 when the user should see when servers went down.
4. **Snapshot** (`pmm_metrics_snapshot`, incident window ±1h, `{service_name="<svc>"}` in every selector):
   - State: `haproxy_backend_status`, `haproxy_server_status` (labels proxy, server); `increase(haproxy_server_check_failures_total[1h])`, `increase(haproxy_server_check_up_down_total[1h])`.
   - Errors: `rate(haproxy_backend_connection_errors_total[5m])`, `rate(haproxy_backend_response_errors_total[5m])`, `rate(haproxy_frontend_request_errors_total[5m])`, `rate(haproxy_server_retry_warnings_total[5m])`, `rate(haproxy_backend_redispatch_warnings_total[5m])`; HTTP mode: `sum by (code) (rate(haproxy_backend_http_responses_total[5m]))`.
   - Queueing: `haproxy_backend_current_queue`, `haproxy_server_current_queue`, `haproxy_backend_queue_time_average_seconds`.
   - Limits: `haproxy_frontend_current_sessions` vs `haproxy_frontend_limit_sessions`; `haproxy_backend_current_sessions` vs `haproxy_backend_limit_sessions`; `haproxy_process_max_connections` (global maxconn); `rate(haproxy_frontend_denied_connections_total[5m])`.
   - Aborts: `rate(haproxy_backend_client_aborts_total[5m])` vs `rate(haproxy_backend_server_aborts_total[5m])`.
5. **Backends.** Map `server` labels to PMM database services by address (`pmm_list_services`) and check them (`mysql_up`, `mysql_global_status_wsrep_local_state` for PXC, `postgresql_up`, …).
6. **HAProxy log** (`pmm_clickhouse_query`, database=otel): `ResourceAttributes['node_name'] = '<node>'` AND `LogAttributes['log.file.name'] LIKE '%haproxy%'`, ORDER BY Timestamp DESC LIMIT 50. Look for "Server <be>/<srv> is DOWN, reason: …", "backend <be> has no server available!", and the two-letter termination state on request lines.
7. **For the user** (self-managed, needs a `stats socket`): `echo "show servers state" | sudo socat stdio /run/haproxy/admin.sock`; `echo "show info" | sudo socat stdio /run/haproxy/admin.sock` (CurrConns vs Maxconn).

## Interpretation
- Status values as mapped by the PMM dashboard: server/backend 1 = UP, 0 = DOWN; frontend 0 STOP, 1 UP, 2 FULL. If the series carry a `state` label (newer HAProxy), the current state is the one whose value is 1.
- Backend DOWN = no server left: HTTP clients get 503, TCP clients are closed immediately. Health-check reasons: Layer4 connection problem = port closed (DB down, firewall); Layer4 timeout = network or overloaded server; Layer7 wrong status = the check endpoint says unhealthy (e.g. PXC clustercheck: node not Synced, mysql-check user missing).
- check_up_down climbing fast = flapping: checks too aggressive for the load, `fall`/`rise` too low, or a borderline backend.
- Queue > 0 = every server is at its `maxconn`; requests wait (queue time) and fail with 503 after `timeout queue`. Root cause is usually a slow backend, not HAProxy.
- current_sessions at limit_sessions (frontend maxconn) or at global maxconn = new connections wait or are refused; denied_connections = ACL or rate-limit rules.
- connection_errors = HAProxy cannot connect to servers; response_errors = server reset or closed mid-response; 5xx > 1–5 % of responses = error-rate incident; client aborts high = client timeouts shorter than backend latency.
- Termination states: `sC` server connect timeout, `SC` connection refused by server, `SH` server aborted before headers, `sQ` expired in queue, `cD` client data timeout.
- Uptime reset = restart or reload (counters reset too; compare rates, not totals, across it).

## Remediation
Self-managed host commands below. k8s (PXC Operator HAProxy): change `spec.haproxy` / its configuration in the CR. Run with approval.
- **Config / restart**: `sudo haproxy -c -f /etc/haproxy/haproxy.cfg && sudo systemctl reload haproxy`; status: `sudo systemctl status haproxy`, `sudo journalctl -u haproxy --since "-1h"`.
- **Server left in MAINT/DRAIN**: `echo "set server <backend>/<server> state ready" | sudo socat stdio /run/haproxy/admin.sock`.
- **Backend really down**: fix the database first. PXC clustercheck: `curl -i http://<db_host>:9200/` must return 200 "Percona XtraDB Cluster Node is synced".
- **Flapping checks**: relax `server <name> <ip>:<port> check inter 2s fall 3 rise 2`.
- **Queueing / limits**: speed up the backend (slow queries, saturation); raise per-server `maxconn` only within the database's max_connections; raise frontend/global `maxconn` together with `ulimit-n`; tune `timeout queue` / `timeout connect`; add backend servers.
- **Client aborts**: align client and HAProxy `timeout client` / `timeout server` with real backend latency.

## Related skills
mysql-availability, mysql-pxc-galera, node-capacity-saturation (HAProxy host CPU / network).
