# Logs Part 2B: rules and detection — Implementation Plan

> **Deferred draft (2026-10-17):** Part 2B is postponed until alerting is revisited (see the spec's Part 2B note).
> Tasks 1, 2, 4 and 5 are written; Tasks 3 and 6–10 wait on the Grafana credential decision. The preset tag (Task 4)
> and the nginx migration (Task 2) move to the Part 3 plan; drop them here when this plan is resumed.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Log rules (built-in and admin-made) tag matching lines in the server collector, count them into
`pmm_log_rule_matches_total`, and own one Grafana alert rule each, so high-signal problems alert with no setup; admins
manage rules on a new Logs → Rules tab and the Assistant can read the lines behind a log-rule alert.

**Architecture:** pmm-managed stores rules in a `log_rules` table (built-ins synced from embedded YAML). A new
`managed/services/logrules` service owns the rules API, validation (Go regexp, then `otelcol-contrib validate` through
supervisord), a 10 s batched collector refresh, and the Grafana alert sync (provisioning HTTP API, leader only). The
collector config gains a preset tag on every receiver, a `transform/pmm_rules` processor, a `count` connector and a
remote-write metrics pipeline, rendered by `managed/otel`.

**Tech Stack:** Go (pmm-managed, reform, gRPC + grpc-gateway), otelcol-contrib 0.148.0 (`transform`, `count`,
`deltatocumulative`, `prometheusremotewrite`), Grafana 12.4.5 provisioning API, VictoriaMetrics, ClickHouse `otel.logs`,
React + MUI + TanStack Query (pmm-ui), HolmesGPT config and skills.

**Spec:** `dev/otel/logs-design.md`, Part 2B (2B.1–2B.10).

## Global Constraints

- Branch `PMM-7-tibi-holmes`, pushed to `origin tibi-holmes`; commits `PMM-7 <summary ≤50 chars>`, signed off
  (`git commit -s`), `git add` and `git commit` in separate calls (a hook blocks them combined).
- Migrations are forward-only: add `149` and `150`; never edit 127–148.
- Proto changes are additive; run `make gen` after editing `.proto` files or mocked interfaces; never hand-edit
  generated files.
- Go conventions from `AGENTS.md`: no inline `if err := …`, no named returns, comments on their own line, one-sentence
  doc comments, no `%q` in error or log messages, logrus `*logrus.Entry`, `status.Error` with gRPC codes in services.
- Rule name `^[a-z0-9_]{1,64}$`; `match` ≤ 1024 chars; window 1 m to 1 h; at most 200 enabled rules; alert severities
  `info`, `warning`, `error`, `critical`.
- Grafana folder UID `pmm-log-rules`, title `PMM Log Rules`, group evaluated every 1 m; alert rule UID `pmm-log-` +
  first 12 hex chars of `sha1(name)`; label `pmm_log_rule=<name>`.
- Metric `pmm_log_rule_matches_total{pmm_rule, node_name, service_name, pmm_collector}`; attribute names
  `log.preset`, `pmm.rule`.
- The runner host runs no PostgreSQL, ClickHouse, Grafana or Docker: `testdb` tests and live checks run on CHAOS VM
  `adre-pmm7` (`/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh`). Compile test
  binaries locally and run them inside `pmm-server` as user `pmm`.
- Never print secrets from the VM (admin password, tokens, model keys); read them on the VM and pass them via files or
  environment there.
- Lint Go with `bin/golangci-lint run <pkgs>`; UI with `cd ui && make lint && make format-check`; no new dependencies.

## Review Focus

- A rule whose regex Go accepts but OTTL or the collector rejects must fail its save with the collector's message and
  leave the running collector untouched (Task 5, `TestCreateLogRule_CollectorRejects`).
- Rule strings with quotes, backslashes, `$`, newlines and non-ASCII must reach the collector unchanged and match the
  same lines as ClickHouse `match()` (Task 4, `TestRuleStatement_Escaping`; Task 10 live).
- Grafana down or slow during a rule save: the save succeeds, the alert state reads `pending`, and the next sync
  creates the rule; a sync never deletes alert rules outside the `pmm-log-rules` folder (Task 6,
  `TestSyncAlerts_GrafanaDown`, `TestSyncAlerts_OnlyOwnFolder`).
- An upgrade must not resurrect a deleted built-in or overwrite an admin-edited one (Task 3,
  `TestSyncBuiltinRules`).
- Ten rule edits within a few seconds cause one collector restart, not ten (Task 5,
  `TestCollectorRefresh_Batches`).

---

### Task 1: Spike — prove the pipeline on the test server

No product code. Findings go into `dev/otel/logs-design.md` 2B.3 ("Verified" list) in one commit; any failed check
stops the plan and goes back to the user with a proposed design change.

**Files:**
- Modify: `dev/otel/logs-design.md` (2B.3)
- Scratch only (VM `/tmp/spike/`, never committed): a collector config, a log file, a curl script

**Interfaces:**
- Produces: the confirmed metric name, label names, remote-write URL, whether `count` reads resource attributes, the
  CPU figure, and the Grafana credential pmm-managed will use for background calls. Tasks 4 and 6 consume them.

- [ ] **Step 1: Pipeline config.** In `pmm-server` on `adre-pmm7`, as user `pmm`, write `/tmp/spike/config.yaml`: a
  `filelog` receiver on `/tmp/spike/in.log` (`start_at: beginning`, `resource: {node_name: spike-node, service.name:
  spike-svc}`, one `add` operator setting `attributes["log.preset"]` to `postgres`); `transform/pmm_rules` with 30
  statements in the spec's shape (29 that never match, the last matching `deadlock detected`); `count/pmm_rules` as in
  spec 2B.3; `deltatocumulative`; `prometheusremotewrite` to `http://127.0.0.1:9090/prometheus/api/v1/write` with
  `external_labels: {pmm_collector: spike}`; `service.telemetry.metrics.level: none`; no OTLP receiver.
- [ ] **Step 2: Validate and run.** `otelcol-contrib validate --config=/tmp/spike/config.yaml`, then run it in the
  background. Expected: validate exits 0; the collector logs "Everything is ready" and no errors.
- [ ] **Step 3: Counts.** Append 5 matching and 5 non-matching lines, wait 30 s, append 3 more matching lines, wait
  30 s. Query VictoriaMetrics (`/prometheus/api/v1/query?query={__name__=~"pmm_log_rule_matches.*"}`).
  Expected: one series, value 8 (cumulative, not 5 then 3), labels include `pmm_rule="spike_rule"` and
  `pmm_collector="spike"`; record the exact metric name and whether `node_name`/`service_name` are present (check 1).
  If they are missing, repeat with `transform/pmm_rules` also copying `resource.attributes["node_name"]` and
  `["service.name"]` onto the record and record that the copy is required.
- [ ] **Step 4: CPU.** Replay 10 minutes of real lines from `otel.logs` (all services) into a second file at the
  original rate, once with the 30 statements and once with `transform/pmm_rules` removed; sample the process CPU from
  `/proc/<pid>/stat` over 120 s each. Expected: the difference is recorded (target: under 5 % of one core).
- [ ] **Step 5: Grafana provisioning API.** With the credential pmm-managed can use outside a request (see Task 6's
  Interfaces), create folder `pmm-log-rules` and one alert rule through `POST /graph/api/v1/provisioning/alert-rules`
  with the spec's query, read it back, check `provenance` is `api`, open Alerting in the UI (screenshot via
  `playwright-cli`) to confirm it is read-only, then `DELETE` it and the folder. Compare `supervisorctl status grafana`
  uptime before and after. Expected: no Grafana restart; rule visible and read-only; delete works.
