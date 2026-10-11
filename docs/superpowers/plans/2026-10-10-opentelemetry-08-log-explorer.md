# OTel 08: Native Log Explorer in PMM UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Admins search, filter, chart and read logs on a native PMM UI page built with `@percona/peak-ui`, without going through Grafana. The page:
- shows a severity histogram;
- filters by node, PMM service, app, source, severity and text, with counts;
- has a details pane with surrounding lines;
- tails live;
- has shareable URLs;
- links to service dashboards and QAN.

The same API becomes the LBAC-enforced read path in plan 06.

**Architecture:**
- A new `LogsService` in pmm-managed builds parameterized ClickHouse SQL from structured filters. It reads `otel.logs` as the read-only `otel_reader` user and resolves node and service names from inventory.
- The PMM UI page `/logs` uses TanStack Query hooks over that API, and peak-ui components (`Table`, `Chip`, `CodeBlock`, `AutoCompleteInput`, `DateTimePickerInput`, `useDetailsPaneNavigation`, `usePerconaTableUrlState`, illustrations).
- The page follows the RTA overview page (`ui/apps/pmm/src/pages/rta/overview/`).
- It is Admin-only in this phase (ADR-07). Plan 06 opens it to restricted users once LBAC applies.

**Tech Stack:** Go (pmm-managed, clickhouse-go), protobuf/grpc-gateway, React, MUI v7, `@percona/peak-ui` 1.0.26+, `@mui/x-charts`, TanStack Query, Vitest.

