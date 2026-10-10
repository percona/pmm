# OTel 04: Database, OS and Application Logs from pmm-admin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:**
- `pmm-admin add logs` / `remove logs` manage log sources on a node.
- `pmm-admin add mysql|postgresql|mongodb` collects the database's own log **by default** when the database runs on the same host.
- pmm-agent asks the database where that log is.
- Built-in presets parse the MySQL slow and general logs, PostgreSQL csvlog and jsonlog, and MongoDB JSON logs.

**Architecture:**
- pmm-agent discovers log paths. It is connected to the database and can `stat` the files.
- pmm-managed asks for them with a new `LogDiscoveryRequest` on the agent channel:
  - inside the add-service transaction;
  - for `--discover`;
  - again on every pmm-agent reconnect, for sources that were discovered.
- pmm-agent keeps the paths it discovered in memory, and lets exactly those paths bypass its allow-list (plan 02, Task 6's `DiscoveredPaths`).

**Tech Stack:** Go (pmm-admin with Kong, pmm-agent, pmm-managed), protobuf, MySQL, PostgreSQL and MongoDB drivers already in `go.mod`, the stanza preset harness from plan 02, Task 2.

**Spec:**
- [analysis](../specs/2026-10-10-opentelemetry-analysis.md): §6.3, F5, F6, decisions D2, D9;
- PMM-15573, PMM-15584, PMM-15585, PMM-15586, PMM-15587;
- the [roadmap](2026-10-10-opentelemetry-00-roadmap.md) for the shared names.

**Depends on:** plan 02 (the `log_sources` API, the presets harness, `Render` with `DiscoveredPaths`).

## Global Constraints

The roadmap's **Shared contract** and **Global constraints** apply. In addition, from the tickets:

**`pmm-admin add logs` flags:**
- `--path`, or `--journald` with `--units`;
- `--preset` (default `raw`);
- `--service-name` / `--service-id`;
- `--discover`;
- `--app`;
- `--custom-labels`.

**Rules:**
- `--service-name` and `--app` can't be used together.
- An unknown service, a service on another node, or an unknown preset fails with an error that names it.
- `--help` lists the presets the server has.

**Discovery:**
- A remote service, or a file that is missing on this host, still gets the service added. The command prints why no logs are collected.
- **MySQL:**
  - error log: `@@log_error`, with a relative path resolved against `@@datadir`; `stderr` means a journald source;
  - slow and general logs: only when `@@log_output` includes `FILE`.
- **PostgreSQL:**
  - `log_directory` is relative to `data_directory` unless absolute;
  - `log_filename` becomes a glob;
  - `log_destination` picks the preset;
  - `logging_collector=off` means journald, or a message.
- **MongoDB:**
  - `getCmdLineOpts` `systemLog.path`, with an `--logpath` argv fallback;
  - `destination: syslog` gives a message and no source.
- The path is read again on every pmm-agent reconnect.

**D2:** `--collect-logs` defaults to `error` for MySQL and to `on` for PostgreSQL and MongoDB. `none` (or `off`) opts out.

## Review Focus

1. **`log_error` set to `./host.err` under a `datadir` that is a symlink** (common with LVM and bind mounts). Expected: the resolved path follows the symlink, and is reported as discovered under that resolved path. Test in Task 2 (`TestResolveMySQLLogPath`).
2. **Two MySQL instances on one host whose error logs sit in one directory.** Expected: one source each, with the rows told apart by `pmm.service_id`. Test in Task 3 (`TestAddTwoLocalMySQLGetSeparateSources`, sqlmock with two services).
3. **A new pmm-admin talking to an older server** (`DiscardUnknown`, F5). Expected: pmm-admin doesn't send `collect_logs`, and prints "log collection needs PMM Server 3.11 or newer". Test in Task 4 (`TestAddMySQLSkipsCollectLogsOnOldServer`).
4. **The service is added with `--skip-connection-check`, or the database is down at add time.** Expected: the service is added, the command prints why logs were not set up, and the user can run `pmm-admin add logs --service-name=… --discover` later. Test in Task 3 (`TestAddServiceDiscoveryFailureDoesNotFailAdd`).
5. **A PostgreSQL `log_filename` with escapes other than `%Y%m%d%H%M%S`** (for example `%a` or `%j`), or none at all. Expected: every escape becomes `*`, and a name without escapes is used as is. Test in Task 2 (`TestPostgresLogGlob`).

---

### Task 1: `pmm-admin add logs`, `remove logs`, and log sources in `list`

**Files:**
- Create:
  - `admin/commands/management/add_logs.go`
  - `admin/commands/management/add_logs_test.go`
  - `admin/commands/management/remove_logs.go`
  - `admin/commands/management/remove_logs_test.go`
- Modify:
  - `admin/commands/management/add.go` (:22-31): a `Logs AddLogsCommand \`cmd\`` entry
  - `admin/commands/management/remove.go` (:41-45) and `admin/cli/cli.go` (:46-61): the D9 restructure
  - `admin/commands/list.go`: a "Log sources" table with node, kind, path or units, preset, owner and state with its reason. The collector's agent line also shows a health summary, "`<n>` collecting, `<m>` not collecting", so health isn't just "running" (PMM-15593).
  - `admin/cmd/pmm-admin/main.go`, or wherever the Kong vars are built: `${logPresets}`

**Interfaces:**
- Consumes the generated client `client.Default.OtelService` (`AddLogSource`, `RemoveLogSources`, `ListLogSources`, `ListLogParserPresets`), and `agentlocal.GetStatus` for `PMMAgentID` and `NodeID` (pattern at `add_mysql.go:183-193`).
- Produces:
  - `type AddLogsCommand struct { Path string; Journald bool; Units []string; Preset string \`default:"raw" help:"Parser preset, one of: ${logPresets}"\`; ServiceName, ServiceID string; Discover string; App string; CustomLabels map[string]string }`
  - `type RemoveLogsCommand struct { Path string; ServiceName, ServiceID string }`
  - `logPresetsHelp(args []string, fetch func(context.Context) ([]string, error)) string`. It fetches with a 2 s timeout only when `args` contain `logs` together with `--help`/`-h`. Otherwise it returns `"see 'pmm-admin add logs --help'"`. When the fetch fails, it returns the built-in names compiled into pmm-admin and the suffix "(server unreachable)".
- **D9 spike, first step of this task:** try to make `remove` a Kong command group that has subcommands `logs` and `traces` and a default `service` form, so `pmm-admin remove mysql name1` keeps working. If Kong rejects positional arguments combined with sibling subcommands, add `logs` and `traces` to `serviceTypesEnum` instead, and dispatch on them in `RemoveCommand.RunCmd`. Record which option was taken in the PR description.

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestAddLogsValidation(t *testing.T) {
      for name, cmd := range map[string]AddLogsCommand{
          "service and app":  {Path: "/var/log/x.log", ServiceName: "mysql-1", App: "billing"},
          "no path nor journald": {},
          "path and journald": {Path: "/var/log/x.log", Journald: true},
          "discover without service": {Discover: "error"},
      } {
          t.Run(name, func(t *testing.T) { _, err := cmd.RunCmd(); require.Error(t, err) })
      }
  }

  func TestLogPresetsHelpFetchesOnlyForHelp(t *testing.T) {
      called := false
      fetch := func(context.Context) ([]string, error) { called = true; return []string{"raw", "mysql_error"}, nil }
      assert.Equal(t, "see 'pmm-admin add logs --help'", logPresetsHelp([]string{"add", "mysql"}, fetch))
      assert.False(t, called)
      assert.Equal(t, "raw, mysql_error", logPresetsHelp([]string{"add", "logs", "--help"}, fetch))
  }
  ```
  Add `TestRemoveServiceStillPositional`, which parses `remove mysql name1` with the real Kong parser, and `TestRemoveLogsByPath`. Add `TestListRendersLogSources`, a golden test of `list` output with three sources, one of them `NOT_ALLOWED`.
- [ ] **Step 2: Run them.** `cd admin && go test ./commands/... ./cli/... -run 'Logs|Remove|List'`. Expected: FAIL.
- [ ] **Step 3: Implement** the spike, then the commands and the `list` table.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS. Then on a CHAOS VM with a Feature Build, run each owner and kind from the PMM-15573 AC: an OS file, a journald unit, an app file and a service-bound file. Expect one collector, and the identity from the table on each row in `otel.logs`.
- [ ] **Step 5: Commit.**
  ```bash
  git add admin
  git commit -s -m "PMM-15573 Add pmm-admin add/remove logs"
  ```

### Task 2: Log discovery on pmm-agent

**Files:**
- Create:
  - `api/agent/v1/log_discovery.proto` (package `agent.v1`)
  - `agent/logdiscovery/mysql.go`
  - `agent/logdiscovery/postgresql.go`
  - `agent/logdiscovery/mongodb.go`
  - `agent/logdiscovery/resolve.go`
  - `agent/logdiscovery/cache.go`
  - `agent/logdiscovery/*_test.go`
- Modify:
  - `api/agent/v1/agent.proto`: add `LogDiscoveryRequest` to the `ServerMessage` request oneof and `LogDiscoveryResponse` to the `AgentMessage` response oneof, next to `ServiceInfoRequest` (:465-481)
  - `agent/client/client.go` (the dispatch at ~:652-761): handle `LogDiscoveryRequest`
  - `agent/agents/mongodb/mongolog/internal/mongolog.go` (:241-267): move `getLogFilePath` to `agent/utils/mongodb/logpath.go` so both callers use it, and handle `--logpath=<x>` as well as `--logpath <x>`
  - `agent/agents/supervisor/supervisor.go`: pass `logdiscovery.Cache.Paths()` as `RenderInput.DiscoveredPaths` and `Cache.Units()` as `RenderInput.DiscoveredUnits`

**Interfaces:**
- Produces:
  - **Proto:**
    - `message LogDiscoveryRequest { inventory.v1.ServiceType type = 1; string dsn = 2 [(extensions.v1.sensitive) = REDACT_TYPE_DSN]; google.protobuf.Duration timeout = 3; TextFiles text_files = 4; bool tls_skip_verify = 5; bool tls = 6; repeated string kinds = 7; }`
    - `message LogDiscoveryResponse { repeated LogDiscoveryEntry entries = 1; string error = 2; }`
    - `message LogDiscoveryEntry { string kind = 1; OtelLogSourceKind source_kind = 2; string path = 3; repeated string units = 4; string preset = 5; bool collectable = 6; string reason = 7; }`
    - `kinds` take the values `error`, `slow` and `general` for MySQL, and `log` for PostgreSQL and MongoDB.
  - **Pure resolvers:**
    - `func ResolveMySQLLogPath(value, datadir string) (path string, journald bool)`
    - `func PostgresLogGlob(logDirectory, dataDirectory, logFilename string) string`
    - `func PostgresPreset(logDestination string) (preset string, journald bool, reason string)`: `stderr` → `postgres`, `csvlog` → `postgres_csv`, `jsonlog` → `postgres_json`, and `syslog` → no source, reason "PostgreSQL logs to syslog; add the OS log instead". When `log_destination` lists several, prefer `jsonlog`, then `csvlog`, then `stderr`.
    - `func IsLocalAddress(host string, socket string) bool`: true for a socket, a loopback address, or an address of one of the host's interfaces.
  - **Discovery:**
    - `func Discover(ctx context.Context, req *agentv1.LogDiscoveryRequest, readable func(string) error) *agentv1.LogDiscoveryResponse`
    - `type Cache struct{…}` with `Add(entries)`, `Paths() map[string]bool` and `Units() map[string]bool`. Entries are replaced per service on each discovery.
  - The `readable` check uses the collector credential from plan 02, Task 7 (`ReadableAs`), so "can't read" means "the collector can't read it".
  - **MySQL:** `SELECT @@log_error, @@datadir, @@log_output, @@slow_query_log, @@slow_query_log_file, @@general_log, @@general_log_file`. Presets are `mysql_error`, `mysql_slow` and `mysql_general`. `log_error=stderr` gives a journald source with units `["mysqld.service", "mysql.service"]`; the unit that `systemctl show -p Id` resolves is kept when available.
  - **PostgreSQL:** `SELECT name, setting FROM pg_settings WHERE name IN ('logging_collector','log_destination','log_directory','log_filename','data_directory')`. Call `pg_current_logfile()` only to confirm the glob, and treat a permission error as "not needed".
  - **MongoDB:** the shared `getLogFilePath`.

- [ ] **Step 1: Write the failing tests.**
  ```go
  func TestResolveMySQLLogPath(t *testing.T) {
      dir := t.TempDir()
      real := filepath.Join(dir, "real"); require.NoError(t, os.Mkdir(real, 0o755))
      link := filepath.Join(dir, "datadir"); require.NoError(t, os.Symlink(real, link))
      p, j := ResolveMySQLLogPath("./host.err", link+"/")
      assert.Equal(t, filepath.Join(real, "host.err"), p)
      assert.False(t, j)
      _, j = ResolveMySQLLogPath("stderr", link)
      assert.True(t, j)
      p, _ = ResolveMySQLLogPath("/var/log/mysql/error.log", link)
      assert.Equal(t, "/var/log/mysql/error.log", p)
  }

  func TestPostgresLogGlob(t *testing.T) {
      assert.Equal(t, "/var/lib/pgsql/16/data/log/postgresql-*.log", PostgresLogGlob("log", "/var/lib/pgsql/16/data", "postgresql-%a.log"))
      assert.Equal(t, "/var/log/postgresql/postgresql-*_*.log", PostgresLogGlob("/var/log/postgresql", "/x", "postgresql-%Y-%m-%d_%H%M%S.log"))
      assert.Equal(t, "/var/log/pg/fixed.log", PostgresLogGlob("/var/log/pg", "/x", "fixed.log"))
  }

  func TestPostgresPreset(t *testing.T) { /* stderr, csvlog, jsonlog, "stderr,jsonlog", syslog */ }
  func TestIsLocalAddress(t *testing.T) { /* 127.0.0.1, ::1, localhost, socket → true; 10.255.255.1 → false */ }
  func TestCacheReplacesPerService(t *testing.T) { /* … */ }
  ```
  Add `TestDiscoverMySQL` with `sqlmock`, for `log_output=TABLE` (slow and general not collectable, with the reason "log_output does not include FILE") and `log_output=FILE,TABLE`. Add `TestDiscoverMongoDBSyslog`, with the shared helper mocked to return destination `syslog`.
- [ ] **Step 2: Run them.** `make gen && cd agent && go test ./logdiscovery/ ./utils/mongodb/`. Expected: FAIL.
- [ ] **Step 3: Implement** the proto, the resolvers, the per-database discovery, the cache, the client dispatch, the move of the mongolog helper, and the supervisor wiring.
- [ ] **Step 4: Run them again.** Same command, plus `go test ./agents/mongodb/... ./client/...`. Expected: PASS. Then on a CHAOS VM with PostgreSQL 13 and 17, using PMM's documented grants (`pg_monitor`): run discovery, and record whether `pg_current_logfile()` needs anything more, for the docs PR (PMM-15585 AC).
- [ ] **Step 5: Commit.**
  ```bash
  git add api/agent agent
  git commit -s -m "PMM-15584 Discover database log paths on pmm-agent"
  ```

### Task 3: Discovery in pmm-managed: add-time, `--discover`, and reconnect

**Files:**
- Create:
  - `managed/services/agents/log_discovery.go`
  - `managed/services/agents/log_discovery_test.go`
- Modify:
  - `api/management/v1/mysql.proto`: `string collect_logs` at the next free number in `AddMySQLServiceParams`. Its values are a comma list of `error`, `slow` and `general`, or `none`. Add `repeated otel.v1.LogSource log_sources` and `repeated string log_collection_messages` at the next free numbers in `MySQLServiceResult` (:104-111).
  - `api/management/v1/postgresql.proto` and `api/management/v1/mongodb.proto`: `bool collect_logs` and the same two result fields.
  - `api/otel/v1/otel.proto`: `DiscoverLogSources(DiscoverLogSourcesRequest{service_id, kinds}) → {log_sources, messages}`
  - `managed/services/management/mysql.go` (:37-205), `postgresql.go` and `mongodb.go`: after `GetInfoFromService` (:116-128), when `collect_logs` is set, call `logDiscovery.Discover`. Upsert the collectable entries through `models.UpsertLogSource` with `Discovery = "mysql_" + kind` (MySQL), `"postgresql"` or `"mongodb"`. Put the reasons in `log_collection_messages`. **A discovery error never fails the add.**
  - `managed/services/agents/registry.go` (`authenticate`): after the agent is registered, rediscover each service that has sources with `discovery != ""` on this pmm-agent, and update a source's path when it changed. Then `RequestStateUpdate`.
  - `managed/services/otel/log_sources_grpc.go`: the `DiscoverLogSources` handler.

**Interfaces:**
- Consumes `models.UpsertLogSource` and `FindLogSources` (plan 02, Task 3), and the `LogDiscoveryRequest`/`Response` from Task 2.
- Produces:
  - `type LogDiscoverer struct{ r *Registry; db *reform.DB }`
  - `func (d *LogDiscoverer) Discover(ctx context.Context, q *reform.Querier, service *models.Service, agent *models.Agent, kinds []string) ([]*agentv1.LogDiscoveryEntry, error)`. It builds the DSN from the service's exporter agent, the way `ServiceInfoBroker` does (`managed/services/agents/service_info_broker.go:148`), with a 5 s timeout. It is version-gated: an older agent gets the message "log discovery needs pmm-agent 3.11.0 or newer".
  - `func (d *LogDiscoverer) RediscoverOnConnect(ctx context.Context, pmmAgentID string) error`

- [ ] **Step 1: Write the failing tests** (sqlmock, with a fake channel for the agent request):
  - `TestAddMySQLCollectLogsCreatesSource`: discovery returns an error-log entry; assert one `log_sources` insert with `discovery='mysql_error'`, the `mysql_error` preset and the service id.
  - `TestAddTwoLocalMySQLGetSeparateSources`: two services, two inserts, distinct `service_id`.
  - `TestAddServiceDiscoveryFailureDoesNotFailAdd`: discovery returns a timeout; the add succeeds, and `log_collection_messages` contains "timed out".
  - `TestAddRemoteServiceNoSource`: the entry is not collectable, with the reason "service is not on this host"; no insert, and the message is passed through.
  - `TestRediscoverOnConnectUpdatesMovedPath`.
- [ ] **Step 2: Run them.** `make gen && cd managed && go test ./services/agents/ ./services/management/ -run 'CollectLogs|Discovery|Rediscover|RemoteService'`. Expected: FAIL.
- [ ] **Step 3: Implement** everything listed above.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS. Write api-tests in `api-tests/management/mysql_test.go` and the PostgreSQL and MongoDB equivalents. Using the server's own `pmm-server-postgresql` service with `DiscoverLogSources`, assert a source with the path `/srv/logs/postgresql.log`, or the message explaining why not (`logging_collector=off` → stdout, analysis F21).
- [ ] **Step 5: Commit.**
  ```bash
  git add api managed api-tests/management
  git commit -s -m "PMM-15584 Create log sources when adding a database"
  ```

### Task 4: `--collect-logs` on `pmm-admin add mysql|postgresql|mongodb` (D2)

**Files:**
- Modify:
  - `admin/commands/management/add_mysql.go`: `CollectLogs string \`default:"error" help:"Database logs to collect from this host: error,slow,general or none"\`` in `AddMySQLCommand` (:94-130), passed into the body (:210-250)
  - `admin/commands/management/add_postgresql.go` and `add_mongodb.go`: `CollectLogs bool \`default:"true" negatable:"" help:"Collect the database log from this host"\``
  - the result templates of the three commands: print `log_sources` and `log_collection_messages`
  - `admin/agentlocal/agentlocal.go`: expose `ServerVersion` (:77) if it isn't exported already