- [ ] **Step 6: Clean up and record.** Stop the spike collector, delete its series
  (`/prometheus/api/v1/admin/tsdb/delete_series?match[]={pmm_collector="spike"}`), remove `/tmp/spike`. Rewrite the
  2B.3 "Verified" list with the five results (exact metric name and labels, write URL, copy needed or not, CPU figure,
  credential).
- [ ] **Step 7: Commit** — `PMM-7 Record log rules spike results`.

### Task 2: nginx error lines get a severity (migration 149)

**Files:**
- Modify: `managed/models/database.go` (migration `149`)
- Test: `managed/models/log_parser_preset_migration_test.go` (new subtest)

**Interfaces:**
- Produces: built-in `nginx_access` preset that parses both `key=value` access lines and
  `2026/10/16 20:41:58 [warn] 114#114: …` error lines; error lines get `attributes.level` and a severity.

- [ ] **Step 1: Write the failing subtest** `nginx_access parses error-format lines` in
  `TestLogParserPresetsServerLogs`:

```go
p := mustPreset(t, q, "nginx_access")
var ops []map[string]any
require.NoError(t, yaml.Unmarshal([]byte(p.OperatorYAML), &ops))
var errRegex string
for _, op := range ops {
	if op["type"] == "regex_parser" {
		errRegex, _ = op["regex"].(string)
		assert.Contains(t, op["if"], `body matches`)
	}
	if op["type"] == "key_value_parser" {
		assert.Contains(t, op["if"], `body matches`)
	}
}
re := regexp.MustCompile(errRegex)
m := re.FindStringSubmatch(`2026/10/16 20:41:58 [warn] 114#114: *259446 a client request body is buffered to a temporary file`)
require.NotNil(t, m)
assert.Equal(t, "warn", m[re.SubexpIndex("level")])
assert.Contains(t, p.OperatorYAML, "fatal: [crit, alert, emerg]")
```

- [ ] **Step 2: Run** it on the VM (compile `go test -c ./managed/models`, run inside `pmm-server` as `pmm` with
  `-test.run TestLogParserPresetsServerLogs`). Expected: FAIL (no `regex_parser`).
- [ ] **Step 3: Add migration 149**: one `UPDATE log_parser_presets SET operator_yaml = $yaml$…$yaml$, updated_at =
  NOW() WHERE name = 'nginx_access' AND built_in AND operator_yaml = $old$<the text migration 128 inserted>$old$`
  (check with `grep -n nginx_access managed/models/database.go` that no later migration changed it). New operators:
  `key_value_parser`, `time_parser` (`attributes.time`, layout `'2006-01-02T15:04:05Z07:00'`) and the level `add`, each
  with `if: 'body matches "^time="'`; then `regex_parser` with the `nginx_error` preset's regex and
  `if: 'body matches "^\\d{4}/\\d{2}/\\d{2} "'`, and its `time_parser` (layout `'2006/01/02 15:04:05'`, same `if`);
  then one `severity_parser` on `attributes.level` with mapping `debug: debug`, `info: [info, notice]`, `warn: warn`,
  `error: error`, `fatal: [crit, alert, emerg]`. One comment line above the migration (why two parsers).
- [ ] **Step 4: Run** the same command. Expected: PASS, and the whole `managed/models` suite passes on the VM.
- [ ] **Step 5: Validate with the real collector.** On the VM, render a one-receiver config with the new operators
  (Task 5's validator will do this in product code; here by hand) and run `otelcol-contrib validate`. Expected: exit 0.
- [ ] **Step 6: Commit** — `PMM-7 Give nginx error lines a severity`.

### Task 3: Rule model, migration 150 and built-in sync

_Pending: written after the credential decision._

### Task 4: Tag, match and count in the collector

**Files:**
- Create: `managed/otel/log_rules.go`, `managed/otel/log_rules_test.go`
- Modify: `managed/otel/server_logs.go` (preset tag on every server receiver), `managed/otel/server_logs_test.go`
- Modify: `managed/services/agents/otelcollector.go` (preset tag on every agent receiver), its test
- Modify: `managed/services/supervisord/otelconfig.go`, `managed/services/supervisord/supervisord.go`,
  `managed/services/supervisord/otelconfig_test.go`

**Interfaces:**
- Consumes: `models.LogRule` and `models.FindEnabledLogRules` (Task 3); Task 1's findings (metric name, write URL,
  whether node/service attributes must be copied onto the record).
- Produces:

```go
// managed/otel/log_rules.go
const (
	LogPresetAttribute       = "log.preset"
	LogRuleAttribute         = "pmm.rule"
	LogRulesProcessor        = "transform/pmm_rules"
	LogRulesConnector        = "count/pmm_rules"
	LogRulesExporter         = "prometheusremotewrite/pmm_rules"
	LogRulesMetricsPipeline  = "metrics/pmm_rules"
	LogRulesRemoteWriteURL   = "http://127.0.0.1:9090/prometheus/api/v1/write" // as confirmed by Task 1
)

