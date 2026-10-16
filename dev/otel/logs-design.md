# Logs: rule-based detection and the Logs page — design

Status: draft for review (2026-10-16). Branch `tibi-holmes` (PMM-7).

## Goal

Give PMM users one place to read, filter and make sense of every collected log — database logs from agents
and PMM Server's own logs — and find known problems in them automatically with predefined, editable rules, not by
running an LLM over every line. A rule match is a normal PMM alert, so notifications, the Alerts page and the
Assistant's *Investigate* work unchanged.

Both halves weigh equally: detection (rules → alerts → Assistant) and a log viewer for incidents.

## Current state

Verified on the test server (adre-fb4259: PMM 3.9.1 build of this branch, otelcol-contrib 0.148.0, Grafana 12.4.5,
grafana-clickhouse-datasource 4.20.0) and in the branch code:

- Agents tail log files with `filelog` receivers built from parser presets (`log_parser_presets`, generated in
  `managed/services/agents/otelcollector.go`) and send OTLP to the server collector, which writes `otel.logs`
  (`managed/services/supervisord/otelconfig.go`, schema in `managed/otel/schema.go`).
- `otel.logs` sorts by `(ServiceName, TimestampTime, …)`, but `ServiceName` is empty on every row, so a node or
  service filter reads the whole time range: one node's errors over 6 h read 1.5M rows / 404 MB (0.2 s at about
  350k lines/h).
- PostgreSQL error reports arrive as separate rows: `ERROR`, then `DETAIL`, `STATEMENT`, `CONTEXT`, `HINT` at INFO
  (SeverityNumber 9).
- `SeverityText` differs per engine (`LOG`/`ERROR`, `I`/`W`, `Note`/`Warning`); `SeverityNumber` is consistent
  (ERROR 17, WARN 13, FATAL 21, INFO 9).
- PMM Server's own logs (`/srv/logs/*.log`) are not collected: `DefaultServerOtelLogSources` is used only by
  `BuildServerOtelConfigYAML`, which nothing calls, and no code creates the server-node collector agent that
  `dev/otel/README.md` describes. The list also misses `victoriametrics.log`, `vmalert.log` and `pmm-init.log`.
- No logs page exists in pmm-ui; no Grafana dashboard queries `otel.logs`; no alert template reads logs.
- The server collector build has every component this design needs: `filelog`, `transform`, `filter`, `count`,
  `deltatocumulative`, `prometheusremotewrite`, `file_storage`.

## Overview

```
pmm-agent: filelog + presets (+ log.preset attribute) ──OTLP──▶ server OTel collector
PMM Server: /srv/logs/*.log ──filelog (server collector)──────▶      │
                                                                     ├─ transform: service.name; pmm.rule tag per rule
                                                                     ├─▶ ClickHouse otel.logs (all lines)
                                                                     └─▶ count ─▶ deltatocumulative ─▶ VictoriaMetrics
                                                                           pmm_log_rule_matches_total{pmm_rule, node_name, service_name}
                                                                                 └─▶ log-rule alert templates ─▶ Alerts, Assistant
pmm-managed: log rules (PostgreSQL) ─▶ server collector config + alert templates
pmm-managed: logs API over ClickHouse ─▶ pmm-ui /logs (Lines, Patterns, Rules)
```

The `pmm.rule` attribute ties the halves together: an alert names its rule, and the Logs page shows exactly the
lines that matched.

Work is split into three parts, each with its own implementation plan, built in order.

## Part 1: log data

### 1.1 `ServiceName`

The server collector's existing `transform` sets `resource.attributes["service.name"]` to `service_name` when
present, else `node_name`. No schema change: the sort key already starts with `ServiceName`, so node and service
filters prune. Rows written before the change keep an empty value until TTL removes them.

The collector agent runs per node, so a "service" filter means that service's node. Tying a log source to a service
(`--log-sources path:preset:service`) is a later change.

### 1.2 One entry per PostgreSQL error