**Spec:**
- [analysis](../specs/2026-10-10-opentelemetry-analysis.md) §7 and decision D20;
- ADR-07, ADR-18, and ADR-21 (proposed);
- the [roadmap](2026-10-10-opentelemetry-00-roadmap.md) for the shared names.
- [PMM-15722](https://perconadev.atlassian.net/browse/PMM-15722) (Logs explorer, under PMM-15567). Where its acceptance criteria differ from this plan, the ticket is current.

**Depends on:** plan 01, Tasks 2–4 (`otel.logs`, `otel_reader`, settings), and plan 02 (data flowing). It can run in parallel with plan 03.

## Global Constraints

The roadmap's constraints apply, together with `ui/AGENTS.md` and peak-ui's `AGENTS.md` and `llms.txt`. In addition:

**Query safety:**
- Clients send structured filters only, never SQL. Every value is a bound parameter.
- Every query has a time range: at most 31 days, and `end` after `start`.
- Every query runs with `SETTINGS max_execution_time = 30` and a 35 s context deadline, as `otel_reader`. That user can only `SELECT` on `otel.*` (plan 01, Task 2).

**Limits:**
- `limit` defaults to 200 and is capped at 1000.
- The histogram has at most 120 buckets.
- Each facet returns at most 50 values.

**Severity classes** follow OTel `SeverityNumber` ranges:

| Class | `SeverityNumber` |
|---|---|
| TRACE | 1–4 |
| DEBUG | 5–8 |
| INFO | 9–12 |
| WARN | 13–16 |
| ERROR | 17–20 |
| FATAL | 21–24 |
| UNKNOWN | 0 |

**Phase 1 access:** rule `"/v1/logs": admin`, and the page uses `<Page roles={[OrgRole.Admin]}>`.

**UI rules:**
- peak-ui components first, MUI for the rest. No ad-hoc CSS.
- Severity colours come from the theme palette (`error`, `warning`, `informative`, `neutral`). Chart colours come from `semanticTokens*.charts`.
- Load the `dataviz` skill before building the histogram.

## Review Focus

1. **A log line of 64 KiB, or one with no newline at all.** Expected: the table shows one truncated line (CSS ellipsis, no layout break), and the details pane shows the whole body in a wrapping `CodeBlock`. Test in Task 4 (`renders very long body truncated in table and full in pane`).
2. **Many lines sharing the same nanosecond timestamp across a page boundary.** Expected: no duplicates and no gaps when loading older pages. Test in Task 1 (`TestSearchCursorSameTimestamp`).
3. **A search text with `%`, `_`, `'` or `\`.** Expected: a literal substring match, case-insensitive. Test in Task 1 (`TestSearchTextIsLiteral`).
4. **OTEL disabled, or no data yet.** Expected: the page shows the "turned off" illustration linking to Settings → OTEL, or the "nothing found" illustration, never an error. Test in Task 3 (`shows disabled state`, `shows empty state`).
5. **Live tail left running for hours.** Expected: it polls only for rows newer than the newest one shown, keeps at most 1000 rows in memory (the oldest drop off), and pauses while the tab is hidden. Test in Task 5 (`live tail appends and caps`, `pauses when hidden`).

---

### Task 1: `LogsService.Search` and `Context`

**Files:**
- Create:
  - `api/logs/v1/logs.proto` (package `logs.v1`)
  - `managed/services/logs/service.go`
  - `managed/services/logs/query.go`
  - `managed/services/logs/cursor.go`
  - `managed/services/logs/deps.go`
  - `managed/services/logs/*_test.go`
  - `api-tests/logs/search_test.go`
- Modify:
  - `api/Makefile` (register the domain the way realtimeanalytics is registered at :22,38,57,75,118)
  - `managed/cmd/pmm-managed/main.go`: an `otel_reader` ClickHouse pool from `PMM_CLICKHOUSE_OTEL_READER_USER/PASSWORD` (max 4 connections), and gRPC and gateway registration
  - `managed/services/grafana/auth_server.go`: `"/v1/logs": admin`

**Interfaces:**
- **Proto:**
  - `enum Severity { SEVERITY_UNSPECIFIED = 0; SEVERITY_TRACE = 1; SEVERITY_DEBUG = 2; SEVERITY_INFO = 3; SEVERITY_WARN = 4; SEVERITY_ERROR = 5; SEVERITY_FATAL = 6; }`
  - `message Filters { google.protobuf.Timestamp start = 1; google.protobuf.Timestamp end = 2; repeated string node_ids = 3; repeated string service_ids = 4; repeated string apps = 5; repeated string sources = 6; repeated Severity severities = 7; string text = 8; }`
  - `message LogRecord { google.protobuf.Timestamp timestamp = 1; Severity severity = 2; string severity_text = 3; string body = 4; string app = 5; string node_id = 6; string node_name = 7; string service_id = 8; string service_name = 9; string source = 10; string trace_id = 11; string span_id = 12; map<string,string> attributes = 13; map<string,string> resource_attributes = 14; }`
  - `rpc Search(SearchRequest{Filters filters = 1; uint32 limit = 2; string cursor = 3;}) returns (SearchResponse{repeated LogRecord records = 1; string next_cursor = 2;})` at `POST /v1/logs:search`. Rows come newest first.
  - `rpc Context(ContextRequest{string node_id = 1; string source = 2; string app = 3; google.protobuf.Timestamp timestamp = 4; uint32 before = 5; uint32 after = 6;}) returns (ContextResponse{repeated LogRecord records = 1;})` at `POST /v1/logs:context`. `before` and `after` default to 50 and are capped at 200. It returns lines from the same node and source (or app) around the timestamp.
- **Go:**
  - `func BuildSearchQuery(f *logsv1.Filters, limit uint32, c Cursor) (string, []any, error)` lives in `query.go`.
    - It filters on `TimestampTime`, `PmmNodeId IN ?`, `PmmServiceId IN ?`, `ServiceName IN ?` and `coalesce(LogAttributes['log.file.path'], LogAttributes['_SYSTEMD_UNIT']) IN ?`.
    - Severity is a `multiIf` over the `SeverityNumber` ranges, and text is `positionCaseInsensitive(Body, ?) > 0`.
    - It orders with `ORDER BY Timestamp DESC`, and ends with `LIMIT ? SETTINGS max_execution_time = 30`.
  - `type Cursor struct { BeforeNs int64; SkipAtTimestamp uint32 }`, base64-JSON encoded. A page returns rows with `Timestamp <= BeforeNs`, skipping the first `SkipAtTimestamp` rows that sit exactly at `BeforeNs`.
  - Node and service names are resolved through `models.FindNodesByIDs` and `models.FindServicesByIDs`, with a 30 s cache.
  - `ClickHouse` in `deps.go` is the same small interface as in plan 01, Task 9.

- [ ] **Step 1: Write the failing tests** (sqlmock for ClickHouse):
  ```go
  func TestBuildSearchQueryBindsEverything(t *testing.T) {
      q, args, err := BuildSearchQuery(&logsv1.Filters{Start: ts(t0), End: ts(t1), ServiceIds: []string{"s1"}, Text: "x' OR 1=1 --"}, 200, Cursor{})
      require.NoError(t, err)
      assert.NotContains(t, q, "OR 1=1")
      assert.Contains(t, args, "x' OR 1=1 --")
      assert.Contains(t, q, "SETTINGS max_execution_time = 30")
  }
  func TestSearchRejectsMissingOrLongRange(t *testing.T) { /* no start → InvalidArgument; 32 days → InvalidArgument */ }
  func TestSearchLimitCapped(t *testing.T) { /* limit 5000 → 1000 bound */ }
  func TestSearchTextIsLiteral(t *testing.T) { /* "%_\\'" passed verbatim as one parameter to positionCaseInsensitive */ }
  func TestSearchCursorSameTimestamp(t *testing.T) { /* 3 rows at the same ns, limit 2: page 1 → 2 rows with cursor {ns, 2}; page 2 → the 3rd row */ }
  func TestSearchResolvesNames(t *testing.T) { /* node_id/service_id → names from inventory mock */ }
  func TestContextSameSourceAround(t *testing.T) { /* … */ }
  ```
- [ ] **Step 2: Run them.** `make gen && cd managed && go test ./services/logs/`. Expected: FAIL.
- [ ] **Step 3: Implement** the proto, the query builder, the cursor, the service, the pool and the registration.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS. Then run the api-test against a Feature Build:
  - search the server's own logs for the last 10 minutes and get records with `node_name` set;
  - a Viewer gets 403;
  - a 32-day range gets 400.
- [ ] **Step 5: Commit.**
  ```bash
  git add api/logs api/Makefile managed api-tests/logs
  git commit -s -m "PMM-15722 Add LogsService search and context API"
  ```

### Task 2: `Histogram` and `Facets`

**Files:**
- Modify:
  - `api/logs/v1/logs.proto`
  - `managed/services/logs/query.go` and `service.go`, and their tests

**Interfaces:**
- **Histogram:**
  - `rpc Histogram(HistogramRequest{Filters filters = 1;}) returns (HistogramResponse{uint32 interval_seconds = 1; repeated HistogramBucket buckets = 2;})` at `POST /v1/logs:histogram`.
  - `message HistogramBucket { google.protobuf.Timestamp start = 1; map<string,uint64> counts = 2; }`. `counts` is keyed by severity class name.
  - `func HistogramInterval(rangeDur time.Duration) time.Duration` returns the smallest of 1s, 5s, 10s, 30s, 1m, 5m, 10m, 30m, 1h, 3h, 6h, 12h and 24h that gives at most 120 buckets.
- **Facets:**
  - `rpc Facets(FacetsRequest{Filters filters = 1;}) returns (FacetsResponse{repeated FacetValue nodes = 1; repeated FacetValue services = 2; repeated FacetValue apps = 3; repeated FacetValue sources = 4; repeated FacetValue severities = 5;})` at `POST /v1/logs:facets`.
  - `message FacetValue { string value = 1; string label = 2; uint64 count = 3; }`.
  - Each facet applies every filter **except its own dimension**, the usual faceted-search rule, so a user can widen the selection.
  - Labels come from inventory names; for apps, sources and severities they are the values themselves.

- [ ] **Step 1: Write the failing tests.**
  - `TestHistogramInterval`: 15 minutes → 10s, 24 hours → 30m, 7 days → 3h.
  - `TestHistogramGroupsBySeverityClass`.
  - `TestFacetsIgnoreOwnDimension`: with `service_ids=[s1]`, the services facet query has no `PmmServiceId IN`, and the apps facet query has it.
  - `TestFacetsCappedAt50`.
- [ ] **Step 2: Run them.** `make gen && cd managed && go test ./services/logs/ -run 'Histogram|Facets'`. Expected: FAIL.
- [ ] **Step 3: Implement** both RPCs.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add api/logs managed/services/logs
  git commit -s -m "PMM-15722 Add log histogram and facets API"
  ```