// LogRules holds the collector config blocks for the enabled log rules, ready to embed; zero value = no rules.
type LogRules struct {
	ProcessorYAML string // entry under "processors:", indented two spaces
	ConnectorYAML string // entry under "connectors:"
	ExporterYAML  string // entry under "exporters:"
}

func (r LogRules) Enabled() bool
func BuildLogRules(rules []*models.LogRule, collectorID string) LogRules
// RuleStatement returns the OTTL statement that tags lines matching r.
func RuleStatement(r *models.LogRule) string

// managed/services/supervisord/otelconfig.go
func buildOtelCollectorConfigYAML(endpoint, username, password string, logRetentionDays, spanRetentionDays, metricsRetentionDays int, serverLogs otel.ServerLogs, logRules otel.LogRules) string
```

Statement shape (one per enabled rule; rules with a `match` first, then severity-only, each group by name):
`set(attributes["pmm.rule"], "<name>") where attributes["pmm.rule"] == nil and (attributes["log.preset"] == "<p1>" or
attributes["log.preset"] == "<p2>")[ and severity_number >= <min>][ and IsMatch(body, "<pattern>")]`, where `contains`
becomes `"(?i)" + regexp.QuoteMeta(match)`. Strings are OTTL double-quoted literals (`\` → `\\`, `"` → `\"`, newline →
`\n`); the statements are emitted through `yaml.v3` so YAML quoting is never hand-built; `$` becomes `$$` because the
collector expands `${…}`. Remote write sets `external_labels: {pmm_collector: <collectorID>}`; `collectorID` is
`haParams.NodeID` in HA, else `pmm-server`.