- Test:
  - `admin/commands/management/add_mysql_test.go`
  - `add_postgresql_test.go`
  - `add_mongodb_test.go`

**Interfaces:**
- Consumes the `collect_logs` body fields (Task 3) and `agentlocal.GetStatus().ServerVersion`.
- Produces `func collectLogsSupported(serverVersion string) bool`, true for 3.11.0-0 and later. When it is false, pmm-admin leaves `collect_logs` unset. If the user passed the flag explicitly, it prints "log collection needs PMM Server 3.11 or newer"; on the default, it prints nothing.

- [ ] **Step 1: Write the failing tests.**
  - `TestAddMySQLDefaultCollectsErrorLog`: the built request body has `CollectLogs == "error"`.
  - `TestAddMySQLCollectLogsNone`: the value is `"none"`.
  - `TestAddMySQLSkipsCollectLogsOnOldServer`: with server version `3.10.0`, the body's `CollectLogs` is empty and stdout has no message. With an explicit `--collect-logs=slow`, the message is printed.
  - `TestAddPostgreSQLNoCollectLogs`: `--no-collect-logs` gives `false`.
  - A result template test: messages are printed under "Logs:".
- [ ] **Step 2: Run them.** `cd admin && go test ./commands/management/ -run CollectLogs`. Expected: FAIL.
- [ ] **Step 3: Implement** the flags, the version check and the templates.
- [ ] **Step 4: Run them again.** Same command, then `make prepare-pr`. Expected: PASS. Then on a CHAOS VM with MySQL 8.4, PostgreSQL 17 and MongoDB 7 (`pmm-qa:chaos-docker-provisioning`), check the acceptance criteria of PMM-15584/85/86:
  - default add → rows in `otel.logs` within 1 minute, labelled with the service;
  - a remote add → a message and no source;
  - `--discover` on a service added before this feature.
