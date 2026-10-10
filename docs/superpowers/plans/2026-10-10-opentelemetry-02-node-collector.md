# OTel 02: Node Collector, Log Sources and Parser Presets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every node runs at most one pmm-agent-managed OpenTelemetry Collector, as the non-root `pmm-agent` user. It ships the node's log sources to PMM Server through a disk-backed queue, tags each record with PMM identity, and survives restarts and server outages without losing lines. PMM Server's own logs flow through the same path.

**Architecture:**
- pmm-managed stores log sources and parser presets in PostgreSQL. It builds a typed `OtelCollectorParams` for each collector and sends it in `SetStateRequest`.
- pmm-agent turns those params into collector YAML (typed structs, no templates). It enforces its local allow-list, drops privileges, runs the binary, and reports a status for each source.
- Built-in presets ship as embedded YAML and are upserted at startup.

**Tech Stack:** Go (pmm-managed, pmm-agent, pmm-admin), protobuf, reform/PostgreSQL, OpenTelemetry Collector (filelog, journald, otlphttp, file_storage), the stanza library (preset tests only, in a separate Go module).

**Spec:**
- [analysis](../specs/2026-10-10-opentelemetry-analysis.md): §3, §5 F1–F7, F9, F12, F14, §6.1, §6.2, §6.6, §6.7, decisions D3, D10, D11, D13;
- PMM-15572, PMM-15574 (backend), PMM-15575, PMM-15577 (client), PMM-15593 (scrape);
- the [roadmap](2026-10-10-opentelemetry-00-roadmap.md) for the shared names.

**Depends on:** plan 01, Tasks 1–4 (binary, users, schema, settings) and Task 9 (the `api/otel/v1` domain exists).

## Global Constraints

The roadmap's **Shared contract** and **Global constraints** apply. In addition:

**From PMM-15572:**
- The collector process runs as `pmm-agent`, never as root.
- No OTLP port is open in this plan; the node traces receiver comes in plan 05.
- Every record carries `pmm.node_id` and `pmm.agent_id`. Service-bound sources add `pmm.service_id` and `pmm.service_name`. App sources add `service.name`.
- A source bound to a service on another node is rejected.
- Checkpoints live in a persistent state dir, never in `paths.tempdir`.
- The disk-backed sending queue is 1024 MiB by default.
- Config is deterministic, and is pushed only to agents whose params changed.
- Files that contain credentials are 0600.
- An older pmm-agent gets `FailedPrecondition` on add, and is never sent the type.

**From PMM-15574:**
- Built-ins are read-only, and can be cloned.
- Custom presets allow only known operator types. `EXPR(` and environment expansion (`${`) are rejected.
- Every preset is validated with the bundled `otelcol validate`.
- Deleting a preset that is in use fails.
- A literal `{{` in a preset works.
- No credential can reach `otel.logs` through a preset.

## Review Focus

1. **A symlink under an allowed prefix that points outside it** (`/var/log/x -> /etc/shadow`). Expected: `NOT_ALLOWED`, because the allow-list checks the resolved path. Test in Task 6 (`TestRenderRejectsSymlinkEscape`).
2. **Globs and `..`** (`/var/log/../etc/*`, `/var/log/mysql/*.log`). Expected: a path containing `..` is rejected. A glob is checked on its static directory prefix. Test in Task 6 (`TestAllowListGlobsAndDotDot`).
3. **Log lines of 64 KiB or more, or binary content.** Expected: each line is truncated at `max_log_size` (64 KiB), and every OTLP request stays under nginx's 10 MB limit. Test in Task 6 (`TestRenderedConfigBoundsRequestSize`), and checked live in plan 01's contract test by adding a 1 MiB line.
4. **Two collectors created concurrently for one pmm-agent** (two `pmm-admin add logs` at once). Expected: exactly one collector. The second request reuses it. Test in Task 3 (`TestAddLogSourceConcurrentCreatesOneCollector`, using the partial unique index).
5. **A log file rotated while the collector is down.** Expected: lines written to the old file before rotation are read from its checkpoint, and the new file is read from its start. This is documented as the guarantee. Test in Task 6 (`TestRenderedFilelogUsesStorageAndStartAtEnd`), which asserts `storage: file_storage`, `start_at: end` and `include_file_path: true`; the live check is in PMM-15578.

---

### Task 1: The `OTEL_COLLECTOR` agent type in inventory

**Files:**
- Modify:
  - `api/inventory/v1/agents.proto`:
    - enum `AGENT_TYPE_OTEL_COLLECTOR = 20` (:16-37);
    - message `OtelCollector`, following `NomadAgent` at :79-100;
    - `otel_collector` in `ListAgentsResponse` (:792-812), in the `GetAgentResponse` oneof (:821-842), and in `ChangeAgentRequest/Response` (:907-955), with `ChangeOtelCollectorParams { optional bool enable = 1; optional common.StringMap custom_labels = 2; }`;
    - no Add RPC: plan 02, Task 3 creates collectors.
  - `api/inventory/v1/agents.go` (sealed interface, :29,44-45)
  - `api/inventory/v1/types/agent_types.go` (:19-63)
  - `managed/models/agent_model.go` (:79-99, `OtelCollectorType`)
  - `managed/models/database.go`: migration **121** (renumber at merge time), `CREATE UNIQUE INDEX agents_one_otel_collector_per_pmm_agent ON agents (pmm_agent_id) WHERE agent_type = 'otel-collector'`
  - `managed/services/converters.go` (:580-588)
  - `managed/services/inventory/grpc/agents_server.go` (:44-60 enum map, `ListAgents` :71-128, Change :240-282)
  - `managed/services/inventory/agents.go` (`ChangeOtelCollector`, following `ChangeNomadAgent` :1774-1800)
  - `admin/commands/list.go` (:300-312, :556-620)
  - `admin/commands/inventory/list_agents.go` (:46,55,134-144,200,302)
  - `admin/commands/inventory/change_agent_otel_collector.go` (new, following `change_agent_nomad_agent.go`)
  - `managed/services/management/agent.go` (`agentToAPI` :141-240)
  - `api-tests/management/helpers.go` (`removeAllAgentsInList`, :125-170)