- [ ] **Step 1: Write the failing tests** in `managed/otel/log_rules_test.go`:
  - `TestBuildLogRules_Empty`: `BuildLogRules(nil, "pmm-server").Enabled()` is false.
  - `TestBuildLogRules_Order`: rules `b_regex` (match), `a_sev` (severity-only, min 21), `a_regex` (match) render in the
    order `a_regex`, `b_regex`, `a_sev` in the parsed `log_statements[0].statements`.
  - `TestRuleStatement`: a regex rule with presets `postgres`, `mysql_error`, min severity 17, match `deadlock detected`
    equals exactly
    `set(attributes["pmm.rule"], "r") where attributes["pmm.rule"] == nil and (attributes["log.preset"] == "postgres" or attributes["log.preset"] == "mysql_error") and severity_number >= 17 and IsMatch(body, "deadlock detected")`;
    a `contains` rule with match `a.b (c)` gives `IsMatch(body, "(?i)a\\.b \\(c\\)")`.
  - `TestRuleStatement_Escaping`: match `say "hi" \d+ $HOME ${X} é` + `"\n"` survives `BuildLogRules` → YAML parse as
    `IsMatch(body, "say \"hi\" \\d+ $$HOME $${X} é\n")`.
  - `TestBuildLogRules_Pipeline`: parsed `ConnectorYAML` has `count/pmm_rules.logs.pmm_log_rule_matches` with condition
    `attributes["pmm.rule"] != nil` and attribute keys `pmm.rule`, `node_name`, `service.name`; `ExporterYAML` has the
    write URL and `external_labels.pmm_collector` = the collector ID.
  - In `server_logs_test.go`: every receiver's operators end with
    `{type: add, field: 'attributes["log.preset"]', value: <preset>, on_error: send_quiet}` (`raw` for `pmm-init.log` and
    `filelog/srv_other`, which then have exactly that one operator).
  - In the agents' collector test: each `filelog/preset_<name>` receiver ends with the same `add` (`raw` for the raw
    receiver).
  - In `otelconfig_test.go`: `TestBuildOtelCollectorConfigYAML_LogRules` — with a non-empty `LogRules`, the logs pipeline
    processors are `[memory_limiter, transform, transform/pmm_rules, batch]`, its exporters
    `[clickhouse/logs, count/pmm_rules]`, and `service.pipelines.metrics/pmm_rules` has receivers `[count/pmm_rules]`,
    processors `[deltatocumulative, batch]`, exporters `[prometheusremotewrite/pmm_rules]`; with an empty `LogRules`
    none of these appear.
