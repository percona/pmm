# OTel 01: Server Receiver and Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** PMM Server receives OTLP/HTTP logs on `/otlp/` from Admin credentials only, masks secrets, and stores them in ClickHouse `otel.logs`. Retention, a size cap and a disk watermark mean the data can never starve QAN. All of this can be turned on and off at runtime.

**Architecture:**
- An OCB-built `otelcol` binary ships in the pmm-client tarball, which the server image installs.
- pmm-managed renders `/etc/otel-collector/config.yaml` from typed structs and runs it as the supervisord program `otel-collector` on `127.0.0.1:4318`.
- nginx proxies `/otlp/` to it behind `auth_request`.
- qan-api2 migrates a separate `otel` database.
- A leader-only cleaner in pmm-managed drops partitions to enforce retention, the size cap and the watermark.

**Tech Stack:** Go (pmm-managed, qan-api2), OpenTelemetry Collector Builder, ClickHouse 25.3, supervisord, nginx, protobuf/grpc-gateway, Helm.

**Spec:**
- [analysis](../specs/2026-10-10-opentelemetry-analysis.md): §3, §5 F8–F20, §6.5, §6.8, decisions D1, D4, D5, D6, D12, D14;
- PMM-15571, PMM-15577 (build and server image), PMM-15594 (server side);
- the [roadmap](2026-10-10-opentelemetry-00-roadmap.md) for the shared names.

## Global Constraints

The roadmap's **Shared contract** and **Global constraints** apply to every task. In addition, from PMM-15571:
- The receiver listens on `127.0.0.1` only, over HTTP. Nothing listens on 4317.
- `/otlp/` returns 401 without credentials and 403 for a Viewer. It accepts OTLP from an Admin or from a pmm-agent.
- Redaction covers bearer tokens, `glsa_` tokens, `Authorization` headers and `password=` values, in the body and in attributes.
- The collector config holds a ClickHouse password, so it is written 0600.
- `collector_enabled` is `optional bool` in the API, so a request that changes only retention keeps it as it was.
- Settings changes run no DDL under the supervisord lock, and every ClickHouse call has a timeout.
- Disable keeps the data. Purge deletes it.
- The cleanup never touches QAN tables.

## Review Focus

These inputs and conditions are implied by the spec but no task naturally tests them. Each line names the task that adds the test.

1. **Upgrade from a release whose stored settings have no `otel` key.** Expected: OTEL is on, with the defaults, and needs no manual step. Test in Task 4 (`TestOtelSettingsDefaultsOnUpgrade`).
2. **A ClickHouse password or address containing `$`, `'`, `"` or `#`.** Expected: the collector receives it verbatim, with no confmap expansion and no YAML breakage. Test in Task 5 (`TestServerCollectorConfigEscapesSecrets`).
3. **External ClickHouse (`PMM_DISABLE_BUILTIN_CLICKHOUSE`) without the `otel_*` users.** Expected: `GET /v1/otel/status` says `otel_writer` can't authenticate, instead of the collector failing silently. Test in Task 9 (`TestGetStatusReportsWriterAuthFailure`).
4. **Clients pushing while the collector is disabled or paused.** Expected: `/otlp/` answers 502, the clients keep the data queued, and nothing reaches ClickHouse. Test in Task 7 (api-test `TestOTLPWhenCollectorDisabled`).
5. **A cap smaller than one day's data, and a retention lowered below existing partitions.** Expected: the next pass drops the old partitions, never today's, and ingest pauses when today alone exceeds the cap. Test in Task 10 (`TestPlanCleanup*`).

---

### Task 1: OCB-built collector in the pmm-client tarball

**Files:**
- Create:
  - `build/otelcol/manifest.yaml`
  - `build/otelcol/expected-components.txt`
  - `build/otelcol/check-components.sh`
  - `.github/workflows/otelcol.yml`
- Modify:
  - `build/scripts/vars` (next to `nomad_commit_hash`, ~:59-63)
  - `build/scripts/build-client-binary` (~:71, :135-136, :169-170)
  - `build/scripts/install_tarball` (~:83-90, the `pt-*|nomad` case)
  - `build/packages/rpm/client/pmm-client.spec` (~:94-111 and `%files`)
  - `build/packages/deb/install`
  - `build/packages/deb/files`

**Interfaces:**
- Produces `/usr/local/percona/pmm/tools/otelcol` on every client package, and on the server image through the tarball. Variables `otelcol_version` and `otelcol_builder_version` go in `build/scripts/vars`.

- [ ] **Step 1: Write the component check.** `check-components.sh <binary>` runs `<binary> components`, extracts the component names, sorts them, and `diff`s them against `expected-components.txt`. The expected list:
  - `receivers: filelog, journald, otlp`
  - `processors: batch, memory_limiter, probabilistic_sampler, resource, tail_sampling, transform`
  - `exporters: clickhouse, otlphttp`
  - `extensions: file_storage`

  It exits non-zero on any difference.
- [ ] **Step 2: Run it against the prototype's `otelcol-contrib` 0.148.0.** Expected: FAIL. The diff lists hundreds of extra components.
- [ ] **Step 3: Write `manifest.yaml`.**
  - `dist.name: otelcol`, `dist.output_path: ./_build`.
  - One `gomod:` line per component above, all at the version in `otelcol_version`. Use the latest collector release at implementation time, with core modules at the matching `v1.x`/`v0.x` pair from its release notes.
  - In `build-client-binary`, build it inside the existing rpmbuild container with `CGO_ENABLED=0`:
    1. `go install go.opentelemetry.io/collector/cmd/builder@v${otelcol_builder_version}`
    2. `builder --config build/otelcol/manifest.yaml`
  - Copy `_build/otelcol` into the tarball's `tools/`. Add `otelcol` to the `pt-*|nomad` case in `install_tarball`, and to the RPM and DEB file lists.
