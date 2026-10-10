# OpenTelemetry in PMM: roadmap and shared contract

> **For agentic workers:** this file is the index. Execute plans 01–06 one at a time, each with superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Every plan's tasks implicitly include the **Shared contract** and **Global constraints** below.

**Goal:** Collect database, OS, application and PMM Server logs, and later traces, through OpenTelemetry Collectors, and store them in ClickHouse next to QAN. The work follows epic [PMM-15567](https://perconadev.atlassian.net/browse/PMM-15567).

**Spec:** [`docs/superpowers/specs/2026-10-10-opentelemetry-analysis.md`](../specs/2026-10-10-opentelemetry-analysis.md), together with PMM-15567 and its child tickets. The analysis records decisions D1–D14; the plans assume the recommended option of each. If a decision goes the other way, re-plan the tasks it lists under "Blocks".

---

## Plans and order

| Plan | Scope | Tickets | Repos | Depends on |
|---|---|---|---|---|
| (Phase 0) | ADRs ADR-01..14 in `docs/adr/`, threat model, capacity harness. These are not code plans; they run alongside plan 01. Merge the ADRs and get the threat-model sign-off before plans 01 and 02 merge. | PMM-15590, 15591, 15592 | pmm | — |
| [01](2026-10-10-opentelemetry-01-server-receiver.md) | OCB collector build; server receiver; `otel` schema; ClickHouse users; settings and `PMM_ENABLE_OTEL`; `/otlp/`; redaction; retention, size cap and watermark; status and purge; HA server bits | PMM-15571, 15577 (build, server image), 15594 (server side) | pmm, percona-helm-charts | D1, D4, D5, D6, D14 decided |
| [02](2026-10-10-opentelemetry-02-node-collector.md) | `OTEL_COLLECTOR` agent type; `log_sources`; parser presets (backend, built-ins); config params and agent-side rendering; privilege drop; status per source; PMM Server's own logs; self-metrics scrape; telemetry | PMM-15572, 15574 (backend), 15575, 15577 (client packaging), 15593 (scrape) | pmm, grafana (Inventory types) | 01 tasks 1–4 |
| [03](2026-10-10-opentelemetry-03-ui-and-grafana.md) | Settings → OTEL tab (settings, presets, log sources); `ClickHouse-OTEL` datasource; Admin-only guard in the Grafana fork; Logs dashboard and links; collector health dashboard and alerts | PMM-15574 (UI), 15576, 15593 (dashboard, alerts) | pmm, grafana | 01, 02 |
| [04](2026-10-10-opentelemetry-04-database-logs.md) | `pmm-admin add/remove logs`; log discovery on the agent; `--collect-logs` on `add mysql/postgresql/mongodb` with a default; database presets | PMM-15573, 15584, 15585, 15586, 15587 | pmm | 02 |
| [05](2026-10-10-opentelemetry-05-traces.md) | Trace schema; server traces pipeline; node traces receiver and sampling; `pmm-admin add/remove traces`; traces dashboard; correlation | PMM-15579, 15580, 15582 | pmm | 01–03. Re-plan with writing-plans once 02 has merged |
| [06](2026-10-10-opentelemetry-06-lbac.md) | LBAC for logs and traces: ingest identity, selector evaluator, logs and traces query API that fails closed, log explorer | none yet: propose a follow-up epic | pmm, grafana | 05. Design-level; re-plan before execution |

**Release gating** (from the epic):
- Phase 0 and plans 01–03 make up 3.11.0 Technical Preview.
- Plan 04 is Phase 1b.
- Plan 05 is Phase 2.
- Plan 06 needs its own epic.

The QA automation ticket (PMM-15578) adds its tests in percona/pmm-qa against each Feature Build. Each plan lists the API tests it adds in this repo.

**Docs:** in [`documentation/`](../../../documentation/), following `documentation/AGENTS.md`, in one PR per phase, **merged only when the release ships** (a merge to `main` publishes live). The docs PR covers:
- the "Logs & traces (OpenTelemetry)" section, marked Technical Preview;
- `PMM_ENABLE_OTEL` and the Helm value;
- the `pmm-admin` commands;
- presets;
- retention, the size cap and the watermark;
- who can read logs;
- the sizing guide;
- the non-root read requirements: `adm` and `systemd-journal` groups, and ACLs on RHEL;
- the `docker run` mounts for journald and database logs;
- a correction of the 3.8.0/3.8.1 statement that PMM accepts no inbound OTel traffic;
- downgrade behaviour: the `otel` database and the `log_sources` and `log_parser_presets` tables stay in place, unused; an older server ignores them, and an older client never receives the agent type.

**Outside this repo:**
- pmm-dump support for `otel.logs` (PMM-15571 asks for it) lives in percona/pmm-dump. File a ticket there once plan 01 has merged.
- Kubernetes sidecar mounts need the Percona operators (PMM-15577). File one ticket per operator, listing the mounts from plan 02, Task 10b.

**Not planned here** (analysis D8):
- **Profiles:** OTLP profiles are Alpha, and the ClickHouse exporter marks them "development". Write an ADR that records this, and revisit it when the exporter reaches beta.
- **eBPF:** it stays in PMM-15588. It reaches PMM as OTLP traces (plan 05) and as metrics for VictoriaMetrics (PMM-15589), so no extra server receiver is needed here.

---

## Shared contract

These names are fixed across all plans. A task that needs a name not listed here defines it in its own **Interfaces** block.

**Agent type**

| Item | Value |
|---|---|
| Proto enum | `AGENT_TYPE_OTEL_COLLECTOR = 20` in `api/inventory/v1/agents.proto` (check that 20 is still free when you start) |
| Model constant | `models.OtelCollectorType AgentType = "otel-collector"` |
| CLI type string | `types.AgentTypeOtelCollector = "AGENT_TYPE_OTEL_COLLECTOR"`, display name `"otel_collector"` |
| Inventory message | `inventory.v1.OtelCollector { agent_id = 1; pmm_agent_id = 2; disabled = 3; map<string,string> custom_labels = 4; AgentStatus status = 10; string process_exec_path = 11; uint32 listen_port = 12; }`. `listen_port` is the collector's self-metrics port. |
| Version gate | `version.OtelCollectorSupportVersion = version.MustParse("3.11.0-0")` in `version/features.go`. Change it if the target release moves. |

**Agent channel** (`api/agent/v1/otel.proto`, new file in package `agent.v1`)

```proto
enum OtelLogSourceKind { OTEL_LOG_SOURCE_KIND_UNSPECIFIED = 0; OTEL_LOG_SOURCE_KIND_FILE = 1; OTEL_LOG_SOURCE_KIND_JOURNALD = 2; }
enum LogSourceState { LOG_SOURCE_STATE_UNSPECIFIED = 0; LOG_SOURCE_STATE_COLLECTING = 1; LOG_SOURCE_STATE_NOT_ALLOWED = 2;
                      LOG_SOURCE_STATE_NOT_READABLE = 3; LOG_SOURCE_STATE_NOT_FOUND = 4; LOG_SOURCE_STATE_JOURNALD_UNAVAILABLE = 5; }
message OtelLogSource { string id = 1; OtelLogSourceKind kind = 2; string path = 3; repeated string units = 4;
                        string operators_yaml = 5; map<string,string> resource_attributes = 6; bool discovered = 7; }
message OtelCollectorParams { repeated OtelLogSource log_sources = 1; map<string,string> resource_attributes = 2;
                              bool send_to_local_receiver = 3; uint32 sending_queue_mib = 4; }
message LogSourceStatus { string log_source_id = 1; LogSourceState state = 2; string reason = 3; }
```

These messages are referenced from:
- `SetStateRequest.AgentProcess.otel_collector = 9` (type `OtelCollectorParams`);
- `StateChangedRequest.log_source_statuses = 6` (type `repeated LogSourceStatus`).

Plan 05 adds `OtelTracesReceiver traces_receiver = 5` to `OtelCollectorParams`. Plan 04 adds `LogDiscoveryRequest` and `LogDiscoveryResponse`.

**Public API** (`api/otel/v1/otel.proto`, package `otel.v1`, service `OtelService`, auth rule `"/v1/otel": admin`)

| RPC | HTTP | Plan |
|---|---|---|
| `GetStatus` | `GET /v1/otel/status` | 01 |
| `Purge` | `POST /v1/otel:purge` | 01 |
| `ListLogParserPresets` / `GetLogParserPreset` | `GET /v1/otel/log-parser-presets`, `GET …/{preset_id}` | 02 |
| `AddLogParserPreset` / `ChangeLogParserPreset` / `RemoveLogParserPreset` | `POST …`, `PUT …/{preset_id}`, `DELETE …/{preset_id}` | 02 |
| `ListLogSources` / `AddLogSource` / `RemoveLogSources` | `GET /v1/otel/log-sources`, `POST /v1/otel/log-sources`, `POST /v1/otel/log-sources:remove` | 02 |
| `DiscoverLogSources` | `POST /v1/otel/log-sources:discover` | 04 |
| `ChangeTracesReceiver` | `POST /v1/otel/traces-receiver` | 05 |

**Settings**
- **Proto:** `server.v1.OtelSettings { bool collector_enabled = 1; uint32 logs_retention_days = 2; uint32 traces_retention_days = 3; uint32 max_disk_gb = 4; uint32 disk_watermark_percent = 5; }` as `Settings.otel = 22`.
- **Partial updates:** `server.v1.ChangeOtelSettings` holds the same fields, each `optional`, as `ChangeSettingsRequest.otel = 16`.
- **Model:** `models.OtelSettings` (JSON key `otel`), with `(*Settings).IsOtelCollectorEnabled() bool`.

| Default | Value |
|---|---|
| `OtelCollectorEnabledDefault` | `true` |
| `OtelLogsRetentionDaysDefault` | `7` |
| `OtelTracesRetentionDaysDefault` | `7` |
| `OtelMaxDiskGBDefault` | `10` (provisional until PMM-15592) |
| `OtelDiskWatermarkPercentDefault` | `80` |

- **Env:** `PMM_ENABLE_OTEL`. Helm: `otel.enabled`.

**ClickHouse**
- Database `otel`, tables `otel.logs` (plan 01), and `otel.otel_traces`, `otel.otel_traces_trace_id_ts`, `otel.otel_traces_service_name` (plan 05).
- Materialized columns `PmmNodeId` and `PmmServiceId`. Sort key `(PmmNodeId, PmmServiceId, ServiceName, TimestampTime)`. Daily partitions `toDate(TimestampTime)`.
- Users `otel_writer` (INSERT on `otel.*`) and `otel_reader` (SELECT on `otel.*`), in one `users.d` drop-in: `build/ansible/roles/clickhouse/files/users.d/otel.xml`.
- Env `PMM_CLICKHOUSE_OTEL_WRITER_USER/PASSWORD` and `PMM_CLICKHOUSE_OTEL_READER_USER/PASSWORD`. Defaults are user = password = the user name, with localhost-only networks, as for the existing `grafana` user. HA overrides them with generated secrets.
- qan-api2 owns the schema: `migrations.RunOtel(dsn string, templateData map[string]any, isCluster bool, clusterName string) error`, with SQL in `qan-api2/migrations/otel/sql/` and migrations table `otel_schema_migrations`.

**Server collector**
- supervisord program `otel-collector`, log `/srv/logs/otel-collector.log`.
- Config `/etc/otel-collector/config.yaml`, mode 0600, owned by `pmm`.
- OTLP/HTTP receiver on `127.0.0.1:4318`; self-metrics on `127.0.0.1:4319`. No gRPC.
- Binary `/usr/local/percona/pmm/tools/otelcol`, from the pmm-client tarball (the Nomad precedent).
- nginx `location /otlp/ → http://127.0.0.1:4318/`.

**Identity attributes** (analysis §3): `pmm.node_id`, `pmm.agent_id`, `pmm.service_id`, `pmm.service_name`, `service.name`, `log.file.path`.

**Packages**

| Package | Holds |
|---|---|
| `managed/services/otel` | server config, `Service` (reconcile loop), `Cleaner` (leader job), the `OtelService` gRPC server |
| `managed/services/otel/presets` | built-in YAML, loader, validator |
| `managed/services/agents/otelcollector.go` | builds `OtelCollectorParams` |
| `agent/agents/otelcollector` | YAML rendering, allow-list, privilege drop |

**pmm-agent config** (`agent/config/config.go`)
- `Paths.OtelCollector`: yaml `otelcol`, default `tools/otelcol`.
- `Paths.OtelCollectorDataDir`: yaml `otelcol_data_dir`, default derived like `nomad_data_dir`.
- `LogSources.AllowedPaths []string`: yaml `log-sources.allowed-paths`, default `["/var/log"]`.
- `LogSources.AllowedJournaldUnits []string`: yaml `log-sources.allowed-journald-units`, default empty, which allows no hand-typed units.
- Env vars `PMM_AGENT_PATHS_OTELCOL`, `PMM_AGENT_PATHS_OTELCOL_DATA_DIR`, `PMM_AGENT_LOG_SOURCES_ALLOWED_PATHS`, `PMM_AGENT_LOG_SOURCES_ALLOWED_JOURNALD_UNITS`.

---

## Global constraints

Every task follows `AGENTS.md`; the rules agents most often miss are repeated here:
- **Git:** branch `PMM-<child ticket>-short-description`; commits titled `PMM-XXXX Short summary` and signed off (`git commit -s`). A new commit for each review round.
- **Go:**
  - reform only, and `go-sqlmock` for managed unit tests (`testdb.Open` only for migrations);
  - `status.Error` with gRPC codes; `logrus` `*Entry` with fields;
  - no inline comments; no inline `err` checks; `any`; `t.Context()`;
  - the license header on every new Go file.
- **Generated code:** run `make gen` after any `.proto`, reform model or mocked interface change; never hand-edit generated files. Run `make prepare-pr` before a Go task is called done.
- **Compatibility:**
  - proto changes are additive only (`buf breaking`);
  - managed migrations are forward-only and numbered after `main`'s latest at merge time (120 today);
  - qan-api2 `otel` migrations are a separate chain starting at `01`;
  - older pmm-agents get a clear `FailedPrecondition` and never receive the new type.
- **Security** (from the epic and PMM-15591):
  - no OTLP port on a non-loopback address unless an admin sets one;
  - no gRPC on the server;
  - `/otlp/` needs Admin;
  - the collector runs as non-root `pmm-agent`;
  - files that hold credentials are 0600;
  - no user string passes through `text/template`;
  - `$` in user strings is escaped as `$$`;
  - logs are readable by Admins only until plan 06.
- **Limits:**
  - nginx body limit 10 MB;
  - server batches of at least 5 s or 10,000 records and under 10 MB;
  - client sending queue on disk, 1024 MiB by default;
  - retention 7 d; watermark 80 %.
- **Supply chain:** the collector is built with OCB from pinned versions, with only these components:
  - receivers `filelog`, `journald`, `otlp`;
  - processors `resource`, `memory_limiter`, `batch`, `transform`, `probabilistic_sampler`, `tail_sampling`;
  - exporters `otlphttp`, `clickhouse`;
  - extension `file_storage`.
  No AGPL dependencies.
- **Product:** OTEL is on by default and labelled **Technical Preview** in the UI.
