---
name: mongodb-query-analysis
description: MongoDB slow operations and plans - QAN top operations by service_id (docs examined vs returned, COLLSCAN, plan summary), explain("executionStats") for the user to run, ESR-rule index advice. Use for "why is MongoDB slow", scan-heavy load, or query-driven ticket exhaustion.
---

# MongoDB query analysis

## When to use
Slow MongoDB operations, high scanned-to-returned ratios, or ticket and queue alerts that QAN ties to specific operations.
Elections, lag, oplog, disk or cert problems: use **mongodb-incidents**.

## Steps
1. Scope: `pmm_list_services(service_type=mongodb)` gives `service_id` (UUID) and `node_id`. For a replica set, check which member serves the load: the primary, or secondaries for secondary reads.
2. QAN top operations: `pmm_clickhouse_query(database=pmm)`. MongoDB rows fill `database` and `tables` (the collection); `schema` is empty:
   ```sql
   SELECT fingerprint, database, any(tables) AS collection, any(queryid) AS queryid,
          SUM(num_queries) AS ops, SUM(m_query_time_sum) AS total_s,
          SUM(m_query_time_sum) / nullIf(SUM(num_queries), 0) AS avg_s,
          SUM(m_docs_examined_sum) AS docs_examined, SUM(m_keys_examined_sum) AS keys_examined,
          SUM(m_docs_returned_sum) AS returned, SUM(m_full_scan_sum) AS collscans,
          any(plan_summary) AS plan, any(example) AS example
   FROM pmm.metrics
   WHERE service_id = '<uuid>' AND period_start >= '<YYYY-MM-DD HH:MM:SS>' AND period_start <= '<YYYY-MM-DD HH:MM:SS>'
   GROUP BY fingerprint, database ORDER BY total_s DESC LIMIT 10
   ```
   Keep the top 3 to 5. No rows after a successful query means MongoDB QAN is off. It needs the profiler (`db.setProfilingLevel(1, {slowms: 100})` on each database) or the mongolog source. Say so, and use the profiler query in Remediation.
3. Context: `pmm_observability_map(engine=mongodb, intent=latency, service_id=<id>)`, then `pmm_metrics_snapshot` on its exprs. Instance-wide scan efficiency: `rate(mongodb_mongod_metrics_query_executor_total{service_id="<id>",state="scanned_objects"}[5m])` divided by `rate(mongodb_mongod_metrics_document_total{service_id="<id>",state="returned"}[5m])` (panel 63 Query Efficiency on `mongodb-instance-summary`).
4. Plan: there is no PMM MongoDB explain tool. Build the command from `example` (filter, sort, projection) and give it to the user:
   ```javascript
   db.getSiblingDB("<db>").<coll>.find(<filter>).sort(<sort>).explain("executionStats")
   db.getSiblingDB("<db>").<coll>.explain("executionStats").aggregate([<pipeline>])
   db.getSiblingDB("<db>").<coll>.getIndexes()
   db.getSiblingDB("<db>").<coll>.aggregate([{ $indexStats: {} }])
   ```
   Use `explain("allPlansExecution")` when the planner picks badly. Ask them to paste `winningPlan` and the `executionStats` totals.

## Interpretation
- `plan` = `COLLSCAN` or `collscans` > 0: no usable index for the filter. This is the usual top offender.
- `docs_examined / returned` > 10 (or `returned` near 0 with large `docs_examined`): low selectivity. The index covers only part of the predicate, or the filter is on an unindexed field. Updates and deletes show up the same way.
- In explain output the same ratios are `totalDocsExamined` and `totalKeysExamined` vs `nReturned`.
- `keys_examined` far above `returned`: an index is used but its key order is wrong. Usually a range field sits before an equality field, or before the sort field.
- `SORT` stage in the plan (instead of an index-provided order): an in-memory sort. It costs CPU and can fail at the sort memory limit unless `allowDiskUse` is set.
- `FETCH` after `IXSCAN` that filters many documents: add the filtered field to the index. A `PROJECTION_COVERED` plan reads no documents.
- High `ops` with low `avg_s`: an application call-rate problem (N+1, polling), not a plan problem.
- `$regex` without an anchor, `$nin`/`$ne`, `$where`, `$expr` on unindexed fields, and large `$in` lists all defeat or weaken indexes.

## Remediation
Index with the ESR rule: **E**quality fields first, then **S**ort fields (same direction or exact reverse), then **R**ange fields:
```javascript
// filter {tenant_id: X, status: "open", created_at: {$gte: D}} sort {priority: -1}
db.getSiblingDB("<db>").<coll>.createIndex(
  { tenant_id: 1, status: 1, priority: -1, created_at: 1 },
  { name: "tenant_status_prio_created" })
```
- The `background` option is ignored since 4.2. Builds hold an exclusive lock only briefly at start and end, but use I/O. On 4.4+ they run on all data-bearing members at once (`commitQuorum`). For very large collections, schedule off-peak or do a rolling build member by member.
- Partial indexes (`partialFilterExpression`) keep indexes small for skewed predicates. TTL indexes (`expireAfterSeconds`) remove expired documents.
- Before dropping an index you think is unused (`$indexStats` `accesses.ops` = 0 since the last restart), hide it to test safely (4.4+): `db.<coll>.hideIndex("<name>")`, then `db.<coll>.dropIndex("<name>")`. `unhideIndex` rolls it back.
- Stopgap while the index builds: `.hint("<index_name>")` in the application query.
- Rewrite and show before and after: anchor regexes (`/^abc/`), project only the needed fields, paginate on an indexed key instead of large `skip()`, and replace unbounded `$in` with batched queries.

Profiler fallback when QAN is empty (profiling must be on for that database):
```javascript
db.getSiblingDB("<db>").system.profile.find(
  { millis: { $gt: 100 } },
  { ns: 1, op: 1, millis: 1, planSummary: 1, docsExamined: 1, keysExamined: 1, nreturned: 1, command: 1 }
).sort({ ts: -1 }).limit(10)
```
Environment: `createIndex` works on self-managed, Operator (via mongosh) and Atlas deployments; Atlas Performance Advisor also suggests indexes. Profiler or `slowms` defaults belong in mongod.conf (self-managed), the Operator CR (Kubernetes), or the Atlas cluster settings.
