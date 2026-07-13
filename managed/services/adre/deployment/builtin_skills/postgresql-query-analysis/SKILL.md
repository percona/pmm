---
name: postgresql-query-analysis
description: 'PostgreSQL slow query and query plan analysis: QAN top statements from
  pmm.metrics by service_id, EXPLAIN (ANALYZE, BUFFERS) via psql (no PMM Postgres
  explain action), pg_stat_statements, index/bloat inspection (pg_stat_user_tables,
  pg_stat_user_indexes), CREATE INDEX CONCURRENTLY and rewrite recommendations. Use
  for Percona_MS_PostgresqlSlowQueries or ''why is PostgreSQL slow''.'
---

# PostgreSQL query analysis (slow queries, plans, indexes)

## Goal

Find and explain **slow / expensive PostgreSQL queries** and give concrete index or rewrite fixes. PMM QAN (`pmm.metrics`) supplies the top offenders; PostgreSQL plan/index depth comes from **operator `psql`** because PMM has **no PostgreSQL EXPLAIN action** (unlike MySQL). This is the PostgreSQL counterpart to the MySQL EXPLAIN drill-down in the **general** skill (Step 3b).

## When to use

- "Why is Postgres slow?", a `Percona_MS_PostgresqlSlowQueries` alert, or a workload/spike investigation that found a Postgres service.
- A specific `queryid` from QAN that needs a plan and index recommendation.

For broad "what happened in this window" analysis (panels + snapshot + multi-engine), start from the **general** skill and come here for the Postgres query drill-down.

## Prerequisites

- PMM with **QAN enabled for PostgreSQL** (`pg_stat_statements`, or `pg_stat_monitor` for richer data). Without it, `pmm.metrics` has no Postgres rows — fall back to `pg_stat_statements` directly via `psql` (Step 3 alt).
- **`service_id`** (UUID) from inventory (`pmm_list_services`). Never use `service_name` as `service_id`.
- QAN **`schema`** column = the **PostgreSQL database name** the statement ran in. Use it as the `dbname` for `psql` / EXPLAIN — not as instance scope. Scope an instance by **`service_id`**, never `WHERE schema = 'postgres'`.
- Determine the **environment** (self-managed / operator / managed cloud) — it decides whether `psql` runs via host, `kubectl exec`, or a cloud client, and whether superuser is available. See the environment matrix below.

## Steps

### 1. Scope

- **pmm-inventory** → **`service_id`**, **`node_id`**, **`service_name`**, address, version, and `service_type=postgresql`. Note cloud markers (`*.rds.amazonaws.com`, Cloud SQL, operator pod) for environment routing.

### 2. QAN — top slow statements (PMM, works in every environment)

Query **`pmm.metrics`** for the analysis window, **filtered by `service_id`**, grouped by **`fingerprint`** and **`schema`**. Use **ClickHouse DateTime** bounds on `period_start` (`'YYYY-MM-DD HH:MM:SS'`).

```sql
SELECT
  fingerprint,
  any(queryid) AS queryid,
  schema,
  any(example) AS example,
  SUM(m_query_time_sum) AS total_time,
  SUM(m_query_time_sum) / COUNT() AS avg_time,
  COUNT() AS exec_count,
  SUM(m_rows_sent_sum) AS rows_sent
FROM pmm.metrics
WHERE service_id = '<service-uuid>'
  AND period_start >= '<window-start-CH-datetime>'
  AND period_start <= '<window-end-CH-datetime>'
GROUP BY fingerprint, schema
ORDER BY total_time DESC
LIMIT 5
```

- Keep the top 3–5 `(queryid, schema, fingerprint, example)`. **Never filter on `example`**; use `fingerprint`.
- Postgres QAN does **not** reliably populate MySQL-style `m_full_scan` — treat **EXPLAIN (Step 3)** as the authoritative scan/cost evidence for Postgres.
- If QAN returns no rows after a **successful** query, say so and use Step 3 alt (`pg_stat_statements`).

### 3. Plan analysis — operator `psql` (no PMM Postgres EXPLAIN action)

For each top statement, run EXPLAIN against the `schema` database. **Operator step** — route per environment (host `psql`, `kubectl exec ... -- psql`, or cloud client; all use the same SQL):

```sql
-- Use the example SQL from QAN; ANALYZE actually executes it, so prefer a read-only/representative one
EXPLAIN (ANALYZE, BUFFERS, VERBOSE, FORMAT TEXT)
<the example query from Step 2>;
```