### Task 3: Logs page, filters, and URL state

**Files:**
- Create:
  - `ui/apps/pmm/src/api/logs.ts`
  - `ui/apps/pmm/src/api/__mocks__/logs.ts`
  - `ui/apps/pmm/src/types/logs.types.ts`
  - `ui/apps/pmm/src/hooks/api/useLogs.ts`
  - `ui/apps/pmm/src/pages/logs/LogsExplorer.tsx`
  - `ui/apps/pmm/src/pages/logs/LogsExplorer.messages.ts`
  - `ui/apps/pmm/src/pages/logs/LogsExplorer.test.tsx`
  - `ui/apps/pmm/src/pages/logs/filters/LogsFilters.tsx`
  - `ui/apps/pmm/src/pages/logs/filters/LogsFilters.test.tsx`
  - `ui/apps/pmm/src/pages/logs/filters/useLogsUrlFilters.ts`
- Modify:
  - `ui/apps/pmm/src/router.tsx`: route `logs`, next to `rta`
  - `ui/apps/pmm/src/contexts/navigation/navigation.constants.ts`: `NAV_LOGS`, with url `${PMM_NEW_NAV_PATH}/logs`, a "Technical Preview" badge, and visible to Admins only

**Interfaces:**
- **Hooks:**
  - `useLogsSearch(filters, { enabled })` is an infinite query over the cursor, with key `['logs','search',filters]`.
  - `useLogsHistogram(filters)` and `useLogsFacets(filters)`.
  - `useLogsContext(ref)`, enabled only while the details pane is open.
