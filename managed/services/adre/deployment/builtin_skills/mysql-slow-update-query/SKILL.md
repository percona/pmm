---
name: mysql-slow-update-query
description: Investigate MySQL slow UPDATE queries using PMM QAN and EXPLAIN
---

# MySQL slow UPDATE query

## Purpose

Use this runbook when investigating slow or problematic UPDATE queries on MySQL services monitored by PMM. It uses QAN (pmm.metrics), inventory, and PMM MySQL action tools.

## Prerequisites

- PMM with QAN (ClickHouse) and MySQL agent.
- Obtain **service_id** (UUID) from the pmm-inventory toolset (pmm_list_services). Never use service_name as service_id.
- When using QAN data: use **service_id** from inventory in **`pmm.metrics` WHERE** to scope the monitored service — **schema** is only the **application database name** inside that instance (for EXPLAIN / `database` tool args), never `schema = 'mysql'` for scoping. Use **fingerprint** for filtering (never filter on example).
- For broader workload context (spike window, correlated metrics/panels), use the **`general`** skill — this skill is a narrow UPDATE/EXPLAIN path.

## Steps

1. **Get service_id**  
   Call `pmm_list_services` (optionally filter by `service_type=mysql`). Use the returned `service_id` (UUID) for all PMM MySQL actions.

2. **Find the query in QAN**  
   Query `pmm.metrics` (ClickHouse). Always **include `fingerprint` in the SELECT** for list/summary queries. When filtering, use **`WHERE fingerprint = '...'`** only; do not use `WHERE example = ...`.  
   From the result, note: `queryid`, `schema` (use this as the database name), `fingerprint`, and any metrics (e.g. m_query_time_sum, m_rows_examined_sum).

3. **Run EXPLAIN**  
   Call `pmm_mysql_explain` (or `pmm_mysql_explain_json`) with:
   - `service_id` from step 1
   - `query_id` = queryid from step 2
   - `database` = value of the **schema** column from step 2

4. **Inspect table and indexes** (if useful)  
   Use `pmm_mysql_show_create_table` and `pmm_mysql_show_index` with the same `service_id` and `database` (schema), and the relevant `table_name`.

5. **Summarize**  
   Combine EXPLAIN output, table structure, and QAN metrics to explain why the UPDATE is slow and suggest fixes (indexes, query changes, etc.).

## Notes

- Always filter on **fingerprint** in pmm.metrics; never on **example**.
- Use **schema** from pmm.metrics as the database name in all PMM MySQL action tools.
