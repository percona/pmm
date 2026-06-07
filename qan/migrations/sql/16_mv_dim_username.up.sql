CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_username TO dim_values AS
SELECT 'username' AS dimension, username AS value, toStartOfHour(period_start) AS period_start, sum(num_queries) AS weight
FROM metrics_raw GROUP BY value, period_start;
