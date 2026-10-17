---
name: otel-logs-analysis
description: "Search logs PMM collects into ClickHouse otel.logs: which log files a node ships, error lines and spikes, message patterns, counts per minute, correlation with an incident window. MySQL error log, PostgreSQL, MongoDB, nginx, PMM Server component logs."
---

# Log search in otel.logs

## When to use
The user asks for log lines or errors, or metrics show a restart, crash or error burst and you need the log evidence. Only nodes that ship files have logs (`pmm-admin add otel logs`); RDS/Azure database logs stay at the provider (cloud-rds-azure).

## Patterns first
`pmm_logs_patterns` (start, end RFC3339; optional node_names, sources, min_severity, text) groups the range's lines into up to 50 patterns (values replaced by `?`) with count, max severity, first/last seen, a sample and `is_new` (absent from the preceding range of the same length). Call it before any SQL: new and error-level patterns are usually the answer, and they tell you which raw lines to fetch. Users can open the same view in PMM: `/pmm-ui/logs/patterns?from=<ms>&to=<ms>&source=<source>`; lines are under `/pmm-ui/logs?…&severity=17`.

## Schema (`pmm_clickhouse_query`, `database=otel`, table `logs`)
- `ServiceName` and `TimestampTime` are the primary key: filter on both first. `ServiceName` is the node name for client logs and the component (pmm-managed, nginx, grafana, qan-api2, …) for PMM Server logs.
- `Timestamp` DateTime64(9), event time parsed by the preset. `TimestampTime` DateTime is in the primary key: always bound it.
- `SeverityNumber`: OTel scale, 9-12 info, 13-16 warn, 17-20 error, 21-24 fatal, 0 = not parsed. `SeverityText` keeps the source spelling (`Error`, `E`, `error`, `Warning`), so filter on the number.
- `Body`: the message. mysql_error, postgres and mongodb presets strip the timestamp/level prefix; pmm_* logs keep the full logfmt line.
- `ResourceAttributes`: inventory labels such as `node_name`, `node_id`, `agent_id`. PMM Server's own logs have `node_name = 'pmm-server'`.
- `LogAttributes`: `log.file.name` (basename), `log.preset` (parser preset), plus preset fields: mysql_error `code`, `component`, `thread_id`; postgres `pid`, `user`, `database`; mongodb `component`, `context`, `attr`; pmm_managed/pmm_agent `level`, `msg`, `component`.
- Presets: mysql_error, syslog_mysql_systemd, postgres, mongodb, nginx_access, nginx_error, grafana, pmm_managed, pmm_agent, clickhouse_server, otel_collector, supervisord, or raw. mysql_error and postgres join continuation lines (multi-line statements, InnoDB dumps) into one entry; other presets store one line per row.
- Server files: pmm-managed.log, pmm-agent.log, qan-api2.log, vmproxy.log, grafana.log, nginx.log, clickhouse-server.log, postgresql14.log, otel-collector.log, supervisord.log.
- Retention defaults to 7 days. Queries stop after 30 s and return at most 1000 rows: narrow the window first. Never run `JSONExtract` or regexes over `Body` across a wide range.

## Steps
1. Node name from inventory or the alert's `node_name` label.
2. Which files does it ship?
   ```sql
   SELECT LogAttributes['log.file.name'] AS file, count() AS n, min(Timestamp) AS first, max(Timestamp) AS last
   FROM logs
   WHERE ServiceName = '<node>' AND TimestampTime >= now() - INTERVAL 1 DAY
   GROUP BY file ORDER BY n DESC LIMIT 20
   ```
   No rows: the node ships no logs (or nothing in 24 h). Tell the user how to add them (Remediation) and continue with metrics.
3. `pmm_logs_patterns` for the window, `sources=<node>`, `min_severity=13` for warnings and errors.
4. Raw lines of the patterns that matter:
   ```sql
   SELECT Timestamp, SeverityText, LogAttributes['log.file.name'] AS file, substring(Body, 1, 400) AS msg
   FROM logs
   WHERE ServiceName = '<node>'
     AND TimestampTime BETWEEN '<YYYY-MM-DD HH:MM:SS>' AND '<YYYY-MM-DD HH:MM:SS>'
     AND (SeverityNumber >= 17 OR (SeverityNumber = 0 AND Body ILIKE '%error%'))
   ORDER BY Timestamp LIMIT 100
   ```
5. Counts per minute for one pattern, to line up with a metric change point from `pmm_metrics_snapshot`:
   ```sql
   SELECT toStartOfMinute(Timestamp) AS minute, count() AS n
   FROM logs WHERE <node and window> AND Body ILIKE '%<pattern>%'
   GROUP BY minute ORDER BY minute LIMIT 500
   ```
   `hasToken(Body, 'deadlock')` uses the token index (whole word, case-sensitive) and is faster than ILIKE on long windows.
6. Quote 3-10 representative lines in time order, with file names, and say how many similar lines there were.

## Interpretation
What to search per source:
- MySQL (mysql_error): `[ERROR]` with `code` (`MY-xxxxx`), "Aborted connection", "Too many connections", "Deadlock", "semaphore wait", "Out of memory", "got signal", "ready for connections" (a start).
- PostgreSQL: FATAL, PANIC, "could not", "deadlock detected", "canceling statement due to statement timeout", "checkpoints are occurring too frequently", "terminated by signal", "database system is ready to accept connections" (a start).
- MongoDB: `LogAttributes['component']` REPL, ELECTION, STORAGE, NETWORK; "Slow query", "Transition to", "WiredTiger error".
- PMM Server: pmm-managed `level=error`; "Connected pmm-agent" (agent reconnects); clickhouse-server `<Error>`; supervisord "exited" / "spawned" (component restarts).
A burst starting before the metric change usually points at the cause; one starting after is a symptom. Startup lines in the window confirm a restart: look just before them for the reason (signal, OOM, assertion).

## Remediation
Ship a log file (self-managed client): `pmm-admin add otel logs --log-sources=/var/log/mysql/error.log:mysql_error` (comma-separate several `path:preset` pairs). pmm-agent must be able to read the file. Kubernetes and managed cloud: logs come from the pod / provider console, not this table.
