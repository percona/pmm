---
name: pmm-server-health
description: "PMM Server itself unhealthy: UI or API errors, all graphs empty or late, QAN not updating, /srv disk filling, pmm-managed, VictoriaMetrics, vmalert, ClickHouse, Grafana or internal PostgreSQL down, HA leader or Raft trouble, server version and update questions."
---

# PMM Server health

## When to use
The monitoring plane is suspect: data missing for everything (not one node), QAN or logs stale, UI slow/5xx, disk on the PMM Server filling, HA cluster questions, server version. One node or service missing data: exporter-scrape-failures.

## Steps
1. **Tool behaviour is evidence.** `pmm_list_nodes` works: pmm-managed and its PostgreSQL answer. Prometheus tools fail: VictoriaMetrics or vmproxy. `pmm_clickhouse_query` fails: ClickHouse. Only renders fail: Grafana/renderer, data is fine.
2. **Components** (instant): `up{job=~"pmm-managed|victoriametrics|vmalert|qan-api2|grafana|clickhouse"}` and `min(up{service_name="pmm-server-postgresql"})`; a job missing entirely = that component runs externally or is not scraped. A restart shows as a reset of `ClickHouseAsyncMetrics_Uptime{job="clickhouse"}` or of `process_start_time_seconds{job="<job>"}`.
3. **Disk** (`/srv` holds all data): `node_filesystem_free_bytes{node_name="pmm-server",mountpoint=~"/srv|/"} / node_filesystem_size_bytes{node_name="pmm-server",mountpoint=~"/srv|/"}`; VictoriaMetrics `vm_free_disk_space_bytes{job="victoriametrics"}` and `sum(vm_data_size_bytes{job="victoriametrics"})`. `pmm_metrics_snapshot` of free bytes over 7 days gives the slope and time to full (dashboard `victoriametrics` panel 81 "Time before run out of space").
4. **Metric ingestion**: `sum(rate(vm_rows_inserted_total{job="victoriametrics"}[5m]))` versus a day earlier; slow inserts `sum(rate(vm_slow_row_inserts_total{job="victoriametrics"}[5m])) / sum(rate(vm_rows_inserted_total{job="victoriametrics"}[5m]))`; `vm_pending_rows{job="victoriametrics",type="storage"}`; errors `sum(rate(vm_http_request_errors_total{job="victoriametrics"}[5m]))`; churn `sum(rate(vm_new_timeseries_created_total{job="victoriametrics"}[5m]))`; memory `sum(vm_cache_size_bytes{job="victoriametrics"}) / max(vm_allowed_memory_bytes{job="victoriametrics"})`; slow queries `increase(vm_slow_queries_total{job="victoriametrics"}[1h])`.
5. **QAN pipeline**: `qan_api2_data_ingestion_requests_len{job="qan-api2"}` close to `qan_api2_data_ingestion_requests_cap{job="qan-api2"}` = backlog; `rate(qan_api2_data_ingestion_buckets_per_batch_sum{job="qan-api2",error="1"}[5m])` > 0 = failed inserts. Freshness (`database=pmm`): `SELECT service_name, max(period_start) AS last FROM pmm.metrics WHERE period_start > now() - INTERVAL 1 DAY GROUP BY service_name ORDER BY last LIMIT 50`; lag of a few minutes is normal. ClickHouse: `ClickHouseMetrics_MemoryTracking{job="clickhouse"}`, `rate(ClickHouseProfileEvents_SlowRead{job="clickhouse"}[5m])`. Logs freshness (`database=otel`): `SELECT max(Timestamp) FROM logs`.
6. **pmm-managed and agents**: `process_resident_memory_bytes{job="pmm-managed"}`; `pmm_managed_agents_connected` versus `count(pmm_managed_inventory_agents{agent_type="pmm-agent"})`; `increase(pmm_managed_agents_disconnects_total[1h])` by `reason`; `pmm_managed_agents_clock_drift_seconds`, `pmm_managed_agents_round_trip_seconds`. Many agents dropping together = server or network side.
7. **Grafana**: `sum by (code) (rate(grafana_api_response_status_total[5m]))`, look at 5xx.
8. **Server logs** (`database=otel`, `ResourceAttributes['node_name'] = 'pmm-server'`), files `pmm-managed.log`, `qan-api2.log`, `clickhouse-server.log`, `grafana.log`, `postgresql14.log`, `nginx.log`, `vmproxy.log`, `pmm-agent.log`, `supervisord.log` (process exits and respawns), `otel-collector.log`. Filter `SeverityNumber >= 17` and the window (otel-logs-analysis). `victoriametrics.log` and `vmalert.log` are not shipped: ask the user.
9. **HA** (PMM_HA_ENABLE, Helm HA): `sum(pmm_ha_leader_status)` must be 1 (0 no leader, > 1 split brain); `changes(pmm_ha_raft_term[10m])` > 5 = leader flapping; `count(pmm_ha_up{role="voter"})` < 3 = quorum at risk. No `pmm_ha_*` series = HA not enabled. Pods, restarts and volume usage: dashboard `pmm-ha-health-overview`. `GET /v1/server/leaderHealthCheck` succeeds only on the leader (load-balancer check).
10. **Versions**: `count by (version) (pmm_managed_inventory_agents{agent_type="pmm-agent"})` lists client versions. Server version and available updates: Configuration > Updates (API `GET /v1/server/version`, `/v1/server/updates`). The server must be the same or newer than every client: upgrade the server first.

## Interpretation
- Free space under ~10 % on `/srv` and falling: VictoriaMetrics and ClickHouse fail writes next, then the UI. Size drivers: retention, metric resolution, number of series (churn), QAN volume.
- Slow inserts > 10 % or cache near allowed memory: VictoriaMetrics is short of RAM for its index; ingestion lags and graphs end early.
- QAN backlog with ClickHouse up: ClickHouse slow or out of memory; with ClickHouse down: nothing new lands in pmm.metrics until it is back (the freshness query shows when it stopped).
- Clock drift of seconds between agents and server shifts or drops data and can break TLS.

## Remediation (PMM never runs these)
- Docker/Podman: `docker exec pmm-server supervisorctl status`; restart one component `docker exec pmm-server supervisorctl restart <pmm-managed|qan-api2|grafana|clickhouse|victoriametrics|vmalert>`; tail `docker exec pmm-server tail -n 200 /srv/logs/<component>.log`; full bundle `https://<pmm-server>/logs.zip`.
- Disk: grow the `/srv` volume; lower Data retention (Configuration > Settings > Advanced settings, default 30 days; `PMM_DATA_RETENTION`); use a coarser metrics resolution; remove unused services. Never delete files under `/srv` by hand.
- RAM: give the container more memory; reduce series (disable unneeded collectors on large MySQL with `--disable-tablestats`).
- Kubernetes/HA: `kubectl -n <ns> get pods`, `kubectl -n <ns> logs <pod> --previous`; change resources through Helm values, not inside pods.
- Clock: fix NTP/chrony on the clients and on the server host.
