# OTEL (OpenTelemetry) in PMM

Phase 1 adds **log collection** via an OTEL collector: agents send logs to the PMM server, which stores them in ClickHouse.

## Architecture

- **Server supervisord OTEL** (`/srv/otelcol/config.yaml`, rendered by pmm-managed): OTLP receiver (for all agents) plus filelog receivers for PMM Server's own logs → transform (sets `pmm_source` from `node_name`, and `service.name` from `service_name`, else `node_name`) → batch → ClickHouse.
- **PMM Server logs**: the server collector reads `/srv/logs/*.log` itself (`managed/otel/server_logs.go`), one filelog receiver per file, with the preset chosen by file name. Unknown `*.log` files are read raw; rotated files (`*.log.1`) are skipped.

  | File | Preset |
  |---|---|
  | `pmm-managed.log` | `pmm_managed` |
  | `pmm-agent.log`, `qan-api2.log`, `vmproxy.log` | `pmm_agent` |
  | `nginx.log` | `nginx_access` |
  | `grafana.log` | `grafana` |
  | `clickhouse-server.log` | `clickhouse_server` |
  | `postgresql14.log` | `postgres` |
  | `supervisord.log` | `supervisord` |
  | `otel-collector.log` | `otel_collector` |
  | `victoriametrics.log`, `vmalert.log` | `victoriametrics` |
  | `pmm-init.log`, any other `*.log` | `raw` |

  - Resource attributes: `node_id`, `node_name` of the server's node, `service.name` = component (`pmm-managed`, `nginx`, …; `pmm-server` for unknown files), plus `ha_node_id` in HA.
  - Read positions are kept in `/srv/otelcol/storage` (`file_storage` extension), so a collector restart neither loses nor repeats lines; on first start the collector begins at the end of each file.
  - nginx keeps only lines with status 400 or above and its error-format lines; vmproxy drops its per-request info lines (nginx already logs those requests).
  - Every operator on these receivers fails quietly (`send_quiet` / `drop_quiet`): the collector reads its own log, so a logged parse error would be read back and fail again. A line that does not parse is stored unparsed.
  - pmm-managed writes a changed config to `config.yaml.new`, checks it with `otelcol-contrib validate`, and only then replaces the old one and restarts the collector. A config the collector rejects is logged and the old one stays.
- **Remote nodes** (and extra files on any node): **pmm-agent**'s OTEL collector tails files with filelog receivers and the same **parser presets**, and sends OTLP to the server.
- **Presets** live in the `log_parser_presets` table (PostgreSQL). Changing a preset re-renders the server collector config and pushes new configs to agents.
- **ClickHouse**: database `otel`, table `otel.logs` with configurable TTL (retention). Schema is created automatically when OTEL is enabled. Filter and alert on `SeverityNumber` (ERROR 17, WARN 13, FATAL 21, INFO 9); `SeverityText` differs per engine (`LOG`/`ERROR`, `I`/`W`, `Note`/`Warning`) and is for display.

## Enabling OTEL

1. **Server**: Enable the OTEL collector and set log retention in PMM settings. This starts the supervisord `otel-collector`, which receives agents' OTLP and reads PMM Server's own logs, and ensures the `otel` ClickHouse schema exists.
2. **Agent** (any node, including the server): Run `pmm-admin add otel` with log paths and presets. Example:
   ```bash
   # Raw logs (no parsing):
   pmm-admin add otel --log-file-paths=/path/to/app.log

   # One preset for all paths:
   pmm-admin add otel --log-file-paths=/var/log/mysql/error.log --parser-preset=mysql_error

   # Per-path preset (path:preset pairs):
   pmm-admin add otel --log-sources=/var/log/mysql/error.log:mysql_error,/var/log/messages:syslog_mysql_systemd,/var/log/app.log:raw

   # MongoDB (systemLog.destination: file):
   pmm-admin add otel logs --log-sources=/var/log/mongo/mongod.log:mongodb
   ```
   Run `pmm-admin add otel --help` to see all available presets. If no log paths or log-sources are given, only OTLP receivers are configured (no file tailing).

## Log parser presets

Presets define the OTEL filelog operator chain (regex, time, severity, etc.) and are stored in `log_parser_presets`. Each path can be bound to a preset (or `raw` for no parsing).

- **Built-in presets**: `mysql_error`, `syslog_mysql_systemd` (journal/syslog-style ISO8601 lines: `timestamp host tag[pid]: message`), `nginx_access`, `nginx_error`, `grafana`, `pmm_managed`, `pmm_agent`, `postgres`, `clickhouse_server`, `otel_collector`, `supervisord`, `victoriametrics` (VictoriaMetrics and vmalert: time, level, caller, message separated by tabs), `mongodb` (MongoDB 4.4+ JSON log: body is `msg`, severity from `s`, `c`/`ctx`/`attr` kept as attributes), `raw`. Manage presets in the UI under **Settings → OTEL** (or add rows via API / migrations).
- **Storage**: Table columns: `id`, `name`, `description`, `operator_yaml`, `built_in`, timestamps. The server validates preset names and stores `log_sources` as JSON in the agent’s `custom_labels`.
- **API**: Add/change OTEL collector with `log_sources`: list of `{ path, preset }`. Config generator groups paths by preset and emits one filelog receiver per preset.

### Why we did this

- **Single model**: Server logs and remote nodes use the same DB presets.
- **Per-path preset**: Different files can use different parsers.
- **Extensible**: New preset = new row in `log_parser_presets` (or a new migration for built-ins).

## Configuration

- **Settings**: `Otel.CollectorEnabled`, `Otel.LogsRetentionDays` (default 7).
- **Agent config**: In the OTEL collector agent’s `custom_labels`: **Legacy** `log_file_paths` (treated as `raw`); **Current** `log_sources` = JSON array of `{"path":"...","preset":"..."}`.

## Implemented vs remaining

### Implemented

- Server-side OTEL collector (supervisord): OTLP and `/srv/logs` filelog receivers → transform → batch → ClickHouse; configurable TTL.
- Client-side OTEL collector (pmm-agent), config pushed from server; filelog receivers with per-preset operator YAML from `log_parser_presets`.
- `log_parser_presets` table: migration 127 (`mysql_error`), 128/129 (nginx, grafana, PMM, postgres, clickhouse_server, otel_collector, supervisord), migration 134 (`syslog_mysql_systemd`), 143 (`mongodb`), 148 (`victoriametrics`; `postgres` starts an entry only at a top-level line, so DETAIL/HINT/STATEMENT/CONTEXT join the error above).
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
