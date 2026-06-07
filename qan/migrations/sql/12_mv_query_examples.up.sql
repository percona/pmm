CREATE MATERIALIZED VIEW IF NOT EXISTS mv_query_examples TO query_examples AS
SELECT queryid, service_id, period_start, example, example_metrics, query_plan, planid, plan_summary, is_truncated, example_type
FROM metrics_raw
WHERE example != '';