- [ ] **Step 4: Add CI.** `.github/workflows/otelcol.yml`, on changes to `build/otelcol/**`, builds the binary for amd64 and runs `check-components.sh`. Expected: PASS. Record the binary size in the PR description (PMM-15577 AC).
- [ ] **Step 5: Commit.**
  ```bash
  git add build/otelcol build/scripts build/packages .github/workflows/otelcol.yml
  git commit -s -m "PMM-15577 Build otelcol with OCB from pinned source"
  ```

### Task 2: ClickHouse users `otel_writer` and `otel_reader`

**Files:**
- Create:
  - `build/ansible/roles/clickhouse/files/users.d/otel.xml`
  - `qan-api2/otel_users_test.go`
- Modify:
  - `build/ansible/roles/clickhouse/tasks/main.yml` (copy the drop-in to `/etc/clickhouse-server/users.d/otel.xml`, owner `pmm`, mode 0640)
  - `qan-api2/Makefile` (`start-clickhouse`: mount `../build/ansible/roles/clickhouse/files/users.d` read-only at `/etc/clickhouse-server/users.d`)

**Interfaces:**
- Produces the users `otel_writer` and `otel_reader`, both with `password_sha256_hex` of their own name and networks `::1`, `127.0.0.1`.
  - `otel_writer`: profile `otel_writer`, `GRANT INSERT ON otel.*`.
  - `otel_reader`: profile `otel_reader` (`readonly=1`, `max_execution_time` changeable in readonly, the same as the `datasource` profile at `default-users.xml:27-34`), `GRANT SELECT ON otel.*`.
  - Both profiles: `max_memory_usage` 1073741824 and `max_execution_time` 60. These values are provisional until PMM-15592.

- [ ] **Step 1: Write the failing test** (`qan-api2/otel_users_test.go`, integration, uses the test container on `127.0.0.1:19000`):
  ```go
  func TestOtelUsersGrants(t *testing.T) {
      admin := openTestDB(t, "clickhouse://default:clickhouse@127.0.0.1:19000/default")
      var grants []string
      require.NoError(t, admin.SelectContext(t.Context(), &grants, "SHOW GRANTS FOR otel_writer"))
      assert.Equal(t, []string{"GRANT INSERT ON otel.* TO otel_writer"}, grants)
      grants = nil
      require.NoError(t, admin.SelectContext(t.Context(), &grants, "SHOW GRANTS FOR otel_reader"))
      assert.Equal(t, []string{"GRANT SELECT ON otel.* TO otel_reader"}, grants)

      writer := openTestDB(t, "clickhouse://otel_writer:otel_writer@127.0.0.1:19000/default")
      _, err := writer.ExecContext(t.Context(), "SELECT count() FROM pmm_test.metrics")
      require.ErrorContains(t, err, "ACCESS_DENIED")
  }
  ```
  `openTestDB` is a test helper in the same file that opens `sqlx` and registers `t.Cleanup(db.Close)`.
- [ ] **Step 2: Run it.** `cd qan-api2 && make test-env-down test-env-up && go test -run TestOtelUsersGrants ./...`. Expected: FAIL, "There is no user `otel_writer`".
- [ ] **Step 3: Write `users.d/otel.xml`** with the two profiles and two users above, plus the Makefile mount and the Ansible copy task.
- [ ] **Step 4: Run it again.** Same command. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add build/ansible/roles/clickhouse qan-api2/Makefile qan-api2/otel_users_test.go
  git commit -s -m "PMM-15571 Add otel_writer and otel_reader users"
  ```

### Task 3: `otel` schema owned by qan-api2

**Files:**
- Create:
  - `qan-api2/migrations/otel/sql/01_logs.up.sql`
  - `qan-api2/migrations/otel/sql/01_logs.down.sql`
  - `qan-api2/migrations/otel_test.go`
- Modify:
  - `qan-api2/migrations/migrations.go` (a second `//go:embed otel/sql/*.sql` and `RunOtel`)
  - `qan-api2/db.go` (`NewDB` runs the otel chain after the QAN chain)
  - `qan-api2/cmd/render-migrations/main.go` (a `-chain=qan|otel` flag)

**Interfaces:**
- Consumes `createDB(dsn, clusterName string) error` (`qan-api2/db.go:114-148`) and `GetEngine(isCluster bool) string` (`migrations.go:78-84`).
- Produces:
  - `func RunOtel(dsn string, templateData map[string]any, isCluster bool, clusterName string) error`. It uses `x-migrations-table=otel_schema_migrations`, with the cluster engine for that table copied from `addSchemaMigrationsParams`.
  - Table `otel.logs`:
    - the upstream exporter's columns: `Timestamp DateTime64(9)`, `TimestampTime DateTime DEFAULT toDateTime(Timestamp)`, `TraceId`, `SpanId`, `TraceFlags`, `SeverityText`, `SeverityNumber`, `ServiceName`, `Body`, `ResourceSchemaUrl`, `ResourceAttributes Map(LowCardinality(String), String)`, `ScopeSchemaUrl`, `ScopeName`, `ScopeVersion`, `ScopeAttributes`, `LogAttributes`, with the upstream codecs and indexes (copy them from the pinned exporter's `internal/sqltemplates`);
    - plus `PmmNodeId LowCardinality(String) MATERIALIZED ResourceAttributes['pmm.node_id']` and `PmmServiceId LowCardinality(String) MATERIALIZED ResourceAttributes['pmm.service_id']`;
    - `ENGINE = {{ .engine }} PARTITION BY toDate(TimestampTime) ORDER BY (PmmNodeId, PmmServiceId, ServiceName, TimestampTime) SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`;
    - no TTL (D1).