A new migration changes the built-in `postgres` preset's `line_start_pattern` so an entry starts only at a
top-level line:

```
^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)? \S+ \[\d+\] (?:\S+@\S+ )?(LOG|ERROR|WARNING|FATAL|PANIC|NOTICE|INFO|DEBUG\d?):
```

The optional `user@db ` covers Debian's and Ubuntu's default `log_line_prefix` (`'%m [%p] %q%u@%d '`).

`DETAIL`, `HINT`, `STATEMENT`, `CONTEXT`, `QUERY` and `LOCATION` lines join the entry above. PostgreSQL writes a
whole report in one `write()`, so its lines are adjacent in the file. The migration only touches the preset if it is
unchanged (same guard as migration 145). Agents get the new operators on their next config push.

### 1.3 Severity

Filters and rules use `SeverityNumber`. `SeverityText` is display only.

### 1.4 PMM Server logs

- The server collector reads `/srv/logs/*.log` itself with `filelog` receivers, excluding rotated files
  (`*.log.[0-9]*`). No extra agent and no OTLP loop through nginx.
- The preset is chosen by file name; unknown files get `raw`:

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
  | `victoriametrics.log`, `vmalert.log` | `victoriametrics` (new) |
  | `pmm-init.log` | `raw` |

  The preset for `qan-api2.log` and `vmproxy.log` is checked against their real format during Part 1 and changed if
  `pmm_agent` does not fit.
- Resource attributes: `node_name` and `node_id` of the server's node (`pmm-server`, or the HA node's name),
  `service.name` = component (`pmm-managed`, `clickhouse-server`, …).
- Read positions are kept by a `file_storage` extension under `/srv/otelcol/storage`, so a collector restart neither
  loses nor repeats lines. `start_at: end` on first start, so a new install does not import old files.
- nginx logs every API call (about 50 MB per rotated file on the test server). A `filter` drops `nginx_access` lines
  with status below 400; 4xx, 5xx and error lines are kept.
- `DefaultServerOtelLogSources` and `BuildServerOtelConfigYAML` are replaced by this; `dev/otel/README.md` is
  updated.

### 1.5 Out of scope

Changing log retention does not alter the existing table's TTL (`schema.go` has no `MODIFY TTL`). Separate ticket.

## Part 2: rules and detection

### 2.1 Rule model

Table `log_rules` (new migration, reform model `LogRule`):

| Column | Type | Notes |
|---|---|---|
| `name` | varchar PK | `^[a-z0-9_]{1,64}$`, used in the metric and the template name |
| `summary` | varchar | shown in the UI and the alert |
| `presets` | jsonb | non-empty list of preset names the rule applies to |
| `min_severity` | int | OTel SeverityNumber; 0 = any |
| `match_type` | varchar | `contains` (case-insensitive) or `regex` |
| `match` | varchar | ≤ 1024 chars |
| `enabled` | bool | |
| `alert_severity` | varchar | `info`, `warning`, `error`, `critical` |
| `alert_threshold` | double | matches per window that fire the alert |
| `alert_window` | interval | 1 m to 1 h |
| `source` | varchar | `builtin` or `user` |
| `deleted_at` | timestamptz | soft delete for built-ins, so upgrades do not bring them back |
| `created_at`, `updated_at`, `updated_by` | | |

Validation on save: Go `regexp` compiles the pattern (`contains` is compiled as `(?i)` + `regexp.QuoteMeta`);
`match` may be empty only when `min_severity` > 0 (a severity-only rule); presets must exist; at most 200 enabled
rules.

### 2.2 Built-in rules and sync

Built-in rules ship as YAML in `managed/data/log-rules/*.yml` and are synced at startup with the pattern of
`planSkillSync` (`managed/services/adre/deployment/seed.go`):

- New shipped rule → inserted.
- Shipped rule changed and the row is still `source = builtin` → updated.
- An admin edit sets `source = user`; upgrades leave it alone.
- A deleted built-in keeps `deleted_at`; upgrades do not restore it.