- Test:
  - `managed/services/converters_test.go`
  - `managed/models/agent_helpers_test.go` (`testdb`, because the unique index is migration behaviour)
  - `admin/commands/list_test.go`

**Interfaces:**
- Produces the shared-contract enum, message and model constant, and `func (s *AgentsService) ChangeOtelCollector(ctx context.Context, agentID string, p *inventoryv1.ChangeOtelCollectorParams) (*inventoryv1.ChangeAgentResponse, error)`.

- [ ] **Step 1: Write the failing tests.**
  - `TestToAPIAgentOtelCollector`: a `models.Agent{AgentType: models.OtelCollectorType, ...}` converts to `*inventoryv1.OtelCollector` with every field.
  - `TestOneOtelCollectorPerPMMAgent` (`testdb.Open`): creating a second `otel-collector` agent for the same `pmm_agent_id` returns a unique-violation error. A collector for another pmm-agent succeeds.
  - `TestListShowsOtelCollector` in pmm-admin: the rendered list contains `otel_collector` and its status.
- [ ] **Step 2: Run them.** `make gen` (the protos fail first), then `go test ./managed/services/ ./managed/models/ ./admin/commands/ -run 'OtelCollector'`. Expected: FAIL.
- [ ] **Step 3: Implement** every file listed above. `AgentTypeName` panics on an unknown type (`types/agent_types.go:66-73`), so register the name.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add api managed admin api-tests/management/helpers.go
  git commit -s -m "PMM-15572 Add the OTEL collector agent type"
  ```

### Task 2: Built-in presets, the validator, and a preset test harness

**Files:**
- Create:
  - `managed/services/otel/presets/builtin/*.yml`: one file each for `mysql_error`, `syslog_mysql_systemd`, `nginx_access`, `nginx_error`, `grafana`, `pmm_managed`, `pmm_agent`, `postgres`, `clickhouse_server`, `otel_collector`, `supervisord`, `syslog`, `json`, `raw`, and the hidden `journald` (`selectable: false`)
  - `managed/services/otel/presets/presets.go`
  - `managed/services/otel/presets/validate.go`
  - `managed/services/otel/presets/validate_test.go`
  - `managed/services/otel/presets/testdata/<name>/input.log`
  - `managed/services/otel/presets/testdata/<name>/expected.json`
  - `build/otelcol/presettest/go.mod`: a **separate module** that requires `github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza` at `otelcol_version`
  - `build/otelcol/presettest/main_test.go`
- Modify: `.github/workflows/otelcol.yml` (run `go test ./...` in `build/otelcol/presettest`, and check that its stanza version equals `otelcol_version` in `build/scripts/vars`)

**Interfaces:**
- Produces:
  - File format:
    ```yaml
    name: mysql_error
    description: …
    min_agent_version: 3.11.0
    selectable: true
    operators: [ … ]
    ```
  - `type Preset struct { Name, Description, OperatorsYAML, MinAgentVersion string; Selectable bool }`
  - `func BuiltIn() ([]Preset, error)`, which reads the embed FS.
  - `type Validator interface { Validate(ctx context.Context, operatorsYAML string) error }`
  - `func NewValidator(otelcolPath string) Validator`. It runs `otelcol validate --config=<tmp>`, where the temp config has one filelog receiver carrying the operators and an `otlphttp` exporter to `http://127.0.0.1:1`.
  - `func CheckCustom(operatorsYAML string) error`, which checks structure and safety:
    - the YAML parses as a non-empty list of maps, each with a `type`;
    - `type` is one of `regex_parser`, `json_parser`, `csv_parser`, `key_value_parser`, `syslog_parser`, `time_parser`, `severity_parser`, `trace_parser`, `scope_name_parser`, `move`, `copy`, `remove`, `retain`, `flatten`, `add`, `recombine`, `noop`;
    - it rejects any string containing `EXPR(`, `${` or `env(`;
    - errors name the operator index and the offending value.
- Built-in fixes from PMM-15574:
  - `mysql_error` parses `log_timestamps=SYSTEM` offsets (`2026-10-10T12:00:00.123456+02:00`) as well as `Z`.
  - `postgres` parses a non-UTC `log_timezone` abbreviation.
  - Both use `recombine` so that a multi-line entry (an InnoDB deadlock dump, a stack trace) becomes one record.
  - `nginx_error` drops lines in the access-log format: a built-in may use a `filter` expression, a custom preset may not.
  - Built-ins pass `CheckCustom` except for the `filter`/`router` exemption, which `BuiltIn()` marks internally.

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestCheckCustomRejects(t *testing.T) {
      for name, y := range map[string]string{
          "unknown op":   "- type: router\n",
          "expr add":     "- type: add\n  field: body.x\n  value: EXPR(env(\"HOME\"))\n",
          "env expansion": "- type: add\n  field: body.x\n  value: ${env:PMM_AGENT_SERVER_PASSWORD}\n",
          "empty":        "",
      } {
          t.Run(name, func(t *testing.T) { require.Error(t, CheckCustom(y)) })
      }
  }

  func TestCheckCustomAcceptsLiteralBraces(t *testing.T) {
      require.NoError(t, CheckCustom("- type: add\n  field: body.x\n  value: '{{ .server_auth_b64 }}'\n"))
  }

  func TestBuiltInsLoad(t *testing.T) {
      ps, err := BuiltIn()
      require.NoError(t, err)
      var names []string
      for _, p := range ps { names = append(names, p.Name) }
      assert.ElementsMatch(t, []string{"mysql_error", "syslog_mysql_systemd", "nginx_access", "nginx_error", "grafana",
          "pmm_managed", "pmm_agent", "postgres", "clickhouse_server", "otel_collector", "supervisord", "syslog", "json", "raw", "journald"}, names)
  }
  ```
  In `build/otelcol/presettest/main_test.go`, `TestPresetsParseSamples`: for each `testdata/<name>/`, build a stanza pipeline from the preset's operators, feed `input.log` line by line, and compare `[]{timestamp(UTC RFC3339Nano), severity_text, body}` with `expected.json`. Write the samples from real logs:
  - MySQL 8.0 and 8.4 error logs, with both `log_timestamps` modes;
  - PostgreSQL 13–17 stderr logs with `log_timezone=Europe/Berlin`;
  - nginx access and error logs;
  - the Grafana console format;
  - logrus text for pmm-managed, pmm-agent, qan-api2 and vmproxy;
  - ClickHouse server logs;
  - supervisord logs;
  - RFC 3164 and 5424 syslog;
  - JSON with `ts`/`level`/`msg` and `timestamp`/`severity`/`message`.

  Each multi-line sample asserts one record.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/otel/presets/` and `cd build/otelcol/presettest && go test ./...`. Expected: FAIL.
- [ ] **Step 3: Implement** the files, `presets.go` (`//go:embed builtin/*.yml`), `validate.go`, and the harness. Start from the prototype's operator chains (`git show origin/tibi-holmes:managed/models/database.go`, migrations 127–129 and 134), and correct them until the samples pass.
- [ ] **Step 4: Run them again.** Same commands. Expected: PASS. If `qan-api2.log` and `vmproxy.log` parse under `pmm_agent`, record that in the PR; otherwise add `pmm_logrus` and use it for both (PMM-15575 asks for this to be confirmed).
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/otel/presets build/otelcol/presettest .github/workflows/otelcol.yml
  git commit -s -m "PMM-15574 Add built-in log parser presets"
  ```

### Task 3: Tables `log_parser_presets` and `log_sources`, and the `OtelService` API

**Files:**
- Create:
  - `managed/models/log_parser_preset_model.go`
  - `managed/models/log_parser_preset_helpers.go`
  - `managed/models/log_source_model.go`
  - `managed/models/log_source_helpers.go`
  - their `_reform.go` files (generated by `make gen`)
  - `managed/services/otel/presets_grpc.go`
  - `managed/services/otel/log_sources_grpc.go`
  - their tests
  - `api-tests/otel/presets_test.go`
  - `api-tests/otel/log_sources_test.go`
- Modify:
  - `managed/models/database.go`: migration **122**, the two tables per analysis §6.2. `units` and `custom_labels` are stored like `agents.custom_labels`.
  - `api/otel/v1/otel.proto` (preset and source RPCs from the roadmap table)
  - `managed/cmd/pmm-managed/main.go`: call `presets.BuiltIn()` at startup and upsert the results through `models.SyncBuiltInPresets`.

**Interfaces:**
- Consumes `presets.BuiltIn`, `presets.CheckCustom`, the `presets.Validator` (Task 2), and `models.OtelCollectorType` (Task 1).
- Produces:
  - **Presets:**
    - `func SyncBuiltInPresets(q *reform.Querier, builtIns []presets.Preset) error`. It upserts each built-in by name. A custom preset with a clashing name is renamed to `<name>_custom`, and a warning is logged.
    - `func FindLogParserPresets(q *reform.Querier) ([]*LogParserPreset, error)`, which returns `UsageCount` through a `COUNT` over `log_sources`.
    - `CreateLogParserPreset`, `ChangeLogParserPreset` and `RemoveLogParserPreset`:
      - change and remove of a built-in return `codes.FailedPrecondition` "built-in presets are read-only; clone it";
      - remove of a preset in use returns `FailedPrecondition` with the count.
  - **Log sources:**
    - `func UpsertLogSource(q *reform.Querier, p UpsertLogSourceParams) (*LogSource, bool, error)` (the bool means "created"), with `UpsertLogSourceParams{ PMMAgentID string; Kind string; Path string; Units []string; PresetName string; ServiceID string; AppName string; Discovery string; CustomLabels map[string]string }`.
      - It creates the collector agent when the pmm-agent has none.
      - It rejects a `ServiceID` whose `node_id` differs from the pmm-agent's `runs_on_node_id` with `InvalidArgument` "service %s runs on node %s, not on this node".
      - It rejects an unknown preset with `NotFound` naming it.
      - It rejects `ServiceID` and `AppName` together with `InvalidArgument`.
      - Adding the same `(collector, kind, path, units)` again updates the row, so the last preset wins.
    - `func RemoveLogSources(q *reform.Querier, f LogSourceFilters) (int, error)` and `func FindLogSources(q *reform.Querier, f LogSourceFilters) ([]*LogSource, error)`, with `LogSourceFilters{ID, NodeID, PMMAgentID, ServiceID, Path string}`.
    - Removing a service deletes its sources through the foreign key.
  - **gRPC:**
    - `AddLogSourceRequest` takes exactly one of `pmm_agent_id` (pmm-admin) or `node_id` (UI). A `node_id` resolves to the pmm-agent whose `runs_on_node_id` is that node; if there is none, the request fails with `FailedPrecondition` "node %s has no pmm-agent".
    - The handlers call `models.PMMAgentSupported(q, pmmAgentID, "OpenTelemetry log collection", version.OtelCollectorSupportVersion)` before any write, and map the result to `FailedPrecondition`.
    - After the transaction they call `state.RequestStateUpdate(ctx, pmmAgentID)` for that pmm-agent only.
  - **Preset changes** call `RequestStateUpdate` for each pmm-agent whose collector uses the preset (`SELECT DISTINCT pmm_agent_id …`), and for no other agent.

- [ ] **Step 1: Write the failing tests.**
  - `log_source_helpers_test.go` (`testdb.Open`, because the tests cover foreign keys, unique and check constraints):
    - `TestUpsertLogSourceOwners`: a file source, a journald source, an app source and a service-bound source on one node give one collector agent, with 4 rows and the right owner columns.
    - `TestUpsertLogSourceLastPresetWins`.
    - `TestUpsertLogSourceRejectsForeignService`, asserting that the error contains both node ids.
    - `TestUpsertLogSourceRejectsServiceAndApp`.
    - `TestRemoveServiceRemovesItsSources`.
    - `TestAddLogSourceConcurrentCreatesOneCollector`: two goroutines with `sync.WaitGroup.Go`, each in its own transaction, give one collector.
  - `log_parser_preset_helpers_test.go`:
    - `TestSyncBuiltInPresetsRenamesClash`: a custom `syslog` exists before sync; afterwards the custom one is `syslog_custom`, keeps its id, and the built-in `syslog` exists.
    - `TestRemovePresetInUseFails`.
    - `TestChangeBuiltInFails`.
  - `presets_grpc_test.go`, with mocks for `Validator` and the state updater: `TestChangePresetPushesOnlyUsers` asserts that `RequestStateUpdate` is called for agent A, which uses the preset, and not for agent B.
- [ ] **Step 2: Run them.** `cd managed && go test ./models/ ./services/otel/ -run 'LogSource|Preset'`. Expected: FAIL.
- [ ] **Step 3: Implement** the models, the migration, the helpers, the proto (then `make gen`), the handlers and the startup sync.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS. Then write the api-tests against a Feature Build (connect a pmm-agent from `api-tests` setup, or use the server's own node):
  - **Presets:** list contains the 14 selectable built-ins; add, change and use a custom preset; removing it while in use gets 400; changing a built-in gets 400; cloning works; invalid YAML gets 400 with a clear message.
  - **Log sources:** add, list and remove; an unknown preset gets 404 naming it; a Viewer gets 403.
- [ ] **Step 5: Commit.**
  ```bash
  git add api/otel managed api-tests/otel
  git commit -s -m "PMM-15574 Add log sources and parser presets API"
  ```

### Task 4: Build `OtelCollectorParams` in pmm-managed

**Files:**
- Create:
  - `api/agent/v1/otel.proto` (the messages from the roadmap)
  - `managed/services/agents/otelcollector.go`
  - `managed/services/agents/otelcollector_test.go`
- Modify:
  - `api/agent/v1/agent.proto`: `AgentProcess.otel_collector = 9`, `StateChangedRequest.log_source_statuses = 6`
  - `version/features.go`: `OtelCollectorSupportVersion`
  - `managed/services/agents/state.go`:
    - a `case models.OtelCollectorType` in `sendSetStateRequest` (~:281-290), skipped with a warning for agents below the version (pattern at :304-309);
    - in `AgentFilters`, `IgnoreOtelCollector: !settings.IsOtelCollectorEnabled()` (pattern `IgnoreNomad` ~:225; filter field at `agent_helpers.go:244-262,345`).

**Interfaces:**
- Consumes `models.FindLogSources` and `FindLogParserPresets` (Task 3), and `models.MergeLabels`.
- Produces `func otelCollectorConfig(agent *models.Agent, node *models.Node, sources []*models.LogSource, presetsByID map[string]*models.LogParserPreset, services map[string]*models.Service, isServerNode bool, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_AgentProcess`. It returns:
  - `Type: AGENT_TYPE_OTEL_COLLECTOR`;
  - `OtelCollector.ResourceAttributes = {"pmm.node_id", "pmm.agent_id"}`;
  - for each source:
    - `Id = log_sources.id`, `Kind`, `Path`, `Units`;
    - `OperatorsYaml` = the preset's operators. A journald source uses the hidden `journald` preset.
    - `ResourceAttributes`: for a service source, `pmm.service_id` and `pmm.service_name`, plus `service.name` set to the program (`mysqld`, `postgres`, `mongod`, by service type; D4). For an app source, `service.name = app_name`. Then the source's custom labels.
    - `Discovered = discovery != ""`.
  - A source whose preset has `MinAgentVersion` above the pmm-agent version is left out, and its DB state is set to `agent_too_old`.
  - `SendToLocalReceiver = isServerNode`, `SendingQueueMib = 1024`.
  - Sources are sorted by id, and map keys are sorted on marshal, so equal input always gives an equal message.
  - `Args` and `TextFiles` are empty: pmm-agent renders the config itself (D11).

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestOtelCollectorConfigIdentity(t *testing.T) {
      p := otelCollectorConfig(collector, node, []*models.LogSource{svcSource, appSource, osSource}, presets, services, false, v3_11)
      require.Len(t, p.OtelCollector.LogSources, 3)
      assert.Equal(t, map[string]string{"pmm.node_id": node.NodeID, "pmm.agent_id": collector.AgentID}, p.OtelCollector.ResourceAttributes)
      assert.Equal(t, map[string]string{"pmm.service_id": "svc1", "pmm.service_name": "mysql-1", "service.name": "mysqld"}, byID(p, "src-svc").ResourceAttributes)
      assert.Equal(t, map[string]string{"service.name": "billing"}, byID(p, "src-app").ResourceAttributes)
      assert.Empty(t, byID(p, "src-os").ResourceAttributes)
      assert.Empty(t, p.Args)
      assert.Empty(t, p.TextFiles)
  }

  func TestOtelCollectorConfigDeterministic(t *testing.T) {
      a := otelCollectorConfig(collector, node, shuffled(sources), presets, services, false, v3_11)
      b := otelCollectorConfig(collector, node, sources, presets, services, false, v3_11)
      assert.True(t, proto.Equal(a, b))
  }
  ```
  Add `TestSendSetStateWithholdsOtelFromOldAgent`: with a pmm-agent at 3.10.0, the request has no `otel-collector` process. Add `TestSendSetStateIgnoresOtelWhenDisabled`.
- [ ] **Step 2: Run them.** `make gen && cd managed && go test ./services/agents/ -run Otel`. Expected: FAIL.
- [ ] **Step 3: Implement** `otelCollectorConfig` and the `state.go` changes.
- [ ] **Step 4: Run them again.** Same command. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add api/agent version managed/services/agents managed/models
  git commit -s -m "PMM-15572 Send typed collector params to pmm-agent"
  ```

### Task 5: pmm-agent config: binary path, data dir, allow-list

**Files:**
- Modify:
  - `agent/config/config.go`:
    - `Paths.OtelCollector` and `Paths.OtelCollectorDataDir` (:112-136);
    - defaults in `applyDefaults` (`tools/otelcol` via the tools branch at :315-326; `<paths_base>/data/otelcol` like `NomadDataDir` at :304-307);
    - `clearDerivedPaths` (:434-466);
    - Kingpin flags with `PMM_AGENT_*` env vars (:517-548);
    - new `type LogSources struct { AllowedPaths []string \`yaml:"allowed-paths"\`; AllowedJournaldUnits []string \`yaml:"allowed-journald-units"\` }`, added to `Config` as `LogSources LogSources \`yaml:"log-sources"\``. The default is `AllowedPaths = ["/var/log"]` when the key is absent; an explicit empty list stays empty.
  - `agent/config/config_test.go`
  - `managed/services/supervisord/pmm_config.go` (:180-192): add `--paths-otelcol-data-dir=/srv/otelcol/data --log-sources-allowed-paths=/srv/logs` to the server's `[program:pmm-agent]` command, and update the golden `managed/testdata/supervisord.d/pmm.ini`.
  - `build/docker/server/entrypoint.sh`: create `/srv/otelcol/data`.

**Interfaces:**
- Produces `cfg.Paths.OtelCollector`, `cfg.Paths.OtelCollectorDataDir` and `cfg.LogSources`. Task 6 reads them.

- [ ] **Step 1: Write the failing tests.**
  - `TestLoadDefaultsOtel`: an empty config file gives `Paths.OtelCollector == "/usr/local/percona/pmm/tools/otelcol"`, `OtelCollectorDataDir == "/usr/local/percona/pmm/data/otelcol"`, and `LogSources.AllowedPaths == []string{"/var/log"}`.
  - `TestLoadKeepsAllowListAcrossSave`: load a file with `allowed-paths: [/var/log, /opt/app/logs]`, call `SaveToFile`, reload, and get the same list. This is the upgrade-survival AC from PMM-15577.
  - `TestEnvAllowList`: `PMM_AGENT_LOG_SOURCES_ALLOWED_PATHS=/srv/logs` gives `[]string{"/srv/logs"}`.
- [ ] **Step 2: Run them.** `cd agent && go test ./config/ -run 'Otel|AllowList'`. Expected: FAIL.
- [ ] **Step 3: Implement** the fields, defaults and flags, then the server program flags and the entrypoint directory.
- [ ] **Step 4: Run them again.** Same command, plus `cd managed && go test ./services/supervisord/`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add agent/config managed/services/supervisord managed/testdata build/docker/server/entrypoint.sh
  git commit -s -m "PMM-15572 Add collector paths and log allow-list"
  ```

### Task 6: Render the collector config on pmm-agent

**Files:**
- Create:
  - `agent/agents/otelcollector/render.go`
  - `agent/agents/otelcollector/allowlist.go`
  - `agent/agents/otelcollector/render_test.go`
  - `agent/agents/otelcollector/allowlist_test.go`
  - `agent/agents/otelcollector/testdata/*.golden.yaml`

**Interfaces:**
- Consumes `agentv1.OtelCollectorParams` (Task 4) and `config.LogSources` (Task 5).
- Produces:
  - `type RenderInput struct { ServerURL *url.URL; Username, Password string; InsecureTLS bool; DataDir string; TelemetryPort uint16; AllowedPaths, AllowedUnits []string; Readable func(path string) error; JournalctlAvailable bool; DiscoveredPaths map[string]bool }`. Plan 04 fills `DiscoveredPaths`; it is nil here.
  - `func Render(p *agentv1.OtelCollectorParams, in RenderInput) (yaml []byte, statuses []*agentv1.LogSourceStatus, err error)`
  - `func CheckAllowed(path string, allowed []string) error`. It cleans the path, rejects a path that is not absolute or contains `..`, resolves symlinks with `filepath.EvalSymlinks` when the path exists, checks the static prefix of a glob, and requires a resolved prefix to be under one of the `allowed` entries.
- Rendering rules (D11):
  - **One pipeline per accepted source,** `logs/<id>`: receivers `[filelog/<id>]` or `[journald/<id>]`, processors `[memory_limiter, resource/pmm, resource/<id>, batch]`, exporters `[otlphttp]`.
  - **`filelog/<id>`:** `include: [path]`, `include_file_path: true`, `start_at: end`, `storage: file_storage`, `max_log_size: 65536`, `operators` = the parsed `OperatorsYaml`.
  - **`journald/<id>`:** `units`, `start_at: end`, `storage: file_storage`, `operators`.
  - **`resource/pmm` and `resource/<id>`:** `actions` with `upsert` for each attribute.
  - **`memory_limiter`:** `check_interval: 1s`, `limit_mib: 128`.
  - **`batch`:** `timeout: 5s`, `send_batch_size: 2000`, `send_batch_max_size: 2000`.
  - **`otlphttp`:**
    - `endpoint`: `http://127.0.0.1:4318` when `SendToLocalReceiver`, otherwise `<ServerURL>/otlp`;
    - `headers.Authorization`: `Basic base64(user:pass)`, and only when not local;
    - `tls.insecure_skip_verify: InsecureTLS`, `compression: gzip`;
    - `retry_on_failure: {enabled: true, initial_interval: 5s, max_interval: 30s, max_elapsed_time: 0}`;
    - `sending_queue: {enabled: true, storage: file_storage, sizer: bytes, queue_size: SendingQueueMib·2²⁰}`. Check that the pinned collector supports the `bytes` sizer; if it doesn't, use `sizer: requests` with `queue_size: 5000`, and record the choice.
  - **`extensions.file_storage`:** `directory: <DataDir>/storage`, `create_directory: true`, `compaction.on_start: true`.
  - **`service.telemetry.metrics`:** a pull reader on `127.0.0.1:<TelemetryPort>`.
  - **No `receivers.otlp`.**
  - **Escaping:** every string that comes from params or credentials has `$` replaced with `$$` before marshalling.
  - **Statuses:** a file source outside the allow-list gives `NOT_ALLOWED`, unless `DiscoveredPaths[path]`. `Readable(path)` failing gives `NOT_READABLE` or `NOT_FOUND`. A journald source gives `JOURNALD_UNAVAILABLE` when `journalctl` is unavailable, and `NOT_ALLOWED` when a unit is not in `AllowedUnits`. An accepted source gives `COLLECTING`.
  - **No accepted sources:** the config still renders with no pipelines, and the caller (Task 7) doesn't start the process.

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestRenderGolden(t *testing.T) { /* params with one file, one journald, one app source; compare with testdata/mixed.golden.yaml */ }

  func TestRenderRejectsSymlinkEscape(t *testing.T) {
      dir := t.TempDir()
      require.NoError(t, os.Symlink("/etc/hostname", filepath.Join(dir, "x.log")))
      _, st, err := Render(paramsWithFile(filepath.Join(dir, "x.log")), RenderInput{AllowedPaths: []string{dir}, Readable: okReadable})
      require.NoError(t, err)
      assert.Equal(t, agentv1.LogSourceState_LOG_SOURCE_STATE_NOT_ALLOWED, st[0].State)
  }

  func TestAllowListGlobsAndDotDot(t *testing.T) {
      require.NoError(t, CheckAllowed("/var/log/mysql/*.log", []string{"/var/log"}))
      require.Error(t, CheckAllowed("/var/log/../etc/shadow", []string{"/var/log"}))
      require.Error(t, CheckAllowed("relative/path.log", []string{"/var/log"}))
      require.Error(t, CheckAllowed("/var/logs-other/x.log", []string{"/var/log"}))
  }

  func TestRenderEscapesDollarAndKeepsBraces(t *testing.T) {
      y, _, err := Render(paramsWithOperators("- type: add\n  field: body.x\n  value: '{{ .server_auth_b64 }} $HOME'\n"), RenderInput{Password: "pa$$word", /* … */})
      require.NoError(t, err)
      assert.Contains(t, string(y), "{{ .server_auth_b64 }} $$HOME")
      assert.NotContains(t, string(y), "server_password")
  }

  func TestRenderedConfigBoundsRequestSize(t *testing.T) { /* assert max_log_size 65536, batch max 2000, compression gzip */ }
  func TestRenderedFilelogUsesStorageAndStartAtEnd(t *testing.T) { /* assert storage, start_at, include_file_path */ }
  func TestRenderLocalReceiverHasNoAuth(t *testing.T) { /* SendToLocalReceiver: endpoint 127.0.0.1:4318, no Authorization header */ }
  func TestRenderHasNoOTLPReceiver(t *testing.T) { /* receivers has no key "otlp" */ }
  ```
- [ ] **Step 2: Run them.** `cd agent && go test ./agents/otelcollector/`. Expected: FAIL.
- [ ] **Step 3: Implement `Render` and `CheckAllowed`.** Model each config section as a struct; parse operators into `[]yaml.Node`, so they are embedded without re-serialising through `map[string]any`.
- [ ] **Step 4: Run them again.** Same command. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add agent/agents/otelcollector
  git commit -s -m "PMM-15572 Render collector config on pmm-agent"
  ```