- [ ] **Step 1: Write the failing test.**
  ```go
  func TestRunOtelCreatesLogs(t *testing.T) {
      dsn := "clickhouse://default:clickhouse@127.0.0.1:19000/otel_test"
      require.NoError(t, RunOtel(dsn, map[string]any{"engine": GetEngine(false)}, false, ""))
      require.NoError(t, RunOtel(dsn, map[string]any{"engine": GetEngine(false)}, false, ""))

      db := openTestDB(t, dsn)
      var ddl string
      require.NoError(t, db.GetContext(t.Context(), &ddl, "SHOW CREATE TABLE otel_test.logs"))
      assert.Contains(t, ddl, "PARTITION BY toDate(TimestampTime)")
      assert.Contains(t, ddl, "ORDER BY (PmmNodeId, PmmServiceId, ServiceName, TimestampTime)")
      assert.NotContains(t, ddl, "TTL ")

      _, err := db.ExecContext(t.Context(), `INSERT INTO otel_test.logs (Timestamp, Body, ResourceAttributes)
          VALUES (now64(9), 'x', map('pmm.node_id','n1','pmm.service_id','s1'))`)
      require.NoError(t, err)
      var node, svc string
      require.NoError(t, db.QueryRowContext(t.Context(), "SELECT PmmNodeId, PmmServiceId FROM otel_test.logs").Scan(&node, &svc))
      assert.Equal(t, []string{"n1", "s1"}, []string{node, svc})
  }
  ```
  Add `TestRunOtelOnExistingQANDatabase`: run `Run` (QAN) first, then `RunOtel`, and assert that both `schema_migrations` and `otel_schema_migrations` exist, each in its own database.