- [ ] **Step 5: Commit.**
  ```bash
  git add admin
  git commit -s -m "PMM-15584 Collect database logs by default on add"
  ```

### Task 5: Database log presets

**Files:**
- Create:
  - `managed/services/otel/presets/builtin/mongodb_json.yml`
  - `managed/services/otel/presets/builtin/mysql_slow.yml`
  - `managed/services/otel/presets/builtin/mysql_general.yml`
  - `managed/services/otel/presets/builtin/postgres_csv.yml`
  - `managed/services/otel/presets/builtin/postgres_json.yml`
  - `managed/services/otel/presets/testdata/<name>/input.log` and `expected.json`, one sample per supported version: MongoDB 6, 7, 8; MySQL 8.0, 8.4; PostgreSQL 13–17 for csv, 15–17 for json
- Modify: `managed/services/otel/presets/validate_test.go` (`TestBuiltInsLoad`, adding the five names)

**Interfaces:**
- The parsing rules (PMM-15587):
  - **`mongodb_json`:** `json_parser`. Timestamp from `t.$date` (RFC 3339). Severity from `s`: F→fatal, E→error, W→warn, I→info, D1–D5→debug. Body from `msg`. `move` `c`, `id` and `ctx` to `attributes["mongodb.component"]`, `["mongodb.id"]` and `["mongodb.context"]`. `attr` stays an attribute.
  - **`mysql_slow`:** `recombine` with `is_first_entry: body matches "^# Time: "`, then `regex_parser` for `# Time:`, `# User@Host:` and `# Query_time:`. Skip the header lines (`/usr/sbin/mysqld, Version:`, `Tcp port:`, `Time  Id Command`) with a built-in `filter`.
  - **`mysql_general`:** `recombine` where a line that doesn't start with an ISO timestamp continues the previous entry, then a tab-separated `regex_parser` into time, thread id, command and argument.
  - **`postgres_csv`:** `csv_parser` with one header per major version. PostgreSQL 13 adds `backend_type`; 14+ adds `leader_pid` and `query_id`. Choose by field count with a built-in `router` on the column count (built-ins are exempt from the expression ban). Quoted fields may contain newlines, so `recombine` until the quotes balance.
  - **`postgres_json`:** `json_parser`; severity from `error_severity`; body from `message`; timestamp from `timestamp`.
  - **Time zones:** MySQL `+02:00` offsets and PostgreSQL abbreviations convert to UTC. The PR description lists the ambiguous abbreviations (`IST`, `CST`, …), resolved to the IANA zone that Go's `time` package gives.

- [ ] **Step 1: Write the failing tests.** Add the testdata, including a multi-line slow-log entry, a multi-line general-log statement, a CSV field with an embedded newline, a CEST timestamp and a MySQL `+02:00` timestamp. Extend `TestBuiltInsLoad`.
- [ ] **Step 2: Run them.** `cd build/otelcol/presettest && go test ./...` and `cd managed && go test ./services/otel/presets/`. Expected: FAIL.
- [ ] **Step 3: Implement** the five preset files.
- [ ] **Step 4: Run them again.** Same commands. Expected: PASS. Check that `pmm-admin add logs --help` on a Feature Build lists the new presets.
- [ ] **Step 5: Commit.**
  ```bash
  git add managed/services/otel/presets
  git commit -s -m "PMM-15587 Add database log parser presets"
  ```
