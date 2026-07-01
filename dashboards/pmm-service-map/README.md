# PMM Service Map Panel

Interactive service topology map panel for Percona Monitoring and Management.

## Features

- **Topology graph**: Visualizes service-to-service connections using recording-rule metrics (`rr_connection_*`)
- **Health indicators**: Nodes and edges colored by error rate (green/amber/red)
- **Edge thickness**: Proportional to request rate (RPS)
- **Namespace grouping**: Services grouped by Kubernetes namespace
- **Edge detail sidebar**: Click an edge to see RPS, p95 latency, error %, bytes, and "why red?" explanation
- **Synchronized trace table**: Shows ClickHouse OTLP traces for the selected edge
- **Filter chips**: All / Errors / Slow for trace filtering
- **Trace ID deep-links**: Click to open in Grafana Explore

## Data sources

- **Prometheus / VictoriaMetrics**: For `rr_connection_l7_requests`, `rr_connection_l7_latency`, `rr_connection_tcp_bytes_sent`, `rr_connection_tcp_bytes_received`, `rr_connection_tcp_failed`
- **ClickHouse**: For `otel.otel_traces`

Default **recording rules** that map [coroot-node-agent](https://github.com/coroot/coroot-node-agent) `container_*` metrics to `rr_connection_*` ship with the PMM Server image as `/srv/prometheus/rules/pmm-service-map.recording-rules.yml` (vmalert loads `/srv/prometheus/rules/*.yml`). User-defined alerting rules in the UI still go to `pmm.rules.yml` only. After replacing a volume, ensure that file is still present (rebuild image or restore from backup); coroot must be scraped (e.g. `pmm_coroot_metrics_listen` on the `otel_collector` agent).

## Build

```bash
yarn install
yarn build
```

## Ship in PMM Server

The panel is built and installed with the rest of Percona dashboards:

- `make -C dashboards release` builds `pmm-app` and `pmm-service-map`.
- The `percona-dashboards` RPM copies `dashboards/pmm-service-map/dist` to `/usr/share/percona-dashboards/panels/pmm-service-map-panel`.
- On first start, `entrypoint.sh` copies `/usr/share/percona-dashboards/panels/*` to `/srv/grafana/plugins/`.
- `grafana.ini` lists `pmm-service-map-panel` under `allow_loading_unsigned_plugins`.

## Manual deploy (testing only)

```bash
tar -czf pmm-service-map-panel.tar.gz -C dist .
kubectl cp pmm-service-map-panel.tar.gz <namespace>/<pmm-pod>:/tmp/
kubectl exec -it <pmm-pod> -n <namespace> -- bash -c '
  mkdir -p /srv/grafana/plugins/pmm-service-map-panel &&
  tar -xzf /tmp/pmm-service-map-panel.tar.gz -C /srv/grafana/plugins/pmm-service-map-panel &&
  supervisorctl restart grafana
'
```