- **`useLogsUrlFilters()`** reads and writes `from`, `to` (ISO or relative such as `now-15m`), `node`, `service`, `app`, `source`, `severity` and `q` in the URL query, so every view is a shareable link. Grafana dashboards deep-link into it (Task 6).
- **Filter bar:**
  - quick ranges (15m, 1h, 6h, 24h, 7d) with an absolute range through `DateTimePickerInput`;
  - an `AutoCompleteInput` multi-select each for nodes, services, apps and sources, whose options come from `useLogsFacets` and show counts;
  - severity toggle chips, using peak-ui `Chip` with theme colours;
  - a debounced (300 ms) text field;
  - the `AutoRefreshSelect` reused from `pages/rta/overview/auto-refresh-select`.
- **States:**
  - OTEL turned off (from `useSettings`): `RealTimeDatabaseOffIllustration`, with a link to `/settings/otel`;
  - no rows: `NothingFoundIllustration`;
  - an API error: an MUI `Alert` with the server message.

- [ ] **Step 1: Write the failing tests.**
  - `reads filters from URL`, `writes filters to URL`.
  - `facet options show counts`.
  - `shows disabled state`, `shows empty state`.
  - `debounces text search`.
- [ ] **Step 2: Run them.** `cd ui && pnpm --filter pmm test -- logs`. Expected: FAIL.
- [ ] **Step 3: Implement** the API module, the hooks, the page shell, the filters, the route and the nav entry.
- [ ] **Step 4: Run them again.** Same command, then `make lint format-check`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm/src
  git commit -s -m "PMM-15722 Add the Logs page with filters"
  ```

### Task 4: Histogram, log table, details pane

**Files:**
- Create:
  - `ui/apps/pmm/src/pages/logs/histogram/LogsHistogram.tsx` and its test
  - `ui/apps/pmm/src/pages/logs/table/LogsTable.tsx`, `LogsTable.constants.tsx` and their test
  - `ui/apps/pmm/src/pages/logs/details-pane/LogDetailsPane.tsx` and its test
- Modify:
  - `ui/apps/pmm/package.json`: add `@mui/x-charts` (MIT licence; D22) with `pnpm --filter pmm add @mui/x-charts`, at the major version that matches the app's `@mui/material`. Note the licence in the PR description, because CI runs a licence check.

**Interfaces:**
- **Histogram:**
  - stacked bars per severity class, coloured from the theme;
  - clicking a bar narrows `from`/`to` to that bucket;
  - a tooltip shows the counts.
  - It is drawn with `@mui/x-charts` `BarChart` (stacked series, one per severity class), coloured from `semanticTokensLight.charts` and `semanticTokensDark.charts` through the theme.
- **Table:**
  - peak-ui `Table`, with `tableName="logs"` and manual pagination;
  - columns: time (local, with milliseconds), severity `Chip`, node, service or app, source, and message (one line, ellipsis);
  - a "Load older" row action that calls the infinite query's `fetchNextPage`.
- **Details pane:**
  - an MUI `Drawer`, with `useDetailsPaneNavigation` for next and previous (the pattern in `pages/rta/overview/RealtimeOverview.tsx:62`);
  - the full body in a `CodeBlock` with `wrap` and `copyable`, and an attributes table;
  - a "Show surrounding lines" section from `useLogsContext`;
  - links:
    - "Open service dashboard", by service type, with the time range;
    - "Open QAN for this service and time" (`/graph/d/pmm-qan/pmm-query-analytics?var-service_name=…&from=…&to=…`);
    - "View trace", hidden until plan 05.

- [ ] **Step 1: Write the failing tests.**
  - `histogram click narrows time range`.
  - `renders very long body truncated in table and full in pane`.
  - `severity chip colour by class`.
  - `pane next/previous walks loaded rows`.
  - `context lines load when pane opens`.
  - `QAN link carries service and range`.
- [ ] **Step 2: Run them.** `cd ui && pnpm --filter pmm test -- logs`. Expected: FAIL.
- [ ] **Step 3: Implement** the histogram, the table and the details pane.
- [ ] **Step 4: Run them again.** Same command, then `make lint format-check`. Expected: PASS. Then on a Feature Build, run `pmm-qa:ui-evidence` to capture screenshots of the page in light and dark mode for the PR.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm
  git commit -s -m "PMM-15722 Add log histogram, table and details pane"
  ```

