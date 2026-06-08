CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_database TO dim_values AS
SELECT 'database' AS dimension, `database` AS value, toStartOfHour(period_start) AS period_start, sum(num_queries) AS weight
FROM metrics_raw GROUP BY value, period_start;
