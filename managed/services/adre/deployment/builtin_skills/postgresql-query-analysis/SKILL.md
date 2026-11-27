---
name: postgresql-query-analysis
description: PostgreSQL slow statements and plans - QAN top queries by service_id, table DDL and indexes via PMM actions, EXPLAIN (ANALYZE, BUFFERS) for the user to run, CREATE INDEX CONCURRENTLY advice. Use for "why is Postgres slow", seq scans, temp spills, or statement timeouts.
---

# PostgreSQL query analysis

## When to use
Slow PostgreSQL, a QAN `queryid` that needs a plan and an index fix, or statements behind `pmm_postgresql_high_statement_timeouts`.
Lock pile-ups, connections or replication: use **postgresql-incidents**. Dead tuples or bloat as the cause: use **postgresql-maintenance**.

## Steps
1. Scope: `pmm_list_services(service_type=postgresql)` gives `service_id` (UUID, never the name) and `node_id`.
2. QAN top statements: `pmm_clickhouse_query(database=pmm)`. PostgreSQL rows fill `database` (DB name) and `tables`; `schema` is empty, so group by `database`:
   ```sql
   SELECT fingerprint, database, any(queryid) AS queryid, any(tables) AS tables,
          SUM(num_queries) AS calls, SUM(m_query_time_sum) AS total_s,
          SUM(m_query_time_sum) / nullIf(SUM(num_queries), 0) AS avg_s,
          SUM(m_rows_sent_sum) AS rows, SUM(m_shared_blks_read_sum) AS blks_read, SUM(m_shared_blks_hit_sum) AS blks_hit,
          SUM(m_temp_blks_written_sum) AS temp_blks_written, any(example) AS example, any(query_plan) AS plan
   FROM pmm.metrics
   WHERE service_id = '<uuid>' AND period_start >= '<YYYY-MM-DD HH:MM:SS>' AND period_start <= '<YYYY-MM-DD HH:MM:SS>'
   GROUP BY fingerprint, database ORDER BY total_s DESC LIMIT 10
   ```
   Keep the top 3 to 5. With pg_stat_statements, `example` is empty and `fingerprint` carries `$1` placeholders. With pg_stat_monitor, `example` can hold real values and `plan` holds a plan when `pgsm_enable_query_plan` is on. No rows after a successful query means QAN is not enabled for this service: say so, and give the user the pg_stat_statements query from Remediation.
3. Table structure for each table in `tables`, using the QAN `database`:
   - `pmm_db_action(action=postgres_show_create_table, service_id=<id>, table_name=<schema.table>, database=<database>)` returns columns, indexes, FKs and constraints.
   - `pmm_db_action(action=postgres_show_index, service_id=<id>, table_name=<schema.table>, database=<database>)` returns the index definitions.
   Unqualified names search every schema.
4. Context (optional): `pmm_observability_map(engine=postgresql, intent=slow_queries, service_id=<id>)` and `pmm_metrics_snapshot` on its exprs. For the tables: `rate(pg_stat_user_tables_seq_scan{service_id="<id>",relname="<t>"}[5m])` vs the same with `pg_stat_user_tables_idx_scan`, and `rate(pg_stat_database_temp_bytes{service_id="<id>"}[5m])` for spills.
5. Plan: PMM has no PostgreSQL EXPLAIN action. Give the user the command for the top statement, against the QAN `database`:
   ```sql
   EXPLAIN (ANALYZE, BUFFERS, VERBOSE) <statement with real values>;
   -- ANALYZE executes the statement: wrap DML in BEGIN; ... ROLLBACK;
   EXPLAIN (GENERIC_PLAN) <fingerprint with $1 placeholders>;   -- PG16+, plan only, no ANALYZE
   ```
   Ask them to paste the output. Then read it with the Interpretation list.

## Interpretation
- High `total_s` from many fast `calls`: reduce call count (N+1 pattern, caching, batching). A few calls with high `avg_s`: fix the plan.
- `blks_read` large vs `blks_hit`: reads come from disk (cold cache, or a scan of a large table). `temp_blks_written` > 0: sorts or hashes spill to disk.
- In the plan:
  - `Seq Scan` on a large table with a selective `Filter` and big `Rows Removed by Filter`: missing or unusable index. Causes: a function on the column, a type mismatch, or leading `%` in LIKE.
  - Estimated rows far from actual rows: stale statistics, or correlated or skewed columns.
  - `Sort Method: external merge` / `Disk:` / `Batches: > 1`: `work_mem` too small for this query.
  - `Nested Loop` with large `loops` on the inner side: missing index on the join key, or bad estimates.
  - `Buffers: shared read` dominant: an I/O-bound plan. `Heap Fetches` high on an Index Only Scan: the visibility map is stale and the table needs VACUUM.
- Index scan present but slow: low selectivity, wrong column order (equality columns first, then range or sort), or the table is bloated.

## Remediation
Index (online, cannot run inside a transaction block; an invalid index left by a failed run must be dropped):
```sql
CREATE INDEX CONCURRENTLY idx_<table>_<cols> ON <schema>.<table> (<equality_col>, <range_or_sort_col>);
-- partial: ... WHERE status = 'active';  expression: ... ((lower(email)));  covering: ... INCLUDE (<col>);
```
Statistics:
```sql
ANALYZE <schema>.<table>;
ALTER TABLE <schema>.<table> ALTER COLUMN <col> SET STATISTICS 1000; ANALYZE <schema>.<table>;
CREATE STATISTICS <name> (dependencies, ndistinct) ON <col_a>, <col_b> FROM <schema>.<table>; ANALYZE <schema>.<table>;
```
Spills: `SET work_mem = '64MB';` per session or `ALTER ROLE <app> SET work_mem = '64MB';`. Do not raise it globally without sizing (it applies per sort node per session).
Rewrite: show before and after with a one-line reason. Examples: make predicates sargable (`created_at >= ...` instead of `date(created_at) = ...`), use keyset pagination instead of large `OFFSET`, replace `NOT IN (subquery)` with `NOT EXISTS`.
Unused indexes (write overhead) to review before dropping:
```sql
SELECT relname, indexrelname, idx_scan, pg_size_pretty(pg_relation_size(indexrelid)) FROM pg_stat_user_indexes
WHERE idx_scan = 0 ORDER BY pg_relation_size(indexrelid) DESC LIMIT 20;
DROP INDEX CONCURRENTLY <schema>.<index>;
```
No QAN data: pull the top statements straight from pg_stat_statements (PG13+ column names):
```sql
SELECT queryid, calls, round(total_exec_time::numeric, 1) AS total_ms, round(mean_exec_time::numeric, 2) AS mean_ms, rows, left(query, 200)
FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT 10;
```
Environment: `CREATE INDEX` and `ANALYZE` work everywhere, RDS and Cloud SQL included. Server-wide `work_mem` or `default_statistics_target` changes go through `ALTER SYSTEM` (self-managed), the Operator CR (Kubernetes), or the parameter group (managed cloud).