Read the plan for:
- **Seq Scan** on a large table with a selective filter → missing/!used index.
- **Rows Removed by Filter** ≫ rows returned → poor selectivity / missing index.
- **Sort** or **Hash** with `external merge Disk: …` → under-sized `work_mem`.
- **Nested Loop** with high loops × rows → join/index problem or bad row estimates (stale stats).
- Estimated vs **actual** rows far apart → run `ANALYZE <table>;` (stale planner stats).

**Step 3 alt — no DB shell or QAN empty:** pull top statements straight from the view:

```sql
SELECT queryid, calls, round(total_exec_time::numeric, 1) AS total_ms,
       round(mean_exec_time::numeric, 2) AS mean_ms, rows,
       left(query, 200) AS query
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;
```

### 4. Index and table inspection

```sql
-- structure + existing indexes for a table in the plan
\d+ schema.table_name
SELECT indexname, indexdef FROM pg_indexes WHERE tablename = '<table>';

-- scan behavior: seq vs index, live/dead tuples (bloat/vacuum signal)
SELECT relname, seq_scan, seq_tup_read, idx_scan, n_live_tup, n_dead_tup, last_autovacuum, last_autoanalyze
FROM pg_stat_user_tables
WHERE relname = '<table>';

-- unused indexes (write overhead, drop candidates) — review before dropping
SELECT relname, indexrelname, idx_scan, pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;
```

High `n_dead_tup` / old `last_autovacuum` points at bloat or autovacuum lag — pair with **postgresql-incidents** if vacuum is the root cause.

### 5. Recommendations (exact commands)

- **Add index** — non-blocking on a live system:
  ```sql
  CREATE INDEX CONCURRENTLY idx_<table>_<cols> ON schema.table_name (col_a, col_b);
  ```
  Use **partial** (`WHERE status = 'active'`) or **expression** (`((data->>'k'))`) indexes when the filter warrants it. `CONCURRENTLY` cannot run inside a transaction block.
- **Query rewrite** — show **before** (QAN example) and **after**, with a one-line rationale (e.g. "sargable predicate lets the planner use idx_orders_created_at").
- **Planner stats** — `ANALYZE schema.table_name;` when estimates were wrong; raise `default_statistics_target` for skewed columns.
- **Sorts spilling to disk** — raise `work_mem` for the session/role (not globally without sizing).
- **Drop** confirmed-unused indexes (`idx_scan = 0`) to cut write amplification — with customer approval.
- **Environment note:** `CREATE INDEX` is allowed on RDS/Aurora/Cloud SQL (no superuser needed); `work_mem`/`default_statistics_target` are **parameter-group / CR** changes on managed-cloud and operator deployments, not `ALTER SYSTEM` on the host.

## Synthesize findings

Name the dominant cost driver (seq scan, sort spill, bad estimate, lock wait) for the top 1–3 statements, cite the EXPLAIN line and QAN total time, and give the exact index/rewrite. If the slowness is contention or vacuum rather than the plan, hand off to **postgresql-incidents**.

### Environment availability matrix (shared)

Before prescribing a command, decide **which surfaces this deployment exposes**. PMM tools (inventory, observability map, metrics snapshot, QAN on `pmm.metrics`, `otel.logs`, Grafana render) are available in **every** environment — always try those first. Host shell, DB superuser, and config-file edits are **not** universal.

| Surface | Self-managed (VM / bare-metal / Docker) | Kubernetes / Operator (PSMDB, PXC, PG operator) | Managed cloud (RDS / Aurora / Cloud SQL / Atlas) |
|---|---|---|---|
| PMM metrics / QAN / observability map | ✅ | ✅ | ✅ (exporter may be **remote/cloud-mode**) |
| `otel.logs` (PMM-collected) | ✅ | ✅ | ⚠️ partial — DB logs often only in the cloud console |
| Host shell (`journalctl`, `grep /var/log/...`, `openssl`, `df`) | ✅ operator | ⚠️ via `kubectl exec` into the pod, not the node | ❌ none |
| DB shell (`psql`, `mongosh`, `mysql`) | ✅ operator | ✅ operator via `kubectl exec` | ✅ but **no superuser** (rds_superuser / atlasAdmin only) |
| `ALTER SYSTEM` / `mongod.conf` / `my.cnf` edits | ✅ | ❌ — change the **Custom Resource**, not the file (operator reverts file edits) | ❌ — change a **parameter group / cluster config**, then reboot/apply |
| `pg_terminate_backend`, `KILL`, force-primary, oplog resize | ✅ with approval | ✅ with approval | ⚠️ often restricted or wrapped by a cloud API |

**Routing rules:**