### Task 7: Run the collector as `pmm-agent`, and report per-source status

**Files:**
- Create:
  - `agent/agents/otelcollector/credential.go`
  - `agent/agents/otelcollector/credential_test.go`
  - `agent/agents/otelcollector/readable.go`
- Modify:
  - `agent/agents/process/process.go`:
    - a `Credential *syscall.Credential` field in `Params` (:75-87);
    - in `toStarting` (:150-178), set `p.cmd.SysProcAttr.Credential` **after** `pdeathsig.Set`, so `Pdeathsig` is kept.
  - `agent/agents/supervisor/supervisor.go`:
    - a `case inventoryv1.AgentType_AGENT_TYPE_OTEL_COLLECTOR` in `processParams` (:1118-1219). It calls `Render`, writes the YAML to `<OtelCollectorDataDir>/<agentID>/config.yaml` with mode 0600, chowns that directory tree to the credential, sets `Path = cfg.Paths.OtelCollector`, `Args = ["--config=<file>"]`, `Env = ["PATH=/usr/bin:/bin"]` and `Credential`, and keeps the statuses on the agent's process info;
    - `version()` (:1262-1283) with a regex for `otelcol version`, in `deps.go:26`;
    - send `StateChangedRequest.LogSourceStatuses` together with the process status (:822-829), and again whenever the statuses change.
  - `managed/services/agents/handler.go` (`stateChanged` ~:129,223): store the statuses through `models.UpdateLogSourceStates(q, statuses)`.
  - `build/packages/rpm/client/pmm-client.spec` and `build/packages/deb/postinst`: after the user is created, `usermod -aG systemd-journal pmm-agent` when `getent group systemd-journal` succeeds, and `-aG adm` on Debian/Ubuntu when `adm` exists.