### 2.3 Rules API

`api/logs/v1/rules.proto`, `LogRulesService` (admin for writes, viewer for reads):

| RPC | HTTP |
|---|---|
| `ListLogRules` | `GET /v1/logs/rules` |
| `CreateLogRule` | `POST /v1/logs/rules` |
| `UpdateLogRule` | `PUT /v1/logs/rules/{name}` (includes enable/disable) |
| `DeleteLogRule` | `DELETE /v1/logs/rules/{name}` |
| `PreviewLogRule` | `POST /v1/logs/rules:preview` — a rule body (saved or not) and a time range; returns the match count and up to 20 sample lines from `otel.logs` |

`ListLogRules` returns, per rule, matches in the last 1 h and 24 h from VictoriaMetrics.

### 2.4 Collector pipeline

pmm-managed renders the server collector config from the enabled rules and restarts the collector through
supervisord when rules change (changes within 10 s are batched into one restart). In HA every node renders the same
config from the shared database. Rule strings are escaped as OTTL string literals; unit tests cover quotes,
backslashes and Unicode. The first matching rule wins, so statements are ordered: rules with a `match` first, then
severity-only rules, each group by name. During a restart agents keep sending: their OTLP exporters retry, so no lines
are lost.

```yaml
processors:
  transform/pmm_rules:
    error_mode: ignore
    log_statements:
      - context: log
        statements:
          # one statement per enabled rule; the first match wins
          - set(attributes["pmm.rule"], "postgresql_deadlock") where attributes["pmm.rule"] == nil and attributes["log.preset"] == "postgres" and severity_number >= 17 and IsMatch(body, "deadlock detected")
  deltatocumulative: {}
connectors:
  count/pmm_rules:
    logs:
      pmm_log_rule_matches:
        description: Log lines matched by a PMM log rule.
        conditions: ['attributes["pmm.rule"] != nil']
        attributes:
          - key: pmm.rule
          - key: node_name
          - key: service.name
exporters:
  prometheusremotewrite/pmm_rules:
    endpoint: http://127.0.0.1:9090/prometheus/api/v1/write
service:
  pipelines:
    logs:
      receivers: [otlp, filelog/srv_<component>...]
      processors: [memory_limiter, transform, filter/nginx_success, transform/pmm_rules, batch]
      exporters: [clickhouse/logs, count/pmm_rules]
    metrics/pmm_rules:
      receivers: [count/pmm_rules]
      processors: [deltatocumulative, batch]
      exporters: [prometheusremotewrite/pmm_rules]
```

Agents tag each line with `attributes["log.preset"]` through an `add` operator in each preset receiver; the server
collector's own receivers do the same. Because the agent config is rendered by pmm-managed, existing agents need no
upgrade.

Verified in Part 2's first task, before anything is built on them:

1. `count` groups by resource attributes (`node_name`, `service.name`); if not, `transform/pmm_rules` copies them
   onto the record.
2. `count` emits delta sums, so `deltatocumulative` is needed before remote write.
3. The resulting series name and labels in VictoriaMetrics (expected
   `pmm_log_rule_matches_total{pmm_rule, node_name, service_name}`), and the write URL.
4. Collector CPU with 30 rules at the test server's log rate.

### 2.5 Alert templates

The alerting service gets a fourth template source next to built-in, user-file and DB templates: one template per
log rule, generated in `CollectTemplates` and collected again after a rule change. New enum value
`TEMPLATE_SOURCE_LOG_RULE` (additive). These templates are read-only in the Alerting UI; they are edited through
their rule.

