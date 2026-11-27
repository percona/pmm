# Logs Part 1: log data and PMM Server logs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every log line in `otel.logs` carries a `ServiceName`, a PostgreSQL error report is one row, and PMM Server's own `/srv/logs` files are collected by the server collector without loss or duplication across restarts.

**Architecture:** pmm-managed already renders `/srv/otelcol/config.yaml` for the supervisord `otel-collector`. It gains filelog receivers for `/srv/logs` (built in `managed/otel` from the parser presets in PostgreSQL), a `file_storage` extension for read positions, a `service.name` transform, and a validate-then-restart step when the rendered config changes. One migration fixes the `postgres` and `pmm_agent` presets and adds a `victoriametrics` preset.

**Tech Stack:** Go (pmm-managed), reform/PostgreSQL migrations, otelcol-contrib 0.148.0 (`filelog`, `file_storage`, `transform`), ClickHouse `otel.logs`.

**Spec:** `dev/otel/logs-design.md`, Part 1 (sections 1.1–1.5).

## Global Constraints

- Branch `PMM-7-tibi-holmes`, pushed to `origin tibi-holmes`; commits `PMM-7 <summary ≤50 chars>`, signed off (`git commit -s`), `git add` and `git commit` in separate calls (a hook blocks them combined).
- Migrations are forward-only: add migration `148`; never edit 127–147.
- Go conventions from `AGENTS.md`: no inline `if err := …`, no named returns, comments on their own line, one-sentence doc comments, no `%q` in error or log messages, logrus `*logrus.Entry`.
- The runner host runs no PostgreSQL, ClickHouse or Docker: `testdb` tests run in CI; live checks run on CHAOS VM `adre-fb4259` via `/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh`.
- Lint with `bin/golangci-lint run ./managed/...` and format with `gofumpt`; no proto or reform model changes, so `make gen` is not needed.
- Server log directory `/srv/logs`; storage directory `/srv/otelcol/storage`; collector binary `/usr/local/percona/pmm/tools/otelcol-contrib`.

## Review Focus

- A rendered config the collector rejects stops all log ingestion, agents' OTLP included: the old config must stay in place and the collector must not restart (Task 4, `TestWriteOtelCollectorConfig_InvalidKeepsOld`).
- A known file that does not exist on this install (`vmalert.log` before vmalert first runs) must not stop the collector or spam its log (Task 6, live check 3).
- First start on a server with large existing logs must not import them (`start_at: end`) (Task 6, live check 2).
- The nginx filter must keep 4xx, 5xx and lines without a `status` field (Task 3, `TestBuildServerLogs` asserts the expression; Task 6, live check 5 with a real 404).
- A collector restart neither loses nor repeats lines of a quiet file (Task 6, live check 4 with marker lines).

---

### Task 1: Preset migration 148

**Files:**
- Modify: `managed/models/database.go` (add migration `148` after `147`)
- Create: `managed/models/log_parser_preset_migration_test.go`

**Interfaces:**
- Produces: built-in presets `postgres` (new `line_start_pattern`), `pmm_agent` (time layout `'2006-01-02T15:04:05Z07:00'`), new `victoriametrics` (id and name `victoriametrics`). Task 3 maps files to these names.

- [ ] **Step 1: Write the failing test** `TestLogParserPresetsServerLogs` (uses `testdb.Open(t, models.SkipFixtures, nil)`, then `models.FindLogParserPresetByName`):