### Task 5: Live tail and export

**Files:**
- Create:
  - `ui/apps/pmm/src/pages/logs/useLiveTail.ts` and its test
  - `ui/apps/pmm/src/pages/logs/export/exportLogs.ts` and its test, following `pages/rta/overview/export/exportRtaQueriesToCsv.ts`

**Interfaces:**
- **Live tail:** `useLiveTail(filters, intervalMs)`.
  - On each tick it calls `Search` with `start = newest shown timestamp`, then prepends the new rows, de-duplicated.
  - It keeps at most 1000 rows.
  - It stops while `document.visibilityState === 'hidden'`.
  - It turns off as soon as the user picks an absolute range.
- **Export:** `exportLogs(records, format: 'csv' | 'ndjson')` downloads the rows currently loaded, with every column and the full body.

- [ ] **Step 1: Write the failing tests.**
  - `live tail appends and caps`.
  - `pauses when hidden`.
  - `absolute range disables live tail`.
  - `csv escapes quotes and newlines`.
- [ ] **Step 2: Run them.** `cd ui && pnpm --filter pmm test -- logs`. Expected: FAIL.
- [ ] **Step 3: Implement** the live tail and the export.
- [ ] **Step 4: Run them again.** Same command, then `make lint format-check`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm/src/pages/logs
  git commit -s -m "PMM-15722 Add live tail and export to Logs"
  ```

### Task 6: Links into the explorer

**Files:**
- Modify:
  - the service summary dashboards and Node Summary (the same files as plan 03, Task 6): their log links open the native explorer, `/logs?service=${service_id}&from=${__from}&to=${__to}`, or `node=${node_id}` from Node Summary;
  - `dashboards/dashboards/OTel/Logs.json`: a panel link "Open in Logs explorer" with the same filters;
  - the Grafana fork's Inventory Services tab (`/srv/runner/4/percona/grafana/public/app/percona/inventory/`): a "Logs" row action for services that have log sources.

  Use the PMM UI base path for the URL (`PMM_NEW_NAV_PATH` in `ui/apps/pmm/src/lib/constants.ts`, and the path Grafana uses to reach the PMM UI). Check it on a Feature Build.

- [ ] **Step 1: Write the failing check.** A dashboards test, `test_service_dashboards_link_to_logs_explorer`, finds a link to `/logs?service=` in each service summary dashboard. Run `python3 -m pytest dashboards/misc -k logs_explorer`. Expected: FAIL.
- [ ] **Step 2: Implement** the links, the Inventory row action (with its Grafana fork test), and run `cleanup-dash.py`.
- [ ] **Step 3: Run the checks again.** The pytest, `cleanup-dash.py --check-only` on every changed dashboard, and the fork test (`yarn test Services`). Expected: PASS. Then on a Feature Build, check that a link from the MySQL summary opens the explorer filtered to that service and time range.
- [ ] **Step 4: Commit** in each repo: `git commit -s -m "PMM-15722 Link dashboards and Inventory to Logs"`.
