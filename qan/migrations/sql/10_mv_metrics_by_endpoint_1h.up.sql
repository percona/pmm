CREATE MATERIALIZED VIEW IF NOT EXISTS mv_metrics_by_endpoint_1h TO metrics_by_endpoint_1h AS
SELECT
  queryid, service_id, `database`, `schema`, cmd_type, username, client_host,
  toStartOfHour(period_start) AS period_start,
  sum(num_queries) AS num_queries,
  sum(m_query_time_sum) AS m_query_time_sum,
  sum(m_query_time_cnt) AS m_query_time_cnt,
  sumMap(m_query_time_sketch) AS m_query_time_sketch
FROM metrics_raw
GROUP BY queryid, service_id, `database`, `schema`, cmd_type, username, client_host, period_start;