**Interfaces:**
- Consumes `Render` (Task 6) and `config.Paths` and `LogSources` (Task 5).
- Produces:
  - `func CollectorCredential(euid int, lookup func(string) (*user.User, error), groupIDs func(*user.User) ([]string, error)) (*syscall.Credential, error)`. It returns `nil, nil` when `euid != 0`. As root, it returns the `pmm-agent` uid, gid and supplementary groups. If the user is missing, it returns an error, which becomes the agent status `INITIALIZATION_ERROR` with the message "user pmm-agent not found; the collector does not run as root".
  - `func ReadableAs(cred *syscall.Credential) func(path string) error`. It checks the owner, group and other read bits on the file (or, for a glob, its directory), and the execute bit on each parent directory. When the file or a parent has a `system.posix_acl_access` xattr, the check is skipped.
  - `func UpdateLogSourceStates(q *reform.Querier, statuses []*agentv1.LogSourceStatus) error`

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestCollectorCredential(t *testing.T) {
      lookup := func(string) (*user.User, error) { return &user.User{Uid: "998", Gid: "997", Username: "pmm-agent"}, nil }
      groups := func(*user.User) ([]string, error) { return []string{"997", "4", "190"}, nil }
      c, err := CollectorCredential(0, lookup, groups)
      require.NoError(t, err)
      assert.Equal(t, &syscall.Credential{Uid: 998, Gid: 997, Groups: []uint32{997, 4, 190}}, c)

      c, err = CollectorCredential(1002, lookup, groups)
      require.NoError(t, err)
      assert.Nil(t, c)

      _, err = CollectorCredential(0, func(string) (*user.User, error) { return nil, user.UnknownUserError("pmm-agent") }, groups)
      require.ErrorContains(t, err, "does not run as root")
  }
  ```
  Also write:
  - `TestReadableAs`, on temp files with modes 0600, 0640 and 0644 and a fake credential.
  - In the supervisor tests, `TestProcessParamsOtelCollector`: the config file exists with mode 0600, `Args[0]` starts with `--config=`, and `Env` is `["PATH=/usr/bin:/bin"]`.
  - In `managed/services/agents`, `TestStateChangedStoresLogSourceStates` (sqlmock): the expected UPDATE on `log_sources` sets `state` and `state_reason` for the given id.
- [ ] **Step 2: Run them.** `cd agent && go test ./agents/otelcollector/ ./agents/supervisor/ ./agents/process/` and `cd managed && go test ./services/agents/ -run StateChanged`. Expected: FAIL.
- [ ] **Step 3: Implement** everything listed above, including the package scripts.
- [ ] **Step 4: Run them again.** Same commands, then `make prepare-pr`. Expected: PASS. Then on a CHAOS VM with a Feature Build (`pmm-qa:chaos-docker-provisioning`):
  - `ps -o user= -C otelcol` prints `pmm-agent`.
  - `ss -ltnp | grep otelcol` shows only `127.0.0.1:<listen_port>`.
  - Add a source with a path outside `/var/log`; `GET /v1/otel/log-sources` shows `NOT_ALLOWED`.
- [ ] **Step 5: Commit.**
  ```bash
  git add agent managed build/packages
  git commit -s -m "PMM-15572 Run collector as pmm-agent, report sources"
  ```

### Task 8: PMM Server's own logs

**Files:**
- Create:
  - `managed/services/otel/server_sources.go`
  - `managed/services/otel/server_sources_test.go`
- Modify:
  - `managed/services/agents/registry.go`: in `authenticate` (~:353-405), call `ensureServerOtelCollector` for the server's pmm-agent (`models.PMMServerAgentID`, or the HA UUID) when OTEL is enabled. Copy `addNomadAgentToPMMAgent` (:526-545).

**Interfaces:**
- Consumes `models.UpsertLogSource` (Task 3) and `SendToLocalReceiver` (Task 4).
- Produces:
  - `type ServerSource struct { File, Preset, App, ServiceID string }`
  - `func DefaultServerSources() []ServerSource`. In `/srv/logs`:

    | File | Preset | `service.name` |
    |---|---|---|
    | `nginx.log` | `nginx_error`, which drops access lines (PMM-15575) | `nginx` |
    | `grafana.log` | `grafana` | `grafana` |
    | `pmm-managed.log` | `pmm_managed` | `pmm-managed` |
    | `pmm-agent.log` | `pmm_agent` | `pmm-agent` |
    | `postgresql.log` | `postgres` | — (bound to the `pmm-server-postgresql` service) |
    | `clickhouse-server.log` | `clickhouse_server` | `clickhouse` |
    | `otel-collector.log` | `otel_collector` | `otel-collector` |
    | `supervisord.log` | `supervisord` | `supervisord` |
    | `qan-api2.log` | `pmm_agent`, or `pmm_logrus` from Task 2 | `qan-api2` |
    | `vmproxy.log` | same as `qan-api2.log` | `vmproxy` |

    The real file is `postgresql.log`. PostgreSQL 18 writes to stdout and supervisord stores it there (`pmm_config.go:106-128`), so PMM-15575's `postgresql14.log` is stale.
  - `func (r *Registry) ensureServerOtelCollector(q *reform.Querier, pmmAgentID string, v *version.Parsed) error`. It is idempotent, upserting each default source.
  - To collect nginx access lines, an admin changes that source's preset to `nginx_access` through the API or the UI.

- [ ] **Step 1: Write the failing tests.**
  - `TestDefaultServerSourcesMatchShippedLogs`: every `File` appears as a `stdout_logfile` in `managed/services/supervisord/pmm_config.go` or `supervisord.go` templates, or in `build/ansible/roles/supervisord/files/supervisord.ini`. Read those files in the test and match with a regex, so that a renamed log file fails the test.
  - `TestEnsureServerOtelCollectorIdempotent` (`testdb`): calling it twice gives one collector and 10 sources.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/otel/ ./services/agents/ -run 'ServerSources|EnsureServerOtel'`. Expected: FAIL.
