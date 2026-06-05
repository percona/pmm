---
name: mongodb-query-analysis
description: 'MongoDB slow operation and query plan analysis: QAN top ops from pmm.metrics
  by service_id, explain(executionStats) via pmm_mongodb_explain action or mongosh,
  COLLSCAN and docsExamined-vs-nReturned detection, $indexStats, profiler system.profile,
  createIndex (ESR rule) and rewrite recommendations. Use for slow MongoDB queries
  or query-load-correlated Opcounters/ReadWriteQueue alerts.'
---

# MongoDB query analysis (slow ops, plans, indexes)

## Goal

Find **slow / expensive MongoDB operations** and give concrete index or query fixes. PMM QAN (`pmm.metrics`) supplies the top offenders; plan depth comes from **`explain("executionStats")`** — via the PMM MongoDB explain action when the Holmes build exposes it, otherwise **operator `mongosh`**. This is the MongoDB counterpart to the MySQL EXPLAIN drill-down in the **general** skill (Step 3b).

## When to use

- "Why is Mongo slow?", high `Percona_MS_MongoDBOpcounters` / `ReadWriteQueueHigh` correlated with query load, or a workload/spike investigation that found a MongoDB service.
- A specific slow operation pattern from QAN that needs a plan and index recommendation.

For replica-set / WiredTiger / oplog / sharding incidents use **mongodb-incidents**; come here for the **query** drill-down.

## Prerequisites

- PMM with **QAN enabled for MongoDB** (database profiler at level 1/2, or `slowms` threshold). Without profiling, `pmm.metrics` has no Mongo rows — fall back to the profiler collection (Step 3 alt).
- **`service_id`** (UUID) from inventory (`pmm_list_services`). Never use `service_name` as `service_id`.
- QAN **`schema`** identifies the **database** (and namespace) the op ran in. Scope an instance by **`service_id`**, never by namespace text.
- Determine the **environment** (self-managed / operator / Atlas) — it decides whether `mongosh` runs via host, `kubectl exec`, or an Atlas connection string, and whether `system.profile` / host access exist. See the environment matrix below.

## Steps

### 1. Scope

- **pmm-inventory** → **`service_id`**, **`node_id`**, **`service_name`**, replica-set name, address, version. Note Atlas (`*.mongodb.net`) or operator pod for environment routing.

### 2. QAN — top slow operations (PMM, works in every environment)

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
  SUM(m_docs_scanned_sum) AS docs_scanned,
  SUM(m_docs_returned_sum) AS docs_returned
FROM pmm.metrics
WHERE service_id = '<service-uuid>'
  AND period_start >= '<window-start-CH-datetime>'
  AND period_start <= '<window-end-CH-datetime>'
GROUP BY fingerprint, schema
ORDER BY total_time DESC
LIMIT 5
```

- Keep the top 3–5 `(fingerprint, schema, example)`. A high **`docs_scanned` / `docs_returned`** ratio is the Mongo "full scan" signal (analogous to MySQL rows-examined vs rows-sent). Some columns may be absent depending on profiler/exporter version — confirm before relying on them.
- **Never filter on `example`**; use `fingerprint`. If QAN returns no rows after a **successful** query, use Step 3 alt.

### 3. Plan analysis — `explain("executionStats")`

For each top operation, run explain against the database/collection from `schema`. Prefer the **`pmm_mongodb_explain`** action if your Holmes deployment exposes it (PMM has a MongoDB explain action); otherwise **operator `mongosh`** (host / `kubectl exec` / Atlas client — same commands):

```javascript
// find
db.getSiblingDB("<db>").<collection>.find(<filter>).sort(<sort>).explain("executionStats")
// aggregate
db.getSiblingDB("<db>").<collection>.explain("executionStats").aggregate([ <pipeline> ])
```

Read `executionStats` for:
- **`stage: "COLLSCAN"`** (no `IXSCAN`) → missing/unused index. This is the primary offender.
- **`totalDocsExamined` ≫ `nReturned`** → low selectivity; index not covering the predicate.
- **`totalKeysExamined` ≫ `nReturned`** → index chosen but not selective (wrong key order).
- **`SORT` stage** with no supporting index → in-memory sort (can hit the 32 MB limit and fail).
- **`executionTimeMillis`** and `rejectedPlans` → planner picking a poor plan; consider a `hint` or better index.

**Step 3 alt — no explain access or QAN empty:** read the profiler directly:

```javascript
db.getSiblingDB("<db>").system.profile.find(
  { millis: { $gt: 100 } }
).sort({ ts: -1 }).limit(10)
```

### 4. Index inspection

```javascript
// existing indexes
db.getSiblingDB("<db>").<collection>.getIndexes()

// index usage since last restart — accesses.ops == 0 means unused (drop candidate)
db.getSiblingDB("<db>").<collection>.aggregate([ { $indexStats: {} } ])
```

Unused indexes add write and storage cost; long-zero `accesses.ops` are drop candidates (review before dropping).

### 5. Recommendations (exact commands)

- **Add index** following the **ESR rule** (Equality fields, then Sort fields, then Range fields) so one index serves filter + sort:
  ```javascript
  db.getSiblingDB("<db>").<collection>.createIndex(
    { tenant_id: 1, created_at: -1 },
    { background: true, name: "tenant_created_idx" }
  )
  ```
  Use **partial** (`partialFilterExpression`) or **TTL** indexes when the access pattern warrants. On large collections schedule a **rolling** build per deployment.
- **Query rewrite** — show **before** (QAN example) and **after**: avoid unanchored `$regex`, unbounded `$in`, or `$where`; project only needed fields; ensure sort fields are index-aligned.
- **Force a plan** with `.hint("index_name")` only as a stopgap while the right index is built.
- **Drop** confirmed-unused indexes (`$indexStats` `accesses.ops = 0`) with customer approval.
- **Environment note:** `createIndex` works on self-managed, operator (via `mongosh`), and **Atlas** (or use Atlas **Performance Advisor**, which suggests indexes from the same profiler data). Server tuning (`slowms`, cache size) is a **CR** change on operator and a **cluster config** change on Atlas — not a host edit.

## Synthesize findings

Name the dominant cost driver (COLLSCAN, low-selectivity index, in-memory sort, fan-out `$in`) for the top 1–3 operations, cite the `executionStats` numbers and QAN total time, and give the exact `createIndex` / rewrite. If slowness is WiredTiger/replication/oplog rather than the query, hand off to **mongodb-incidents**.

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

Deliver only the analysis in the standard ADRE shape: short **Summary**, **Key findings** (top ops with plan facts — stage, docsExamined vs nReturned, index used), **Evidence** (QAN table + `explain` output in fenced `text` blocks + index defs in `javascript`), **Recommendations** (numbered, full commands). No skill names, no progress narration.