- **Always lead with PMM/QAN/`otel.logs`.** They work everywhere and need no host/DB access.
- **Gate host-shell and superuser steps.** Label them **"operator, self-managed / k8s only"**. On managed cloud, **skip** them and use the cloud-native equivalent (console logs, Performance Insights / Cloud Monitoring, parameter groups, provider CLI).
- **Config changes:** on **operator** deployments edit the **CR** (e.g. `PerconaServerMongoDB`, `PerconaXtraDBCluster`, `PerconaPGCluster`) and let the operator roll it out; on **managed cloud** edit the **parameter group / flag** and apply per the provider; only on **self-managed** edit the file directly and reload/restart.
- **Detect the environment** from inventory (`service_type`, `node_model`/cloud labels, address like `*.rds.amazonaws.com`, `*.mongodb.net`) before recommending a step. If unknown, state the assumption and give the **PMM-only** path plus the operator path as an option.
- **Never tell the user to run a command their environment can't execute** (e.g. `journalctl` on RDS). Offer the reachable alternative instead.
### PMM observability and panel embed rules (shared)

**Evidence hierarchy:** inventory → **`pmm_observability_map`** → **`pmm_metrics_snapshot`** on panel `expr` → **`pmm_render_grafana_panel`** (embed PNGs, best-effort) → QAN + MySQL EXPLAIN → scoped fallbacks (`pmm_discover_series_labels`, `pmm_list_metric_names` prefix max 50) → **`execute_prometheus_range_query`** at most one per turn, last resort.

**Render failures are non-blocking:** 502, timeout, blank PNG, or curl error on render must **never** skip snapshot, QAN, EXPLAIN, or final synthesis. State **`rendered M/N`**; deliver analysis from snapshot + QAN even when **`rendered 0/N`**.

**FORBIDDEN:** unfiltered `GET /api/v1/label/__name__/values`, full dashboard JSON to the LLM, guessing panel IDs or metric names.

**Observability map:** `GET /v1/grafana/observability-map` via **`pmm_observability_map`**. Pass **`engine`** (`mysql`, `postgresql`, `mongodb`, `valkey`, `node`), **`intent`**, **`service_id`**. Use returned **`primary.dashboard_uid`**, **`panels[].id`**, **`panels[].expr`**. Fall back to **`pmm_list_dashboard_panels`** only when the map warns or intent is unknown (e.g. ProxySQL — no map route yet).

**Metrics snapshot:** **`pmm_metrics_snapshot`** on panel `expr` — returns server-computed **stats** (min/max/mean/median/p25/p75/p95/p99), **change_points**, **anomalies**; not raw matrices. Run **immediately after map** — does not require render. `start`/`end`: RFC3339 or Unix (not `now-24h`). Requires ADRE enabled.

**Render panels (best-effort when this skill uses Grafana):**
1. Inventory first: **`service_id`**, **`node_id`**, **`agent_id`**, **`service_name`**, **`node_name`**, **`version`** → pass all as `var_*` overrides to **`pmm_render_grafana_panel`**. Missing **`agent_id`** often causes blank/timeout renders.
2. Tool: **`POST /v1/grafana/render/resolve`** only (via **`pmm_render_grafana_panel`**).
3. Render **sequentially**; `from`/`to` without quotes (e.g. `now-6h`, `now`). Retry once with shorter window or fixed vars on failure — then **continue snapshot/QAN/EXPLAIN**.
4. **Embed every successful render** in the user-visible answer:
   ```markdown
   ![Panel title](/v1/grafana/render/blob/{hash}.png)
   [Open in Grafana](dashboard_url)
   ```
   Use **`image_url`** and **`dashboard_url` exactly** from tool JSON. Do not skip images for successful renders; do not use `dashboard_url` as img src, do not rebuild URLs.
5. State **`rendered M/N`** if any panel failed; **do not abort** the investigation.

**Scoped metric fallback (when map warns):** **`pmm_discover_series_labels`** with `service_id` + **`metric_prefix`** from map fallback, or **`pmm_list_metric_names`** with prefix only (max 50).

**QAN (database services):** For workload-correlated incidents, query **`pmm.metrics`** top slow patterns by **`service_id`** — see **`general`** skill for query shape and MySQL EXPLAIN Step 3b when applicable. **Always run** even when render fails.
## Output

Deliver only the analysis in the standard ADRE shape: short **Summary**, **Key findings** (top statements with plan/index facts — access path, rows, scan vs index), **Evidence** (QAN table + EXPLAIN in fenced `text` blocks + DDL in `sql`), **Recommendations** (numbered, full SQL). No skill names, no progress narration.
