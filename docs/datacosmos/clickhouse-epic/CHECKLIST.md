# ClickHouse epic — implementation-control checklist

Status keys: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked.
Update this file as work progresses; it is the single source of truth for epic
state. Each phase ends with `go build ./...` + `go vet` clean, a commit, and a
user checkpoint.

## Phase 0 — research & documentation set

- [x] Switch to `feat/clickhouse-collector`
- [x] Deep per-phase research (3 planning agents)
- [x] `docs/datacosmos/clickhouse-epic/` written (OVERVIEW, PHASE-1..4,
      INTEGRATION-TESTS, CHECKLIST)
- [x] Round 1 — plan reviewed against the code + ClickHouse docs;
      contradictions corrected — agent-type number collision, native-endpoint
      version myth, native/exporter metric-naming unification, qan-api2 not
      DB-agnostic (see OVERVIEW "Review findings")
- [x] Round 2 — reuse maximization; PHASE-2 rewritten reuse-first (adapt the
      official `clickhouse-mixin`, 8→5 dashboards), attribution made mandatory,
      OVERVIEW reuse-first rule added (see OVERVIEW "Review findings", round 2)
- [x] Commit the documentation set; user checkpoint before Phase 1 code

## Phase 1 — Metrics + Inventory + API + pmm-admin

- [x] 1.1 Inventory protos — `SERVICE_TYPE_CLICKHOUSE_SERVICE`,
      `AGENT_TYPE_CLICKHOUSE_EXPORTER`, messages
- [x] 1.2 `api/management/v1/clickhouse.proto` + `service.proto` oneof; `make gen`
- [x] 1.3 Inventory type registries (`service_types.go`, `agent_types.go`)
- [x] 1.4 pmm-managed models + `ClickHouseOptions` + migration `118`
- [x] 1.5 Converters (`ToAPIService`/`ToAPIAgent` + reverse map)
- [x] 1.6 `managed/services/management/clickhouse.go` (addClickHouse + probe) + `service.go`
- [x] 1.7 `managed/services/agents/clickhouse.go` + `state.go` + `prometheus.go`
- [x] 1.8 pmm-agent — `config.go` paths, `supervisor.go`, `deps.go`
- [x] 1.9 `clickhouse_exporter` — expand collector + `agent/cmd/clickhouse_exporter/main.go`
- [x] 1.10 pmm-admin — `add_clickhouse.go` (management + inventory) + registration
- [~] 1.V Validation: `make gen` idempotent, build, `go vet`, `go-sumtype`,
      golangci-lint (0 new issues), exporter `--version`/`--help`, unit tests
      green — all done; migration 118 / inventory round-trip / `up{...}==1`
      need a live PMM stack (covered by 1.IT)
- [ ] 1.IT Integration tests IT-1.1 … IT-1.8 green (needs Docker matrix)
- [x] Phase 1 committed; user checkpoint

## Phase 2 — Grafana dashboards

- [x] 2.0 Obtain bases (cloned `clickhouse-mixin` — no LICENSE, used as
      design reference only) + scaffold `dashboards/dashboards/ClickHouse/` + 5 UIDs fixed
- [x] 2.1 `ClickHouse_Instance_Summary.json` (PMM-shell rebuild — anchor)
- [x] 2.2 `ClickHouse_Query_Performance.json`
- [x] 2.3 `ClickHouse_Replication.json`
- [x] 2.4 `ClickHouse_Instances_Overview.json` (PMM-shell fleet view)
- [x] 2.5 `ClickHouse_Instances_Compare.json` (PMM-shell fleet view)
- [x] 2.6 Cross-linking (data links)
- [x] 2.7 Register all 5 in `pmm-app/src/plugin.json`
- [x] 2.8 Normalize every file via `cleanup-dash.py`; write `ATTRIBUTION.md`
- [~] 2.V Validation: `cleanup-dash --check-only` passes all 5, unique UIDs,
      VM datasource only, attribution present — done. `make -C dashboards build`
      needs `yarn install` (node env); covered by 2.IT
- [ ] 2.IT Integration tests IT-2.1 … IT-2.5 green (needs live PMM stack)
- [x] Phase 2 committed; user checkpoint

## Phase 3 — Query Analytics (QAN)

- [x] 3.A API/proto — `AGENT_TYPE_QAN_CLICKHOUSE_QUERYLOG_AGENT`,
      `MetricsBucket.ClickHouse`, qan/v1 fields; `make gen`
- [x] 3.B pmm-agent — `agent/agents/clickhouse/querylog/` agent + supervisor wiring
- [x] 3.C pmm-managed — agent model/helpers, agents config, `state.go`,
      inventory/management plumbing, `qan/client.go` `fillClickHouse`, `--qan` flag
- [x] 3.D qan-api2 — migrations (columns + `agent_type` Enum8),
      `data_ingestion.go`, `base.go`, `analytics/base.go`
- [x] 3.E Unit tests (`makeBuckets`, `fingerprint`, `percentile`) — in querylog_test.go
- [~] 3.V Validation: build, vet, go-sumtype, golangci-lint (0 new issues),
      unit tests green — done; bucket/accuracy/double-count/WAITING checks
      need a live ClickHouse (covered by 3.IT)
- [ ] 3.IT Integration tests IT-3.1 … IT-3.9 green (needs live ClickHouse)
- [x] Phase 3 committed; user checkpoint

## Phase 4 — Distribution, tests, docs

- [x] 4.1 Confirm `agent/cmd/clickhouse_exporter/main.go` (Phase 1 hand-off)
- [x] 4.2 `build/Makefile.clickhouse` (new)
- [x] 4.3 `build/scripts/build-client-binary` — `gobuild_component clickhouse_exporter`
- [x] 4.4 RPM spec — install the binary
- [x] 4.5 DEB packaging — `rules` + `install`
- [x] 4.6 Docker client image check (EL9 copies the whole exporters/ dir — automatic)
- [x] 4.7 Verify supervisor path matches the packaged path (matches)
- [x] 4.8 Unit tests — exporter (`main_test.go` new, `config_test.go` from Phase 1)
- [x] 4.9 Unit tests — QAN agent (`querylog_test.go` from Phase 3)
- [x] 4.10 Extend `run-matrix.sh` — exporter build + widened `-run` filter
- [x] 4.11 CI workflow for ClickHouse tests (`.github/workflows/clickhouse.yml`)
- [x] 4.12 Docs — `.github/instructions/clickhouse.instructions.md`, user docs
- [~] 4.V Validation: build, `--version`/`--help`, `go test`/`vet`/lint clean —
      done; package-contents/install-perms need a real RPM/DEB build (4.IT)
- [ ] 4.IT Integration tests IT-4.1 … IT-4.6 green (needs Docker matrix + package build)
- [x] Phase 4 committed; user checkpoint

## Epic completion

- [ ] All 4 phases merged to `main` via PR
- [ ] ClickHouse integration matrix (`run-matrix.sh`) green — release gate
- [ ] A `v*-dc*` release published with full ClickHouse support
- [ ] (optional) upstream PR proposed to `percona/pmm`
