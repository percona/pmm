# Phase 2 — Grafana dashboards for ClickHouse

**Outcome:** a set of ClickHouse Grafana dashboards under
`dashboards/dashboards/ClickHouse/`, registered in the `pmm-app` plugin,
rendering live ClickHouse metrics — **adapted from existing, ready-made
ClickHouse dashboards** rather than authored from scratch.

## Reuse-first principle

Phase 2 does **not** invent dashboards. ClickHouse's metrics are already
well-covered by official, ready-made dashboards that target the very metric
names PMM emits (`ClickHouseMetrics_*` / `ClickHouseProfileEvents_*` /
`ClickHouseAsyncMetrics_*` — confirmed in Phase 1). Two layers are reused:

- **Panels / PromQL** — from ClickHouse's own dashboards (below).
- **Shell** (templating cascade, datasource variable, layout, tags, UID
  conventions, `cleanup-dash.py` normalization) — from PMM's existing
  PostgreSQL/Valkey dashboards.

Phase 2 = *adapt + organize*, not *author*.

### Base dashboards used  *(attribution is mandatory)*

| Base | Source | Role |
|---|---|---|
| **`ClickHouse/clickhouse-mixin`** — `dashboard.json` | `github.com/ClickHouse/clickhouse-mixin` | **primary** — ClickHouse's official dashboard, "nearly identical to ClickHouse's own internal dashboards", filters 1000+ metrics to ~125 mission-critical; targets the native `/metrics` endpoint |
| ClickHouse + Keeper Comprehensive | Grafana Labs dashboard `23285` | secondary — Keeper/replication panels reference |
| ClickHouse (community) | Grafana Labs dashboard `14192` | secondary — cross-check panel coverage |
| PMM `PostgreSQL/*.json`, `Valkey/*.json` | this repo, `dashboards/dashboards/` | the PMM shell — templating, datasource var, tags, UIDs |

**Before incorporating**: verify each base's license (the `clickhouse-mixin`
license must be confirmed — likely Apache-2.0, compatible with PMM's
`dashboards/` Apache-2.0; if unclear, treat panels as reference and rebuild).
Record every base + URL + license + commit/version in
`dashboards/dashboards/ClickHouse/ATTRIBUTION.md`, and keep an attribution line
in each dashboard JSON `description`.

## Design

- Dashboards bind to the **VictoriaMetrics / Prometheus** datasource (the
  `ClickHouseMetrics_*` / `ClickHouseProfileEvents_*` / `ClickHouseAsyncMetrics_*`
  metrics — emitted identically by the native endpoint and the Phase 1
  exporter) — **not** the bundled `grafana-clickhouse-datasource` plugin.
- Template variables cascade `environment → cluster → node_name → service_name`,
  keyed off `up{service_type="clickhouse"}` (scraper-synthesized) — there is
  **no** `clickhouse_up` metric.
- Tags on every dashboard: `ClickHouse`, `Percona`, `Services`. Stable
  lowercase-hyphen UIDs, fixed at creation.

## Development line (ordered)

**Step 0 — obtain bases + scaffold.** Clone `ClickHouse/clickhouse-mixin`;
record license + commit in `ATTRIBUTION.md`. Create
`dashboards/dashboards/ClickHouse/`. Add a `ClickHouse` row to
`.github/instructions/dashboards.instructions.md`. Fix UIDs:
`clickhouse-instance-summary`, `clickhouse-query-performance`,
`clickhouse-replication`, `clickhouse-instances-overview`,
`clickhouse-instances-compare`.

**Step 1 — `ClickHouse_Instance_Summary.json`** *(primary — the adapted mixin)*.
Take `clickhouse-mixin/dashboard.json` and adapt to PMM:
- repoint every panel datasource to the PMM Prometheus/VM datasource variable;
- replace the mixin's instance/job variables with the PMM cascade
  (`environment → cluster → node_name → service_name`), keyed off
  `up{service_type="clickhouse"}` — reuse the variable block from
  `PostgreSQL/PostgreSQL_Instance_Summary.json`;
- set PMM `tags`, the fixed UID, `editable:false`, `refresh:false`;
- keep the mixin's panels and PromQL essentially intact (that is the reuse);
- run `cleanup-dash.py`.

**Step 2 — organize / split out.** If `dashboard.json` is large, split its
panel groups into focused PMM-style dashboards — **reusing the mixin's panels
and queries verbatim**, only regrouping:
- `ClickHouse_Query_Performance.json` — query/select/insert/failed rates,
  durations (mixin query panels).
- `ClickHouse_Replication.json` — replica queue, delay, readonly, Keeper
  (mixin + dashboard `23285` panels). Degrades to no-data on single-node.
Only split where it genuinely improves clarity; do not pad the set.

**Step 3 — fleet views** (PMM-specific — no mixin equivalent). Clone the PMM
shell of `PostgreSQL_Instances_Overview.json` / `PostgreSQL_Instances_Compare.json`
and drop in the mixin's headline panels:
- `ClickHouse_Instances_Overview.json` — one row per `service_name`.
- `ClickHouse_Instances_Compare.json` — side-by-side, series by `service_name`.

**Step 4 — cross-linking.** Data links: Instance Summary ↔ the split-outs;
Overview/Compare → Instance Summary.

**Step 5 — registration.** Add each JSON to `dashboards/pmm-app/src/plugin.json`
`includes` with `path: dashboards/ClickHouse/<File>.json` (avoid the existing
mis-pathed entry bug in that file).

**Step 6 — normalize.** `python3 dashboards/misc/cleanup-dash.py <file>` on
every file (CI enforces it). Write `ATTRIBUTION.md`.

## Validation criteria

1. `make -C dashboards build` succeeds; `pmm-app/dist/dashboards/ClickHouse/`
   contains every JSON.
2. `cleanup-dash.py --check-only <file>` returns 0 for all.
3. `yarn lint:check` + `yarn test:ci` pass.
4. Each JSON: valid, unique non-null `uid`, `editable:false`, `refresh:false`,
   `timezone:"browser"`, tags include `ClickHouse`/`Percona`/`Services`, and a
   `description` attribution line.
5. `ATTRIBUTION.md` lists every base dashboard with URL + license + version;
   every base license is confirmed compatible.
6. No panel references `grafana-clickhouse-datasource`; all use the VM
   datasource variable.
7. Every template-variable / panel query references a metric Phase 1 emits.
8. No duplicate `uid` across `dashboards/dashboards/`.

## Integration tests

See [INTEGRATION-TESTS.md](INTEGRATION-TESTS.md) — IT-2.x: plugin-build artifact
contains the ClickHouse dashboards; live provisioning (all load, no datasource
errors); metric-binding (headline panels render under load); template cascade
on single-node and cluster.

## Risks

- **License of the base dashboards** — must be confirmed before incorporating
  `clickhouse-mixin` content; if unclear, the mixin is used only as a reference
  to rebuild equivalent panels. Attribution is mandatory either way.
- **Blocked on the Phase 1 metric contract** — dashboards need the native
  metric families realised; if Phase 1 trims them, trim panels to match.
- **Metric-name parity is mandatory** — native and exporter modes must emit
  identical names, else the reused dashboards work for only one mode.
- **Mixin schema drift** — the mixin targets a recent Grafana; `fix-panels.py`
  + `cleanup-dash.py` must reconcile it with PMM's Grafana version.
