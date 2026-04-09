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

## Build

```bash
yarn install
yarn build
```

## Deploy to PMM

```bash
tar -czf pmm-service-map-panel.tar.gz -C dist .
kubectl cp pmm-service-map-panel.tar.gz <namespace>/<pmm-pod>:/tmp/
kubectl exec -it <pmm-pod> -n <namespace> -- bash -c '
  mkdir -p /srv/grafana/plugins/pmm-service-map-panel &&
  tar -xzf /tmp/pmm-service-map-panel.tar.gz -C /srv/grafana/plugins/pmm-service-map-panel &&
  supervisorctl restart grafana
'
```