- [ ] **Step 2: Run it.** `cd qan-api2 && go test -run TestRunOtel ./migrations/`. Expected: FAIL, `undefined: RunOtel`.
- [ ] **Step 3: Implement `RunOtel`.** Call `createDB` first, because the otel DSN's database may not exist yet. Then wire it into `NewDB`: derive the otel DSN by replacing the database path of the QAN DSN with the constant `otel`, and run the chain after the QAN chain. Add `-chain=otel` to `render-migrations` so CI can pipe the SQL into a container.
- [ ] **Step 4: Run it again.** Same command, then `go test ./...` in `qan-api2`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add qan-api2
  git commit -s -m "PMM-15571 Create the otel schema in qan-api2"
  ```

### Task 4: OTEL settings, `PMM_ENABLE_OTEL`, env lock

**Files:**
- Modify:
  - `api/server/v1/server.proto` (`OtelSettings` as `Settings.otel = 22`; `ChangeOtelSettings` as `ChangeSettingsRequest.otel = 16`)
  - `managed/models/settings.go` (defaults consts ~:26-37, struct ~:63-126, `IsOtelCollectorEnabled`, `fillDefaults` ~:219-251)
  - `managed/models/settings_helpers.go` (`ChangeSettingsParams` ~:54-108, `UpdateSettings`, `ValidateSettings`)
  - `managed/utils/envvars/parser.go` (a `case "PMM_ENABLE_OTEL":` like `PMM_ENABLE_ALERTING` at :201-207)
  - `managed/utils/env/env.go`
  - `managed/services/server/server.go` (`convertSettings` ~:420-454, `validateChangeSettingsRequest` ~:514-566, `ChangeSettings`)
- Test:
  - `managed/models/settings_test.go`
  - `managed/utils/envvars/parser_test.go`
  - `managed/services/server/server_test.go`
  - `api-tests/server/settings/otel_settings_test.go`

**Interfaces:**
- Produces:
  - `type OtelSettings struct { CollectorEnabled *bool \`json:"collector_enabled"\`; LogsRetentionDays uint32 \`json:"logs_retention_days"\`; TracesRetentionDays uint32 \`json:"traces_retention_days"\`; MaxDiskGB uint32 \`json:"max_disk_gb"\`; DiskWatermarkPercent uint32 \`json:"disk_watermark_percent"\`; IngestPaused bool \`json:"ingest_paused"\`; IngestPausedReason string \`json:"ingest_paused_reason"\` }`.
  - `func (s *Settings) IsOtelCollectorEnabled() bool`.
  - `ChangeSettingsParams.Otel *ChangeOtelParams`, with pointer fields that mirror the proto `optional`s.
  - `IngestPaused` and `IngestPausedReason` are internal: Task 10 writes them, they never appear in the API settings, and `GetStatus` exposes them.
- Validation:
  - retention 1–365 days for each signal;
  - `max_disk_gb` ≥ 1;
  - `disk_watermark_percent` 50–95;
  - an out-of-range value returns `codes.InvalidArgument`, naming the field.

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestOtelSettingsDefaultsOnUpgrade(t *testing.T) {
      s := &models.Settings{}
      require.NoError(t, json.Unmarshal([]byte(`{"data_retention":2592000000000000}`), s))
      s.FillDefaults()
      assert.True(t, s.IsOtelCollectorEnabled())
      assert.Equal(t, models.OtelSettings{CollectorEnabled: s.Otel.CollectorEnabled, LogsRetentionDays: 7,
          TracesRetentionDays: 7, MaxDiskGB: 10, DiskWatermarkPercent: 80}, s.Otel)
  }

  ```
  In `managed/models/settings_helpers_test.go`, inside the existing `TestSettings` (it uses `testdb.Open(t, models.SkipFixtures, nil)` at :31), add:
  ```go
  t.Run("otel retention change keeps collector_enabled", func(t *testing.T) {
      _, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{Otel: &models.ChangeOtelParams{CollectorEnabled: new(false)}})
      require.NoError(t, err)
      s, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{Otel: &models.ChangeOtelParams{LogsRetentionDays: new(uint32(3))}})
      require.NoError(t, err)
      assert.False(t, s.IsOtelCollectorEnabled())
      assert.Equal(t, uint32(3), s.Otel.LogsRetentionDays)
  })
  t.Run("otel retention out of range", func(t *testing.T) {
      _, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{Otel: &models.ChangeOtelParams{LogsRetentionDays: new(uint32(0))}})
      tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Invalid argument: otel.logs_retention_days must be between 1 and 365."), err)
  })
  ```
  `new(value)` is the `new(expr)` form from Go 1.26. The repo is on Go 1.27, and `registry.go:526` already uses it (`new(models.NomadAgentType)`). Use the error helper that the surrounding subtests use.

  Also add:
  - `TestParseEnvVars` cases: `PMM_ENABLE_OTEL=false` gives `envSettings.Otel.CollectorEnabled=false`; an invalid value gives an error naming the variable.
  - In `server_test.go`, `TestValidateChangeSettingsRequest`: with env `PMM_ENABLE_OTEL=false`, a request with `otel.collector_enabled=true` gets `FailedPrecondition`.
- [ ] **Step 2: Run them.** `cd managed && go test ./models/ ./utils/envvars/ ./services/server/ -run 'Otel|ParseEnvVars|ValidateChangeSettings'`. Expected: FAIL to compile, `undefined: models.OtelSettings`.
- [ ] **Step 3: Implement.** Add the proto messages and run `make gen`. Add the model, the defaults, validation, the env case and the env lock. Then add the response conversion: `IngestPaused` is not exported in `Settings`.
- [ ] **Step 4: Run them again.** Same command. Expected: PASS. Then write the api-test (`api-tests/server/settings/otel_settings_test.go`):
  - `GET /v1/server/settings` has `otel.collector_enabled=true` and `logs_retention_days=7`;
  - a `PUT` with only `{"otel":{"logs_retention_days":3}}` keeps `collector_enabled`;
  - `logs_retention_days: 0` gets 400.

  Restore the defaults in `t.Cleanup`.
- [ ] **Step 5: Commit.**
  ```bash
  git add api/server managed/models managed/utils managed/services/server api-tests/server/settings
  git commit -s -m "PMM-15571 Add OTEL settings and PMM_ENABLE_OTEL"
  ```

### Task 5: Server collector config generator

**Files:**
- Create:
  - `managed/services/otel/server_config.go`
  - `managed/services/otel/server_config_test.go`
  - `managed/services/otel/testdata/server-config.golden.yaml`

**Interfaces:**
- Produces:
  - `type ServerConfigParams struct { ClickHouseAddr, Database, Username, Password string; ReceiverEndpoint, TelemetryEndpoint string }`
  - `func ServerCollectorConfig(p ServerConfigParams) ([]byte, error)`, which marshals typed structs with `gopkg.in/yaml.v3`. Keys are sorted, so the output is deterministic.
  - `func escapeConfmap(s string) string`, which replaces `$` with `$$`. It is applied to every string that comes from env or settings.
- Config content:
  - `receivers.otlp.protocols.http.endpoint` = `ReceiverEndpoint` (`127.0.0.1:4318`). No `grpc` key.
  - Processors, in this order: `memory_limiter` (`check_interval: 1s`, `limit_mib: 256`), then `transform/redact`, then `batch` (`timeout: 5s`, `send_batch_size: 10000`, `send_batch_max_size: 10000`).
  - `transform/redact`, `log_statements`, context `log`:
    - `replace_pattern(body, "(?i)(bearer\\s+)[A-Za-z0-9._~+/=-]+", "$$1[REDACTED]")`
    - `replace_pattern(body, "glsa_[A-Za-z0-9_]+", "[REDACTED]")`
    - `replace_pattern(body, "(?i)(authorization\"?\\s*[:=]\\s*\"?)[^\\s\",]+(\\s+[^\\s\",]+)?", "$$1[REDACTED]")`
    - `replace_pattern(body, "(?i)(password\\s*=\\s*)[^\\s&;,]+", "$$1[REDACTED]")`
    - the same four as `replace_all_patterns(attributes, "value", …)` and `replace_all_patterns(resource.attributes, "value", …)`.
  - `exporters.clickhouse`: `endpoint: tcp://<addr>?dial_timeout=10s`, `database: otel`, `username`, `password`, `logs_table_name: logs`, `create_schema: false`, `timeout: 10s`, `retry_on_failure` (5s/30s/300s), `sending_queue: {enabled: true, queue_size: 1000}`.
  - `service.telemetry.metrics`: a pull reader on `TelemetryEndpoint` (`127.0.0.1:4319`), level `normal`.
  - `service.pipelines.logs: [otlp] → [memory_limiter, transform/redact, batch] → [clickhouse]`.

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestServerCollectorConfigGolden(t *testing.T) {
      b, err := ServerCollectorConfig(ServerConfigParams{ClickHouseAddr: "127.0.0.1:9000", Database: "otel",
          Username: "otel_writer", Password: "otel_writer", ReceiverEndpoint: "127.0.0.1:4318", TelemetryEndpoint: "127.0.0.1:4319"})
      require.NoError(t, err)
      golden.Assert(t, string(b), "server-config.golden.yaml")
  }

  func TestServerCollectorConfigEscapesSecrets(t *testing.T) {
      b, err := ServerCollectorConfig(ServerConfigParams{Password: `p$a'"#ss${env:HOME}`, /* other fields as above */})
      require.NoError(t, err)
      var cfg map[string]any
      require.NoError(t, yaml.Unmarshal(b, &cfg))
      pw := cfg["exporters"].(map[string]any)["clickhouse"].(map[string]any)["password"]
      assert.Equal(t, `p$$a'"#ss$${env:HOME}`, pw)
  }

  func TestServerCollectorConfigHasNoGRPC(t *testing.T) {
      // unmarshal, assert receivers.otlp.protocols has only "http"
  }
  ```
  Use the golden-file helper this repo already uses, if there is one (`grep -rn golden managed --include=*_test.go`). Otherwise compare against `os.ReadFile`, with an `-update` flag.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/otel/ -run ServerCollectorConfig`. Expected: FAIL, the package does not exist.
