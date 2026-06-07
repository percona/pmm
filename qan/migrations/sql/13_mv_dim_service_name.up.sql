-- One small MV per filterable dimension, all feeding dim_values.
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_service_name TO dim_values AS
SELECT 'service_name' AS dimension, service_name AS value, toStartOfHour(period_start) AS period_start, sum(num_queries) AS weight
FROM metrics_raw GROUP BY value, period_start;
