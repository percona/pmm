CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dim_client_host TO dim_values AS
SELECT 'client_host' AS dimension, client_host AS value, toStartOfHour(period_start) AS period_start, sum(num_queries) AS weight
FROM metrics_raw GROUP BY value, period_start;
