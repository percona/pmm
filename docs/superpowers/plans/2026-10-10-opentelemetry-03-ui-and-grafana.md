# OTel 03: Settings → OTEL UI, Grafana Datasource, Logs Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Admins manage OTEL, parser presets and log sources in Settings → OTEL. They read logs in a Grafana Logs dashboard backed by a read-only ClickHouse datasource that only Admins can query. They see collector health and get alerts when records are dropped.

**Architecture:**
- **PMM UI** (`ui/apps/pmm`): a new Settings tab built on TanStack Query hooks over `/v1/otel/*` and `/v1/server/settings`.
- **Grafana:**
  - a provisioned `ClickHouse-OTEL` datasource with uid `pmm-otel` and user `otel_reader`;
  - a plugin-client middleware in the Grafana fork that denies non-Admin queries to `pmm-otel`, needed because nginx can't see query bodies (analysis F8);
  - dashboards in `dashboards/dashboards/OTel/`.

**Tech Stack:** React, MUI, `@percona/peak-ui`, react-hook-form with zod, TanStack Query, Vitest; Grafana dashboards JSON and the ClickHouse datasource plugin 4.21.3; Go (Grafana fork middleware, pmm-managed).

**Spec:**
- [analysis](../specs/2026-10-10-opentelemetry-analysis.md): §6.4, §7.3, F8, F12, decision D7;
- PMM-15574 (UI), PMM-15576, PMM-15593 (dashboard, alerts);
- the [roadmap](2026-10-10-opentelemetry-00-roadmap.md) for the shared names.

**Depends on:** plan 01 (settings, status, purge, `otel_reader`) and plan 02 (preset and source API, statuses, self-metrics).

## Global Constraints

The roadmap's **Shared contract** and **Global constraints** apply, together with `ui/AGENTS.md` and `dashboards/dashboards/AGENTS.md`. In addition:
- The tab shows a **Technical Preview** label.
- The datasource uses `otel_reader`, so even an Admin can't write or run DDL through it.
- A Viewer can't query `pmm-otel`, neither from a dashboard nor through `/graph/api/ds/query`.
- Dashboards pass `cleanup-dash.py --check-only`.
- No textbox variable reaches SQL unescaped: use Grafana's `${var:sqlstring}`.
- No unsigned plugin.
- **The UI doesn't edit custom labels in this plan.** `axios-case-converter` rewrites map keys (the prototype hit this); custom labels remain a pmm-admin feature.

## Review Focus

1. **`PMM_ENABLE_OTEL` set in the environment while an Admin flips the switch.** Expected: the API returns `FailedPrecondition`, the UI shows that message in an error toast, and the switch reverts. Test in Task 2 (`shows env lock error and reverts`).
2. **A preset name that is valid YAML but a duplicate, or a built-in name.** Expected: the server error text is shown in the dialog, and the dialog stays open with the input kept. Test in Task 3 (`keeps dialog open on server validation error`).
3. **A log source whose state is `NOT_READABLE` or `NOT_ALLOWED`.** Expected: the row shows the state and the reason text from the agent. Test in Task 4 (`renders source state reason`).
4. **A search text containing `'`, `\` or `%`.** Expected: the Logs dashboard query stays valid and matches literally. Test in Task 6 (a dashboard check that every `$search` use is `${search:sqlstring}`).
5. **A Viewer opening a dashboard link to Logs.** Expected: the panels show "access denied", not data, and no SQL runs. Test in Task 5 (middleware unit test for a Viewer), and an api-test in Task 6 (`TestViewerCannotQueryOtelDatasource`).

---

### Task 1: API module and TanStack hooks

**Files:**
- Create:
  - `ui/apps/pmm/src/api/otel.ts`
  - `ui/apps/pmm/src/api/__mocks__/otel.ts`
  - `ui/apps/pmm/src/hooks/api/useOtel.ts`
  - `ui/apps/pmm/src/hooks/api/useOtel.test.tsx`
  - `ui/apps/pmm/src/types/otel.types.ts`
- Modify:
  - `ui/apps/pmm/src/types/settings.types.ts`: `otel: OtelSettings` in `Settings`, and `otel?: Partial<OtelSettings>` in `UpdateSettingsPayload`

**Interfaces:**
- Produces:
  - API functions:
    - `getOtelStatus(): Promise<OtelStatus>`
    - `purgeOtel(): Promise<void>`
    - `listLogParserPresets(): Promise<LogParserPreset[]>`
    - `addLogParserPreset(p: LogParserPresetInput)`
    - `changeLogParserPreset(id: string, p: LogParserPresetInput)`
    - `removeLogParserPreset(id: string)`
    - `listLogSources(f?: { nodeId?: string; serviceId?: string }): Promise<LogSource[]>`
    - `addLogSource(p: LogSourceInput)`
    - `removeLogSources(p: { logSourceId: string })`
  - Hooks:
    - `useOtelStatus`
    - `usePurgeOtel`
    - `useLogParserPresets`
    - `useAddLogParserPreset`
    - `useChangeLogParserPreset`
    - `useRemoveLogParserPreset`
    - `useLogSources`
    - `useAddLogSource`
    - `useRemoveLogSources`
  - Query keys are `['otel', 'status']`, `['otel', 'presets']` and `['otel', 'sources']`. Each mutation invalidates the keys it changes; a preset change also invalidates the sources key, because the preset name appears in the source rows.
  - Types mirror the proto messages in camelCase.

- [ ] **Step 1: Write the failing test.** In `useOtel.test.tsx`, following `useUpdates.test.tsx`:
  - `useLogParserPresets` returns the mocked list;
  - `useRemoveLogParserPreset` calls `removeLogParserPreset('id1')` and invalidates `['otel','presets']`.
- [ ] **Step 2: Run it.** `cd ui && pnpm --filter pmm test -- useOtel`. Expected: FAIL, the module is not found.
- [ ] **Step 3: Implement** the API module, the types, the hooks and the mocks, following `api/rta.ts` and `hooks/api/useRealtime.ts`.
- [ ] **Step 4: Run it again.** Same command, then `cd ui && make lint && make format-check`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm/src
  git commit -s -m "PMM-15574 Add OTEL API hooks to PMM UI"
  ```

