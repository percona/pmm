CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_schema TO dim_values AS
SELECT 'schema' AS dimension, `schema` AS value, toStartOfHour(period_start) AS period_start, sum(num_queries) AS weight
FROM metrics_raw GROUP BY value, period_start;
