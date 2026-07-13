# OTEL (OpenTelemetry) in PMM

Phase 1 adds **log collection** via an OTEL collector: agents send logs to the PMM server, which stores them in ClickHouse.

## Architecture

- **Server supervisord OTEL** = **receiver only**: OTLP receiver (for all agents) → transform (sets `pmm_source` from `node_name`) → batch → ClickHouse. No filelog on the server; all log collection uses the same pipeline below.
- **All log collection** (including server logs) is done by **pmm-agent** (pmm-client) using **filelog** receivers and **parser presets** from the DB. On the server host there are two processes: (1) supervisord `otel-collector` (receiver only), (2) pmm-agent’s OTEL collector (filelog for server logs, e.g. nginx, grafana, pmm-managed, postgres), which sends OTLP to localhost. Remote nodes run only the pmm-agent OTEL collector and send to the server.
- **Presets** live in the `log_parser_presets` table (PostgreSQL). The server’s default log_sources (nginx, grafana, etc.) are created in **UpdateConfigurations** when OTEL is enabled (idempotent: create OTEL collector agent on the server node only if missing).
- **ClickHouse**: database `otel`, table `otel.logs` with configurable TTL (retention). Schema is created automatically when OTEL is enabled.

## Enabling OTEL

1. **Server**: Enable the OTEL collector and set log retention in PMM settings. This starts the supervisord `otel-collector` (receiver), ensures the `otel` ClickHouse schema exists, and ensures the **server node** has an OTEL collector agent with default log_sources (nginx, grafana, pmm-managed, pmm-agent, postgres) so server logs are collected by pmm-agent on the server.
2. **Agent** (any node, including the server): Run `pmm-admin add otel` with log paths and presets. Example:
   ```bash
   # Raw logs (no parsing):
   pmm-admin add otel --log-file-paths=/path/to/app.log

   # One preset for all paths:
   pmm-admin add otel --log-file-paths=/var/log/mysql/error.log --parser-preset=mysql_error

   # Per-path preset (path:preset pairs):
   pmm-admin add otel --log-sources=/var/log/mysql/error.log:mysql_error,/var/log/messages:syslog_mysql_systemd,/var/log/app.log:raw
   ```
   Run `pmm-admin add otel --help` to see all available presets. If no log paths or log-sources are given, only OTLP receivers are configured (no file tailing).

## Log parser presets

Presets define the OTEL filelog operator chain (regex, time, severity, etc.) and are stored in `log_parser_presets`. Each path can be bound to a preset (or `raw` for no parsing).

- **Built-in presets**: `mysql_error`, `syslog_mysql_systemd` (journal/syslog-style ISO8601 lines: `timestamp host tag[pid]: message`), `nginx_access`, `nginx_error`, `grafana`, `pmm_managed`, `pmm_agent`, `postgres`, `clickhouse_server`, `otel_collector`, `supervisord`, `raw`. Manage presets in the UI under **Settings → OTEL** (or add rows via API / migrations).
- **Storage**: Table columns: `id`, `name`, `description`, `operator_yaml`, `built_in`, timestamps. The server validates preset names and stores `log_sources` as JSON in the agent’s `custom_labels`.
- **API**: Add/change OTEL collector with `log_sources`: list of `{ path, preset }`. Config generator groups paths by preset and emits one filelog receiver per preset.

### Why we did this

- **Single model**: Server and remote nodes both use pmm-agent + DB presets; no hardcoded server-side filelog.
- **Per-path preset**: Different files can use different parsers.
- **Extensible**: New preset = new row in `log_parser_presets` (or a new migration for built-ins).

## Configuration

- **Settings**: `Otel.CollectorEnabled`, `Otel.LogsRetentionDays` (default 7).
- **Agent config**: In the OTEL collector agent’s `custom_labels`: **Legacy** `log_file_paths` (treated as `raw`); **Current** `log_sources` = JSON array of `{"path":"...","preset":"..."}`.

## Implemented vs remaining

### Implemented

- Server-side OTEL collector (supervisord): **receiver only** (OTLP → transform → batch → ClickHouse); configurable TTL.
- Server log collection by pmm-agent on the server with default log_sources (nginx, grafana, pmm-managed, pmm-agent, postgres), created in UpdateConfigurations when OTEL is enabled.
- Client-side OTEL collector (pmm-agent), config pushed from server; filelog receivers with per-preset operator YAML from `log_parser_presets`.
- `log_parser_presets` table: migration 127 (`mysql_error`), 128/129 (nginx, grafana, PMM, postgres, clickhouse_server, otel_collector, supervisord), migration 130 (`syslog_mysql_systemd`).
- API: `LogSource`, `log_sources` in `AddOtelCollectorParams`; validation and storage in custom_labels.
- `pmm-admin add otel` with `--log-file-paths`, `--log-sources`, `--parser-preset`; help lists all built-in presets.
- Backward compatibility: agents with only `log_file_paths` still work (treated as raw).

### Still to implement (future)

- **More built-in presets**: e.g. mysql_general, mongodb, orchestrator.

### Recently added

- **Preset management API** and **UI** (**Settings → OTEL**): list/get/add/change/remove custom presets; built-in rows editable; delete blocked while an OTEL collector still references the preset.
- **Traces, profiles, eBPF**: Same agent type and server pipeline can be extended later.

## See also

- [TESTING.md](TESTING.md) for how to test Phase 1.
