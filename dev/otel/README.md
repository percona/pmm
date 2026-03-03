# OTEL (OpenTelemetry) in PMM

Phase 1 adds **log collection** via an OTEL collector: agents send logs to the PMM server, which stores them in ClickHouse.

## Architecture

- **pmm-agent** runs an `otelcol-contrib` process (same node as other exporters). It collects:
  - OTLP (gRPC/HTTP) for push-based log ingestion
  - Optional **filelog** for user-specified log files (e.g. database error logs)
- Logs are sent to the **PMM server** OTLP endpoint (`/otlp/`), protected by existing auth (auth_request).
- The **server** may run its own otel-collector (supervisord) to receive and forward to ClickHouse; alternatively the endpoint can be implemented by nginx proxying to a collector.
- **ClickHouse**: database `otel`, table `otel.logs` with configurable TTL (retention). Schema is created automatically when OTEL is enabled in settings.

## Enabling OTEL

1. **Server**: Enable the OTEL collector and set log retention in PMM settings (e.g. Settings API or UI). This turns on the supervisord `otel-collector` program and ensures the `otel` ClickHouse schema exists.
2. **Agent**: On a host with pmm-agent, run:
   ```bash
   pmm-admin add otel [--log-file-paths=/path/to/error.log] [--custom-labels=key=value]
   ```
   If `--log-file-paths` is omitted, only OTLP receivers are configured (no file tailing).

## Configuration

- **Settings**: `Otel.CollectorEnabled`, `Otel.LogsRetentionDays` (default 7).
- **Agent config**: Log file paths and custom labels are stored in the OTEL collector agent's custom_labels (`log_file_paths` = comma-separated paths).

## Extensibility

The design is ready to extend to **traces**, **profiles**, and **eBPF** later: same agent type, additional pipelines and exporters as needed.

## See also

- [TESTING.md](TESTING.md) for how to test Phase 1.