```yaml
name: pmm_log_rule_postgresql_deadlock
summary: PostgreSQL deadlock detected (logs)
# [[ ]] are template delimiters, so the PromQL range is "[" + [[ .window ]] + "]"
expr: sum by (node_name, service_name) (increase(pmm_log_rule_matches_total{pmm_rule="postgresql_deadlock"}[[[ .window ]]])) >= [[ .threshold ]]
params: threshold (float, default from the rule), window (string, default from the rule)
for: 0s
severity: from the rule
labels: {pmm_log_rule: postgresql_deadlock}
annotations:
  description: <summary>; open the matching lines: /pmm-ui/logs?rule=postgresql_deadlock&node=<node>&from=<t-15m>&to=<t+5m>
```

Alert rules are not created automatically; users create them from these templates as from any other. Deleting a log
rule deletes its template; the Rules tab warns first when an alert rule uses it, because that rule would stop firing.

### 2.6 Built-in rules (v1)

| Rule | Preset | Match |
|---|---|---|
| `mysql_deadlock` | mysql_error | `Transactions deadlock detected` |
| `mysql_access_denied` | mysql_error | `Access denied for user` (threshold 10 / 5 m) |
| `mysql_replica_sql_error` | mysql_error | `Replica SQL for channel` + ERROR |
| `mysql_semaphore_wait` | mysql_error | `Semaphore wait has lasted` |
| `mysql_crash_recovery` | mysql_error | `crash recovery` |
| `mysql_out_of_memory` | mysql_error | `Out of memory` |
| `postgresql_deadlock` | postgres | `deadlock detected` |
| `postgresql_auth_failed` | postgres | `password authentication failed` (threshold 10 / 5 m) |
| `postgresql_too_many_clients` | postgres | `too many clients already` |
| `postgresql_statement_timeout` | postgres | `canceling statement due to statement timeout` (threshold 5 / 5 m) |
| `postgresql_lock_wait` | postgres | `still waiting for .*Lock` |
| `postgresql_checkpoints_too_frequent` | postgres | `checkpoints are occurring too frequently` |
| `postgresql_temp_file` | postgres | `temporary file: path` (threshold 20 / 5 m) |
| `postgresql_shm_exhausted` | postgres | `could not resize shared memory segment` |
| `postgresql_fatal_panic` | postgres | severity ≥ FATAL |
| `postgresql_unclean_shutdown` | postgres | `was not properly shut down` |
| `mongodb_auth_failed` | mongodb | `Failed to authenticate` (threshold 10 / 5 m) |
| `mongodb_election` | mongodb | `Starting an election` |
| `mongodb_stepdown` | mongodb | `Stepping down from primary` |
| `mongodb_query_executor_error` | mongodb | `Plan executor error` |
| `mongodb_wiredtiger_error` | mongodb | `WiredTiger error` |
| `pmm_managed_errors` | pmm_managed | severity ≥ ERROR (threshold 20 / 5 m) |
| `clickhouse_errors` | clickhouse_server | severity ≥ ERROR (threshold 20 / 5 m) |
| `victoriametrics_errors` | victoriametrics | severity ≥ ERROR (threshold 10 / 5 m) |
| `nginx_5xx` | nginx_access | severity ≥ ERROR, which the preset sets for status ≥ 500 (threshold 20 / 5 m) |
| `pmm_component_crash` | supervisord | `exited: .* \(exit status` |

Threshold 1 per 5 m unless noted. Patterns are checked against real lines from the incident generator before they
ship.

## Part 3: logs API and Logs page

### 3.1 Logs API

`api/logs/v1/logs.proto`, `LogsService`, viewer access:

| RPC | HTTP | Returns |
|---|---|---|
| `SearchLogs` | `POST /v1/logs:search` | Lines, newest first: timestamp, node, service, preset, severity, body, attributes, rule; 200 per page with a timestamp cursor |
| `GetLogVolume` | `POST /v1/logs:getVolume` | Counts per time bucket (about 100 buckets) by severity class: error (≥ 17), warning (13–16), info (9–12), other (< 9) |
| `GetLogPatterns` | `POST /v1/logs:getPatterns` | Top 50 patterns: count, max severity, first and last seen, one sample, `new` when absent from the previous window of equal length |
| `GetLogContext` | `POST /v1/logs:getContext` | ±50 lines from the same node and file around one entry |

