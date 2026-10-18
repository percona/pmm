# Logs: rule-based detection and the Logs page — design

Status: Part 1 implemented (2026-10-16); Part 3 implemented (2026-10-17, plan `logs-part3-plan.md`); Part 2B deferred (2026-10-17).
Branch `tibi-holmes` (PMM-7).

## Goal

Give PMM users one place to read, filter and make sense of every collected log — database logs from agents
and PMM Server's own logs — and find known problems in them automatically with predefined, editable rules, not by
running an LLM over every line. A rule match is a normal PMM alert, so notifications, the Alerts page and the
Assistant's *Investigate* work unchanged; high-signal built-in rules alert with no setup.

Both halves weigh equally: detection (rules → alerts → Assistant) and a log viewer for incidents.

## Current state (before Part 1)

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
                                                                                 └─▶ Grafana alert rule per alert-on log rule ─▶ Alerts, Assistant
pmm-managed: log rules (PostgreSQL) ─▶ server collector config + Grafana alert rules (provisioning API)
pmm-managed: logs API over ClickHouse ─▶ pmm-ui /logs (Rules in 2B; Lines, Patterns in Part 3)
```

The `pmm.rule` attribute ties the halves together: an alert names its rule, and the Logs page shows exactly the
lines that matched.

Work is split into parts, each with its own implementation plan, built in this order: Part 1 (log data, done),
Part 3 (logs API and Logs page), then Part 2B (rules and detection, with alerting) and Part 2A (database log
discovery).

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
- Added during Part 1's live verification: vmproxy's per-request info lines are dropped like nginx's below-400 lines
  (about 2,500 a minute; nginx logs the same requests), and every operator on the server receivers handles errors
  quietly (`send_quiet` / `drop_quiet`), because the collector reads its own log and logged parse errors fed back
  into it.

### 1.5 Out of scope

Changing log retention does not alter the existing table's TTL (`schema.go` has no `MODIFY TTL`). Separate ticket.

## Part 2B: rules and detection

**Deferred (2026-10-17)** until alerting is revisited; Part 3 is built first. Open issue to settle then: on this
branch pmm-managed holds no Grafana credential for background calls (every Grafana call uses the caller's request
auth, Grafana has no auth proxy, and the only stored token is the Assistant's, which exists only once the Assistant is
provisioned), so "alerts with no setup" needs one of: a dedicated service account minted from the first admin request,
a Grafana provisioning file with a Grafana restart on each alert change (upstream master's pattern for its built-in
HA alerts), or an explicit "Enable log alerts" step. A draft plan for the parts that do not depend on it is in
`dev/otel/logs-part2b-plan.md`. The preset tag (2B.3) and the nginx error-line severity (2B.9) move to Part 3.

Built before Part 2A. Decisions from the 2026-10-17 review: a rule match alerts with no setup for a high-signal
subset of built-in rules; each rule owns its alert (no alert templates); the Assistant gets a bounded tool for the
lines behind a log-rule alert.

### 2B.1 Rule model

Table `log_rules` (new migration, reform model `LogRule`):

| Column | Type | Notes |
|---|---|---|
| `name` | varchar PK | `^[a-z0-9_]{1,64}$`, used in the metric and the alert |
| `summary` | varchar | shown in the UI and the alert |
| `presets` | jsonb | non-empty list of preset names the rule applies to (`raw` allowed) |
| `min_severity` | int | OTel SeverityNumber; 0 = any |
| `match_type` | varchar | `contains` (case-insensitive) or `regex` |
| `match` | varchar | ≤ 1024 chars |
| `enabled` | bool | a disabled rule neither tags nor counts lines |
| `alert_enabled` | bool | the rule's Grafana alert exists (2B.4) |
| `alert_severity` | varchar | `info`, `warning`, `error`, `critical` |
| `alert_threshold` | double | matches per window that fire the alert |
| `alert_window` | interval | 1 m to 1 h |
| `source` | varchar | `builtin` or `user` |
| `deleted_at` | timestamptz | soft delete for built-ins, so upgrades do not bring them back |
| `created_at`, `updated_at`, `updated_by` | | |

Validation on save, in order:

1. Go `regexp` compiles the pattern (`contains` is compiled as `(?i)` + `regexp.QuoteMeta`). The collector's OTTL
   `IsMatch` and ClickHouse's `match()` use the same RE2 syntax.
2. `match` may be empty only when `min_severity` > 0 (a severity-only rule); presets must exist; at most 200 enabled
   rules.
3. pmm-managed renders the server collector config with the candidate rule set and runs `otelcol-contrib validate` on
   it (Part 1's validator). A rejection fails the save with `InvalidArgument` and the validator's message.

Preset edits (`ChangeLogParserPreset`, `AddLogParserPreset`) get the same collector check: a one-receiver config with
the candidate operators, so presets the server config does not use (`mysql_error`, `mongodb`) are checked too.

Matching order: rules with a `match` first, then severity-only rules, each group by name; the first match wins. A
failed PostgreSQL login therefore counts as `postgresql_auth_failed`, not as `postgresql_fatal_panic`.

### 2B.2 Built-in rules and sync

Built-in rules ship as YAML in `managed/data/log-rules/*.yml` (with `alert_enabled` per rule) and are synced at
startup with the pattern of `planSkillSync` (`managed/services/adre/deployment/seed.go`):

- New shipped rule → inserted.
- Shipped rule changed and the row is still `source = builtin` → updated.
- An admin edit sets `source = user`; upgrades leave it alone.
- A deleted built-in keeps `deleted_at`; upgrades do not restore it.
- **Reset** puts a built-in back to its shipped definition: `source = builtin`, `deleted_at` cleared.

### 2B.3 Collector pipeline

pmm-managed renders the server collector config from the enabled rules. Changes go through Part 1's
validate-then-swap and the pending collector restart; changes within 10 s are batched into one restart. During a
restart agents keep sending (their OTLP exporters retry) and the server's own receivers resume from their stored read
positions. A restart resets the cumulative counters; `increase()` handles resets.

- Every receiver adds `attributes["log.preset"]` through an `add` operator: one per preset receiver on agents, one per
  file receiver on the server (`raw` for unparsed files). The agent config is rendered by pmm-managed, so existing
  agents need no upgrade.
- `transform/pmm_rules` holds one OTTL statement per enabled rule, in matching order, and sets
  `attributes["pmm.rule"]`; the attribute is stored in `otel.logs`, so Part 3 shows exactly the matched lines. Rule
  strings are escaped as OTTL string literals; unit tests cover quotes, backslashes and Unicode. Rules match `body`,
  which is the message for presets that extract one.
- `count/pmm_rules` turns tagged lines into `pmm_log_rule_matches`, `deltatocumulative` makes it cumulative, and
  `prometheusremotewrite/pmm_rules` writes it to VictoriaMetrics.
- HA: each node's collector counts only the lines it receives, so the remote-write exporter sets the external label
  `pmm_collector` (the HA node ID; `pmm-server` when standalone) and alerts sum across nodes. Without it two nodes
  would write the same series.

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
    external_labels:
      pmm_collector: pmm-server
service:
  pipelines:
    logs:
      receivers: [otlp, filelog/srv_...]
      processors: [memory_limiter, transform, transform/pmm_rules, batch]
      exporters: [clickhouse/logs, count/pmm_rules]
    metrics/pmm_rules:
      receivers: [count/pmm_rules]
      processors: [deltatocumulative, batch]
      exporters: [prometheusremotewrite/pmm_rules]
```

`service_name` on agent-collected lines is the node name until a log source is tied to its service (Part 2A does that
for discovered sources).

Verified in a spike on the test server, before anything is built on it:

1. `count` groups by resource attributes (`node_name`, `service.name`); if not, `transform/pmm_rules` copies them onto
   the record.
2. `count` emits delta sums, so `deltatocumulative` is needed before remote write, and counts arrive correctly.
3. The series name and labels in VictoriaMetrics (expected
   `pmm_log_rule_matches_total{pmm_rule, node_name, service_name, pmm_collector}`) and the write URL.
4. Collector CPU with 30 rules at the test server's log rate.
5. pmm-managed can create, update and delete Grafana alert rules through the provisioning HTTP API with a server-side
   credential, without restarting Grafana, and they show read-only in Alerting.

The HA label cannot be tested on one VM; it is verified in the next HA test.

### 2B.4 Alerts

Each rule owns its alert: a rule that is enabled with `alert_enabled` has exactly one Grafana alert rule. There are no
alert templates for log rules and no `TEMPLATE_SOURCE_LOG_RULE`.

- Folder **PMM Log Rules** (fixed UID `pmm-log-rules`), one rule group evaluated every 1 m.
- Grafana rule: UID `pmm-log-` + the first 12 hex characters of `sha1(name)`; title = summary; label
  `pmm_log_rule=<name>` and `severity` from the rule; query on PMM's VictoriaMetrics datasource (UID resolved as in
  `provisioning_datasource.go`):
  `sum by (node_name, service_name) (increase(pmm_log_rule_matches_total{pmm_rule="<name>"}[<window>])) >= <threshold>`;
  `for: 0s`; no data = OK (no matches is normal); execution error = Error. Each alert carries `node_name` and
  `service_name`, so notification policies can route on them. Annotations: summary and match count (a link to the
  matching lines comes with Part 3).
- Sync: pmm-managed compares the enabled, alert-on rules with the rules in the folder and creates, updates or deletes
  through Grafana's provisioning HTTP API (`/api/v1/provisioning/alert-rules`). Rules created this way carry
  provenance, so Grafana shows them read-only; they are edited through the log rule. The provisioning *file* path used
  for the HA and components bundles is not used: applying it restarts Grafana.
- Runs after every rule change, at startup, and every 5 min (repairs drift such as a deleted folder). In HA only the
  leader syncs; alert rules live in the shared Grafana database.
- When Grafana is unreachable the rule save still succeeds (the database is the source of truth) and the sync retries
  with backoff. `ListLogRules` reports each rule's alert state: `synced`, `pending` or `failed` with the reason.
- With the collector disabled, alert rules stay (they cannot fire) and the Rules tab shows a banner.
- Deleting a rule deletes its alert; silences that referenced it no longer match anything.
- Quieting a rule for one service uses Alertmanager silences; per-service exceptions in rules are not in v1.

### 2B.5 Rules API

`api/logs/v1/rules.proto`, `LogRulesService`; viewer for reads, admin for writes, both added to the auth server's
rules for `/v1/logs/`:

| RPC | HTTP |
|---|---|
| `ListLogRules` | `GET /v1/logs/rules` — each rule with matches in the last 1 h and 24 h (VictoriaMetrics), alert state, and built-in / edited / custom |
| `CreateLogRule` | `POST /v1/logs/rules` |
| `UpdateLogRule` | `PUT /v1/logs/rules/{name}` — includes enable/disable, alert on/off and thresholds |
| `DeleteLogRule` | `DELETE /v1/logs/rules/{name}` — custom rules are removed, built-ins hidden (`deleted_at`) |
| `ResetLogRule` | `POST /v1/logs/rules/{name}:reset` — built-ins only |
| `PreviewLogRule` | `POST /v1/logs/rules:preview` — a rule body (saved or not) and a range of 1 h or 24 h; returns the match count and up to 20 sample lines |
| `ListLogRuleMatches` | `GET /v1/logs/rules/{name}/matches` — see 2B.7 |

Preview and matches query `otel.logs` with ClickHouse `match()` / `positionCaseInsensitive()`, filtered by
`LogAttributes['log.preset']` and `SeverityNumber`, with `max_execution_time = 10`, `readonly = 1` and a rows-read
limit. They see only lines stored after 2B ships, since older rows have no preset tag.

### 2B.6 Rules tab

Route `/pmm-ui/logs`, a top-level **Logs** item in the left menu; in 2B the page has only the **Rules** tab (Part 3
adds Lines and Patterns to the same page). Layout from the 2026-10-16 mockups, with the per-rule alert toggle in place
of **Create alert rule**; MUI and `@percona/peak-ui`, API module and
TanStack Query hooks, Vitest tests, per `ui/AGENTS.md`.

- Table: summary, presets, match, matches 1 h / 24 h, enable switch, alert on/off with severity and threshold, alert
  state, built-in / edited / custom badge.
- Editor drawer: the fields and a live preview (debounced) with sample lines; server validation errors inline,
  including the collector's rejection message.
- Actions: delete; reset for built-ins; confirmation before deleting a rule whose alert is on.
- Banner when the OTEL collector is disabled.

### 2B.7 Assistant

The Assistant already reads `otel.logs` through `pmm_clickhouse_query` (toolset `pmm-clickhouse`); 2B adds a bounded
tool and corrects that toolset's guidance.

- `ListLogRuleMatches`: query `service_name`, `from`/`to` (≤ 24 h), `limit` (default 20, max 100). Returns the newest
  matched lines: timestamp, severity, file name, body (cut to 4 KB). With `around=<timestamp>` it returns up to 20
  lines before and after one match from the same service and file. Same limits as preview; filters on `ServiceName`,
  the table's sort key. Part 3's logs API extends this rather than replacing it.
- Holmes tool `pmm_log_rule_matches` in `pmm-clickhouse`, calling the endpoint like the other PMM tools (curl with
  the API token).
- Investigation guidance: for an alert with a `pmm_log_rule` label, the first step is `pmm_log_rule_matches` for that
  rule, service and window; the report quotes the lines (for a deadlock, its DETAIL and STATEMENT).
- `pmm_clickhouse_query` guidance for logs: filter by `ServiceName` and `TimestampTime` first, use `SeverityNumber`,
  not `SeverityText`, use `LogAttributes['pmm.rule']` for rule matches, and avoid `JSONExtract` over wide ranges.

### 2B.8 Built-in rules (v1)

| Rule | Preset | Match | Threshold | Alert |
|---|---|---|---|---|
| `mysql_deadlock` | mysql_error | `Transactions deadlock detected` | 1 / 5 m | on, error |
| `mysql_out_of_memory` | mysql_error | `Out of memory` | 1 / 5 m | on, critical |
| `mysql_access_denied` | mysql_error | `Access denied for user` | 10 / 5 m | off |
| `mysql_replica_sql_error` | mysql_error | `Replica SQL for channel` + ERROR | 1 / 5 m | off |
| `mysql_semaphore_wait` | mysql_error | `Semaphore wait has lasted` | 1 / 5 m | off |
| `mysql_crash_recovery` | mysql_error | `crash recovery` | 1 / 5 m | off |
| `postgresql_deadlock` | postgres | `deadlock detected` | 1 / 5 m | on, error |
| `postgresql_too_many_clients` | postgres | `too many clients already` | 1 / 5 m | on, critical |
| `postgresql_shm_exhausted` | postgres | `could not resize shared memory segment` | 1 / 5 m | on, error |
| `postgresql_fatal_panic` | postgres | severity ≥ FATAL | 1 / 5 m | on, critical |
| `postgresql_unclean_shutdown` | postgres | `was not properly shut down` | 1 / 5 m | on, critical |
| `postgresql_auth_failed` | postgres | `password authentication failed` | 10 / 5 m | off |
| `postgresql_statement_timeout` | postgres | `canceling statement due to statement timeout` | 5 / 5 m | off |
| `postgresql_lock_wait` | postgres | `still waiting for .*Lock` | 1 / 5 m | off |
| `postgresql_checkpoints_too_frequent` | postgres | `checkpoints are occurring too frequently` | 1 / 5 m | off |
| `postgresql_temp_file` | postgres | `temporary file: path` | 20 / 5 m | off |
| `mongodb_election` | mongodb | `Starting an election` | 1 / 5 m | on, warning |
| `mongodb_stepdown` | mongodb | `Stepping down from primary` | 1 / 5 m | on, warning |
| `mongodb_auth_failed` | mongodb | `Failed to authenticate` | 10 / 5 m | off |
| `mongodb_query_executor_error` | mongodb | `Plan executor error` | 1 / 5 m | off |
| `mongodb_wiredtiger_error` | mongodb | `WiredTiger error` | 1 / 5 m | off |
| `pmm_component_crash` | supervisord | `exited: .* \(exit status` | 1 / 5 m | on, error |
| `pmm_managed_errors` | pmm_managed | severity ≥ ERROR | 20 / 5 m | off |
| `clickhouse_errors` | clickhouse_server | severity ≥ ERROR | 20 / 5 m | off |
| `victoriametrics_errors` | victoriametrics | severity ≥ ERROR | 10 / 5 m | off |
| `nginx_5xx` | nginx_access | severity ≥ ERROR (the preset sets it for status ≥ 500) | 20 / 5 m | off |

Every pattern is checked against real lines from the incident generator before it ships. A default-on rule that
matches during normal operation (for example `postgresql_fatal_panic` on "role does not exist" or "terminating
connection due to administrator command") is narrowed or switched to alert off, and the table is updated.

### 2B.9 Part 1 follow-ups

- nginx error-format lines (`2026/10/16 20:41:58 [warn] …`) in `nginx.log` get a severity: a new migration adds a
  second parser to the built-in `nginx_access` preset, selected with the stanza `if:` field, that sets their level.
  Access lines are unchanged; only an unedited preset is updated.
- `dev/otel/README.md`: lines failing a parse are dropped, not stored, for presets that use `on_error: drop`
  (`grafana`, `otel_collector`).

### 2B.10 Done

Verified on the test server:

1. The spike's five checks pass, or the design is changed first.
2. For every log-based scenario of the incident generator (`dev/adre/issues/`), its rule's matches appear on the
   Rules tab within 2 min; where the rule is alert-on, the Grafana alert fires with no setup. Matches outside
   incident windows are false positives; report a per-rule table.
3. Switching an alert on or off shows in Alerting within seconds, and Grafana's uptime shows no restart.
4. **Investigate** on a deadlock alert quotes the DETAIL and STATEMENT lines.
5. Collector CPU with all built-ins enabled stays near the Part 1 level, and its log stays free of errors.

## Part 2A: database log discovery

Built after 2B, with its own spec and plan. Decisions so far (2026-10-17):

- Log collection is on by default for newly added MySQL, PostgreSQL and MongoDB services; `--disable-logs` (and a
  global setting) turns it off. Already-registered services are offered collection in the UI.
- pmm-agent asks the database where its log is (`log_error`; `logging_collector`, `log_directory`, `log_filename` or
  `pg_current_logfile()`; `systemLog.path`), tries known default locations, and tails the file with the right preset,
  tied to its service.
- When no readable file is found, the service shows why and the fix: the setting to change, or
  `--log-file=<path>`. No journald or Docker log readers in v1.
- Needs a new pmm-agent; older agents keep the manual `pmm-admin add otel` setup. Tested with a Feature Build.

## Part 3: logs API and Logs page

Built next (2026-10-17), without rules: no rule filter, rule chips or Rules tab until Part 2B.

### 3.0 Prerequisites taken from Part 2B

- Every collector receiver adds `attributes["log.preset"]` (the preset name, `raw` for unparsed files) through an `add`
  operator: one per preset receiver on agents, one per file receiver on the server. The agent config is rendered by
  pmm-managed, so existing agents need no upgrade. This feeds the log-type filter and the Patterns tab.
- nginx error-format lines in `nginx.log` (`2026/10/16 20:41:58 [warn] …`) get a severity: a new migration adds a second
  parser to the built-in `nginx_access` preset, selected with the stanza `if:` field; access lines are unchanged; only
  an unedited preset is updated.

### 3.1 Logs API

`api/logs/v1/logs.proto`, `LogsService`, viewer access:

| RPC | HTTP | Returns |
|---|---|---|
| `SearchLogs` | `POST /v1/logs:search` | Lines, newest first: timestamp, node, source, preset, severity, body, attributes; 200 per page with a timestamp cursor |
| `GetLogVolume` | `POST /v1/logs:getVolume` | Counts per time bucket (about 100 buckets) by severity class: error (≥ 17), warning (13–16), info (9–12), other (< 9) |
| `GetLogPatterns` | `POST /v1/logs:getPatterns` | Top 50 patterns: count, max severity, first and last seen, one sample, `new` when absent from the previous window of equal length |
| `GetLogContext` | `POST /v1/logs:getContext` | ±50 lines from the same node and file around one entry |
| `GetLogFacets` | `POST /v1/logs:getFacets` | Distinct node names, sources (`ServiceName`) and presets in the range with line counts, top 200 each, for the pickers |
| `FindLogQuery` | `POST /v1/logs:findQuery` | The QAN query (query ID, service, fingerprint) a PostgreSQL statement line ran, or none |

Every request takes the same filter: time range (required; default 1 h, maximum the retention), node names, sources
(`ServiceName`: the component for PMM Server's own logs, the node name for agent logs until log sources are tied to
services), presets, minimum severity, text (case-insensitive contains, served by the `tokenbf` index on `Body`),
pattern. Rows written before the preset tag existed match no preset filter.

### 3.2 Queries

pmm-managed builds parameterized ClickHouse SQL; the browser never sends SQL. Queries run with
`max_execution_time = 30` and `readonly = 1`.

Pattern = message with variable parts replaced, applied in this order (`managed/otel/patterns.go`): logfmt `msg="…"`
values are unquoted, also one nested escaped level, so their text stays; quoted strings → `'?'`; timestamps, UUIDs,
hex ≥ 8 chars, IPv4 addresses, underscore-joined numbers (ClickHouse part names), numbers with a unit (`4048ms`) and
numbers → `?`; lists of masked values collapse to one; then the first 200 characters. `otel.logs.PatternHash` is a
MATERIALIZED `cityHash64` of that expression, so masking runs once per line at insert. `GetLogPatterns` groups by the
hash and masks only the sample lines it returns; the pattern filter compares hashes. pmm-managed sets the column's
expression at startup, which rewrites no data: parts written before the column existed, or before the rules changed,
keep computing or holding the old value until they expire or `ALTER TABLE otel.logs MATERIALIZE COLUMN PatternHash`
runs.

The OpenTelemetry `drain` processor (alpha) was measured against the hash on adre-pmm7 on 2026-10-18 and not adopted.
The test replayed 2.2 M lines (6 h) through otelcol-contrib 0.163.1 and ran one hour live (0.37 M lines):

- The hash found 18,510 patterns in the replay, and Drain with the same masks found 534 templates. ClickHouse server
  lines alone made 16,840 of the hashes, because the masks miss part names (`20261018_161096_1`); Grafana `t=`
  timestamps and numbers with a unit (MySQL `4717ms`) split patterns the same way. On database logs they agree
  (MongoDB 134 and 134, PostgreSQL 160 and 139).
- Templates change as Drain learns. 4.4% of the replayed lines (1.9% live) carry an older template than the latest one
  for their hash, and templates that keep the hour of a timestamp (`18T12:<num>:50Z`) are replaced as time passes. Each
  collector learns its own tree, so HA collectors and restarts disagree unless a snapshot is shared.
- With the quoted-string mask both merge logfmt messages, since `msg="…"` becomes one token: one pmm-agent pattern
  held 1,140 different messages, and Drain cut pmm-managed's 181 patterns to 28 templates.
- Each processor costs about 25 µs of CPU per line; at 75 lines/s the live collector used 6–7% of a core with and
  without it.

The rules above came out of this test. On a later 6 h window (2.3 M lines) they give 933 patterns where the earlier
rules gave 28,921 (ClickHouse server 16,119 → 57, Grafana 11,808 → 176, MySQL error logs 231 → 12), while the logfmt
components gain the messages they had merged (pmm-managed 56 → 130). They cost 4.2 µs of CPU per line on one core,
against 2.0 µs for the earlier rules.

`FindLogQuery` matches a PostgreSQL `statement:`, `execute …:` or `duration: … statement:` line to a QAN fingerprint
by text (`managed/services/logs/qan.go`). PostgreSQL's `%Q` log prefix is no help: statement lines log `qid=0`, and
services monitored through pg_stat_monitor store its own query ID in QAN, not the core one. It reads the node's
PostgreSQL queries in `pmm.metrics` 30 minutes either side of the line, most executed first, and normalizes both
sides: comments and case go, literals, `$n`, `true`, `false`, `null` and the `extract` field become `?`, lists of `?`
collapse. Fingerprints of 1024 characters or more match as a prefix, since pg_stat_monitor cuts them at
`pgsm_query_max_len` (1024 at least) and pmm-agent at its maximum query length (2048 by default; a setting under
1024 leaves its cut fingerprints unmatched). On adre-pmm7 on 2026-10-18 this matched 98 of 100 statement patterns,
99.9% of statement lines other than `statement: ;`; the miss differs from QAN's text only in a column alias, which
PostgreSQL leaves out of the query ID.

### 3.3 Access control

Viewers read logs. Log rows carry no environment or cluster labels, so LBAC filters cannot be
applied to them. `/v1/logs` joins `lbacPrefixes` in `managed/services/grafana/auth_server.go`, so the auth server
sets `X-Proxy-Filter` for users with LBAC filters; when that header is present the logs API returns
`PermissionDenied` instead of possibly someone else's logs.

### 3.4 Page

Route `/pmm-ui/logs`, a top-level **Logs** item in the left menu. Built with MUI and `@percona/peak-ui`; no new
dependencies.

The filter bar and the chart stay pinned; only the lines or patterns below them scroll. The page has no heading.

- **Filter bar:** the time-range control from native QAN, node and source pickers filled from `GetLogFacets`, log type
  (preset), minimum severity, search box. Filters live in the URL, so alert links open the right view; hidden patterns
  are `hide=<pattern hash>` and show as one chip.
- **Volume chart:** bars stacked by severity, drawn like QAN's sparklines; drag to zoom the time range; collapsible.
  The last opened line's pattern is drawn over the dimmed bars on its own scale (right axis), with a marker at the
  line's time and a chip that clears it.
- **Lines tab:** time, node, source, severity (coloured), message; a logfmt line shows its `msg` followed by its other
  fields. Expanding a row shows the line's pattern with its count in the range and against the previous one, first and
  last seen, and the three sources with the most of it (`GetLogPatterns` with the line's `pattern_hash`), **Show only
  this pattern**, **Hide this pattern**, its attributes, **Show context**, **Ask Assistant** (opens the Assistant chat
  with the line and time window as context), **Open dashboard** (the service's or node's dashboard at
  that time), **Open QAN** (PostgreSQL statement lines only: the query's details in QAN, native when enabled, found
  by `FindLogQuery` when the row opens; without a match, QAN for the service, or for the node when it runs more than
  one PostgreSQL service), **Copy line** and **Copy link**. Older lines load as the list reaches its end.
- **Patterns tab:** pattern, count, severity, first and last seen, `new` badge; clicking one filters Lines by it.
- **Empty state:** when nothing is collected, a link to Settings → OTEL → Log collectors.

### 3.5 Assistant

A `pmm_logs_patterns` tool in the shipped Holmes config calls `GetLogPatterns`, so the Assistant reads summarized
patterns instead of writing SQL against `otel.logs`. The `pmm_clickhouse_query` guidance for logs is corrected:
filter by `ServiceName` and `TimestampTime` first, use `SeverityNumber`, not `SeverityText`, and avoid `JSONExtract`
over wide ranges. The `otel-logs-analysis` skill is updated to use the tool.

## Testing

Part 3 (Part 2B's checks are in 2B.10):

- **Unit:** the preset tag on server and agent receivers; query builders (SQL and arguments for every filter, the
  cursor, facets); the pattern expression; LBAC denial; UI components and hooks (Vitest).
- **Database:** the nginx preset migration (run in the test server's container).
- **API tests** (`api-tests/logs/`): search, volume, patterns, context and facets against a live server, viewer access,
  and `PermissionDenied` for a user with LBAC filters.
- **Live, on the test VM with the incident generator (`dev/adre/issues/`):** for incident windows from
  `incidents.jsonl`, the Logs page finds the expected evidence (deadlock DETAIL/STATEMENT, `Access denied`, election
  lines) through filters, patterns and context; nginx error lines carry a severity; screenshots of each tab; logs API
  latency on 7 days of test-server data (target: under 2 s for search and volume over 24 h).

## Not in v1

- Mapping a log file to one service when a node runs several.
- QAN links from MySQL slow-log and MongoDB slow-query lines.
- Rules spanning several lines or entries (same PID within 1 s); would need scheduled queries.
- LBAC for logs (needs labels on log rows).
- Live tail; saved searches; log-derived dashboards in Grafana.
- Per-service exceptions in a rule (use silences).
- Rules suggested by the Assistant.
- journald and Docker log readers (Part 2A reports the fix instead).
