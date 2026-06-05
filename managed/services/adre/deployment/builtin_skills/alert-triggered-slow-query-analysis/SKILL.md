---
name: alert-triggered-slow-query-analysis
description: 'Alert-triggered slow query analysis: from queryid get pmm.metrics context,
  run EXPLAIN, show tables/indexes, analyse and give concrete recommendations and
  query rewrites'
---

# Alert-triggered slow query analysis

## Purpose

Use this runbook when an alert provides a **queryid** (e.g. from QAN or a slow-query alert). The goal is to: (1) get context and a slow example from the PMM QAN database (pmm.metrics), (2) run EXPLAIN and inspect tables/indexes via PMM MySQL actions, (3) analyse and optimise the query, and (4) give **concrete recommendations** with exact commands and, when applicable, a **rewritten query** (before/after).

## Prerequisites

- PMM with QAN (ClickHouse) and MySQL agent.
- Alert payload (or user context) that includes **queryid**.
- Obtain **service_id** (UUID) from the pmm-inventory toolset (pmm_list_services). Never use service_name as service_id.
- When using QAN data: use the **schema** column as the **MySQL/Postgres application database name** for EXPLAIN and table tools — **not** as instance scope. Do **not** use `WHERE schema = 'mysql'` to mean “this MySQL service”; use **`service_id`** (from inventory or from rows matching **queryid**) for that. See Holmes **`pmm-clickhouse`** llm_instructions.
- **Hard rule — where `database` comes from:** Pass `database` to PMM MySQL actions **only** from the **`schema`** value returned by your successful `pmm.metrics` query for this investigation. **Never** infer the database from the fingerprint or `example` SQL. In SQL, a **single unqualified** identifier after `FROM` names the **table** (or view), not the database — e.g. `SELECT c FROM sbtest2` means table **`sbtest2`** in whatever **default database** the session used; `sbtest2` is not the database name. Only a **qualified** form `database.table` names both: e.g. **`sbtest.sbtest2`** → database **`sbtest`**, table **`sbtest2`**. (In MySQL, “schema” in metadata terms is the same as **database**; it is not the bare table name after `FROM`.) PMM’s **`schema`** column is that **database**; use it for MySQL `database` tool args — never substitute the unqualified table identifier from the text.
- Use **fingerprint** for filtering by pattern (never filter on **example**). When the same logical pattern could exist in multiple application databases, keep **`schema`** tied to the row you chose (aggregate or drill down with **`fingerprint` + `schema`** together).
- For **period workload / spike context** around the alert window, use the **`general`** skill (observability map + panels + QAN overview) — this skill focuses on a single **queryid** drill-down.

## Steps

### 1. Extract queryid (and optional service/time from alert)

- From the alert or investigation context, note the **queryid** (hash).
- If the alert includes service_id or service_name, use that to resolve **service_id** via pmm_list_services (or pmm_get_service). If not, you will resolve service_id in step 2 from pmm.metrics.

### 2. Get context and a slow example from pmm.metrics

Query the ClickHouse database **pmm.metrics** to get everything needed for this queryid:

- **Required columns**: `queryid`, `schema` (database where the query ran), `fingerprint`, `example` (one example SQL for display).
- **Metrics**: aggregate or pick a slow example — e.g. `m_query_time_sum`, `m_query_time_p99`, `m_rows_examined_sum`, `m_full_scan_sum`, `m_no_index_used_sum`, execution count.
- **Filter**: use `WHERE queryid = '<queryid from alert>'`. Optionally narrow by `period_start` to the alert time range or last 24h to choose a representative slow window.
- **Slow example**: if you need a single “slow” representative row, order by `m_query_time_sum` DESC (or similar) and take one row; note its `schema` and `queryid` (they should match the alert). Use that **schema** as the database name for all subsequent PMM MySQL actions.

Example pattern (aggregated per **queryid**, **schema**, and **fingerprint** — same as grouping top slow patterns by **fingerprint, schema** in workload investigations: avoids mixing metrics or picking the wrong DB when one SQL shape runs against more than one database):

```sql
SELECT
  queryid,
  schema,
  fingerprint,
  any(example) AS example,
  SUM(m_query_time_sum) AS total_time,
  SUM(m_query_time_sum) / COUNT() AS avg_time,
  COUNT() AS exec_count,
  SUM(m_rows_examined_sum) AS rows_examined,
  SUM(m_full_scan_sum) AS full_scan_count,
  SUM(m_no_index_used_sum) AS no_index_used_count
FROM pmm.metrics
WHERE queryid = '<queryid from alert>'
  AND period_start >= now() - INTERVAL 24 HOUR
GROUP BY queryid, schema, fingerprint
ORDER BY total_time DESC
LIMIT 10
```

From the result: keep **queryid**, **schema** (database), **fingerprint**, **example** (to show users the actual SQL), and the metrics. If multiple rows differ by **schema** (same queryid/fingerprint, different application DB), pick the row that matches the alert context or highest **total_time**, and use **that row’s `schema`** for every MySQL action in steps 4–5; do not merge across schemas. If you analyse more than one schema, repeat steps 4–6 per row and label each by **schema**.

### 3. Resolve service_id

- If not already known from the alert, use **pmm_list_services** (optionally filter by service_type=mysql or by name). Match the service that owns this QAN data (e.g. by service_id from pmm.metrics if you selected it in the query).
- Use the **service_id** (UUID) for all PMM MySQL action calls.

### 4. Run EXPLAIN

- Call **pmm_mysql_explain** or **pmm_mysql_explain_json** with:
  - `service_id` from step 3  
  - `query_id` = **queryid** from step 2 (the hash — never use fingerprint or example)  
  - `database` = **schema** from step 2  