### Task 2: Settings → OTEL tab: settings, status, purge

**Files:**
- Create:
  - `ui/apps/pmm/src/pages/settings/components/otel/OtelTab.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/OtelSettingsForm.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/OtelSettingsForm.schema.ts`
  - `ui/apps/pmm/src/pages/settings/components/otel/OtelStatusCard.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/OtelTab.messages.ts`
  - `ui/apps/pmm/src/pages/settings/components/otel/OtelTab.test.tsx`
- Modify:
  - `ui/apps/pmm/src/pages/settings/Settings.types.ts` (add `'otel'` to `TabValue`)
  - `ui/apps/pmm/src/pages/settings/Settings.tsx` (a `<Tab data-testid="settings-tab-otel" value="otel">` with the label and a "Technical Preview" `Chip`, and the content switch)
  - `ui/apps/pmm/src/pages/settings/Settings.messages.ts`

**Interfaces:**
- Consumes `useSettings` and `useUpdateSettings` (`hooks/api/useSettings.ts`), and `useOtelStatus` and `usePurgeOtel` (Task 1).
- The zod schema:
  - `logsRetentionDays` and `tracesRetentionDays`: integers 1–365;
  - `maxDiskGb`: an integer ≥ 1;
  - `diskWatermarkPercent`: an integer 50–95.

  Submit sends only the changed fields under `otel`, so the partial update keeps everything else.
- The status card shows:
  - `diskBytes`, formatted;
  - `diskUsedPercent`;
  - a warning `Alert` with `ingestPausedReason` when paused;
  - an error `Alert` with `lastError` when it is set.

  Purge asks for confirmation (copy `ServiceNowDisconnect.tsx:71-97`) before calling `usePurgeOtel`.

- [ ] **Step 1: Write the failing tests.**
  - `renders Technical Preview label`.
  - `submits only changed retention`: change logs retention to 3 and assert that `updateSettings` was called with `{ otel: { logsRetentionDays: 3 } }`.
  - `rejects retention 0`.
  - `shows env lock error and reverts`: the mocked `updateSettings` rejects with a `FailedPrecondition` body; assert that the toast shows the message and the switch is back on.
  - `shows paused reason`.
  - `purge requires confirmation`.