Every request takes the same filter: time range (required; default 1 h, maximum the retention), node names, service
names, presets, minimum severity, text (case-insensitive contains, served by the `tokenbf` index on `Body`), rule,
pattern.

### 3.2 Queries

pmm-managed builds parameterized ClickHouse SQL; the browser never sends SQL. Queries run with
`max_execution_time = 30` and `readonly = 1`.

Pattern = message with variable parts replaced, applied in this order: quoted strings → `'?'`, UUIDs, hex ≥ 8 chars,
IPv4 addresses, numbers → `?`; then the first 200 characters. One Go helper builds the expression for both
`GetLogPatterns` and the pattern filter, so they agree.

### 3.3 Access control

Viewers read logs; admins change rules. Log rows carry no environment or cluster labels, so LBAC filters cannot be
applied to them. `/v1/logs` joins `lbacPrefixes` in `managed/services/grafana/auth_server.go`, so the auth server
sets `X-Proxy-Filter` for users with LBAC filters; when that header is present the logs API returns
`PermissionDenied` instead of possibly someone else's logs.

### 3.4 Page

Route `/pmm-ui/logs`, a top-level **Logs** item in the left menu. Built with MUI and `@percona/peak-ui`; no new
dependencies.

- **Filter bar:** the time-range control from native QAN, node and service pickers from inventory, log type
  (preset), minimum severity, search box. Filters live in the URL, so alert links open the right view.
- **Volume chart:** bars stacked by severity, drawn like QAN's sparklines; drag to zoom the time range.
- **Lines tab:** time, node, service, severity (coloured), message; a chip on rule-matched lines. Expanding a row shows
  its attributes, **Show context**, **Ask Assistant** (opens the Assistant chat with the line and time window as
  context), **Open dashboard** (the node's dashboard at that time) and **Open QAN** (when the row has a service).
  **Load older** fetches the next page.
- **Patterns tab:** pattern, count, severity, first and last seen, `new` badge; clicking one filters Lines by it.
- **Rules tab:** rules with scope, match, enabled and matches in 1 h / 24 h. Admins create and edit rules in a
  dialog with **Preview**. Each rule links to its lines and to **Create alert rule** (Alerting with its template
  selected).
- **Empty state:** when nothing is collected, a link to Settings → OTEL → Log collectors.

### 3.5 Assistant

A `pmm_logs_patterns` tool in the shipped Holmes config calls `GetLogPatterns`, so the Assistant reads summarized
patterns instead of writing SQL against `otel.logs`. The `otel-logs-analysis` skill is updated to use it and the
`pmm.rule` attribute.

## Testing

- **Unit:** collector config rendering (rules → OTTL, escaping, server receivers, nginx filter); rule validation and
  built-in sync; template generation; query builders (SQL and arguments for every filter); pattern expression;
  UI components (Vitest).
- **API tests** (`api-tests/logs/`): rules CRUD and preview, search, volume, patterns, context against a live server.
- **Live, on a CHAOS VM with the incident generator (`dev/adre/issues/`):**
  - Every scenario with a rule must increase that rule's metric inside its window; matches outside incident windows
    are false positives. Report a per-rule table.
  - An alert created from a log-rule template fires during its scenario, and **Investigate** works from it.
  - Server logs from every `/srv/logs` component reach `otel.logs` with the right preset; restarting the collector
    neither loses nor duplicates lines.
  - PostgreSQL errors arrive as one row with DETAIL and STATEMENT.
  - Logs page: each tab, filters, context, alert link; screenshots.
  - Collector CPU and memory with rules off and on; logs API latency on 7 days of test-server data.

## Not in v1

- Mapping a log file to one service when a node runs several.
- Rules spanning several lines or entries (same PID within 1 s); would need scheduled queries.
- LBAC for logs (needs labels on log rows).
- Live tail; saved searches; log-derived dashboards in Grafana.
- Automatic creation of alert rules from log rules.