```go
pg := mustPreset(t, q, "postgres")
re := regexp.MustCompile(pg.LineStartPattern)
assert.True(t, re.MatchString(`2026-10-16 12:38:54.405 UTC [233190] ERROR:  deadlock detected`))
assert.True(t, re.MatchString(`2026-10-16 12:38:54.405 UTC [233190] LOG:  checkpoint starting: wal`))
for _, l := range []string{"DETAIL", "HINT", "STATEMENT", "CONTEXT", "QUERY", "LOCATION"} {
	assert.False(t, re.MatchString(`2026-10-16 12:38:54.405 UTC [233190] `+l+`:  x`), l)
}

agent := mustPreset(t, q, "pmm_agent")
assert.Contains(t, agent.OperatorYAML, `layout: '2006-01-02T15:04:05Z07:00'`)
for _, ts := range []string{"2026-10-16T19:56:51Z", "2026-10-16T19:56:49.437+00:00"} {
	_, err := time.Parse("2006-01-02T15:04:05Z07:00", ts)
	require.NoError(t, err, ts)
}

vm := mustPreset(t, q, "victoriametrics")
assert.True(t, vm.BuiltIn)
require.NoError(t, models.ValidateLogParserOperatorYAML(vm.OperatorYAML))
vmRe := regexp.MustCompile(regexParserRegex(t, vm.OperatorYAML))
m := vmRe.FindStringSubmatch("2026-10-16T13:20:06.550Z\twarn\t/app/vmalert/config/log/logger.go:44\tno groups found in /srv/prometheus/rules/*.yml")
require.NotNil(t, m)
assert.Equal(t, "warn", m[vmRe.SubexpIndex("level")])
assert.Equal(t, "no groups found in /srv/prometheus/rules/*.yml", m[vmRe.SubexpIndex("message")])
```

Test helpers in the same file: `mustPreset(t, q, name)` (`FindLogParserPresetByName`, `require` non-nil) and `regexParserRegex(t, operatorYAML)` (unmarshal the operators into `[]map[string]any`, return the `regex` of the `regex_parser` entry).

- [ ] **Step 2: Run it to verify it fails** — `go test ./managed/models -run TestLogParserPresetsServerLogs` in CI or the devcontainer. Expected: FAIL (`victoriametrics` not found; postgres pattern matches DETAIL lines).

- [ ] **Step 3: Add migration 148** with three statements:
  - `UPDATE log_parser_presets SET line_start_pattern = $p$^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)? \S+ \[\d+\] (LOG|ERROR|WARNING|FATAL|PANIC|NOTICE|INFO|DEBUG\d?):$p$, updated_at = NOW() WHERE name = 'postgres' AND built_in AND line_start_pattern = $p$^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)? \S+ \[\d+\]$p$` (only an unedited preset changes).
  - `UPDATE … SET operator_yaml = replace(operator_yaml, $o$layout: '2006-01-02T15:04:05.000Z07:00'$o$, $n$layout: '2006-01-02T15:04:05Z07:00'$n$), updated_at = NOW() WHERE name = 'pmm_agent' AND built_in` (Go accepts fractional seconds the layout omits, so `vmproxy.log` lines without milliseconds parse).
  - `INSERT … ('victoriametrics', 'victoriametrics', 'VictoriaMetrics / vmalert log (time, level, caller, message separated by tabs)', <yaml>, true, NOW(), NOW()) ON CONFLICT DO NOTHING`, where `<yaml>` is: `regex_parser` with `regex: '^(?P<timestamp>\S+)\t(?P<level>\w+)\t(?P<caller>\S+)\t(?P<message>.*)$'` from `body` to `attributes`; `time_parser` on `attributes.timestamp`, layout `'2006-01-02T15:04:05Z07:00'`, `gotime`; `severity_parser` on `attributes.level`, `preset: none`, mapping `info: info`, `warn: warn`, `error: error`, `fatal: [fatal, panic]`; `move` `attributes.message` → `body`; `remove` `attributes.timestamp` and `attributes.level`, each `on_error: send_quiet` (the same shape as migrations 129 and 145).
  - One comment line above the migration: why the postgres pattern lists levels (detail lines join the entry above).

- [ ] **Step 4: Run the test** — same command. Expected: PASS.

- [ ] **Step 5: Commit** — `PMM-7 Join PostgreSQL detail lines, add VM preset`.

### Task 2: `service.name` on every log line