- [ ] **Step 2: Run** `go test ./managed/otel/... ./managed/services/supervisord/... ./managed/services/agents/...`
  (agents' testdb tests on the VM). Expected: FAIL (undefined / missing operators).
- [ ] **Step 3: Implement** `log_rules.go`; add the `add` operator in `writeServerReceiver` (before `quietOperators`
  runs, so it gets `send_quiet`) and in the agents' receiver loop; extend `buildOtelCollectorConfigYAML`; in
  `supervisord.go` load `models.FindEnabledLogRules` next to the presets (same error handling as presets: a load error
  keeps the old config) and pass `otel.BuildLogRules(rules, collectorID)`. If Task 1 recorded that `count` does not
  read resource attributes, append `set(attributes["node_name"], resource.attributes["node_name"]) where
  attributes["pmm.rule"] != nil` and the same for `service.name` after the rule statements, and cover them in
  `TestBuildLogRules_Pipeline`.
- [ ] **Step 4: Run** the same tests. Expected: PASS.
- [ ] **Step 5: Validate with the real collector.** Build pmm-managed, deploy it to `adre-pmm7` (Task 3's built-ins are
  in the database), check `/srv/otelcol/config.yaml` has 26 statements and the metrics pipeline, and that
  `otel-collector` restarted with no error lines in `/srv/logs/otel-collector.log`. Expected: as stated.
- [ ] **Step 6: Commit** — `PMM-7 Tag and count log rule matches`.

### Task 5: Rule service core — validate, save, apply

**Files:**
- Create: `managed/services/logrules/service.go`, `managed/services/logrules/deps.go`,
  `managed/services/logrules/validate.go`, tests `service_test.go`, `validate_test.go`
- Modify: `managed/services/supervisord/supervisord.go` (validators, collector-only update), its tests
- Modify: `managed/services/server/log_parser_presets.go`, `managed/services/server/deps.go`,
  `managed/services/server/server.go` (`Params.OtelValidator`), `log_parser_presets_test.go`
- Modify: `.mockery.yaml` (new package `managed/services/logrules`: `collectorService`, `alertSyncer`, `settingsGetter`
  if used; `managed/services/server`: `otelValidator`), then `make gen`
- Modify: `managed/cmd/pmm-managed/main.go` (construct `logrules.New`, `go logRules.Run(ctx)`)

**Interfaces:**
- Consumes: `models.LogRule`, `models.CreateLogRule`, `models.UpdateLogRule`, `models.SoftDeleteLogRule`,
  `models.ResetLogRule`, `models.FindLogRule`, `models.FindEnabledLogRules`, `models.FindLogRules` (Task 3);
  `otel.BuildLogRules` (Task 4).
- Produces:

```go
// managed/services/supervisord
// ValidateLogRules renders the server collector config with rules in place of the stored ones and runs the collector's validator.
func (s *Service) ValidateLogRules(rules []*models.LogRule) error
// ValidateLogParserOperators checks preset operators in a one-receiver collector config.
func (s *Service) ValidateLogParserOperators(operatorYAML, lineStartPattern string) error
// UpdateOtelCollector re-renders only the collector config and restarts the collector when it changed.
func (s *Service) UpdateOtelCollector(settings *models.Settings) error

// managed/services/logrules
type collectorService interface {
	ValidateLogRules(rules []*models.LogRule) error
	UpdateOtelCollector(settings *models.Settings) error
}
type alertSyncer interface {
	// SyncAlerts makes the Grafana alert rules match the enabled, alert-on log rules (Task 6).
	SyncAlerts(ctx context.Context) error
}
func New(db *reform.DB, collector collectorService, alerts alertSyncer) *Service
func (s *Service) Run(ctx context.Context)
func (s *Service) CreateRule(ctx context.Context, r *models.LogRule, login string) (*models.LogRule, error)
func (s *Service) UpdateRule(ctx context.Context, name string, params models.UpdateLogRuleParams, login string) (*models.LogRule, error)
func (s *Service) DeleteRule(ctx context.Context, name string) error
func (s *Service) ResetRule(ctx context.Context, name string) (*models.LogRule, error)
// ValidateRule checks one rule without the collector: name, presets, match, severities, window, threshold.
func ValidateRule(q *reform.Querier, r *models.LogRule) error
```

Behavior: every write validates (`ValidateRule`, the 200-enabled cap, then `collector.ValidateLogRules` with the
candidate set) before touching the database, returns `codes.InvalidArgument` with the reason (the collector's output
for a rejection), then signals `Run`. `Run` waits until 10 s pass with no further signal, then calls
`collector.UpdateOtelCollector(settings)` and `alerts.SyncAlerts`; it also calls `SyncAlerts` every 5 min and once
at start. The debounce and tick durations are fields so tests can shorten them. `alerts` may be nil until Task 6.

- [ ] **Step 1: Write the failing tests:**
  - `validate_test.go` `TestValidateRule` (table; sqlmock for the preset lookup): bad name `Bad-Name`, empty presets,
    unknown preset, regex `(`, `contains` with 1025 chars, empty match with `min_severity` 0, window 30 s and 2 h,
    threshold 0, severity `fatal` → each an error naming the field; a valid regex rule and a valid severity-only rule →
    nil.
  - `service_test.go` (testdb on the VM, mocks for `collectorService` and `alertSyncer`):
    - `TestCreateLogRule_CollectorRejects`: the mock's `ValidateLogRules` returns `errors.New("bad statement")` →
      `status.Code(err) == codes.InvalidArgument`, message contains `bad statement`, and `models.FindLogRule` returns
      not found; `UpdateOtelCollector` never called.
    - `TestCreateLogRule_Applies`: valid rule → saved, and within the shortened debounce `UpdateOtelCollector` and
      `SyncAlerts` are each called once.
    - `TestCollectorRefresh_Batches`: ten `UpdateRule` calls 10 ms apart with a 200 ms debounce →
      `UpdateOtelCollector` called exactly once.
    - `TestEnabledRuleCap`: with 200 enabled rules, creating an enabled one fails with `InvalidArgument`; creating a
      disabled one succeeds.
  - supervisord: `TestValidateLogParserOperators_Config` (the rendered one-receiver config parses and contains the
    operators and `multiline.line_start_pattern`; validator injected as a func field so the test does not need the
    binary).
  - server: `TestChangeLogParserPreset_CollectorRejects` — the `otelValidator` mock returns an error →
    `InvalidArgument`, preset unchanged.
- [ ] **Step 2: Run** `go test ./managed/services/logrules/... ./managed/services/supervisord/... ./managed/services/server/...`
  (testdb ones on the VM). Expected: FAIL (undefined).
- [ ] **Step 3: Implement.** In supervisord, render to a temp file in `/srv/otelcol` and reuse
  `validateOtelCollectorConfig`; `UpdateOtelCollector` takes `supervisordConfigsM`, writes with
  `writeOtelCollectorConfig`, and calls `restartOtelCollectorIfPending(settings.IsOtelCollectorEnabled())`. In server,
  call `s.otelValidator.ValidateLogParserOperators` in `AddLogParserPreset` and `ChangeLogParserPreset` before saving.
  Wire `logrules.New(db, supervisord, nil)` and `go logRules.Run(ctx)` in `main.go`.
- [ ] **Step 4: Run** the same command and `go build ./managed/cmd/pmm-managed`. Expected: PASS, build succeeds.
- [ ] **Step 5: Commit** — `PMM-7 Validate and apply log rule changes`.

### Tasks 6–10

_Pending: Grafana alert sync, rules API, Rules tab, Assistant tool, pattern tuning and live verification are written
after the credential decision._