- Interpret the plan: note **table** names, **type** (e.g. ALL, index, range), **possible_keys**, **key**, **rows**, **Extra** (e.g. Using filesort, Using temporary). These drive the next step and the final recommendations.

### 5. Get table structure and indexes

- From the EXPLAIN result, list the **tables** involved (the **`table`** column in EXPLAIN output — that is the **table** name inside the database you set in step 2, not the MySQL database name).
- For each table, call:
  - **pmm_mysql_show_create_table** with `service_id`, `table_name` = name from EXPLAIN’s `table` column, and **`database`** = **schema from step 2** (unchanged for all tables in this plan).  
  - **pmm_mysql_show_index** with the same parameters  
- Optionally, for large tables or to justify “full scan” cost: **pmm_mysql_show_table_status** for row counts and index cardinality.

### 6. Analyse and optimise

- Correlate EXPLAIN with table definitions and indexes:
  - Full table scans (type=ALL, or high rows_examined in QAN): consider indexes on WHERE/ORDER BY/GROUP BY/JOIN columns.
  - Filesort or temporary: consider indexes that allow in-order access or avoid temporary tables.
  - No index used (possible_keys=NULL or key=NULL): strong candidate for a new index or query rewrite.
- Form **concrete recommendations**:
  - **Indexes**: give the exact **ALTER TABLE** (or **CREATE INDEX**) statement, e.g.  
    `ALTER TABLE db.table_name ADD INDEX idx_column (column_list);`
  - **Query rewrite**: if the query should be changed, show **before** (original example from step 2) and **after** (new query), with a short explanation (e.g. “avoids full scan by using indexed column”, “removes redundant condition”).
  - **Other**: config changes, partitioning, or “run in maintenance window” only when relevant; keep commands concrete (e.g. full SQL or shell command).

### 7. Output: summary and concrete recommendations

- **Summary**: 1–2 sentences on why the query is slow (e.g. full scan on large table, missing index on X).
- **Evidence**: key lines from EXPLAIN and QAN metrics (rows examined, full scan count, etc.).
- **Recommendations** (bulleted or numbered):
  - Each recommendation must include **exact command(s)** or **full SQL**.
  - If the fix is a **query rewrite**, include:
    - **Before**: the example query (or fingerprint) from step 2.
    - **After**: the new, optimised query.
    - Short rationale (e.g. “uses index on status + created_at”).
- If you used **pmm_mysql_show_table_status**, you can cite table size/rows to justify why an index or rewrite matters.

## Notes

- **query_id**: Always pass the **queryid** (hash) from pmm.metrics to pmm_mysql_explain; never use fingerprint or example.
- **database**: Always use the **schema** column from pmm.metrics as the database name for pmm_mysql_explain, pmm_mysql_show_create_table, pmm_mysql_show_index, and pmm_mysql_show_table_status. Never use a column named `database` from `pmm.metrics` for this; never substitute a **table** name from the SQL text for **schema**.
- **Filtering in pmm.metrics**: When filtering by query pattern, use **fingerprint**; never filter on **example**. If you drill into one aggregated row, scope with **`fingerprint` and `schema`** together so you do not mix databases.
- **Multiple schemas**: If the same queryid appears in more than one schema, either analyse the highest-impact one and state that, or repeat the steps per schema and summarise.

## Optional extensions

- **SHOW TABLE STATUS**: Use **pmm_mysql_show_table_status** for tables that show large row estimates in EXPLAIN; use row count and data length to justify “add index” or “avoid full scan”.
- **Time window**: Prefer the alert’s time range (if provided) when selecting a slow example; otherwise use a recent window (e.g. last 24 hours).
- **Escalation**: If the fix requires schema changes, replication, or application deployment, say so explicitly and suggest involving the DBA or app team; if useful, mention tools like pt-query-digest for deeper analysis.

---

## Interview: questions to refine this runbook

Answer these so we can tailor the runbook to your environment and workflows.

1. **Alert payload**  
   Does the alert include only **queryid**, or also **service_id** / **service_name** / **schema**? If we always have service_id, we can shorten step 3; if we only have queryid, the runbook should keep “resolve service from pmm.metrics or inventory”.

2. **Table names**  
   Do you expect pmm.metrics to expose “tables used” (e.g. from system.query_log) in a column, or is “get table names from EXPLAIN” the only source? If there is a column, we can add it to the step 2 query.

3. **Time range**  
   Should the runbook always use the **alert’s time window** (e.g. from labels) when picking a slow example, or is “last 24h” the default when no window is given?

4. **Multiple schemas**  
   For the same queryid in multiple databases: analyse **one** (e.g. highest load) or **all**? Should the runbook say “analyse one and state which” or “repeat for each schema”?

5. **Output format**  
   Is this runbook for **humans** (ADRE/Holmes report) only, or also for **scripts**? Any required structure (e.g. markdown sections, bullet points, or a specific “Recommendations” block)?

6. **Concrete commands**  
   Should every recommendation that touches schema or config include **exact SQL/shell** (e.g. full `ALTER TABLE ... ADD INDEX`), or are short hints acceptable in some cases?

7. **Query rewrite**  
   When suggesting a rewrite: should we always show **before + after** full query text, or is “describe the change + show only the changed part” enough?

8. **Escalation / tools**  
   When should the runbook say “involve DBA” or “run pt-query-digest / Percona Toolkit”? Any internal links (e.g. to Grafana QAN or PMM docs) to add?

9. **Other ideas**  
   Anything else to add? Examples: check replication lag before recommending heavy indexes; suggest EXPLAIN ANALYZE (if available) for actual row counts; or add a step to validate the suggested index with a dry-run or test query.
