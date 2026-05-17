# Phase 2 — Grafana dashboards for ClickHouse

**Outcome:** a set of ClickHouse Grafana dashboards under
`dashboards/dashboards/ClickHouse/`, registered in the `pmm-app` plugin,
rendering live ClickHouse metrics from VictoriaMetrics — on par with the
MySQL/PostgreSQL dashboard sets.

## Design

- ClickHouse monitoring dashboards bind to the **VictoriaMetrics / Prometheus**
  datasource (the `clickhouse_*` metrics from the Phase 1 exporter) — **not**
  the bundled `grafana-clickhouse-datasource` plugin (that is for ad-hoc direct
  ClickHouse querying, a separate concern).
- Dashboard JSON: one file per dashboard in `dashboards/dashboards/ClickHouse/`,
  each registered in `dashboards/pmm-app/src/plugin.json` `includes`.
- Template variables cascade off a `clickhouse_up` metric:
  `environment → cluster → node_name → service_name` (the Valkey/MySQL pattern).
- Tags on every dashboard: `ClickHouse`, `Percona`, `Services`.
- UIDs are stable lowercase-hyphen, fixed at creation, never changed.

### Metric contract  *(Phase 1 prerequisite)*

Phase 2 cannot author panels against metrics that do not exist. Phase 1's
`clickhouse_exporter` must emit at least:

- `clickhouse_up` (gauge 1/0 — **drives all templating**),
  `clickhouse_version_info{version=...}`.
- Queries: `clickhouse_query_count`, `clickhouse_query_duration_seconds`,
  `clickhouse_select_query_count`, `clickhouse_insert_query_count`,
  `clickhouse_failed_query_count`.
- Memory: `clickhouse_memory_tracking_bytes`, `clickhouse_memory_resident_bytes`,
  `clickhouse_mark_cache_bytes`, `clickhouse_uncompressed_cache_bytes`.
- Parts/tables (labels `database`,`table`): `clickhouse_parts_count`,
  `clickhouse_parts_bytes`, `clickhouse_table_rows`, `clickhouse_active_parts`,
  `clickhouse_merge_count`, `clickhouse_mutation_count`.
- Replication (labels `database`,`table`): `clickhouse_replica_queue_size`,
  `clickhouse_replica_delay_seconds`, `clickhouse_readonly_replica`.
- System: `clickhouse_connections`, `clickhouse_background_pool_tasks`,
  `clickhouse_scrape_duration_seconds`.

All series carry `service_name`, `node_name`, `cluster`, `environment` (VM
external labels). If Phase 1 delivers metrics incrementally, author the
dashboards in the Step order below — each needs only its own subset.

## Development line (ordered)

**Step 0 — scaffold.** Create `dashboards/dashboards/ClickHouse/`. Add a
`ClickHouse` row to `.github/instructions/dashboards.instructions.md`. Fix UIDs
up front: `clickhouse-instance-summary`, `clickhouse-query-performance`,
`clickhouse-memory`, `clickhouse-tables-parts`, `clickhouse-replication`,
`clickhouse-system-resources`, `clickhouse-instances-overview`,
`clickhouse-instances-compare`.

**Step 1 — `ClickHouse_Instance_Summary.json`** *(anchor)* — clone
`PostgreSQL/PostgreSQL_Instance_Summary.json`. Status/uptime, version, QPS,
query latency, memory tracked vs resident, active parts, connections, pool
tasks. Build this first and get the templating/layout reviewed; later
dashboards clone it.

**Step 2 — `ClickHouse_Query_Performance.json`** — SELECT/INSERT/failed query
rates, duration percentiles, slow-query indicators. (Metrics counterpart of QAN.)

**Step 3 — `ClickHouse_Memory.json`** — memory tracking, mark/uncompressed
cache, resident memory, allocator stats.

**Step 4 — `ClickHouse_Tables_and_Parts.json`** — per-`database`/`table` parts
count/size, active vs inactive, rows, merges, mutations. Adds `database`/`table`
multi-value template variables.

**Step 5 — `ClickHouse_Replication.json`** — replica queue size, replication
delay, readonly replicas, Keeper/ZooKeeper session. Cluster-relevant; panels
degrade gracefully (no-data) on single-node.

**Step 6 — `ClickHouse_System_Resources.json`** — background pool tasks,
threads, file descriptors, ClickHouse data-dir disk usage. Link out to OS
dashboards for host-level metrics rather than duplicating.

**Step 7 — `ClickHouse_Instances_Overview.json`** — fleet view, one row per
`service_name` (`repeat`), `$service_name` multi-value/`All`.

**Step 8 — `ClickHouse_Instances_Compare.json`** — side-by-side comparison,
series keyed by `service_name`.

**Step 9 — cross-linking.** Data links: Instance Summary → the drill-down
dashboards; Overview/Compare → Instance Summary.

**Step 10 — registration.** Add 8 `includes` entries to
`dashboards/pmm-app/src/plugin.json` with `path: dashboards/ClickHouse/<File>.json`.
(Note: an existing entry mis-registers a PostgreSQL dashboard under
`dashboards/MongoDB/` — do not copy that mistake.)

**Step 11 — normalize.** Run `python3 dashboards/misc/cleanup-dash.py <file>`
on every file before commit (CI enforces it).

### Per-dashboard workflow

Design in Grafana UI against a live ClickHouse + Phase 1 exporter → export with
`dashboards/misc/export-dash.py` → `convert-dash-to-PMM.py` + `fix-panels.py` →
`cleanup-dash.py` → place in `dashboards/dashboards/ClickHouse/` → register in
`plugin.json` → `make -C dashboards build` to confirm bundling.

## Validation criteria

1. `make -C dashboards build` succeeds; `pmm-app/dist/dashboards/ClickHouse/`
   has all 8 JSONs.
2. `cleanup-dash.py --check-only <file>` returns 0 for every file (CI `check`).
3. `yarn lint:check` + `yarn test:ci` pass.
4. Each JSON: valid, unique non-null `uid`, `editable:false`, `refresh:false`,
   `timezone:"browser"`, tags include `ClickHouse`/`Percona`/`Services`.
5. No panel references `grafana-clickhouse-datasource` — all use the VM datasource var.
6. Every template-variable query references a metric Phase 1 actually emits.
7. No duplicate `uid` across `dashboards/dashboards/`.

## Integration tests

See [INTEGRATION-TESTS.md](INTEGRATION-TESTS.md) — IT-2.x: plugin-build artifact
contains the ClickHouse dashboards; live provisioning (all 8 load, no
datasource errors); metric-binding (headline panels render data under load);
template cascade on single-node and cluster.

## Risks

- **Blocked on the Phase 1 metric contract** — dashboards need real metrics
  incl. `clickhouse_up`. The contract above is the explicit hand-off; if Phase 1
  trims it, trim the dashboards to match.
- `plugin.json` has an existing path bug — ClickHouse entries must use the
  correct `dashboards/ClickHouse/` path.
- ClickHouse's OLAP performance model differs from row-store OLTP; panels must
  reflect ClickHouse reality (parts/merges/memory) — not be a MySQL clone.
