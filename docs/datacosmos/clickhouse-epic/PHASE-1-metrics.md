# Phase 1 — Metrics + Inventory + API + pmm-admin

**Outcome:** `pmm-admin add clickhouse <name> <address>` registers a ClickHouse
service in PMM inventory, and its metrics reach VictoriaMetrics — via the
ClickHouse native Prometheus endpoint **or** a PMM-managed `clickhouse_exporter`.

## Design

### Dual metrics source — two agent types, not one

PMM keys process spawning, scrape-config, status and converters off
`AgentType`. The two metric sources are modelled as two agent kinds:

| Source | Agent type | Process? | Scrape target |
|---|---|---|---|
| Managed exporter (old ClickHouse) | `AGENT_TYPE_CLICKHOUSE_EXPORTER` (new, `=20`) | `clickhouse_exporter` spawned by pmm-agent | `127.0.0.1:{listen_port}/metrics` |
| Native endpoint (CH 22.6+) | `AGENT_TYPE_EXTERNAL_EXPORTER` (existing — reused) | none | `{address}:{native_port}/metrics` |

Reusing `external-exporter` for the native case means **zero new code** in the
scrape/status path (`prometheus.go` already handles `ExternalExporterType`).
Trade-off: native ClickHouse shows in inventory as `external-exporter`; a
dedicated `AGENT_TYPE_CLICKHOUSE_NATIVE` can be added later if the UI needs it.

### Auto-probe + override

`AddClickHouseService` carries `metrics_source` (`UNSPECIFIED|NATIVE|EXPORTER`)
and `native_metrics_port` (default `9363`). On `UNSPECIFIED`, during the
connection check, probe `HTTP GET {address}:{native_metrics_port}/metrics`
(3 s timeout) → reachable ⇒ NATIVE, else EXPORTER. Explicit values skip the
probe; forced NATIVE that fails the probe ⇒ `FailedPrecondition`, full rollback.
CLI: `--metrics-source=auto|native|exporter`, `--native-metrics-port`.

### Template

**Valkey** is the freshest and simplest end-to-end DB integration in the
codebase — use it as the scaffold throughout (service type 7, agent type 17,
migration 111).

## Development line (ordered — tree compiles after each numbered step)

### 1. Inventory protos
- `api/inventory/v1/services.proto` — `SERVICE_TYPE_CLICKHOUSE_SERVICE = 8`;
  `message ClickHouseService` (modelled on `ValkeyService`); add
  `ClickHouseService clickhouse = 8` to the list/oneof slots.
- `api/inventory/v1/agents.proto` — `AGENT_TYPE_CLICKHOUSE_EXPORTER = 20`;
  `message ClickHouseExporter` (modelled on `ValkeyExporter`);
  `AddClickHouseExporterParams` / `ChangeClickHouseExporterParams` + oneof slots.

### 2. Management proto + regenerate
- New `api/management/v1/clickhouse.proto` (from `mysql.proto`):
  `AddClickHouseServiceParams` (+ `MetricsSource metrics_source`,
  `uint32 native_metrics_port`), `ClickHouseServiceResult`.
- `api/management/v1/service.proto` — import it; add `clickhouse = 9` to the
  `AddServiceRequest` / `AddServiceResponse` oneofs.
- `cd api && make gen` (buf) — regenerates `*.pb.go`, gRPC, swagger JSON
  clients. Commit generated files in the same commit. **Never hand-edit them.**

### 3. Inventory type registries
- `api/inventory/v1/types/service_types.go` — `ServiceTypeClickHouseService` + display-name map.
- `api/inventory/v1/types/agent_types.go` — `AgentTypeClickHouseExporter` + map.

### 4. pmm-managed models
- `managed/models/service_model.go` — `ClickHouseServiceType ServiceType = "clickhouse"`.
- `managed/models/agent_model.go` — `ClickHouseExporterType AgentType = "clickhouse_exporter"`;
  `ClickHouseOptions` struct (TLS material + `NativeEndpoint bool`,
  `NativeMetricsPort uint16`) with `Value()/Scan()/IsEmpty()`; add the field to
  the `Agent` struct; DSN + `Files()` cases; `ClickHouseOptionsFromRequest`.
- `managed/models/database.go` — **migration `118`**: `ALTER TABLE agents ADD
  COLUMN clickhouse_options JSONB` (+ regenerate the reform metadata for `Agent`).

### 5. Converters
- `managed/services/converters.go` — `ToAPIService` and `ToAPIAgent` cases for
  the new types; reverse `ServiceType → models` map entry.

### 6. Management service
- New `managed/services/management/clickhouse.go` (from `valkey.go`): the
  `addClickHouse` flow with auto-probe; native ⇒ `models.CreateExternalExporter`,
  exporter ⇒ `models.CreateAgent(..., ClickHouseExporterType, ...)`.