- [ ] **Step 3: Implement `ServerCollectorConfig`** with typed structs, one per config section.
- [ ] **Step 4: Run them again.** Same command. Expected: PASS. Task 8 proves that the redaction regexes and `$$1` behave as intended in the real collector.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/otel
  git commit -s -m "PMM-15571 Generate the server collector config"
  ```

### Task 6: `otel.Service`: supervisord program, runtime toggle, HA reconcile

**Files:**
- Create:
  - `managed/services/otel/service.go`
  - `managed/services/otel/deps.go`
  - `managed/services/otel/service_test.go`
  - `managed/testdata/supervisord.d/otel-collector.ini`
- Modify:
  - `managed/services/supervisord/supervisord.go`:
    - a `{{define "otel-collector"}}` block in `templates`, with command `/usr/local/percona/pmm/tools/otelcol --config=/etc/otel-collector/config.yaml`, `autostart = {{ .OtelCollectorEnabled }}`, `stdout_logfile = /srv/logs/otel-collector.log`, `stopsignal = INT`, `stopwaitsecs = 30`;
    - in `UpdateConfiguration` (~:96-147), when the collector is disabled or paused, remove `otel-collector.ini` **and** run `supervisorctl update otel-collector` (F17);
    - the `OtelCollectorEnabled` param in `marshalConfig` (~:417-486).
  - `managed/services/server/deps.go` and `server.go` `UpdateConfigurations` (~:791-819): call `otel.UpdateConfiguration` after nomad and before supervisord.
  - `managed/services/server/logs.go` (~:185-200): add `otel-collector.ini` and the collector config, with the password replaced by `[REDACTED]`.
  - `managed/cmd/pmm-managed/main.go`: construct `otel.Service`, pass it to `server.Params`, and run `Run` for every replica (not as a leader service).
  - `build/docker/server/entrypoint.sh` (~:80-84): create `/etc/otel-collector`, owner `pmm`, mode 0700.

**Interfaces:**
- Consumes:
  - `ServerCollectorConfig` (Task 5);
  - `models.GetSettings`, and the `OtelSettings` fields `CollectorEnabled` and `IngestPaused` (Task 4);
  - the `supervisord` interface `RestartSupervisedService(ctx context.Context, name string) error`.
- Produces:
  - `func New(params Params) *Service`, with `Params{DB *reform.DB; Supervisord supervisordService; ConfigPath string; ClickHouse ServerConfigParams}`;
  - `func (s *Service) UpdateConfiguration(ctx context.Context, settings *models.Settings) error`. It writes the config atomically with mode 0600, only when the content changed, and restarts the program if it changed while enabled;
  - `func (s *Service) Run(ctx context.Context)`, a ticker that calls `UpdateConfiguration` when the OTEL settings differ from the last applied ones.
  - Two unexported fields exist so tests can drive the loop: `reconcileInterval time.Duration`, set to 30 s by `New`, and `onApplied func(enabled bool)`. The hook is nil in production and is called after each apply with the effective running state.

  The collector runs only when `IsOtelCollectorEnabled() && !Otel.IngestPaused`.
- Credentials: `PMM_CLICKHOUSE_OTEL_WRITER_USER/PASSWORD` with defaults `otel_writer`/`otel_writer`, read once at startup through `envvars.GetEnv` and added to the parser allow-list (`parser.go:118-125`).

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestUpdateConfigurationWritesOnceWith0600(t *testing.T) {
      dir := t.TempDir()
      sv := &mockSupervisordService{}
      s := New(Params{Supervisord: sv, ConfigPath: filepath.Join(dir, "config.yaml"), ClickHouse: testCH})
      settings := &models.Settings{}
      settings.FillDefaults()
      require.NoError(t, s.UpdateConfiguration(t.Context(), settings))
      fi, err := os.Stat(filepath.Join(dir, "config.yaml"))
      require.NoError(t, err)
      assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
      require.NoError(t, s.UpdateConfiguration(t.Context(), settings))
      sv.AssertNumberOfCalls(t, "RestartSupervisedService", 0)
  }

  func TestRunAppliesSettingsChangedByAnotherReplica(t *testing.T) {
      sqlDB := testdb.Open(t, models.SkipFixtures, nil)
      db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
      sv := &mockSupervisordService{}
      applied := make(chan bool, 4)
      s := New(Params{DB: db, Supervisord: sv, ConfigPath: filepath.Join(t.TempDir(), "config.yaml"), ClickHouse: testCH})
      s.reconcileInterval = 10 * time.Millisecond
      s.onApplied = func(enabled bool) { applied <- enabled }

      ctx, cancel := context.WithCancel(t.Context())
      t.Cleanup(cancel)
      go s.Run(ctx)
      assert.True(t, <-applied)

      _, err := models.UpdateSettings(db, &models.ChangeSettingsParams{Otel: &models.ChangeOtelParams{CollectorEnabled: new(false)}})
      require.NoError(t, err)
      assert.False(t, <-applied)
  }
  ```
  Add `managed/testdata/supervisord.d/otel-collector.ini` and a case in `supervisord_test.go` `TestConfig`. Add a test that a disabled collector removes the `.ini` and calls `update otel-collector`: either inject the command runner, or assert through the existing reload seam in the test.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/otel/ ./services/supervisord/`. Expected: FAIL.
- [ ] **Step 3: Implement** the service, the template, the F17 fix, the wiring, `logs.go` and the entrypoint directory. Generate the `supervisordService` mock with mockery (`.mockery.yaml`).
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed build/docker/server/entrypoint.sh
  git commit -s -m "PMM-15571 Run otel-collector under supervisord"
  ```

