CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_cmd_type TO dim_values AS
SELECT 'cmd_type' AS dimension, cmd_type AS value, toStartOfHour(period_start) AS period_start, sum(num_queries) AS weight
FROM metrics_raw GROUP BY value, period_start;