- `managed/services/management/service.go` — add to `supportedServices` map and
  the `AddService` switch (`case *managementv1.AddServiceRequest_Clickhouse`).

### 7. Exporter config + state dispatch
- New `managed/services/agents/clickhouse.go` — `clickhouseExporterConfig(...)`
  (from `valkey.go`) building the `clickhouse_exporter` process args.
- `managed/services/agents/state.go` — `ClickHouseExporterType` in the exporter
  case + dispatch.
- `managed/services/victoriametrics/prometheus.go` — `scrapeConfigForClickHouseExporter`
  helper; native case already covered by `ExternalExporterType`.

### 8. pmm-agent
- `agent/config/config.go` — `Paths.ClickHouseExporter` (default
  `clickhouse_exporter`, env `PMM_AGENT_PATHS_CLICKHOUSE_EXPORTER`, CLI flag).
- `agent/agents/supervisor/supervisor.go` — `processParams` + `version` cases
  for `AGENT_TYPE_CLICKHOUSE_EXPORTER`.
- `agent/agents/supervisor/deps.go` — `clickhouseExporterRegexp`.

### 9. clickhouse_exporter binary
- Promote `agent/agents/clickhouse/` — expand the collector to scrape
  `system.metrics`, `system.events`, `system.asynchronous_metrics`,
  `system.parts`, `system.replicas` (always-populated tables — **not**
  `system.query_log`, which is empty on a fresh server); emit `clickhouse_up`,
  `clickhouse_version_info`, and the metric families in the
  [metric contract](PHASE-2-dashboards.md#metric-contract). English comments.
- New `agent/cmd/clickhouse_exporter/main.go` — `package main`, HTTP `/metrics`
  via `promhttp`, `--web.listen-address`/`--version` flags, emits the
  `clickhouse_exporter, version X` banner the version regex expects.

### 10. pmm-admin
- New `admin/commands/management/add_clickhouse.go` and
  `admin/commands/inventory/add_service_clickhouse.go` (from `add_valkey.go`):
  `--metrics-source`, `--native-metrics-port`, `--username/--password`, TLS,
  `--metrics-mode`, `--environment/--cluster/--replication-set`,
  `--custom-labels`, `--skip-connection-check`.
- `admin/commands/management/add.go`, `management.go`, `remove.go` — register
  the `clickhouse` command and service type.

## Validation criteria

1. `cd api && make gen` is idempotent (no diff on a second run); full repo
   builds `pmm-managed`, `pmm-agent`, `pmm-admin`, `clickhouse_exporter`.
2. Fresh PMM Server DB reaches schema `118`; `agents.clickhouse_options` exists;
   upgrade from `117` succeeds.
3. `pmm-admin add clickhouse` → `pmm-admin inventory list services/agents`
   shows the ClickHouse service + the right agent (exporter or external).
4. `make check` (golangci-lint) clean; new files carry the AGPL header.
5. Unit tests: converters, `add_clickhouse_test.go`, the probe logic.
6. No regression in `services_test.go`, `agents_test.go`, state/VM suites.
7. `up{service_type="clickhouse"} == 1` in VictoriaMetrics within ~30 s, both modes.

## Integration tests

See [INTEGRATION-TESTS.md](INTEGRATION-TESTS.md) — IT-1.x. Summary: exporter
mode end-to-end; native mode end-to-end; auto-probe picks native; auto-probe
falls back to exporter; forced-native fails fast with rollback; bad-credentials
rollback; `remove clickhouse`; the version matrix (< 22.6 exporter-only, ≥ 22.6
both).

## Risks

- **Proto-regen ripple** — `make gen` regenerates many files (gRPC, swagger).
  Regenerate on a clean tree with the pinned toolchain; review the diff is only
  ClickHouse + oneof additions. Enum numbers (`8`, `20`) are wire contracts —
  must be new, never reused. Proto + admin changes land in one commit.
- **`reform` model** — `Agent` is reform-generated; the `ClickHouseOptions`
  field, regenerated reform metadata, and migration `118` must land together.
- **Version skew** — new pmm-agent ↔ old pmm-managed: the supervisor `default`
  case must reject unknown agent types gracefully (it does). Document that
  ClickHouse monitoring needs server + agent both on the Phase-1 release.
- **clickhouse_exporter immaturity** — the draft is a 2-metric toy; Step 9 is
  the largest new code. Scrape always-populated `system.*` tables, never
  `system.query_log` (empty on fresh servers).
- **Packaging** — the binary must reach `/usr/local/percona/pmm/exporters/` and
  match the supervisor's path templates, else mode (b) fails at runtime. Full
  packaging is Phase 4; Phase 1 must at least produce a buildable binary.