### Task 7: `/otlp/` behind `auth_request` with the Admin role

**Files:**
- Modify:
  - `build/ansible/roles/nginx/files/conf.d/pmm.conf`:
    - an upstream `otel-collector { server 127.0.0.1:4318; keepalive 32; }` next to the others (:1-48);
    - `location /otlp/ { proxy_pass http://otel-collector/; proxy_http_version 1.1; proxy_set_header Connection ""; client_body_buffer_size 10m; }`, modelled on `/victoriametrics/api/v1/write` (:244-250). It inherits the server-level `auth_request`.
  - `managed/services/grafana/auth_server.go`: `"/otlp/": admin` in `rules` (:54-133).
- Test:
  - `managed/services/grafana/auth_server_test.go` (`TestResolveRule` ~:108)
  - `api-tests/server/otlp_test.go`

- [ ] **Step 1: Write the failing tests.** Add `TestResolveRule` cases `/otlp/v1/logs → admin` and `/otlp → admin`. In `api-tests/server/otlp_test.go`:
  - `TestOTLPAuth`: a POST of `{"resourceLogs":[]}` to `/otlp/v1/logs` gives 401 without credentials and 403 with a Viewer created in the test (copy the user helper from `api-tests/server/auth_test.go`). Admin gets 200.
  - `TestOTLPWhenCollectorDisabled`: disable the collector through settings, wait for `supervisorctl status` to show it stopped, POST as Admin, and expect 502. Re-enable it in `t.Cleanup`.
- [ ] **Step 2: Run the unit test.** `cd managed && go test ./services/grafana/ -run TestResolveRule`. Expected: FAIL, `/otlp/v1/logs` resolves to `grafanaAdmin`.
- [ ] **Step 3: Implement** the rule and the nginx location.
- [ ] **Step 4: Run them again.** The unit test is expected to PASS. Then run `make env-up-rebuild` (or a Feature Build) and `make api-test` (`go test ./api-tests/server -run OTLP`). Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add build/ansible/roles/nginx managed/services/grafana api-tests/server/otlp_test.go
  git commit -s -m "PMM-15571 Proxy /otlp/ to the collector for Admins"
  ```

### Task 8: Contract test against the pinned collector and the migrated schema

**Files:**
- Create: `managed/services/otel/contract_test.go`
- Modify: `.github/workflows/otelcol.yml` (Task 1). It starts the qan-api2 test ClickHouse with the `users.d` mount, runs `go run ./qan-api2/cmd/render-migrations -chain=otel | clickhouse client -d otel`, and then runs the test with the env vars set.

**Interfaces:**
- Consumes `ServerCollectorConfig` (Task 5), the `otelcol` binary (Task 1) and `otel.logs` (Task 3).
- The test is skipped unless `PMM_OTEL_CONTRACT_BINARY` and `PMM_OTEL_CONTRACT_CLICKHOUSE` are set.

- [ ] **Step 1: Write the test.**
  ```go
  func TestCollectorContract(t *testing.T) {
      bin, chAddr := os.Getenv("PMM_OTEL_CONTRACT_BINARY"), os.Getenv("PMM_OTEL_CONTRACT_CLICKHOUSE")
      if bin == "" || chAddr == "" { t.Skip("contract test needs PMM_OTEL_CONTRACT_BINARY and PMM_OTEL_CONTRACT_CLICKHOUSE") }
      // render config with Username otel_writer, start bin with exec.CommandContext(t.Context(), bin, "--config", path)
      // wait for 127.0.0.1:4318 to accept; POST OTLP JSON body:
      //   "contract-<uuid> Authorization: Bearer secret-token-123 password=hunter2 glsa_abc_123 ^anchor$"
      //   with resource attributes pmm.node_id=n1, pmm.service_id=s1
      // poll ClickHouse (as default) up to 30s: SELECT Body, PmmNodeId FROM otel.logs WHERE Body LIKE 'contract-<uuid>%'
      assert.NotContains(t, body, "secret-token-123")
      assert.NotContains(t, body, "hunter2")
      assert.NotContains(t, body, "glsa_abc_123")
      assert.Contains(t, body, "^anchor$")
      assert.Equal(t, "n1", node)
      _, err := net.DialTimeout("tcp", "127.0.0.1:4317", time.Second)
      require.Error(t, err)
  }
  ```
- [ ] **Step 2: Run it locally against a collector without the redaction processor**: temporarily drop `transform/redact` from the pipeline in a scratch copy of the config. Expected: FAIL on `NotContains`. This proves the test can catch a missing redaction.
- [ ] **Step 3: Restore the generator's config.** No production code changes, unless the test exposes a regex or `$$` problem; fix that in `server_config.go`.
- [ ] **Step 4: Run it.** `PMM_OTEL_CONTRACT_BINARY=… PMM_OTEL_CONTRACT_CLICKHOUSE=127.0.0.1:19000 go test ./managed/services/otel/ -run TestCollectorContract`, then check that the CI workflow is green. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/otel/contract_test.go .github/workflows/otelcol.yml
  git commit -s -m "PMM-15571 Add collector-schema contract test"
  ```