**Files:**
- Modify: `managed/services/supervisord/otelconfig.go` (the `transform` processor's `log_statements`)
- Test: `managed/services/supervisord/otelconfig_test.go`

**Interfaces:**
- Produces: resource attribute `service.name` (ClickHouse `ServiceName`) = `service_name` when present, else `node_name`. Parts 2 and 3 filter on it.

- [ ] **Step 1: Write the failing test** `TestBuildOtelCollectorConfigYAML_ServiceName`: the rendered YAML contains, in this order,

```
set(resource.attributes["service.name"], resource.attributes["service_name"]) where resource.attributes["service.name"] == nil and resource.attributes["service_name"] != nil
set(resource.attributes["service.name"], resource.attributes["node_name"]) where resource.attributes["service.name"] == nil and resource.attributes["node_name"] != nil
```

- [ ] **Step 2: Run** `go test ./managed/services/supervisord -run TestBuildOtelCollectorConfigYAML` — Expected: FAIL.
- [ ] **Step 3: Add the two statements** after the existing `pmm_source` statement.
- [ ] **Step 4: Run** the same command — Expected: PASS (existing tests still pass).
- [ ] **Step 5: Commit** — `PMM-7 Fill ServiceName on collected logs`.

### Task 3: Filelog receivers for `/srv/logs`

**Files:**
- Create: `managed/otel/server_logs.go`, `managed/otel/server_logs_test.go`
- Delete: `managed/otel/server_sources.go`; `BuildServerOtelConfigYAML` in `managed/otel/config.go` and `TestBuildServerOtelConfigYAML` in `config_test.go` (keep `IndentYAML`, `FilelogMultilineYAML`, `quoteYAMLString`, `sanitizePresetName`)

**Interfaces:**
- Consumes: presets from Task 1 by name.
- Produces:

```go
const (
	ServerLogDir     = "/srv/logs"
	ServerLogStorage = "file_storage/srv_logs"
)

// ServerNode identifies the PMM Server node whose logs are collected.
type ServerNode struct {
	NodeID   string
	NodeName string
	// HANodeID is set in HA mode so each server's lines stay apart.
	HANodeID string
}

// ServerLogs holds the filelog receivers for PMM Server's own logs, ready to embed in the collector config.
type ServerLogs struct {
	// ReceiversYAML is the receiver entries for the "receivers:" block, indented two spaces.
	ReceiversYAML string
	ReceiverIDs   []string
}

func BuildServerLogs(presets []*models.LogParserPreset, node ServerNode) ServerLogs
```

File → preset table (ordered slice, the spec's 1.4 table): `pmm-managed.log` → `pmm_managed`; `pmm-agent.log`, `qan-api2.log`, `vmproxy.log` → `pmm_agent`; `nginx.log` → `nginx_access`; `grafana.log` → `grafana`; `clickhouse-server.log` → `clickhouse_server`; `postgresql14.log` → `postgres`; `supervisord.log` → `supervisord`; `otel-collector.log` → `otel_collector`; `victoriametrics.log`, `vmalert.log` → `victoriametrics`; `pmm-init.log` → none.

Each known file gets receiver `filelog/srv_<stem with non-alphanumerics as _>` with: `include: ["/srv/logs/<file>"]`, `start_at: end`, `storage: file_storage/srv_logs`, `resource: {node_id, node_name, service.name: <stem>}` (+ `ha_node_id` when set), the preset's `multiline` and `operators`. A missing or empty preset yields a receiver without operators. `nginx.log` gets one more operator, appended last:

```yaml
- type: filter
  expr: 'attributes.status != nil and int(attributes.status) < 400'
```

A last receiver `filelog/srv_other` takes every other file: `include: ["/srv/logs/*.log"]`, `exclude:` every known path, no operators, `service.name: pmm-server`. Rotated files (`*.log.1`) match neither.

- [ ] **Step 1: Write the failing tests** (parse `"receivers:\n" + ReceiversYAML` with `gopkg.in/yaml.v3`):
  - `TestBuildServerLogs`: with presets `pmm_managed` (operators), `postgres` (line start pattern) and `nginx_access` and `ServerNode{NodeID: "pmm-server", NodeName: "pmm-server"}`:
    - `ReceiverIDs` has 14 entries, ending with `filelog/srv_other`;
    - `filelog/srv_pmm_managed` has `include` `["/srv/logs/pmm-managed.log"]`, `start_at` `end`, `storage` `file_storage/srv_logs`, resource `service.name` `pmm-managed` and `node_name` `pmm-server`, and the preset's operators;
    - `filelog/srv_postgresql14` has `multiline.line_start_pattern` equal to the preset's;
    - `filelog/srv_nginx`'s last operator has type `filter` and the exact expression above;
    - `filelog/srv_other` has `include` `["/srv/logs/*.log"]` and `exclude` listing all 13 known paths.
  - `TestBuildServerLogs_NoPresets`: with nil presets every receiver exists and none has `operators`.
  - `TestBuildServerLogs_HANode`: `HANodeID: "pmm-ha-1"` puts `ha_node_id: pmm-ha-1` on every receiver's resource.
- [ ] **Step 2: Run** `go test ./managed/otel -run TestBuildServerLogs` — Expected: FAIL (undefined).
- [ ] **Step 3: Implement** `BuildServerLogs` and the file table; delete the dead code listed above.
- [ ] **Step 4: Run** `go test ./managed/otel/...` — Expected: PASS.
- [ ] **Step 5: Commit** — `PMM-7 Build receivers for PMM Server logs`.

### Task 4: Render, validate and apply the server collector config

**Files:**
- Modify: `managed/services/supervisord/otelconfig.go`, `managed/services/supervisord/supervisord.go`, `managed/cmd/pmm-managed/main.go` (the `supervisord.New` call), `managed/services/supervisord/supervisord_test.go` and `devcontainer_test.go` (new `New` argument)
- Test: `managed/services/supervisord/otelconfig_test.go`

**Interfaces:**
- Consumes: `otel.BuildServerLogs`, `otel.ServerLogs`, `otel.ServerNode`, `otel.ServerLogStorage` (Task 3); `models.FindAllLogParserPresets`, `models.FindNodeByID`, `models.PMMServerNodeID`.
- Produces:

```go
func buildOtelCollectorConfigYAML(endpoint, username, password string, logRetentionDays, spanRetentionDays, metricsRetentionDays int, serverLogs otel.ServerLogs) string
func New(configDir string, params *models.Params, db *reform.DB) *Service
// writeOtelCollectorConfig writes the config when it changed and the validator accepts it; it reports whether it wrote.
func writeOtelCollectorConfig(path string, data []byte, validate func(path string) error) (bool, error)
```

Rendered additions: top-level `extensions: {file_storage/srv_logs: {directory: /srv/otelcol/storage}}`; `service.extensions: [file_storage/srv_logs]`; `serverLogs.ReceiversYAML` under `receivers:` after `otlp`; logs pipeline `receivers: [otlp, <serverLogs.ReceiverIDs…>]`. With empty `ServerLogs` the output keeps today's shape plus the extension.

`writeOtelCollectorConfig`: equal to the current file → `false, nil`; otherwise write `path + ".new"`, run `validate`, rename over `path` on success, remove the `.new` and return the error on failure. The production validator runs `/usr/local/percona/pmm/tools/otelcol-contrib validate --config=<path>` with a 30 s timeout.

`Service.writeOtelCollectorConfigFile(settings)`: with `s.db` non-nil, load presets and the server node (fall back to `pmm-server` when the node is missing), `HANodeID = s.haParams.NodeID` when `s.haParams.Enabled`; `os.MkdirAll("/srv/otelcol/storage", 0o755)`; then `writeOtelCollectorConfig`. `UpdateConfiguration`: when it wrote and `settings.IsOtelCollectorEnabled()`, run `supervisorctl restart otel-collector` and log a warning if that fails. Tests pass `nil` for `db`.

- [ ] **Step 1: Write the failing tests:**
  - `TestBuildOtelCollectorConfigYAML_ServerLogs`: with a `ServerLogs` holding one receiver `filelog/srv_x`, the YAML parses; `service.pipelines.logs.receivers` equals `[otlp, filelog/srv_x]`; `service.extensions` equals `[file_storage/srv_logs]`; `extensions.file_storage/srv_logs.directory` equals `/srv/otelcol/storage`.
  - `TestWriteOtelCollectorConfig_Unchanged`: second write of identical bytes returns `false` and does not call the validator.
  - `TestWriteOtelCollectorConfig_Changed`: new bytes with an accepting validator return `true` and the file holds the new bytes.
  - `TestWriteOtelCollectorConfig_InvalidKeepsOld`: a validator returning an error leaves the old bytes, no `.new` file, returns the error and `false`.
  - Existing `TestBuildOtelCollectorConfigYAML*` pass `otel.ServerLogs{}`.
- [ ] **Step 2: Run** `go test ./managed/services/supervisord/...` — Expected: FAIL (signature).
- [ ] **Step 3: Implement** the signature changes, `writeOtelCollectorConfig`, the restart and the `main.go` wiring (pass `db`, created at `main.go:976`, to the `supervisord.New` call at `main.go:1029`).
- [ ] **Step 4: Run** `go test ./managed/services/supervisord/... ./managed/otel/...` and `go build ./managed/cmd/pmm-managed` — Expected: PASS, build succeeds.
- [ ] **Step 5: Commit** — `PMM-7 Collect PMM Server logs in the collector`.

### Task 5: Re-render after a preset change

**Files:**
- Modify: `managed/services/server/log_parser_presets.go` (`ChangeLogParserPreset`)
- Test: the existing `ChangeLogParserPreset` test in `managed/services/server/` (find it with `grep -rn ChangeLogParserPreset managed/services/server/*_test.go`)

**Interfaces:**
- Consumes: `(*Server).UpdateConfigurations(ctx)` (already calls `supervisord.UpdateConfiguration` and `agentsState.UpdateAgentsState`).

- [ ] **Step 1: Update the test** so changing a preset expects `supervisord.UpdateConfiguration` once (mockery mock in that package) in addition to the agents' state update.
- [ ] **Step 2: Run** `go test ./managed/services/server -run LogParserPreset` — Expected: FAIL (unexpected or missing call).
- [ ] **Step 3: Replace** `s.agentsState.UpdateAgentsState(ctx)` in `ChangeLogParserPreset` with `s.UpdateConfigurations(ctx)`, keeping the same error message.
- [ ] **Step 4: Run** the same command — Expected: PASS.
- [ ] **Step 5: Commit** — `PMM-7 Refresh server collector on preset change`.

### Task 6: Docs, lint and live verification

**Files:**
- Modify: `dev/otel/README.md` (Architecture: the server collector reads `/srv/logs` itself with the file → preset table; read positions in `/srv/otelcol/storage`; nginx keeps status ≥ 400; filter and rule on `SeverityNumber`, not `SeverityText`)

- [ ] **Step 1: Lint and test** — `bin/golangci-lint run ./managed/otel/... ./managed/services/supervisord/... ./managed/services/server/... ./managed/models/...` and `go test` for the same packages. Expected: no new findings compared with the branch before Task 1 (record the before count first); tests pass except the known `testdb` ones that need PostgreSQL.
- [ ] **Step 2: Deploy** — build `pmm-managed` with the version ldflags, copy it into `pmm-server` on `adre-fb4259`, `supervisorctl restart pmm-managed`. Expected: `pmm-managed` RUNNING; `select max(id) from schema_migrations` returns 148; `/srv/otelcol/config.yaml` lists the 14 `filelog/srv_*` receivers; `otel-collector` RUNNING with no error in `/srv/logs/otel-collector.log`.
- [ ] **Step 3: Live checks** (ClickHouse queries on `otel.logs`, last 10 minutes):
  1. Rows with `ResourceAttributes['node_name'] = 'pmm-server'` exist for `pmm-managed`, `grafana`, `clickhouse-server`, `qan-api2`, `vmproxy`, `victoriametrics` and `nginx`, each with a parsed `SeverityNumber` (not 0 for preset-parsed files).
  2. No row for any server file is older than the deploy (no import of old content).
  3. `otel-collector.log` has no repeated errors about a missing file.
  4. Append `adre-marker-1` to `/srv/logs/pmm-init.log`, wait 10 s, `supervisorctl restart otel-collector`, append `adre-marker-2`, wait 10 s: each marker appears exactly once.
  5. `curl -sk https://localhost/does-not-exist` inside the server: an nginx row with `status=404` exists; no nginx row with status < 400 after the deploy.
  6. New rows from agents (`pdpgsql_pmm_17_1`) have `ServiceName` set.
  7. After the agents reconnect, `/root/adre-issues/issues.sh run pg_deadlocks 2` gives `deadlock detected` rows whose `Body` contains `DETAIL:` and `STATEMENT:` and no separate `DETAIL` rows from then on.
- [ ] **Step 4: Commit and push** — `PMM-7 Document PMM Server log collection`; `git push origin HEAD:tibi-holmes`.
