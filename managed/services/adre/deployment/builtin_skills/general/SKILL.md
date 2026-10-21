---
name: general
description: "Workload and time-window analysis for any service or node: 'what happened last night', spikes, slow periods, 'check the workload', explain these graphs. Map, snapshot, QAN top patterns, EXPLAIN only the 1-3 dominant queries. Not for one-line facts or firing alerts."
---

# Workload and time-window analysis

## When to use
Period questions ("last 24h", "last night", "the spike at 14:05", "why was it slow"), workload overviews, "what do these graphs show". Not for single facts (answer with 1-2 calls) or firing alerts (pmm-alert-triage). Deeper work: mysql-query-analysis, postgresql-query-analysis, mongodb-query-analysis; host saturation: node-capacity-saturation.

## Steps
1. **Scope.** Convert the window to absolute UTC once and reuse it in every call. `pmm_list_services` (with `service_type` when the user named an engine) gives `service_id`, `service_name`, `node_id`, `node_name`. Nothing named: list services and keep the ones whose throughput changed in step 3; ask only if that is still ambiguous.
2. **Map.** `pmm_observability_map(engine, intent="workload", service_id)`. Add `engine="node", intent="cpu_memory"` (or `disk_io`) for the host, unless the node is RDS/Azure (cloud-rds-azure). ProxySQL/HAProxy have no route: `pmm_search_dashboards` + `pmm_list_dashboard_panels`.
3. **Snapshot.** `pmm_metrics_snapshot` on the 3-5 panel `expr` that answer the question (throughput, connections/threads, latency, CPU, disk I/O). Read `change_points` and `anomalies` first: they date the event. p95/max far above median = short spike; median shifted = sustained change. Narrow the window to the first change point +/- 30 min for the next steps.
4. **QAN top patterns** (database services): one `pmm_clickhouse_query`, `database=pmm`:
   ```sql
   SELECT fingerprint, any(queryid) AS queryid, schema,
          SUM(num_queries) AS calls,
          SUM(m_query_time_sum) AS total_s,
          SUM(m_query_time_sum) / greatest(SUM(num_queries), 1) AS avg_s,
          SUM(num_queries_with_errors) AS errors
   FROM pmm.metrics
   WHERE service_id = '<uuid>'
     AND period_start >= '<YYYY-MM-DD HH:MM:SS>' AND period_start < '<YYYY-MM-DD HH:MM:SS>'
   GROUP BY fingerprint, schema ORDER BY total_s DESC LIMIT 10
   ```
   Work out each pattern's share of total time. To separate "new or grown" from "always there", run the same query on an equal baseline window (same hours one day or one week earlier) and compare `calls` and `avg_s` per fingerprint. Useful extra columns: MySQL `SUM(m_rows_examined_sum)`, `SUM(m_full_scan_sum)`; PostgreSQL `SUM(m_shared_blks_read_sum)`; MongoDB `SUM(m_docs_examined_sum)`.
5. **EXPLAIN only what dominates**: the 1-3 patterns with >= 20 % of total time, or a clear jump in `calls`/`avg_s` versus baseline.
   - MySQL: `pmm_db_action(action="mysql_explain_json", service_id, query_id=<queryid>, database=<schema>)`, then `mysql_show_create_table` / `mysql_show_index` with `table_name` for the tables in the plan.
   - PostgreSQL / MongoDB: no explain action. Continue in postgresql-query-analysis / mongodb-query-analysis, or give the user `EXPLAIN (ANALYZE, BUFFERS) <query>` / `db.<coll>.find(...).explain("executionStats")`.
   - No pattern dominates: skip EXPLAIN; the problem is load or resources, not one query.
6. **Render** only panels that show the finding (the spike, the saturation, before vs after), usually 1-3: `pmm_render_grafana_panel(dashboard_uid, panel_id, from, to, service_id or node_id)`.
7. **Logs** only when metrics point to errors, restarts or crashes: one `otel` query for the node and window (otel-logs-analysis).

## Interpretation
- Throughput up, latency flat: more demand and the system coped. Throughput flat, latency up: contention (locks, I/O, CPU steal, replication apply). Throughput down, latency up: saturation or blocking; check running threads / active sessions and node CPU and iowait.
- One fingerprint >= 50 % of time: query or plan problem (EXPLAIN it). Many fingerprints slower at once: shared cause such as host, storage, locks, a backup or batch job, failover.
- Step change at an exact minute: deploy, config change or restart. Restart check with an instant query on `mysql_global_status_uptime`, `pg_postmaster_uptime_seconds` or `mongodb_instance_uptime_seconds` (value smaller than the time since the event = restarted).
- Spikes repeating at fixed times: scheduled jobs (backups, cron batches, vacuum, TTL).
- QAN empty while metrics show traffic: the service has no QAN agent (`pmm_list_agents(service_id)`). Say so; never report "no slow queries".
- Gaps in every panel at the same time: collection problem, not workload (exporter-scrape-failures).

## Remediation
Give commands only for the cause found. Query or index fixes come from the EXPLAIN evidence as full `CREATE INDEX` / `ALTER TABLE` statements. Resource limits: node-capacity-saturation. Overlapping scheduled jobs: name the job and the time slot to move it to. For "is this normal" questions, show the baseline comparison from step 4 instead of guessing thresholds.