### Task 9: `OtelService`: status and purge

**Files:**
- Create:
  - `api/otel/v1/otel.proto`
  - `managed/services/otel/grpc.go`
  - `managed/services/otel/grpc_test.go`
  - `api-tests/otel/status_test.go`
- Modify:
  - `api/Makefile` (register the domain the way realtimeanalytics is registered at :22,38,57,75,118)
  - `managed/cmd/pmm-managed/main.go` (gRPC and gateway registration, ~:352-355,454)
  - `managed/services/grafana/auth_server.go` (`"/v1/otel": admin`)

**Interfaces:**
- Produces:
  - `rpc GetStatus(GetStatusRequest) returns (GetStatusResponse)` at `GET /v1/otel/status`. The response: `bool collector_enabled = 1; bool collector_running = 2; bool ingest_paused = 3; string ingest_paused_reason = 4; uint64 disk_bytes = 5; float disk_used_percent = 6; string last_error = 7;`.
  - `rpc Purge(PurgeRequest) returns (PurgeResponse)` at `POST /v1/otel:purge`. It truncates every `MergeTree`-family table in `system.tables WHERE database = 'otel'`.
  - `type ClickHouse interface { QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row; ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error); QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) }` in `deps.go`. The existing ClickHouse client (`main.go:1098`) satisfies it.
  - `last_error` says "otel_writer cannot authenticate to ClickHouse" when a probe as `otel_writer` (`SELECT 1`, 5 s timeout) fails.
  - Every ClickHouse call uses a context with a 10 s timeout.

- [ ] **Step 1: Write the failing tests** (`grpc_test.go`, with `go-sqlmock` standing in for ClickHouse):
  - `TestGetStatusSumsOtelParts`: mock `SELECT sum(bytes_on_disk) FROM system.parts WHERE database = 'otel' AND active` → 1024, and the `system.disks` free and total query. Assert `DiskBytes == 1024` and a correct `DiskUsedPercent`.
  - `TestGetStatusReportsWriterAuthFailure`: the injected writer probe returns an auth error, and `LastError` contains `otel_writer`.
  - `TestPurgeTruncatesOnlyOtelTables`: mock the `system.tables` query → `logs`. Expect exactly `TRUNCATE TABLE otel.logs`, and no statement touching `pmm.`.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/otel/ -run 'GetStatus|Purge'`. Expected: FAIL.
- [ ] **Step 3: Implement.** Write the proto, run `make gen`, then write the handlers, the registration and the auth rule.
- [ ] **Step 4: Run them again.** Same command: PASS. Then the api-test: `GET /v1/otel/status` as Admin gets 200 with `collector_enabled`; as a Viewer it gets 403. Purge as Admin leaves `count()`, read back through status `disk_bytes` after the parts merge, at 0. Run it against a Feature Build.
- [ ] **Step 5: Commit.**
  ```bash
  git add api/otel api/Makefile managed api-tests/otel
  git commit -s -m "PMM-15571 Add OTEL status and purge API"
  ```

### Task 10: Leader-only cleaner: retention, size cap, watermark, pause

**Files:**
- Create:
  - `managed/services/otel/cleaner.go`
  - `managed/services/otel/cleanup_plan.go`
  - `managed/services/otel/cleanup_plan_test.go`
  - `managed/services/otel/cleaner_test.go`
- Modify: `managed/cmd/pmm-managed/main.go`: `haService.AddLeaderService(ha.NewContextService("otel-cleaner", cleaner.Run))`, next to `"cleaner"` (~:1324).

**Interfaces:**
- Produces:
  - `type Partition struct { Table string; ID string; Day time.Time; Bytes uint64 }`
  - `type CleanupInput struct { Partitions []Partition; DiskTotal, DiskFree uint64; LogsRetentionDays, TracesRetentionDays, MaxDiskGB, WatermarkPercent uint32; Paused bool; Now time.Time }`
  - `type CleanupPlan struct { Drop []Partition; Pause bool; Resume bool; Reason string }`
  - `func PlanCleanup(in CleanupInput) CleanupPlan`, a pure function.
  - `func NewCleaner(db *reform.DB, ch ClickHouse, interval time.Duration) *Cleaner` and `func (c *Cleaner) Run(ctx context.Context) error`, with interval 10 minutes and a first pass at start.
  - The cleaner reads partitions with `SELECT table, partition_id, partition, sum(bytes_on_disk) FROM system.parts WHERE database = 'otel' AND active GROUP BY 1,2,3`, and the disk with `SELECT free_space, total_space FROM system.disks WHERE name = 'default'`. It runs `ALTER TABLE otel.<table> DROP PARTITION ID '<id>'` for each planned drop.
  - It writes `IngestPaused` and `IngestPausedReason` into the settings row. Task 6's reconcile loop then stops or starts the collector on every replica.
  - It exports `pmm_managed_otel_ingest_paused` (gauge) and `pmm_managed_otel_dropped_partitions_total{reason}` (counter).
- The algorithm, which the tests determine:
  1. Drop each partition whose `Day` is older than `Now` minus the retention for its table. `logs` uses the logs retention; tables starting with `otel_traces` use the traces retention.
  2. While the remaining otel bytes exceed `MaxDiskGB·2³⁰`, or `(DiskTotal−DiskFree−dropped)/DiskTotal` exceeds `WatermarkPercent`, drop the oldest remaining partition across all tables. Ties go to the larger partition first. **Never drop a partition whose `Day` is today.**
  3. If either limit is still exceeded: `Pause = true`, and `Reason` names the limit.
  4. If `Paused` was set and both values are under 90 % of their limits: `Resume = true`.

- [ ] **Step 1: Write the failing table test.**
  ```go
  func TestPlanCleanup(t *testing.T) {
      day := func(d int) time.Time { return time.Date(2026, 10, 10-d, 0, 0, 0, 0, time.UTC) }
      now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
      gb := uint64(1 << 30)
      for name, tc := range map[string]struct{ in CleanupInput; drop []string; pause, resume bool }{
          "retention lowered drops old partitions": {
              in: CleanupInput{Partitions: []Partition{{Table: "logs", ID: "a", Day: day(5), Bytes: 1}, {Table: "logs", ID: "b", Day: day(1), Bytes: 1}},
                  DiskTotal: 100 * gb, DiskFree: 90 * gb, LogsRetentionDays: 3, TracesRetentionDays: 7, MaxDiskGB: 10, WatermarkPercent: 80, Now: now},
              drop: []string{"a"},
          },
          "cap drops oldest first, never today": {
              in: CleanupInput{Partitions: []Partition{{Table: "logs", ID: "old", Day: day(2), Bytes: 6 * gb}, {Table: "logs", ID: "today", Day: day(0), Bytes: 6 * gb}},
                  DiskTotal: 100 * gb, DiskFree: 80 * gb, LogsRetentionDays: 7, TracesRetentionDays: 7, MaxDiskGB: 10, WatermarkPercent: 80, Now: now},
              drop: []string{"old"},
          },
          "today alone over cap pauses": {
              in: CleanupInput{Partitions: []Partition{{Table: "logs", ID: "today", Day: day(0), Bytes: 12 * gb}},
                  DiskTotal: 100 * gb, DiskFree: 80 * gb, LogsRetentionDays: 7, TracesRetentionDays: 7, MaxDiskGB: 10, WatermarkPercent: 80, Now: now},
              pause: true,
          },
          "watermark drops even under cap": {
              in: CleanupInput{Partitions: []Partition{{Table: "logs", ID: "old", Day: day(3), Bytes: 2 * gb}},
                  DiskTotal: 10 * gb, DiskFree: 1 * gb, LogsRetentionDays: 7, TracesRetentionDays: 7, MaxDiskGB: 10, WatermarkPercent: 80, Now: now},
              drop: []string{"old"},
          },
          "paused and well under limits resumes": {
              in: CleanupInput{Partitions: []Partition{{Table: "logs", ID: "today", Day: day(0), Bytes: 1 * gb}},
                  DiskTotal: 100 * gb, DiskFree: 90 * gb, LogsRetentionDays: 7, TracesRetentionDays: 7, MaxDiskGB: 10, WatermarkPercent: 80, Paused: true, Now: now},
              resume: true,
          },
      } {
          t.Run(name, func(t *testing.T) {
              p := PlanCleanup(tc.in)
              var ids []string
              for _, d := range p.Drop { ids = append(ids, d.ID) }
              assert.Equal(t, tc.drop, ids)
              assert.Equal(t, tc.pause, p.Pause)
              assert.Equal(t, tc.resume, p.Resume)
          })
      }
  }
  ```
  Add `TestCleanerNeverTouchesQAN` (`cleaner_test.go`, sqlmock): after one pass, every executed statement starts with `ALTER TABLE otel.`.
- [ ] **Step 2: Run them.** `cd managed && go test ./services/otel/ -run 'PlanCleanup|Cleaner'`. Expected: FAIL.
- [ ] **Step 3: Implement** `PlanCleanup` and the `Cleaner` (queries, drops, the settings write, metrics), and register the leader service.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/otel managed/cmd/pmm-managed/main.go
  git commit -s -m "PMM-15571 Enforce OTEL retention, size cap and watermark"
  ```

