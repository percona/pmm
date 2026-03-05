# OTEL (OpenTelemetry) in PMM

Phase 1 adds **log collection** via an OTEL collector: agents send logs to the PMM server, which stores them in ClickHouse.

## Architecture

- **pmm-agent** runs an `otelcol-contrib` process (same node as other exporters). It collects:
  - OTLP (gRPC/HTTP) for push-based log ingestion
  - Optional **filelog** for user-specified log files (e.g. database error logs), with **per-path parser presets**
- Logs are sent to the **PMM server** OTLP endpoint (`/otlp/`), protected by existing auth (auth_request).
- The **server** runs its own otel-collector (supervisord) to receive and forward to ClickHouse.
- **ClickHouse**: database `otel`, table `otel.logs` with configurable TTL (retention). Schema is created automatically when OTEL is enabled in settings.

## Enabling OTEL

1. **Server**: Enable the OTEL collector and set log retention in PMM settings (e.g. Settings API or UI). This turns on the supervisord `otel-collector` program and ensures the `otel` ClickHouse schema exists.
2. **Agent**: On a host with pmm-agent, run:
   ```bash
   # Raw logs (no parsing):
   pmm-admin add otel --log-file-paths=/path/to/app.log

   # One preset for all paths:
   pmm-admin add otel --log-file-paths=/var/log/mysql/error.log --parser-preset=mysql_error

   # Per-path preset (path:preset pairs):
   pmm-admin add otel --log-sources=/var/log/mysql/error.log:mysql_error,/var/log/app.log:raw
   ```
   If no log paths or log-sources are given, only OTLP receivers are configured (no file tailing).

## Log parser presets (client-side)

To parse log files (e.g. MySQL error log) instead of sending raw lines, each path can be bound to a **preset**. Presets define the OTEL filelog operator chain (regex, time, severity, etc.).

- **Storage**: Presets are stored in the `log_parser_presets` table (PostgreSQL). Each row has: `id`, `name`, `description`, `operator_yaml` (YAML fragment for the filelog `operators:` block), `built_in`, timestamps.
- **Built-in preset**: `mysql_error` — MySQL 8 error log format (`timestamp thread_id [Subsystem] [CODE] [Component] message`). Seeded in migration 127.
- **`raw`**: No preset (no operators); log lines are sent as-is.
- **API**: Add/change OTEL collector with `log_sources`: list of `{ path, preset }`. The server validates preset names against `log_parser_presets` and stores `log_sources` as JSON in the agent’s `custom_labels`. Config generator groups paths by preset and emits one filelog receiver per preset with the corresponding operators.

### Why we did this

- **Per-path preset**: Different files can use different parsers (e.g. `mysql_error` for error.log, `raw` for access.log).
- **DB-backed presets**: Enables future API/UI to add or edit presets without code changes; built-in presets are protected by `built_in` (no delete).
- **Extensible**: Adding a new preset = new row in `log_parser_presets` (or a new migration for built-ins). Config generator already uses preset name → operator YAML; no extra code for new presets.

## Configuration

- **Settings**: `Otel.CollectorEnabled`, `Otel.LogsRetentionDays` (default 7).
- **Agent config**: Stored in the OTEL collector agent’s `custom_labels`:
  - **Legacy**: `log_file_paths` = comma-separated paths (treated as preset `raw`).
  - **Current**: `log_sources` = JSON array of `{"path":"...","preset":"..."}`. When present, it is used; otherwise `log_file_paths` is used as raw.

## Server-side log parsing

For `/srv/logs/nginx.log` we only parse **logfmt access lines**. The **filelog/nginx_access** receiver uses `key_value_parser`. Other line formats in the same file are not parsed for now.

## Implemented vs remaining

### Implemented

- Server-side OTEL collector (supervisord), config generation, filelog receivers for nginx, grafana, pmm-managed, pmm-agent, postgres; ClickHouse exporter; configurable TTL.
- Client-side OTEL collector (pmm-agent), config pushed from server; filelog receivers with per-preset operator YAML.
- `log_parser_presets` table and migration 127 with seed preset `mysql_error`.
- API: `LogSource` message, `log_sources` in `AddOtelCollectorParams`; validation and storage in custom_labels.
- `pmm-admin add otel` with `--log-file-paths`, `--log-sources` (path:preset), and `--parser-preset`.
- Backward compatibility: agents with only `log_file_paths` still work (treated as raw).

### Still to implement (future)

- **Preset management API**: ListLogParserPresets, AddLogParserPreset, UpdateLogParserPreset, DeleteLogParserPreset (forbid delete if `built_in`). Table and model already support it.
- **UI**: Add/edit presets; attach path+preset when adding OTEL collector.
- **More built-in presets**: e.g. mysql_general, mongodb, orchestrator (add rows in a new migration with their operator YAML).
- **Traces, profiles, eBPF**: Same agent type and server pipeline can be extended later.

## See also

- [TESTING.md](TESTING.md) for how to test Phase 1.