- [ ] **Step 3: Implement** the source list and the ensure hook.
- [ ] **Step 4: Run them again.** Same command. Expected: PASS. Then on a Feature Build, after 2 minutes:
  ```bash
  docker exec pmm-server clickhouse-client -q "SELECT ServiceName, count() FROM otel.logs WHERE Timestamp > now() - INTERVAL 10 MINUTE GROUP BY 1"
  ```
  Expect one row per component, no nginx access lines, and a `SeverityText` set on each row.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/otel managed/services/agents/registry.go
  git commit -s -m "PMM-15575 Collect PMM Server's own logs"
  ```

### Task 9: Collector self-metrics into VictoriaMetrics

**Files:**
- Modify:
  - `managed/services/victoriametrics/prometheus.go`: a scrape config for `models.OtelCollectorType` in the agent switch (~:285-296), job `otel-collector`, target `127.0.0.1:<listen_port>`, path `/metrics`, with `node_id`, `agent_id` and `agent_type` labels from `MergeLabels`
  - `managed/services/victoriametrics/scrape_configs.go`: a server scrape of `127.0.0.1:4319`, job `otel-collector-server`, added in `addInternalServicesToScrape` (:311-326) when OTEL is enabled
  - `managed/services/victoriametrics/prometheus_test.go`
  - `managed/services/victoriametrics/scrape_configs_test.go`

**Interfaces:**
- Consumes `OtelCollector.listen_port`, which pmm-agent assigns from its port range in Task 7 (`TelemetryPort = port`).

- [ ] **Step 1: Write the failing tests.** `TestScrapeConfigOtelCollector` asserts job, target and labels. `TestInternalScrapeOtelServer` asserts `127.0.0.1:4319` when enabled and no job when disabled.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/victoriametrics/ -run Otel`. Expected: FAIL.
- [ ] **Step 3: Implement** both scrape configs.
- [ ] **Step 4: Run them again.** Same command. Expected: PASS. On a Feature Build, `otelcol_receiver_accepted_log_records_total` is queryable in VictoriaMetrics for each node collector and for the server collector.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/victoriametrics
  git commit -s -m "PMM-15593 Scrape OTEL collector self-metrics"
  ```

### Task 10: Inventory type in the Grafana fork, and telemetry

**Files:**
- In `/srv/runner/2/percona/grafana`, on branch `PMM-15572-otel-collector-type`, modify:
  - `public/app/percona/inventory/Inventory.types.ts` (:28-50): `OtelCollector = 'otel-collector'`
  - `public/app/percona/inventory/Tabs/Agents.constants.ts` (:25-47): name `OTEL collector`
  - the matching tests
- In percona/pmm, modify `managed/services/telemetry/config.default.yml`. Add metrics:
  - `otel_collector_enabled` (from settings);
  - `otel_log_sources_count` (`SELECT count(*) FROM log_sources`);
  - `otel_daily_ingested_bytes`, through the ClickHouse telemetry datasource: `SELECT sum(bytes_on_disk) FROM system.parts WHERE database='otel' AND active AND partition = toString(yesterday())`.

- [ ] **Step 1: Write the failing test** in the Grafana fork: the Agents tab renders an `otel-collector` agent as "OTEL collector". Run it with `yarn test Agents`. Expected: FAIL.
- [ ] **Step 2: Implement** the type and name. Run the test again. Expected: PASS. Commit with `git commit -s -m "PMM-15572 Show the OTEL collector in Inventory"`.
- [ ] **Step 3: Write the failing test** for telemetry: extend the telemetry config test in `managed/services/telemetry` so it asserts the three new metric names load. Run it. Expected: FAIL.
- [ ] **Step 4: Implement** the config entries. Run `cd managed && go test ./services/telemetry/`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/telemetry
  git commit -s -m "PMM-15571 Report OTEL usage in telemetry"
  ```
