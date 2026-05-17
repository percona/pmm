# ClickHouse as a first-class monitored database in PMM — Epic Overview

## Goal

Make **ClickHouse** a fully-supported monitored database in PMM, on par with
MySQL and PostgreSQL: metrics, inventory/API, `pmm-admin add clickhouse`, Query
Analytics (QAN), and Grafana dashboards. Quality target: upstream PR standard,
so the work can later be proposed to `percona/pmm`.

Branch: `feat/clickhouse-collector`. Integrated into `main` via PR per phase set.

## Why

The fork currently has only a draft ClickHouse `prometheus.Collector`
(`agent/agents/clickhouse/{collector,config}.go`) — two toy metrics, not wired
into pmm-agent, no inventory type, no QAN, no dashboards. ClickHouse cannot be
added to PMM as a monitored service. This epic closes that gap.

## Architecture (where ClickHouse plugs into PMM)

```
ClickHouse server
   │   metrics                                  query analytics
   ├── native /metrics  ──┐                     system.query_log
   │   (CH 22.6+)         │                          │
   └── clickhouse_exporter│                     QAN agent (built-in,
       (PMM-managed,      │                     in pmm-agent)
        older CH)         │                          │
                          ▼                          ▼
                     vmagent ──► VictoriaMetrics  pmm-agent ──► pmm-managed
                          │                                       │
                     Grafana dashboards                       qan-api2 (ClickHouse store)
                                                                   │
                                                              QAN UI panel
```

**Metrics — two sources, selectable per service.** ClickHouse's native
Prometheus endpoint is **config-gated, not version-gated** (validated below):
it exists since ClickHouse ~19.14 (2019) but only when the operator enables the
`<prometheus>` section in the server config. So a given instance may or may not
expose it:

- **Native endpoint** — when `<prometheus>` is enabled (default `:9363/metrics`).
  PMM scrapes it; no PMM process. Modelled by reusing PMM's `external-exporter`.
  ClickHouse emits metrics under the prefixes `ClickHouseMetrics_*`,
  `ClickHouseProfileEvents_*`, `ClickHouseAsyncMetrics_*`.
- **Managed `clickhouse_exporter`** — for instances where `<prometheus>` is not
  enabled. A PMM-managed process agent, like `mysqld_exporter`. **It emits the
  same metric names as the native endpoint** (`ClickHouseMetrics_*` etc.) so a
  single dashboard set works regardless of source.

`AddClickHouseService` auto-probes the native endpoint and picks the source;
`pmm-admin add clickhouse --metrics-source=auto|native|exporter` overrides it.

