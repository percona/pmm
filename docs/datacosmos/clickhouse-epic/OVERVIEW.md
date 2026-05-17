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

**Metrics — two sources, selectable per service** (a ClickHouse instance may or
may not expose the native Prometheus endpoint depending on its version/config):

- **Native endpoint** — ClickHouse 22.6+ serves Prometheus metrics directly
  (`<prometheus>` config, default `:9363/metrics`). PMM scrapes it; no PMM
  process. Modelled by reusing PMM's existing `external-exporter` machinery.
- **Managed `clickhouse_exporter`** — for older ClickHouse without the native
  endpoint. A PMM-managed process agent, exactly like `mysqld_exporter`.

`AddClickHouseService` auto-probes the native endpoint and picks the source;
`pmm-admin add clickhouse --metrics-source=auto|native|exporter` overrides it.

## The four phases

| Phase | Scope | Outcome |
|---|---|---|
| **1 — Metrics + Inventory + API + pmm-admin** | proto (`SERVICE_TYPE_CLICKHOUSE_SERVICE`, `AGENT_TYPE_CLICKHOUSE_EXPORTER`), `clickhouse_exporter` binary, pmm-managed models/services, pmm-admin command, pmm-agent supervisor wiring | `pmm-admin add clickhouse` works; metrics reach VictoriaMetrics (native or exporter) |
| **2 — Grafana dashboards** | 8 dashboards under `dashboards/dashboards/ClickHouse/`, registered in `pmm-app` | ClickHouse dashboards render live metrics |
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
  1** (dashboards need a real metric contract incl. `clickhouse_up`; packaging
  needs the buildable binary).
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

## Honest scale

~40-60 files across `api/`, `agent/`, `managed/`, `admin/`, `qan-api2/`,
`dashboards/`, `build/`. Proto enum additions ripple through generated gRPC and
swagger clients. This is a multi-phase engineering effort, delivered and
verified phase by phase — not a single drop.
