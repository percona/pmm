# Testing OTEL Phase 1

## Prerequisites

1. Run **make gen** from the repo root so that generated API types (e.g. `AddAgentParamsBodyOtelCollector`, `AddAgentOKBodyOtelCollector`, inventory v1 agents.pb.go) exist.
2. Build and run PMM server and pmm-agent (e.g. from source or use a dev setup).
3. Ensure the server has **OTEL enabled** in settings (`Otel.CollectorEnabled` = true) and optional **Otel.LogsRetentionDays**.
4. Ensure the server’s **otel-collector** (supervisord) and **nginx** `/otlp/` location are deployed (e.g. via Ansible or your deployment).
5. **otelcol-contrib** is included in pmm-client and pmm-server packages/tarballs; when using a dev build, ensure the client tarball was built with the OTEL download step so `tools/otelcol-contrib` exists.

## 1. Add OTEL collector from the agent host

From the machine where pmm-agent runs:

```bash
# OTLP-only collector (e.g. before adding log files)
pmm-admin management add otel ebpf

# With log files (e.g. MySQL error log); merges into the single node collector
pmm-admin management add otel logs --log-file-paths=/var/log/mysql/error.log

# With custom labels
pmm-admin management add otel logs --custom-labels=env=prod,team=db
```

Check that the agent is listed and the collector process is running (e.g. `pmm-admin list` or Inventory API).

## 2. Send logs via OTLP (optional)

If you have an OTLP client or another collector that can send to the PMM server:

- Endpoint: `https://<pmm-server>/otlp/` (HTTP OTLP). Use the same credentials as for the PMM API.
- The server’s nginx proxies `/otlp/` to the otel-collector; auth_request applies.

## 3. Verify data in ClickHouse

- Connect to ClickHouse (e.g. `clickhouse-client` or Grafana).
- Database `otel` and table `otel.logs` should exist (created automatically when OTEL is enabled).
- Example query:
  ```sql
  SELECT Timestamp, ServiceName, Body FROM otel.logs ORDER BY Timestamp DESC LIMIT 10;
  ```

## 4. Grafana

- The **ClickHouse-OTEL** datasource (if provisioned) uses the same ClickHouse with default database `otel` for building dashboards over `otel.logs`.

## Troubleshooting

- **Collector not starting**: Check pmm-agent logs and that `otelcol-contrib` exists at the path in config (e.g. `tools/otelcol-contrib`). It is shipped with pmm-client and pmm-server; for custom builds, the build-client-binary script downloads it from OpenTelemetry releases.
- **No logs in ClickHouse**: Ensure the server’s otel-collector is running and writing to ClickHouse; check server logs and that `PMM_CLICKHOUSE_*` env is set for the collector.
- **Auth errors on /otlp/**: Use valid PMM API credentials (e.g. the same used by pmm-agent).