> Validated against the code and ClickHouse docs — see [review findings](#review-findings).

## The four phases

| Phase | Scope | Outcome |
|---|---|---|
| **1 — Metrics + Inventory + API + pmm-admin** | proto (`SERVICE_TYPE_CLICKHOUSE_SERVICE`, `AGENT_TYPE_CLICKHOUSE_EXPORTER`), `clickhouse_exporter` binary, pmm-managed models/services, pmm-admin command, pmm-agent supervisor wiring | `pmm-admin add clickhouse` works; metrics reach VictoriaMetrics (native or exporter) |
| **2 — Grafana dashboards** | 5 dashboards under `dashboards/dashboards/ClickHouse/` (adapted from ClickHouse's official `clickhouse-mixin`), registered in `pmm-app` | ClickHouse dashboards render live metrics |
| **3 — Query Analytics (QAN)** | `MetricsBucket.ClickHouse` proto, built-in QAN agent reading `system.query_log`, qan-api2 ingestion changes | QAN shows ClickHouse query-level analytics |
| **4 — Distribution, tests, docs** | bundle `clickhouse_exporter` into pmm-client (RPM/DEB/build scripts), unit tests, extend the integration matrix, docs | shipped in the pmm-client package; fully tested |

Each phase is detailed in its own document:
[PHASE-1-metrics.md](PHASE-1-metrics.md) ·
[PHASE-2-dashboards.md](PHASE-2-dashboards.md) ·
[PHASE-3-qan.md](PHASE-3-qan.md) ·
[PHASE-4-distribution.md](PHASE-4-distribution.md).
Integration-test strategy: [INTEGRATION-TESTS.md](INTEGRATION-TESTS.md).
Live tracking: [CHECKLIST.md](CHECKLIST.md).

## Dependencies between phases

- **Phase 1 is the foundation.** It defines the proto types, the metric set,
  and the `clickhouse_exporter` binary. Phases 2 and 4 are **blocked on Phase
  1** (dashboards need the metric contract realised; packaging needs the
  buildable binary).
- **Phase 2** consumes the Phase 1 metric contract; can start once Phase 1
  emits the agreed metric families.
- **Phase 3** is largely independent of Phase 2 but extends Phase 1's proto and
  pmm-agent wiring; do it after Phase 1.
- **Phase 4** packages everything; runs last.

## Execution rules

- All work on `feat/clickhouse-collector`; small atomic commits.
- After each phase: `go build ./...` + `go vet` clean; phase committed; the
  CHECKLIST updated; **checkpoint with the user** before the next phase.
- **Release gate** (standing rule): no `v*-dc*` tag is published unless the
  ClickHouse integration matrix (`agent/agents/clickhouse/testdata/run-matrix.sh`)
  passes locally — extended each phase per INTEGRATION-TESTS.md.
- Proto/`*.pb.go` are regenerated with `make gen` (buf), never hand-edited.
- **Reuse-first** (standing rule): adapt existing assets rather than author
  from scratch — PMM's MySQL/PostgreSQL/Valkey code, services and dashboard
  shell as templates; ClickHouse's official ready-made dashboards for panels
  and PromQL. Every reused *external* asset is recorded with source + license
  + version (attribution is mandatory). See PHASE-2 "Reuse-first principle".

## Honest scale

~40-60 files across `api/`, `agent/`, `managed/`, `admin/`, `qan-api2/`,
`dashboards/`, `build/`. Proto enum additions ripple through generated gRPC and
swagger clients. This is a multi-phase engineering effort, delivered and
verified phase by phase — not a single drop.

## Review findings

The plan was reviewed against the actual code and external ClickHouse
documentation. Outcome:

**Verified correct** (against the code):
- `SERVICE_TYPE_CLICKHOUSE_SERVICE = 8` — `ServiceType` highest used is `7`.
- `AGENT_TYPE_CLICKHOUSE_EXPORTER = 20` — `AgentType` highest used is `19`.
- DB migration `118` — `managed/models/database.go` latest is `117`.
- `MetricsBucket.ClickHouse` = field `5` — `api/agent/v1/collector.proto` has
  `common=1, mysql=2, mongodb=3, postgresql=4`.
- `api/qan/v1/collector.proto` ClickHouse fields use `310+` — current max `309`.

**Corrected** (assumptions that were wrong):
1. *Native endpoint is config-gated, not version-gated.* ClickHouse's
   `<prometheus>` endpoint exists since ~v19.14 (2019), not "22.6+". The
   distinction between the two metric sources is whether the operator enabled
   `<prometheus>`, not the ClickHouse version. (The user's intent — "some
   ClickHouse without the native endpoint" — still holds, for config reasons.)
2. *Metric-naming unification.* ClickHouse's native endpoint emits
   `ClickHouseMetrics_*` / `ClickHouseProfileEvents_*` / `ClickHouseAsyncMetrics_*`.
   For one dashboard set to work in both modes, the managed `clickhouse_exporter`
   must emit those **same** names — not a custom `clickhouse_*` scheme. The
   Phase 2 metric contract was rewritten accordingly.
3. *No `clickhouse_up` metric.* `up` is synthesized by the scraper for every
   target; dashboard templating keys off `up{service_type="clickhouse"}` /
   `ClickHouseAsyncMetrics_Uptime`, not a non-existent `clickhouse_up`.
4. *Agent-type number collision.* Phase 1 and Phase 3 both drafted `=20`.
   Resolved: exporter `=20`, QAN agent `AGENT_TYPE_QAN_CLICKHOUSE_QUERYLOG_AGENT = 21`.
5. *qan-api2 is NOT DB-agnostic.* `qan-api2/models/data_ingestion.go` hardcodes
   `agent_type` as an `Enum8` (values 0-5, no ClickHouse). Phase 3 therefore
   requires qan-api2 migrations + an `Enum8` extension — as PHASE-3 already
   states; the earlier "no qan-api2 change" assumption is rejected.

**Round 2 — reuse maximization & dashboard sourcing.** A second review pass
focused on reusing what already exists:

6. *Dashboards are adapted, not authored.* ClickHouse publishes an official
   ready-made dashboard — `ClickHouse/clickhouse-mixin` (`dashboard.json`) —
   built on the very native metric families PMM emits (`ClickHouseMetrics_*`
   etc.). PHASE-2 was rewritten to **adapt** that mixin (re-point datasource,
   swap in PMM's templating cascade) instead of authoring panels from scratch,
   and the dashboard set was cut from 8 speculative files to **5** focused
   ones (Instance Summary, Query Performance, Replication, Instances Overview,
   Instances Compare). The mixin's panels/PromQL are reused verbatim.
7. *Attribution is mandatory.* Every reused external base (the mixin, Grafana
   Labs dashboards `23285`/`14192`) is recorded — source URL, license, commit/
   version — in `dashboards/dashboards/ClickHouse/ATTRIBUTION.md` plus an
   attribution line in each dashboard JSON `description`. Each base's license
   must be confirmed compatible with PMM's Apache-2.0 `dashboards/` tree before
   incorporation; if unclear, the base is used only as a reference to rebuild.
