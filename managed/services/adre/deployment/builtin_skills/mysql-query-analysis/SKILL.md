---
name: mysql-query-analysis
description: MySQL slow query analysis (no alert template). QAN top queries for a MySQL service or one queryid, EXPLAIN via PMM Actions, SHOW CREATE TABLE / SHOW INDEX, and exact index (ALTER TABLE) or rewrite fixes for slow SELECT, UPDATE or DELETE, full scans, high rows examined.
---

# MySQL slow query analysis (QAN → EXPLAIN → schema → fix)

## When to use
A MySQL query or service is slow, a queryid/fingerprint from QAN or an alert, slow UPDATE/DELETE, full scans or high rows examined. MySQL only: PostgreSQL → **postgresql-query-analysis**, MongoDB → **mongodb-query-analysis**. For a period-wide spike across many metrics start with **general**, then come here for the top queries.

## Steps
1. **Service.** `pmm_list_services(service_type=mysql)` → service_id. Only a queryid known:
   ```sql
   SELECT service_id, service_name, schema, SUM(num_queries) AS calls FROM pmm.metrics
   WHERE queryid = '<queryid>' AND period_start >= now() - INTERVAL 1 DAY
   GROUP BY service_id, service_name, schema
   ```
2. **Top patterns** (`pmm_clickhouse_query`, database=pmm; for one query add `AND queryid = '<queryid>'`):
   ```sql
   SELECT fingerprint, schema, any(queryid) AS queryid, any(example) AS example,
     SUM(num_queries) AS calls, SUM(m_query_time_sum) AS total_s,
     SUM(m_query_time_sum) / SUM(num_queries) AS avg_s, SUM(m_lock_time_sum) AS lock_s,
     SUM(m_rows_examined_sum) / SUM(num_queries) AS examined_per_call,
     SUM(m_rows_sent_sum) / SUM(num_queries) AS sent_per_call, SUM(m_rows_affected_sum) AS affected,
     SUM(m_full_scan_sum) AS full_scans, SUM(m_no_index_used_sum) AS no_index, SUM(m_tmp_disk_tables_sum) AS tmp_disk
   FROM pmm.metrics
   WHERE service_id = '<uuid>' AND period_start >= '<YYYY-MM-DD HH:MM:SS>' AND period_start <= '<YYYY-MM-DD HH:MM:SS>'
   GROUP BY fingerprint, schema ORDER BY total_s DESC LIMIT 5
   ```
   Regression check: rerun for the same hours one day/week earlier; or bucket one fingerprint with `toStartOfInterval(period_start, INTERVAL 5 MINUTE)` to see when avg_s changed.
3. **EXPLAIN** the top 3 by total_s: `pmm_db_action(action="mysql_explain_json", service_id=<uuid>, query_id=<queryid>, database=<schema>)`; `mysql_explain` gives the compact table. `database` is the QAN `schema` of that row, never a name parsed from the SQL (`FROM t` names a table). Known errors:
   - "placeholders count is not correct": QAN holds only a normalized digest (perfschema without examples). Give the user `EXPLAIN FORMAT=TREE <query with real values>;` to run.
   - "exceeded max length and got trimmed": example truncated; raise `--max-query-length` on the service.
   - "database name is not included": schema empty; pass `database` explicitly.
   - Non-DML statements cannot be explained. UPDATE/DELETE/INSERT…SELECT are converted to the equivalent SELECT (`is_dml: true`).
4. **Schema.** For every table in the plan's `table` column: `pmm_db_action(action="mysql_show_create_table", service_id, table_name=<t>, database=<schema>)` and `mysql_show_index`; add `mysql_show_table_status` when size or row count matters (DDL duration, scan cost).
5. **Lock-bound instead of plan-bound** (lock_s is a large share of total_s): `pmm_observability_map(engine=mysql, intent=locks, service_id=<uuid>)` and snapshot its panels; the fix is the blocking transaction, not an index.

## Interpretation
- QAN: examined_per_call ≫ sent_per_call (×100+) = missing or unselective index; calls up with flat avg_s = load (N+1 queries, missing cache), not the plan; avg_s up with flat calls = plan change, data growth or contention; tmp_disk > 0 = GROUP BY/ORDER BY/DISTINCT spilling (often TEXT/BLOB columns); full_scans / no_index > 0 confirm scans.
- EXPLAIN `type`: ALL = full table scan, index = full index scan, range/ref/eq_ref/const = index access. `key` NULL with `possible_keys` NULL = no usable index; possible_keys set but key NULL = optimizer rejected it (low selectivity, stale stats, type/collation mismatch). rows × filtered ≈ rows examined per execution.
- `Extra`: Using filesort = ORDER BY not served by an index; Using temporary = GROUP BY/DISTINCT temp table; Using join buffer = join column not indexed on the inner table; Using index = covering (good). JSON `attached_condition` exposes functions on columns and implicit casts.
- DML: InnoDB locks every row it scans, so an UPDATE/DELETE full scan also causes lock waits and replica lag; index the WHERE columns or batch.
- Index design: equality columns first, then one range column, then ORDER BY columns; extend to cover selected columns when few. Check SHOW INDEX so the new index is not a left prefix of an existing one.

## Remediation
Give exact, schema-qualified SQL; for rewrites show **before** (QAN example) and **after** with a one-line reason. Same SQL on self-managed, k8s and RDS/Aurora.
- **Index**: `ALTER TABLE <schema>.<table> ADD INDEX idx_<cols> (<eq_col>, <range_col>, <order_col>), ALGORITHM=INPLACE, LOCK=NONE;`. Large tables or busy primaries: `pt-online-schema-change --alter "ADD INDEX idx_<cols> (<cols>)" D=<schema>,t=<table> --execute` (or gh-ost); DDL replicates, so watch replica lag; on PXC avoid TOI DDL at peak.
- **Stale statistics** (right index exists, not chosen): `ANALYZE TABLE <schema>.<table>;`.
- **Rewrites**:
  - function on an indexed column: `WHERE DATE(created_at) = '2026-01-01'` → `WHERE created_at >= '2026-01-01' AND created_at < '2026-01-02'`
  - implicit cast: `WHERE phone = 5551234` on a VARCHAR → `WHERE phone = '5551234'`; align charset/collation of join columns
  - `OR` across different columns → `UNION ALL` of two indexed branches
  - deep `LIMIT 100000, 20` → keyset `WHERE id > <last_id> ORDER BY id LIMIT 20`
  - `SELECT *` → only needed columns (enables covering indexes)
  - leading wildcard `LIKE '%abc'` → FULLTEXT index or a reversed/derived column
- **Large UPDATE/DELETE**: batch by primary key, e.g. `DELETE FROM <schema>.<table> WHERE <cond> LIMIT 5000;` in a loop, or `pt-archiver`.
- **Verify**: rerun `pmm_db_action` EXPLAIN after the change and compare avg_s / examined_per_call in QAN before vs after.

## Related skills
general (period-wide workload), mysql-availability (connection pileups), mysql-replication-topology (DDL/DML lag).
