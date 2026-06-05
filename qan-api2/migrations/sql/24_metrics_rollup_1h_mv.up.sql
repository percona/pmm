-- Materialized view feeding metrics_rollup_1h (created in migration 23).
-- Aggregates raw per-bucket metrics into 1-hour partial aggregate states at
-- insert time. ClickHouse maps a TO-target view to the table BY COLUMN NAME, so
-- the bucket must be named period_start. The hour bucket is computed in the inner
-- subquery (and only grouped in the outer one) so the period_start alias does not
-- shadow the source column inside toStartOfHour().
CREATE MATERIALIZED VIEW metrics_rollup_1h_mv TO metrics_rollup_1h AS
SELECT
  period_start,
  queryid,
  service_name,
  `database`,
  `schema`,
  username,
  service_type,
  environment,
  cluster,
  replication_set,
  fingerprint,
  sumState(num_queries) AS num_queries,
  sumState(m_query_time_cnt) AS m_query_time_cnt,
  sumState(m_query_time_sum) AS m_query_time_sum,
  minState(m_query_time_min) AS m_query_time_min,
  maxState(m_query_time_max) AS m_query_time_max,
  avgState(m_query_time_p99) AS m_query_time_p99,
  sumState(m_lock_time_cnt) AS m_lock_time_cnt,
  sumState(m_lock_time_sum) AS m_lock_time_sum,
  minState(m_lock_time_min) AS m_lock_time_min,
  maxState(m_lock_time_max) AS m_lock_time_max,
  avgState(m_lock_time_p99) AS m_lock_time_p99,
  sumState(m_rows_examined_cnt) AS m_rows_examined_cnt,
  sumState(m_rows_examined_sum) AS m_rows_examined_sum,
  minState(m_rows_examined_min) AS m_rows_examined_min,
  maxState(m_rows_examined_max) AS m_rows_examined_max,
  avgState(m_rows_examined_p99) AS m_rows_examined_p99,
  sumState(m_rows_sent_cnt) AS m_rows_sent_cnt,
  sumState(m_rows_sent_sum) AS m_rows_sent_sum,
  minState(m_rows_sent_min) AS m_rows_sent_min,
  maxState(m_rows_sent_max) AS m_rows_sent_max,
  avgState(m_rows_sent_p99) AS m_rows_sent_p99
FROM
(
  SELECT
    toStartOfHour(period_start) AS period_start,
    queryid,
    service_name,
    `database`,
    `schema`,
    username,
    service_type,
    environment,
    cluster,
    replication_set,
    fingerprint,
    num_queries,
    m_query_time_cnt,
    m_query_time_sum,
    m_query_time_min,
    m_query_time_max,
    m_query_time_p99,
    m_lock_time_cnt,
    m_lock_time_sum,
    m_lock_time_min,
    m_lock_time_max,
    m_lock_time_p99,
    m_rows_examined_cnt,
    m_rows_examined_sum,
    m_rows_examined_min,
    m_rows_examined_max,
    m_rows_examined_p99,
    m_rows_sent_cnt,
    m_rows_sent_sum,
    m_rows_sent_min,
    m_rows_sent_max,
    m_rows_sent_p99
  FROM metrics
)
GROUP BY
  period_start,
  queryid,
  service_name,
  `database`,
  `schema`,
  username,
  service_type,
  environment,
  cluster,
  replication_set,
  fingerprint;