- [ ] **Step 2: Run them.** `cd ui && pnpm --filter pmm test -- OtelTab`. Expected: FAIL.
- [ ] **Step 3: Implement** the tab, the form, the status card and the messages, following `SshKeyForm.tsx` and `AdvancedSettingsForm.tsx`.
- [ ] **Step 4: Run them again.** Same command, plus `pnpm --filter pmm test -- Settings.test`, then `make lint format-check`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm/src/pages/settings
  git commit -s -m "PMM-15571 Add Settings → OTEL tab"
  ```

### Task 3: Parser presets section

**Files:**
- Create:
  - `ui/apps/pmm/src/pages/settings/components/otel/presets/PresetsSection.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/presets/PresetDialog.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/presets/PresetDialog.schema.ts`
  - `ui/apps/pmm/src/pages/settings/components/otel/presets/PresetsSection.test.tsx`

**Interfaces:**
- Consumes the preset hooks (Task 1).
- The table columns are name, description, a "Built-in" badge and usage count. The actions:
  - **Built-ins:** Clone. It opens `PresetDialog` with the name `<name>_copy` and the same operators.
  - **Custom presets:** Edit and Delete. Delete is disabled, with a tooltip, when `usageCount > 0`.
- Dialog fields:
  - `name`: `^[a-z][a-z0-9_]*$`, at most 64 characters;
  - `description`;
  - `operatorsYaml`: a multiline `TextInput` with 16 rows and a monospace font.

  The server error body (`message`) is shown under the field it names, or above the form.

- [ ] **Step 1: Write the failing tests.**
  - `clone prefills name and operators`.
  - `built-in has no edit or delete`.
  - `delete disabled when in use`.
  - `keeps dialog open on server validation error`: the mocked add rejects with `{"message":"operator 0: type 'router' is not allowed"}`; assert that the message is shown and the YAML is still in the field.
- [ ] **Step 2: Run them.** `cd ui && pnpm --filter pmm test -- PresetsSection`. Expected: FAIL.
- [ ] **Step 3: Implement** the section and the dialog.
- [ ] **Step 4: Run them again.** Same command, then `make lint format-check`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm/src/pages/settings/components/otel/presets
  git commit -s -m "PMM-15574 Manage log parser presets in the UI"
  ```

### Task 4: Log sources section (D7)

**Files:**
- Create:
  - `ui/apps/pmm/src/pages/settings/components/otel/sources/LogSourcesSection.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/sources/LogSourceDialog.tsx`
  - `ui/apps/pmm/src/pages/settings/components/otel/sources/LogSourceDialog.schema.ts`
  - `ui/apps/pmm/src/pages/settings/components/otel/sources/LogSourcesSection.test.tsx`

**Interfaces:**
- Consumes `useLogSources`, `useAddLogSource`, `useRemoveLogSources` and `useLogParserPresets` (Task 1), and the existing `useServices` and `useAgents` hooks (`hooks/api/useServices.ts`, `useAgents.ts`). The node list comes from the pmm-agents' `runsOnNodeId`.
- The table is grouped by node. Its columns:
  - kind;
  - path, or the journald units;
  - preset;
  - owner: "Node", "Service: <name>" or "App: <name>";
  - state, as a coloured `Chip`, with the reason as text.
- Dialog fields:
  - node;
  - kind: file or journald;
  - for a file: the path, with help text "Must be under an allowed path in pmm-agent.yaml on that node (default /var/log)";
  - for journald: the units, as a comma list;
  - preset: only selectable presets, and only for files;
  - owner: node, service or app. The service list is filtered to the chosen node, and the app name is free text.

  Submit calls `addLogSource({ nodeId, … })`.

- [ ] **Step 1: Write the failing tests.**
  - `renders source state reason`: a row with state `NOT_ALLOWED` and the reason "path /opt/x is not under an allowed path" shows both.
  - `service list filtered by node`.
  - `submits node-bound app source`: assert the payload `{ nodeId: 'n1', kind: 'file', path: '/var/log/app.log', presetName: 'json', appName: 'billing' }`.
  - `remove asks for confirmation`.
