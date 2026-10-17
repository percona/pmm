# Logs Part 3: logs API and Logs page — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `/pmm-ui/logs` page where users read, filter and make sense of every collected log (volume chart, Lines
with context, Patterns), served by a read-only logs API over ClickHouse `otel.logs`, plus an Assistant tool for
patterns.

**Architecture:** A new `LogsService` (`api/logs/v1/logs.proto`, implemented in `managed/services/logs`) builds
parameterized ClickHouse SQL from one filter and runs it on the existing `otel` pool with a 30 s limit. Collector
receivers tag each line with its preset so the page can filter by log type. pmm-ui gets a Logs page with URL-held
filters, an SVG volume chart, a Lines tab and a Patterns tab.

**Tech Stack:** Go (pmm-managed, gRPC + grpc-gateway, `database/sql` ClickHouse driver), otelcol-contrib filelog
operators, React + MUI + `@percona/peak-ui` + TanStack Query, Vitest, HolmesGPT config and skills.

**Spec:** `dev/otel/logs-design.md`, Part 3 (3.0–3.5) and the Part 3 testing list.

## Global Constraints

- Branch `PMM-7-tibi-holmes`, pushed to `origin tibi-holmes`; commits `PMM-7 <summary ≤50 chars>`, signed off
  (`git commit -s`), `git add` and `git commit` in separate calls (a hook blocks them combined).
- Migrations are forward-only: add `149`; never edit 127–148.
- Proto changes are additive; run `make gen` after editing `.proto` files or mocked interfaces; never hand-edit
  generated files; `buf lint` STANDARD rules apply (service suffix `Service`, request/response naming).
- Go conventions from `AGENTS.md`: no inline `if err := …`, no named returns, comments on their own line, one-sentence
  doc comments, no `%q` in error or log messages, logrus `*logrus.Entry`, `status.Error` with gRPC codes.
- Every ClickHouse query is parameterized (never string-built from user input) and ends with
  `SETTINGS max_execution_time = 30, readonly = 1`.
- Viewers read logs; a request carrying `X-Proxy-Filter` (LBAC) gets `PermissionDenied`.
- Severity classes: error ≥ 17, warning 13–16, info 9–12, other < 9 (OTel `SeverityNumber`).
- Attribute names: preset `LogAttributes['log.preset']`, file `LogAttributes['log.file.name']`, node
  `ResourceAttributes['node_name']`, source = `ServiceName`.
- No new npm or Go dependencies. UI: MUI + peak-ui, TanStack Query hooks in `ui/apps/pmm/src/hooks/api/`, camelCase in
  TypeScript (the axios instance converts to snake_case).
- The runner host runs no PostgreSQL, ClickHouse or Docker: `testdb` tests and live checks run on CHAOS VM
  `adre-pmm7` (`/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh`); compile test
  binaries locally and run them inside `pmm-server` as user `pmm`. Never print VM secrets.
- Lint: Go `bin/golangci-lint run <pkgs>`; UI `cd ui && make lint && make format-check && make test`.

## Review Focus

- Search text with `%`, `_`, quotes, backslashes and regex characters is matched literally and never reaches SQL
  unparameterized (Task 3, `TestBuildWhere_TextLiteral`).
- Many lines share one timestamp (bursts, multi-line imports): paging returns each line exactly once
  (Task 3, `TestSearchQuery_Cursor`; Task 11 live with a 1,000-line burst).
- A 7-day range on a busy server hits the 30 s limit: the API answers `DeadlineExceeded` with advice to narrow the
  range, and the page shows that message instead of spinning (Task 5, `TestSearchLogs_Timeout`; Task 8 error state).
- Multi-kilobyte bodies (PostgreSQL statements, MongoDB JSON): lists carry at most 4 KB per body with a truncation
  flag; context shows up to 64 KB (Task 3 `TestSearchQuery_Truncates`, Task 9 expanded row).
- A user with LBAC filters gets `PermissionDenied` from every logs call and the page explains it (Task 5,
  `TestLBACDenied`; Task 6 api-test; Task 8).

---

### Task 1: Tag every line with its preset

**Files:**
- Modify: `managed/otel/server_logs.go` (`writeServerReceiver`), `managed/otel/server_logs_test.go`
- Modify: `managed/services/agents/otelcollector.go` (receiver loop), its test