### Task 11: pmm-ha chart (percona/percona-helm-charts)

**Files** (in `/srv/runner/2/percona/percona-helm-charts`, on branch `PMM-15594-otel-ha`):
- Modify:
  - `charts/pmm-ha/values.yaml` (`otel.enabled: true` and `clickhouse.otelUsers`)
  - `charts/pmm-ha/templates/clickhouse-cluster.yaml`: a `users.d/otel-users.xml` drop-in, copying the datasource-user drop-in at ~:91-129, with grants `INSERT ON otel.*` and `SELECT ON otel.*`
  - `charts/pmm-ha/templates/_helpers.tpl`: secrets generated with the lookup-or-`randAlphaNum 32` pattern at ~:269-281
  - `charts/pmm-ha/templates/statefulset.yaml`: `PMM_ENABLE_OTEL` and `PMM_CLICKHOUSE_OTEL_*` env entries from values and secrets
  - `charts/pmm-ha/README.md`: ClickHouse shards must be 1; size the ClickHouse volume (20 Gi default) against `max_disk_gb`
  - `Chart.yaml`: version bump

**Interfaces:**
- Consumes the env names from the roadmap's Shared contract.

- [ ] **Step 1: Write the check.** Run `helm template t charts/pmm-ha | grep -E 'PMM_ENABLE_OTEL|PMM_CLICKHOUSE_OTEL_WRITER_PASSWORD|GRANT INSERT ON otel'`. Expected: no output (FAIL).
- [ ] **Step 2: Implement** the values, the drop-in, the secrets, the env entries and the README.
- [ ] **Step 3: Run the check again, then `helm lint charts/pmm-ha`.** Expected: all three patterns are present and the lint is clean. Then run `helm template … --set otel.enabled=false` and check that it renders `PMM_ENABLE_OTEL: "false"`.
- [ ] **Step 4: Commit.**
  ```bash
  git add charts/pmm-ha
  git commit -s -m "PMM-15594 Add OTEL users and settings to pmm-ha"
  ```
  The HA failover AC ("deleting the leader during ingest leaves no gap") is tested by QA in PMM-15578, with plan 02's disk queue in place.