- [ ] **Step 2: Run them.** `cd ui && pnpm --filter pmm test -- LogSourcesSection`. Expected: FAIL.
- [ ] **Step 3: Implement** the section and the dialog.
- [ ] **Step 4: Run them again.** Same command, then `make lint format-check`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add ui/apps/pmm/src/pages/settings/components/otel/sources
  git commit -s -m "PMM-15574 Manage log sources in the UI"
  ```

### Task 5: `ClickHouse-OTEL` datasource, with Admin-only queries in the Grafana fork

**Files:**
- In percona/pmm, modify:
  - `build/ansible/roles/grafana/files/datasources.yml`: add the datasource `name: ClickHouse-OTEL`, `uid: pmm-otel`, `type: grafana-clickhouse-datasource`, with `jsonData` `{username: ${PMM_CLICKHOUSE_OTEL_READER_USER}, host: ${PMM_CLICKHOUSE_HOST}, port: ${PMM_CLICKHOUSE_PORT}, defaultDatabase: otel, logs: {defaultDatabase: otel, defaultTable: logs, otelEnabled: true}}` and `secureJsonData.password: ${PMM_CLICKHOUSE_OTEL_READER_PASSWORD}`. Also add it to `deleteDatasources` (:2-4), so it is re-provisioned.
  - `managed/services/supervisord/supervisord.go`: add `PMM_CLICKHOUSE_OTEL_READER_USER/PASSWORD` to the grafana template env (:340-343) and to the params (:453-460).
  - `managed/testdata/supervisord.d/grafana.ini`
- In `/srv/runner/2/percona/grafana`, on branch `PMM-15576-otel-admin-only`:
  - Create `pkg/services/pluginsintegration/clientmiddleware/percona_otel_admin_middleware.go` and its `_test.go`.
  - Modify `pkg/services/pluginsintegration/pluginsintegration.go` (:222-223): register the middleware next to `NewPerconaForwarderHTTPClientMiddleware`.

**Interfaces:**
- Produces `func NewPerconaOtelAdminMiddleware(uid string) plugins.ClientMiddleware`, constructed with `"pmm-otel"`. For `QueryData`, `CallResource` and `CheckHealth`:
  - when `req.PluginContext.DataSourceInstanceSettings != nil`, its `UID == uid`, and `req.PluginContext.User == nil || req.PluginContext.User.Role != "Admin"`, it returns an error that maps to HTTP 403 with the message "only Admins can query OpenTelemetry data";
  - otherwise it calls the next handler.

- [ ] **Step 1: Write the failing tests.**
  - In the fork: `TestOtelAdminMiddleware` has cases Viewer (403), Editor (403), Admin (passes) and another datasource as Viewer (passes), for `QueryData` and for `CallResource`. Use the fake handler pattern from the other `clientmiddleware` tests.
  - In pmm: the `TestConfig` golden `grafana.ini` contains `PMM_CLICKHOUSE_OTEL_READER_USER`.
- [ ] **Step 2: Run them.** `go test ./pkg/services/pluginsintegration/clientmiddleware/ -run OtelAdmin` in the fork, and `cd managed && go test ./services/supervisord/`. Expected: FAIL.
- [ ] **Step 3: Implement** the middleware and its registration, the datasource provisioning and the supervisord env.
- [ ] **Step 4: Run them again.** Same commands. Expected: PASS. Then on a Feature Build that includes both PRs:
  - As Admin, `GET /graph/api/datasources/uid/pmm-otel/health` gives `OK`.
  - The query `INSERT INTO otel.logs (Body) VALUES ('x')` through the datasource fails with a readonly error.
- [ ] **Step 5: Commit** in each repo.
  ```bash
  git commit -s -m "PMM-15576 Provision the ClickHouse-OTEL datasource"
  git commit -s -m "PMM-15576 Allow only Admins to query OTEL data"
  ```

### Task 6: Logs dashboard and links

**Files:**
- Create:
  - `dashboards/dashboards/OTel/Logs.json` (uid `pmm-otel-logs`)
  - `api-tests/otel/datasource_test.go`
- Modify:
  - `dashboards/pmm-app/src/plugin.json` (`includes`)
  - `dashboards/dashboards/MySQL/MySQL_Instance_Summary.json`
  - `dashboards/dashboards/PostgreSQL/PostgreSQL_Instance_Summary.json`
  - `dashboards/dashboards/MongoDB/MongoDB_Instance_Summary.json`
  - `dashboards/dashboards/OS/Node_Summary.json`

  Check the exact file names with `ls dashboards/dashboards/*/`.
  - `managed/services/grafana/client.go`, with a startup call from `managed/cmd/pmm-managed/main.go`: set the `OTel` folder permissions to Admin-only (`POST /api/folders/{uid}/permissions` with `[{"role":"Admin","permission":1}]`). Use the existing Grafana client, and the same startup hook as the other Grafana provisioning calls.

**Interfaces:**
- **Variables:**
  - `node_name`: Metrics datasource, `label_values(node_uname_info, node_name)`, multi-select, include All.
  - `node_id`: hidden, `label_values(node_uname_info{node_name=~"$node_name"}, node_id)`.
  - `service_id`: `pmm-otel`, `SELECT DISTINCT PmmServiceId FROM otel.logs WHERE $__timeFilter(TimestampTime) AND PmmNodeId IN (${node_id:singlequote})`, with the text from `ResourceAttributes['pmm.service_name']`.
  - `app`: distinct `ServiceName`.
  - `source`: distinct `LogAttributes['log.file.path']`.
  - `severity`: custom list, TRACE to FATAL.
  - `search`: textbox.
- **Panels:**
  - log volume by severity: a time series of `count()` grouped by `toStartOfInterval(TimestampTime, INTERVAL $__interval_s second)` and `SeverityText`;
  - a logs panel with `Timestamp`, `SeverityText`, `Body` and the attributes as labels, `LIMIT 1000`.
- **Text search:** `positionCaseInsensitive(Body, ${search:sqlstring}) > 0` when `search` is not empty.
- **Links:**
  - each service summary links to `/graph/d/pmm-otel-logs?var-service_id=${service_id}&${__url_time_range}`;
  - Node Summary links with `var-node_name=${node_name}`.

- [ ] **Step 1: Write the failing checks.**
  - `api-tests/otel/datasource_test.go` `TestViewerCannotQueryOtelDatasource`: a Viewer POSTs to `/graph/api/ds/query` with a `pmm-otel` query and gets 403; an Admin gets 200.
  - Add a test to the dashboards Python suite (`dashboards/misc/test_*.py`, following the existing tests), `test_otel_search_is_sqlstring`: every occurrence of `$search` or `${search` in `OTel/*.json` is `${search:sqlstring}`.
- [ ] **Step 2: Run them.** `python3 -m pytest dashboards/misc -k otel`, and the api-test against the Task 5 build. Expected: FAIL, because the dashboard doesn't exist yet. The api-test passes once Task 5 has merged.
- [ ] **Step 3: Implement** the dashboard, the links, the `plugin.json` entry and the folder permission call. Then run `python3 dashboards/misc/cleanup-dash.py dashboards/dashboards/OTel/Logs.json`.
- [ ] **Step 4: Run them again.** `python3 dashboards/misc/cleanup-dash.py --check-only` on every changed dashboard, the pytest, and `cd dashboards/pmm-app && yarn lint:check`. Expected: PASS. Then on a Feature Build, check each filter manually, and check that a link from the MySQL summary opens Logs filtered to that service and time range.
- [ ] **Step 5: Commit.**
  ```bash
  git add dashboards managed/services/grafana managed/cmd/pmm-managed/main.go api-tests/otel/datasource_test.go
  git commit -s -m "PMM-15576 Add the Logs dashboard and service links"
  ```

### Task 7: Collector health dashboard and alert templates

**Files:**
- Create:
  - `dashboards/dashboards/OTel/OTel_Collectors.json` (uid `pmm-otel-collectors`, Metrics datasource)
  - `managed/data/alerting-templates/otel_collector_dropping_records.yml`
  - `managed/data/alerting-templates/otel_collector_queue_full.yml`
  - `managed/data/alerting-templates/otel_ingest_paused.yml`
- Modify:
  - `dashboards/pmm-app/src/plugin.json`
  - the alerting template tests (`managed/services/alerting/*_test.go`; the loader is at `managed/services/alerting/service.go:200-242`)

**Interfaces:**
- **Panels**, per `node_name`:
  - accepted log records: `rate(otelcol_receiver_accepted_log_records_total[$__rate_interval])`;
  - refused records: `otelcol_receiver_refused_log_records_total`;
  - failed sends: `otelcol_exporter_send_failed_log_records_total`;
  - queue size against capacity: `otelcol_exporter_queue_size / otelcol_exporter_queue_capacity`;
  - memory limiter refusals: `otelcol_processor_refused_log_records_total`;
  - the server collector's ClickHouse insert errors: the `job="otel-collector-server"` series of the same failed-send metric;
  - `pmm_managed_otel_ingest_paused`.

  Check each metric name against the pinned collector's `/metrics` on a Feature Build before you commit.
- **Templates:**
  - dropping records: `sum by (node_name)(increase(otelcol_exporter_send_failed_log_records_total[10m])) > 0`, severity warning;
  - queue full: `max by (node_name)(otelcol_exporter_queue_size / otelcol_exporter_queue_capacity) > 0.9 for 5m`, severity warning;
  - ingest paused: `max(pmm_managed_otel_ingest_paused) == 1 for 1m`, severity critical.

- [ ] **Step 1: Write the failing test.** Extend the built-in template loading test so it asserts that the three new template names load and that each expression parses (`promql.ParseExpr`). Run `cd managed && go test ./services/alerting/`. Expected: FAIL.
- [ ] **Step 2: Implement** the templates and the dashboard. Run `cleanup-dash.py` on the dashboard.
- [ ] **Step 3: Run the checks.** The same Go test, and `cleanup-dash.py --check-only`. Expected: PASS. Then, on a Feature Build, stop `otel-collector` on the server; within 1 minute the queue panel grows on the clients, and it drains after a restart (PMM-15593 AC).
- [ ] **Step 4: Commit.**
  ```bash
  git add dashboards managed/data/alerting-templates managed/services/alerting
  git commit -s -m "PMM-15593 Add collector health dashboard and alerts"
  ```