**Interfaces:**
- Produces: `LogAttributes['log.preset']` on every new row (preset name; `raw` for `pmm-init.log`,
  `filelog/srv_other` and agents' raw receiver). Task 3 filters on it.

- [ ] **Step 1: Write the failing tests.** `server_logs_test.go`: every receiver's operators end with
  `{type: add, field: 'attributes["log.preset"]', value: <preset>, on_error: send_quiet}`; `filelog/srv_pmm_init` and
  `filelog/srv_other` have exactly that one operator. Agents' test: each `filelog/preset_<name>` receiver's operators
  end with the same `add` (`raw` for the raw receiver, which then has only that operator).
- [ ] **Step 2: Run** `go test ./managed/otel/... ./managed/services/agents/...` (agents' testdb tests on the VM).
  Expected: FAIL.
- [ ] **Step 3: Implement.** Server: append the `add` before `quietOperators` runs (so it gets `send_quiet`); pass the
  preset name into `writeServerReceiver`. Agents: append it to each receiver's operators, creating `operators:` for raw.
- [ ] **Step 4: Run** the same tests. Expected: PASS.
- [ ] **Step 5: Live check.** Deploy pmm-managed to `adre-pmm7`; after 2 minutes,
  `SELECT LogAttributes['log.preset'] p, count() FROM otel.logs WHERE TimestampTime > now() - INTERVAL 2 MINUTE GROUP BY p`.
  Expected: `postgres`, `mongodb`, `mysql_error` (if MySQL logged), `pmm_managed`, `nginx_access`, `raw`, … and no empty
  value for rows newer than the deploy; `otel-collector.log` has no new error lines.
- [ ] **Step 6: Commit** — `PMM-7 Tag log lines with their preset`.

### Task 2: nginx error lines get a severity (migration 149)

**Files:**
- Modify: `managed/models/database.go` (migration `149`)
- Test: `managed/models/log_parser_preset_migration_test.go` (new subtest)

**Interfaces:**
- Produces: built-in `nginx_access` parses `key=value` access lines and `2026/10/16 20:41:58 [warn] 114#114: …` error
  lines; error lines get `attributes.level` and a severity.

- [ ] **Step 1: Write the failing subtest** `nginx_access parses error-format lines` in
  `TestLogParserPresetsServerLogs`: unmarshal the preset's operators; the `key_value_parser` and the `regex_parser`
  each have an `if` containing `body matches`; the `regex_parser` regex applied to
  `2026/10/16 20:41:58 [warn] 114#114: *259446 a client request body is buffered to a temporary file` captures
  `level` = `warn`; the YAML contains `fatal: [crit, alert, emerg]`.
- [ ] **Step 2: Run** on the VM (`go test -c ./managed/models`, run inside `pmm-server` as `pmm` with
  `-test.run TestLogParserPresetsServerLogs`). Expected: FAIL (no `regex_parser`).
- [ ] **Step 3: Add migration 149**: `UPDATE log_parser_presets SET operator_yaml = $yaml$…$yaml$, updated_at = NOW()
  WHERE name = 'nginx_access' AND built_in AND operator_yaml = $old$<text migration 128 inserted>$old$` (confirm with
  `grep -n nginx_access managed/models/database.go` that no later migration changed it). Operators: `key_value_parser`,
  `time_parser` (`attributes.time`, layout `'2006-01-02T15:04:05Z07:00'`) and the level `add`, each with
  `if: 'body matches "^time="'`; then a `regex_parser` with the `nginx_error` preset's regex and
  `if: 'body matches "^\\d{4}/\\d{2}/\\d{2} "'`, and its `time_parser` (layout `'2006/01/02 15:04:05'`, same `if`); then
  one `severity_parser` on `attributes.level`: `debug: debug`, `info: [info, notice]`, `warn: warn`, `error: error`,
  `fatal: [crit, alert, emerg]`. One comment line above the migration.
- [ ] **Step 4: Run** the same command, then the whole `managed/models` suite on the VM. Expected: PASS.
- [ ] **Step 5: Live check.** Deploy; trigger an nginx error line (e.g. a request body over the buffer size, or wait for
  the "buffered to a temporary file" warnings); `SELECT SeverityText, SeverityNumber, count() FROM otel.logs WHERE
  ServiceName = 'nginx' AND TimestampTime > <deploy> GROUP BY 1, 2`. Expected: error-format lines have `warn`/13 or
  `error`/17, none at 0; access lines unchanged.
- [ ] **Step 6: Commit** — `PMM-7 Give nginx error lines a severity`.

### Task 3: Query builders

**Files:**
- Create: `managed/services/logs/query.go`, `managed/services/logs/query_test.go`

**Interfaces:**
- Produces (all pure; no database):

```go
// Filter is the shared filter of every logs call.
type Filter struct {
	Start, End  time.Time
	NodeNames   []string
	Sources     []string
	Presets     []string
	MinSeverity int32
	Text        string
	Pattern     string
}

// Cursor positions a search page after the last returned line.
type Cursor struct {
	Time time.Time
	Hash uint64
}

func (c Cursor) Encode() string
func DecodeCursor(s string) (Cursor, error)

// PatternExpr returns the ClickHouse expression that turns column into its pattern; GetLogPatterns and the pattern filter both use it.
func PatternExpr(column string) string

func buildWhere(f Filter) (string, []any)
func searchQuery(f Filter, after *Cursor, limit int) (string, []any)
func volumeQuery(f Filter) (string, []any, time.Duration)
func patternsQuery(f Filter) (string, []any)
func contextQuery(at time.Time, nodeName, fileName string, n int) (string, []any)
func facetsQueries(f Filter) (nodes, sources, presets string, args []any)
```

Rules the code must follow (from the spec; one place each):
- `buildWhere`: `TimestampTime BETWEEN toDateTime(?) AND toDateTime(?)` and `Timestamp BETWEEN ? AND ?`;
  `ServiceName IN (?)`, `ResourceAttributes['node_name'] IN (?)`, `LogAttributes['log.preset'] IN (?)` when non-empty;
  `SeverityNumber >= ?` when > 0; `positionCaseInsensitiveUTF8(Body, ?) > 0` for text (the `tokenbf` index is
  case-sensitive and cannot serve this; time and `ServiceName` prune instead, checked in Task 11);
  `PatternExpr('Body') = ?` for pattern.
- `searchQuery`: selects `Timestamp`, node, `ServiceName`, preset, file name, `SeverityNumber`, `SeverityText`,
  `substringUTF8(Body, 1, 4096)`, `lengthUTF8(Body) > 4096`, `LogAttributes`, and
  `cityHash64(Body, ServiceName, LogAttributes['log.file.name']) AS h`; `ORDER BY Timestamp DESC, h DESC LIMIT ?`;
  with a cursor adds `(Timestamp, h) < (?, ?)`. Limit default 200, max 500.
- `volumeQuery`: step = the smallest of 1s, 5s, 10s, 30s, 1m, 5m, 10m, 30m, 1h, 3h, 6h, 12h, 1d that gives ≤ 100 buckets;
  `toStartOfInterval(Timestamp, INTERVAL <step> SECOND)` with `countIf` per severity class.
- `PatternExpr`: in this order, quoted strings (`'…'`, `"…"`) → `'?'`, UUIDs → `?`, hex runs ≥ 8 → `?`, IPv4 → `?`,
  numbers → `?`, then `substringUTF8(…, 1, 200)`.
- `patternsQuery`: over `[Start − (End − Start), End]`, `countIf` in the current and previous window per pattern,
  `max(SeverityNumber)`, `min`/`max(Timestamp)` and `any(Body)` over the current window, `HAVING current > 0`,
  `ORDER BY current DESC LIMIT 50`; `is_new` = previous = 0.
- `contextQuery`: the `n` lines at or before `at` and the `n` after, same node and file, bodies up to 65536 chars.
- `facetsQueries`: time range only (the pickers list everything in the range); `GROUP BY` value, `ORDER BY count()
  DESC LIMIT 200`.
- Every query ends with `SETTINGS max_execution_time = 30, readonly = 1`.

- [ ] **Step 1: Write the failing tests** (`query_test.go`): `TestBuildWhere_Empty` (only the time conditions, 4 args);
  `TestBuildWhere_All` (every filter, exact SQL and args); `TestBuildWhere_TextLiteral` (text `50%_off "x" \d+` appears
  only as an argument, never in the SQL string); `TestSearchQuery_Cursor` (round-trip `Encode`/`DecodeCursor`; the
  query has `(Timestamp, h) < (?, ?)` with the cursor's values; a malformed cursor is an error);
  `TestSearchQuery_Truncates` (contains `substringUTF8(Body, 1, 4096)` and the truncation flag);
  `TestSearchQuery_Limit` (0 → 200, 1000 → 500); `TestVolumeQuery_Step` (1 h → 1 m, 24 h → 30 m, 7 d → 3 h, 10 min → 10 s);
  `TestPatternsQuery_PreviousWindow` (args start at `Start − (End − Start)`); `TestPatternExpr_Shared` (the patterns
  query and `buildWhere` with a pattern contain the identical `PatternExpr('Body')`); `TestQueries_Settings` (every
  builder's SQL ends with the settings clause).
- [ ] **Step 2: Run** `go test ./managed/services/logs/...`. Expected: FAIL (undefined).
- [ ] **Step 3: Implement** `query.go`.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Check the expressions in ClickHouse.** On the VM, run `SELECT <PatternExpr> FROM (SELECT arrayJoin([...])
  AS Body)` for: `connection 42 from 10.0.0.7 user 'bob'`, `id 3f2a9c1e-1b2c-4d5e-8f90-123456789abc done`,
  `hash deadbeef1234 ok`, `took 12.5 ms`. Expected: `connection ? from ? user '?'`, `id ? done`, `hash ? ok`,
  `took ? ms`. Run one `searchQuery` and one `volumeQuery` built for the last hour against `otel.logs`. Expected:
  rows, no error.
- [ ] **Step 6: Commit** — `PMM-7 Build ClickHouse queries for the logs API`.

### Task 4: Logs API definition

**Files:**
- Create: `api/logs/v1/logs.proto`, `api/logs/v1/json/header.json` (copy `api/realtimeanalytics/v1/json/header.json`,
  title `PMM Logs API`)
- Modify: `api/Makefile` (add `logs/v1` to the gen `SPECS`, the `clean` `SPECS`, and both swagger mixins),
  `api/descriptor.bin` (`make -C api descriptors`); generated files via `make gen`

**Interfaces:**
- Produces: package `logs.v1`, service `LogsService` (shape copied from `api/realtimeanalytics/v1/realtimeanalytics.proto`):

| RPC | HTTP (`body: "*"`) | Request | Response |
|---|---|---|---|
| `SearchLogs` | `POST /v1/logs:search` | `LogFilter filter`, `string cursor`, `int32 limit` (0–500) | `repeated LogLine lines`, `string next_cursor` |
| `GetLogVolume` | `POST /v1/logs:getVolume` | `LogFilter filter` | `repeated VolumeBucket buckets`, `google.protobuf.Duration step` |
| `GetLogPatterns` | `POST /v1/logs:getPatterns` | `LogFilter filter` | `repeated LogPattern patterns` |
| `GetLogContext` | `POST /v1/logs:getContext` | `Timestamp time` (required), `string node_name`, `string file_name` (min_len 1), `int32 lines` (0–100; 0 = 50) | `repeated LogLine lines`, `int32 anchor_index` |
| `GetLogFacets` | `POST /v1/logs:getFacets` | `LogFilter filter` | `repeated Facet node_names`, `repeated Facet sources`, `repeated Facet presets` |

Messages: `LogFilter {Timestamp start (required); Timestamp end (required); repeated string node_names; repeated string
sources; repeated string presets; int32 min_severity; string text; string pattern}`; `LogLine {Timestamp time; string
node_name; string source; string preset; string file_name; int32 severity_number; string severity_text; string body;
bool body_truncated; map<string,string> attributes}`; `VolumeBucket {Timestamp start; int64 error; int64 warning; int64
info; int64 other}`; `LogPattern {string pattern; int64 count; int32 max_severity_number; Timestamp first_seen; Timestamp
last_seen; string sample; bool is_new}`; `Facet {string value; int64 count}`.

- [ ] **Step 1: Write** the proto, `header.json` and the `api/Makefile` entries.
- [ ] **Step 2: Run** `make gen` and `go tool buf lint -v api`. Expected: generated `api/logs/v1/*.pb.go`,
  `*.pb.gw.go`, `*.pb.validate.go`, `json/client/...`, swagger updated; lint clean; `go build ./...` succeeds.
- [ ] **Step 3: Run** `make -C api descriptors`. Expected: `api/descriptor.bin` changes (additive).
- [ ] **Step 4: Commit** — `PMM-7 Add the logs API definition` (proto, header, Makefile, descriptor and generated files).

### Task 5: Logs service, auth and wiring

**Files:**
- Create: `managed/services/logs/service.go`, `managed/services/logs/service_test.go`
- Modify: `managed/services/grafana/auth_server.go` and `auth_server_test.go`, `managed/cmd/pmm-managed/main.go`

**Interfaces:**
- Consumes: Task 3 builders; Task 4 generated `logsv1` types.
- Produces:

```go
type Service struct {
	logsv1.UnimplementedLogsServiceServer
	db *sql.DB
	l  *logrus.Entry
}

func New(db *sql.DB) *Service
```

Behavior: each RPC first checks incoming metadata `x-proxy-filter`; non-empty → `codes.PermissionDenied`
("Logs are not available to users with label-based access filters."). Converts `LogFilter` to `Filter` (end before
start → `InvalidArgument`; no retention check is needed because ClickHouse's TTL already drops older partitions), runs the builder's SQL on `db`, maps rows. A ClickHouse timeout
(error text contains `TIMEOUT_EXCEEDED` or code 159) → `codes.DeadlineExceeded` ("The query took longer than 30 s;
narrow the time range or add filters."); a bad cursor → `InvalidArgument`. `next_cursor` is set only when a full page
was returned.

Auth and wiring: `rules["/v1/logs"] = viewer` and `rules["/logs.v1."] = viewer` in `auth_server.go`; add `/v1/logs` to
`lbacPrefixes`; in `main.go` add `grpc_gateway.WithIncomingHeaderMatcher` that forwards `X-Proxy-Filter` as metadata
`x-proxy-filter` and falls back to `grpc_gateway.DefaultHeaderMatcher`; register
`logsv1.RegisterLogsServiceServer(gRPCServer, logs.New(clickhouseOTelClient))` and add
`logsv1.RegisterLogsServiceHandler` to the gateway registrar list.

- [ ] **Step 1: Write the failing tests** (`service_test.go`, go-sqlmock over `*sql.DB`): `TestSearchLogs` (two rows
  mapped to `LogLine`, `next_cursor` empty for a short page, set for a full page); `TestSearchLogs_Timeout` (driver error
  `code: 159, TIMEOUT_EXCEEDED` → `codes.DeadlineExceeded`); `TestGetLogVolume` (buckets and `step`);
  `TestGetLogPatterns` (`is_new` mapped); `TestGetLogContext` (`anchor_index` points at the line at `time`);
  `TestGetLogFacets`; `TestLBACDenied` (context with metadata `x-proxy-filter: abc` → every RPC returns
  `codes.PermissionDenied` and no query runs); `TestBadRange` (end before start → `InvalidArgument`).
  `auth_server_test.go` `TestResolveRule`: `POST /v1/logs:search` → viewer; `GET /logs.zip` stays admin.
- [ ] **Step 2: Run** `go test ./managed/services/logs/... ./managed/services/grafana/...` (live-Grafana auth tests are
  skipped locally). Expected: FAIL.
- [ ] **Step 3: Implement** `service.go`, the auth entries and the `main.go` wiring.
- [ ] **Step 4: Run** the same tests and `go build ./managed/cmd/pmm-managed`. Expected: PASS, build succeeds.
- [ ] **Step 5: Live check.** Deploy; `curl -ks -u admin:<password from the VM file> -X POST https://127.0.0.1/v1/logs:getVolume
  -d '{"filter":{"start":"<now-1h>","end":"<now>"}}'` and `:search` with `{"limit":5}` inside the VM. Expected: buckets
  and 5 lines with preset and source.
- [ ] **Step 6: Commit** — `PMM-7 Serve logs from ClickHouse`.

### Task 6: API tests

**Files:**
- Create: `api-tests/logs/logs_test.go`
- Modify: `api-tests/init.go` (import `logsClient "github.com/percona/pmm/api/logs/v1/json/client"` and set
  `logsClient.Default = logsClient.New(transport, nil)`)

**Interfaces:**
- Consumes: Task 4 generated client `logs_service`.

- [ ] **Step 1: Write the tests**: `TestSearchLogs` (last hour returns lines; paging with `next_cursor` returns no line
  twice); `TestGetLogVolume` (bucket count ≤ 100); `TestGetLogPatterns` (≤ 50, sorted by count); `TestGetLogContext`
  (around a line from search, `anchor_index` within bounds); `TestGetLogFacets` (contains source `pmm-managed`);
  `TestBadRange` (`AssertAPIErrorf(t, err, 400, codes.InvalidArgument, …)`); `TestViewerCanRead` (a viewer user created
  through Grafana's API in the test reads volume).
- [ ] **Step 2: Run** from the runner against the VM: `go test -count=1 -v ./api-tests/logs/... -pmm.server-url
  https://admin:<password>@pmmqa-adre-pmm7.tp.int.percona.com -pmm.server-insecure-tls` (password read into an env
  var from the VM, not echoed). Expected: PASS.
- [ ] **Step 3: Commit** — `PMM-7 Add logs API tests`.

### Task 7: UI API module and hooks

**Files:**
- Create: `ui/apps/pmm/src/api/logs.ts`, `ui/apps/pmm/src/api/logs.test.ts`, `ui/apps/pmm/src/hooks/api/useLogs.ts`,
  `ui/apps/pmm/src/hooks/api/useLogs.test.tsx`

**Interfaces:**
- Consumes: the REST API of Task 5.
- Produces (pattern of `api/logParserPresets.ts` and `hooks/api/useInvestigations.ts`):

```ts
export interface LogFilter { start: string; end: string; nodeNames?: string[]; sources?: string[]; presets?: string[];
  minSeverity?: number; text?: string; pattern?: string; }
export interface LogLine { time: string; nodeName: string; source: string; preset: string; fileName: string;
  severityNumber: number; severityText: string; body: string; bodyTruncated: boolean; attributes: Record<string, string>; }
export const searchLogs: (filter: LogFilter, cursor?: string, limit?: number) => Promise<{ lines: LogLine[]; nextCursor?: string }>;
export const getLogVolume: (filter: LogFilter) => Promise<{ buckets: VolumeBucket[]; step: string }>;
export const getLogPatterns: (filter: LogFilter) => Promise<LogPattern[]>;
export const getLogContext: (time: string, nodeName: string, fileName: string, lines?: number) => Promise<{ lines: LogLine[]; anchorIndex: number }>;
export const getLogFacets: (filter: LogFilter) => Promise<LogFacets>;
export interface VolumeBucket { start: string; error: number; warning: number; info: number; other: number; }
export interface LogPattern { pattern: string; count: number; maxSeverityNumber: number; firstSeen: string;
  lastSeen: string; sample: string; isNew: boolean; }
export interface LogFacet { value: string; count: number; }
export interface LogFacets { nodeNames: LogFacet[]; sources: LogFacet[]; presets: LogFacet[]; }
export const severityClass: (severityNumber: number) => 'error' | 'warning' | 'info' | 'other';

export const LOGS_KEYS = { all: ['logs'] as const, search: (f: LogFilter) => [...], volume: ..., patterns: ..., facets: ..., context: ... };
// useLogSearch pages with nextCursor (useInfiniteQuery).
export function useLogSearch(filter: LogFilter): UseInfiniteQueryResult<...>;
export function useLogVolume(filter: LogFilter): UseQueryResult<...>;
export function useLogPatterns(filter: LogFilter, enabled: boolean): UseQueryResult<...>;
export function useLogFacets(filter: LogFilter): UseQueryResult<...>;
export function useLogContext(line: LogLine | null): UseQueryResult<...>;
```

Calls pass `{ disableNotifications: true }` so the page shows errors in place.

- [ ] **Step 1: Write the failing tests**: `logs.test.ts` `severityClass` (17 → error, 13 → warning, 9 → info, 0 → other);
  `useLogs.test.tsx` with `vi.mock('api/logs')`: `useLogSearch` fetches the next page with the returned `nextCursor` and
  stops when it is absent.
- [ ] **Step 2: Run** `cd ui && make test`. Expected: FAIL.
- [ ] **Step 3: Implement** the module and hooks.
- [ ] **Step 4: Run** `cd ui && make test && make lint`. Expected: PASS.
- [ ] **Step 5: Commit** — `PMM-7 Add logs API client and hooks`.

### Task 8: Logs page, filter bar and volume chart

**Files:**
- Create: `ui/apps/pmm/src/pages/logs/LogsPage.tsx`, `LogsPage.test.tsx`, `components/LogsFilterBar.tsx`,
  `components/LogsVolumeChart.tsx`, `components/LogsVolumeChart.test.tsx`, `utils/logsFilter.ts`,
  `utils/logsFilter.test.ts`
- Modify: `ui/apps/pmm/src/lib/constants.ts` (`PMM_LOGS_PATH`), `ui/apps/pmm/src/router.tsx` (`logs/:tab?`, before the
  `graph/*` and `*` fallbacks), `ui/apps/pmm/src/contexts/navigation/navigation.constants.ts` (`NAV_LOGS`, icon
  `@mui/icons-material/ReceiptLong`) and `navigation.provider.tsx` (pushed inside `if (user && settings)`, so viewers see it)

**Interfaces:**
- Consumes: Task 7 hooks.
- Produces: `filterFromSearchParams(params: URLSearchParams, now: Date): LogFilter` and
  `searchParamsFromFilter(filter, range): URLSearchParams` in `utils/logsFilter.ts`;
  `LogsVolumeChart: FC<{ buckets: VolumeBucket[]; step: string; onZoom: (fromMs: number, toMs: number) => void }>`; URL keys `range` (`15m`, `1h`,
  `6h`, `24h`, `7d`; default `1h`) or absolute `from`/`to` (ms), repeated `node`, `source`, `preset`, `severity`, `q`,
  `pattern`; tab from the path (`lines` default, `patterns`).

Page: `<Page title="Logs" wide>`; filter bar (quick-range select plus QAN-style `datetime-local` from/to using
`toDatetimeLocalValue`/`toUnixTimestamp` from `pages/qan/utils/qanTools`, multi-select node and source pickers from
`useLogFacets`, log-type multi-select, minimum-severity select (Any, Info, Warning, Error, Fatal → 0, 9, 13, 17, 21),
search box debounced 400 ms); volume chart: hand-drawn SVG like `QanMetricSparkline` — stacked bars per bucket coloured
by severity class from the theme, hover tooltip with counts, dragging across bars sets an absolute `from`/`to`;
tabs Lines / Patterns (URL-driven like `pages/settings/Settings.tsx`). Errors from the hooks show as an `Alert` in place:
`PermissionDenied` → "Logs are not available to users with label-based access filters."; `DeadlineExceeded` → the
server's message. Empty state when the range has no lines: link to Settings → OTEL.

- [ ] **Step 1: Write the failing tests**: `logsFilter.test.ts` (round-trip of every key; `range=24h` with a fixed `now`
  gives start = now − 24 h; absolute from/to win over `range`; unknown severity ignored);
  `LogsVolumeChart.test.tsx` (renders one bar group per bucket; a drag from bucket 2 to 5 calls `onZoom` with bucket 2's
  start and bucket 5's end); `LogsPage.test.tsx` (`vi.mock('api/logs')`: renders the filter bar and the Lines tab;
  a `PermissionDenied` error shows the LBAC message; a `DeadlineExceeded` error shows the server's message; route
  `/logs/patterns` selects the Patterns tab; the nav item is present for `TEST_USER_VIEWER`).
- [ ] **Step 2: Run** `cd ui && make test`. Expected: FAIL.
- [ ] **Step 3: Implement** the page, filter bar, chart, route and nav item.
- [ ] **Step 4: Run** `cd ui && make test && make lint && make format-check`. Expected: PASS.
- [ ] **Step 5: Commit** — `PMM-7 Add the Logs page with filters and volume`.

### Task 9: Lines tab, context and links

**Files:**
- Create: `ui/apps/pmm/src/pages/logs/components/LogsLinesTab.tsx`, `LogLineRow.tsx`, `LogContextDialog.tsx`,
  `utils/logLinks.ts`, tests `LogsLinesTab.test.tsx`, `logLinks.test.ts`
- Modify: `ui/apps/pmm/src/pages/adre/components/AdreChatPanel.tsx` (prefill the input from `?ask=`, do not send),
  its test

**Interfaces:**
- Consumes: `useLogSearch`, `useLogContext` (Task 7).
- Produces in `utils/logLinks.ts`:

```ts
export const nodeDashboardUrl: (nodeName: string, fromMs: number, toMs: number) => string;
// `/graph/d/node-instance-summary/node-summary?var-node_name=<node>&from=<ms>&to=<ms>`
export const qanUrl: (nodeName: string, fromMs: number, toMs: number) => string;
// buildNativeQanPath({ from, to, filter_node_name: nodeName })
export const assistantAskUrl: (line: LogLine, fromMs: number, toMs: number) => string;
// `${PMM_ASSISTANT_PATH}?ask=<encoded question quoting the line, its node, source and time window>`
export const DATABASE_PRESETS: string[]; // mysql_error, postgres, mongodb, syslog_mysql_systemd
```

Lines tab: table rows (time in the browser's zone with a UTC tooltip, node, source, severity chip coloured by class,
first line of the message); the filter's `text` highlighted; **Load older** while `hasNextPage`. Expanding a row shows
the full returned body (with "truncated — open context for the full line" when `bodyTruncated`), attributes as
key/value chips, and actions: **Show context** (dialog with ±50 lines from `useLogContext`, anchor line highlighted and
scrolled into view), **Ask Assistant** (`assistantAskUrl`, ±15 min around the line), **Open dashboard**
(`nodeDashboardUrl`, ±15 min), **Open QAN** (only when `preset` is in `DATABASE_PRESETS`).

- [ ] **Step 1: Write the failing tests**: `logLinks.test.ts` (exact URLs for a fixed line and window; `ask` contains
  the body and the node); `LogsLinesTab.test.tsx` (`vi.mock('api/logs')`: two pages via Load older, expand shows actions,
  Open QAN hidden for preset `nginx_access` and shown for `postgres`, context dialog highlights the anchor);
  `AdreChatPanel` test (rendered with `?ask=hello` the input holds `hello` and no request is sent).
- [ ] **Step 2: Run** `cd ui && make test`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `cd ui && make test && make lint && make format-check`. Expected: PASS.
- [ ] **Step 5: Commit** — `PMM-7 Add the Lines tab with context and links`.

### Task 10: Patterns tab

**Files:**
- Create: `ui/apps/pmm/src/pages/logs/components/LogsPatternsTab.tsx`, `LogsPatternsTab.test.tsx`

**Interfaces:**
- Consumes: `useLogPatterns` (Task 7), the URL filter (Task 8).

Table: pattern (monospace, `?` placeholders dimmed), count, max severity chip, first and last seen, `new` badge;
sorted by count; clicking a row sets `pattern` in the URL and switches to the Lines tab. The query runs only while the
tab is open.

- [ ] **Step 1: Write the failing test** (`vi.mock('api/logs')`): rows render with the `new` badge only where
  `isNew`; clicking a row navigates to `/logs/lines?…&pattern=<pattern>`; `getLogPatterns` is not called while the
  Lines tab is active.
- [ ] **Step 2: Run** `cd ui && make test`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `cd ui && make test && make lint && make format-check`. Expected: PASS.
- [ ] **Step 5: Commit** — `PMM-7 Add the Patterns tab`.

### Task 11: Assistant patterns tool, docs and live verification

**Files:**
- Modify: `managed/services/adre/deployment/default_config.yaml` (`pmm-clickhouse` toolset: new tool
  `pmm_logs_patterns`, corrected logs guidance), `managed/services/adre/deployment/builtin_skills/otel-logs-analysis/SKILL.md`,
  `dev/otel/README.md` (Logs page and API), `dev/adre/README.md` (new tool)
- Test: `managed/services/adre/deployment/renderer_test.go` (new `TestDefaultConfigLogsPatternsTool`)

**Interfaces:**
- Consumes: `POST /v1/logs:getPatterns` (Task 5).

`pmm_logs_patterns` parameters: `start`, `end` (RFC3339), optional `node_names`, `sources`, `min_severity`, `text`; the
command curls the endpoint with `Authorization: Bearer ${PMM_API_TOKEN}` like the other PMM tools and returns the
patterns. Guidance: start log questions with `pmm_logs_patterns`; for raw lines use `pmm_clickhouse_query` filtered by
`ServiceName` and `TimestampTime` first, `SeverityNumber` rather than `SeverityText`, never `JSONExtract` over wide
ranges.

- [ ] **Step 1: Write the failing test** `TestDefaultConfigLogsPatternsTool` in `renderer_test.go`: `defaultConfigYAML`
  parses, and toolset `pmm-clickhouse` has a tool named `pmm_logs_patterns` whose command references
  `/v1/logs:getPatterns`.
- [ ] **Step 2: Run** `go test ./managed/services/adre/deployment/...`. Expected: FAIL.
- [ ] **Step 3: Implement** the tool, guidance, skill and README changes.
- [ ] **Step 4: Run** the same test. Expected: PASS.
- [ ] **Step 5: Live verification on `adre-pmm7`** (deploy pmm-managed and the UI dist; check that
  `adre_holmes_config.updated_by` is `system` so `SyncDefaultConfig` applied the new config, otherwise apply it
  through Configuration → Assistant):
  1. For three incident windows from `/root/adre-issues/incidents.jsonl` (a `pg_deadlocks`, a `mysql_auth_failures` or
     `mysql_deadlocks`, a `mongo_stepdown`), the page shows the evidence: Lines filtered by source and severity shows
     the deadlock with DETAIL/STATEMENT, `Access denied`, election lines; Patterns shows them, `new` where absent before.
  2. Show context around the deadlock shows the neighbouring statements; Open dashboard and Open QAN open the right
     views; Ask Assistant opens the chat with the prefilled question.
  3. Paging through a 1,000-line burst (`issues.sh run pg_auth_failures 1`) returns every line once (count the
     distinct `(time, body)` pairs across pages against a ClickHouse count).
  4. Latency: search and volume over 24 h and 7 d (the retention on the VM) measured with `curl -w %{time_total}`;
     target under 2 s for 24 h; record the figures.
  5. A viewer user sees the page; an LBAC-filtered user gets the message (create both through the UI or API).
  6. Ask the Assistant "what happened in the PostgreSQL logs in the last hour"; it calls `pmm_logs_patterns`.
  7. Screenshots of Lines, an expanded line, context, Patterns and the volume zoom (`pmm-qa:ui-evidence`), named
     `PMM-7-logs-<view>.png` in the session scratchpad.
- [ ] **Step 6: Commit** — `PMM-7 Add Assistant logs patterns tool and docs`, then push.
