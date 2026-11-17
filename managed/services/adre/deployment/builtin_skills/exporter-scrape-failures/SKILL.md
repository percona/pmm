---
name: exporter-scrape-failures
description: "pmm_agent_down, and *_down alerts where the exporter is not scraped: missing data, gaps or 'No data' on dashboards, exporter down, scrape timeouts, pmm-agent disconnected from PMM Server, push (vmagent) vs pull metrics mode, vmagent queue drops."
---

# Exporter, scrape and agent connectivity failures

## When to use
pmm_agent_down; a `*_down` alert with `up` 0 or missing for the service; dashboards empty or gappy for one node or service. Exporter up but the database refuses it (`mysql_up` / `pg_up` / `mongodb_up` / `redis_up` 0 with `up` 1): the engine availability skill. Gaps for every node at once: pmm-server-health.

## Data path
- pmm-agent on the client keeps a gRPC stream to pmm-managed over the server's HTTPS port (443 by default) and starts exporters on local ports 42000-51999.
- Push mode (default for client agents): vmagent on the client scrapes the exporters and remote-writes to `https://<server>/victoriametrics/api/v1/write`, buffering up to 1 GB on disk while the server is unreachable; beyond that it drops data (permanent gaps).
- Pull mode (exporters run by PMM Server's own agent, i.e. remote, RDS and Azure services, or `--metrics-mode=pull`): VictoriaMetrics on the server scrapes the exporter port, so the server must reach the client's port.
- Scrape timeout = 0.9 x interval (resolutions default 5 s / 10 s / 60 s): slow collectors fail at the high-resolution interval first.

## Steps
1. Scope: `node_name` / `service_name` from the alert or user; ids from `pmm_list_nodes` / `pmm_list_services`.
2. `pmm_list_agents()` and find the `pmm_agent` entry whose `runs_on_node_id` is the node: `connected`. Then `pmm_list_agents(pmm_agent_id=<id>)`: per exporter `status`, `disabled`, `push_metrics_enabled`, `listen_port`; plus `vm_agent` status.
3. Instant queries:
   - `pmm_managed_inventory_agents{node_name="N"}`: pmm-agent 1 connected / 0 not; other agents carry the status code: 2 running, 1 starting, 3 waiting (crashed, restarting), 7 initialization error, 4 stopping, 5 done, 6 unknown (its pmm-agent is not connected).
   - `up{node_name="N"}` or `up{service_name="S"}` by `job`: 0 = the scrape failed; series missing = nothing scraped it (push: vmagent down or not delivering).
   - Exporter's own errors (DB query failed: credentials, grants, connection limit): `mysql_exporter_last_scrape_error`, `pg_exporter_last_scrape_error`, `mongodb_exporter_last_scrape_error`, `proxysql_exporter_last_scrape_error` (1 = error); node collectors `node_scrape_collector_success == 0`.
   - Slow collection: `mysql_exporter_collector_duration_seconds` by `collector`, `pg_exporter_last_scrape_duration_seconds`, `mongodb_exporter_last_scrape_duration_seconds`, compared with the timeout.
   - vmagent (push): `rate(vm_promscrape_scrapes_timed_out_total{node_name="N"}[5m])`, `rate(vm_promscrape_scrapes_failed_total{node_name="N"}[5m])`, `rate(vm_promscrape_dial_errors_total{node_name="N"}[5m])`, `vmagent_remotewrite_pending_data_bytes{node_name="N"}` (growing = cannot reach the server), `increase(vm_persistentqueue_bytes_dropped_total{node_name="N"}[1h])` > 0 = data lost. vmagent's own metrics travel the same path, so they go missing too when the push path is broken.
   - Server side: `pmm_managed_agents_connected`, `increase(pmm_managed_agents_disconnects_total[1h])` by `reason`, `pmm_managed_agents_clock_drift_seconds`.
4. When did it start: `pmm_metrics_snapshot` on `up{node_name="N"}` for the window; compare with a reboot (`node_boot_time_seconds{node_name="N"}`) or a deploy. Dashboards for evidence: `prometheus-status` (Prometheus Exporter Status) and `vmagent` (VictoriaMetrics Agents Overview).
5. Logs (`database=otel`): pmm-managed's view, `ResourceAttributes['node_name'] = 'pmm-server' AND LogAttributes['log.file.name'] = 'pmm-managed.log' AND Body ILIKE '%<pmm_agent_id>%'` ("Connected pmm-agent" marks reconnects). Remote/RDS/Azure exporters run in the server's pmm-agent: `LogAttributes['log.file.name'] = 'pmm-agent.log'` on `pmm-server`. A client's own pmm-agent log exists only if that node ships it.

## Interpretation
- pmm-agent disconnected, exporters status 6: host down, pmm-agent stopped, or network/TLS/proxy between client and server on 443. All of the node's data stops, QAN too.
- pmm-agent connected, `up` 0: exporter crashed (status 3/7), port blocked (pull), or scrape timeout. `*_exporter_last_scrape_error` 1: exporter runs but its DB login or grants fail.
- pmm-agent connected, `up` missing, push mode: vmagent not running or its remote write fails (pending bytes grow, drops follow after 1 GB).
- Gaps only at high resolution, or only for MySQL with many tables: collector too slow for the interval. Table-stats collectors are switched off above 1000 tables by default, but the count is taken when the service is added (`table_count`, `tablestats_group_disabled` in `pmm_list_agents`); a schema that grew later keeps them on.
- Clock drift of more than a few seconds: samples rejected or shifted; fix time sync.
- Agent newer than the server: unsupported, upgrade the server first.

## Remediation (self-managed client host; PMM runs none of these)
- State: `pmm-admin status` (server connection, agent list) and `pmm-admin list` (services and agent status).
- Service: `systemctl status pmm-agent`, `journalctl -u pmm-agent --since "1 hour ago"`, `sudo systemctl restart pmm-agent`. Docker client: `docker logs <pmm-client container>` and restart it. Kubernetes: the pmm-client sidecar's logs; change settings in the Operator CR.
- Network: client to server 443 (gRPC and push); pull mode: server to client ports 42000-51999. Test from the client with `curl -sk https://<pmm-server>/v1/server/readyz`.
- Slow MySQL scrapes: re-add with `pmm-admin add mysql ... --disable-tablestats` or `--disable-tablestats-limit=<n>`, or `--disable-collectors=<names>`; or choose a coarser metrics resolution (Configuration > Settings > Metrics resolution).
- Switch the metrics path when one direction is blocked: `pmm-admin add <type> ... --metrics-mode=push|pull`.
- Time: `timedatectl` / `chronyc tracking` on client and server host.
- Diagnostics bundle for support: `pmm-admin summary`.
