-- 1-hour rollup of the hot-path report metrics (query_time, lock_time,
-- rows_examined, rows_sent, num_queries), keyed by queryid + commonly-filtered
-- low-cardinality dimensions. Lets wide-range GROUP BY queryid reports read
-- pre-aggregated rows instead of re-scanning raw per-minute buckets.
-- The materialized view that feeds this table is added in migration 24.
CREATE TABLE metrics_rollup_1h (
  `period_start` DateTime COMMENT 'Start of the 1-hour rollup bucket',
  `queryid` LowCardinality(String),
  `service_name` LowCardinality(String),
  `database` LowCardinality(String),
  `schema` LowCardinality(String),
  `username` LowCardinality(String),
  `service_type` LowCardinality(String),
  `environment` LowCardinality(String),
  `cluster` LowCardinality(String),
  `replication_set` LowCardinality(String),
  `fingerprint` LowCardinality(String),
  `num_queries` AggregateFunction(sum, Float32),
  `m_query_time_cnt` AggregateFunction(sum, Float32),
  `m_query_time_sum` AggregateFunction(sum, Float32),
  `m_query_time_min` AggregateFunction(min, Float32),
  `m_query_time_max` AggregateFunction(max, Float32),
  `m_query_time_p99` AggregateFunction(avg, Float32),
  `m_lock_time_cnt` AggregateFunction(sum, Float32),
  `m_lock_time_sum` AggregateFunction(sum, Float32),
  `m_lock_time_min` AggregateFunction(min, Float32),
  `m_lock_time_max` AggregateFunction(max, Float32),
  `m_lock_time_p99` AggregateFunction(avg, Float32),
  `m_rows_examined_cnt` AggregateFunction(sum, Float32),
  `m_rows_examined_sum` AggregateFunction(sum, Float32),
  `m_rows_examined_min` AggregateFunction(min, Float32),
  `m_rows_examined_max` AggregateFunction(max, Float32),
  `m_rows_examined_p99` AggregateFunction(avg, Float32),
  `m_rows_sent_cnt` AggregateFunction(sum, Float32),
  `m_rows_sent_sum` AggregateFunction(sum, Float32),
  `m_rows_sent_min` AggregateFunction(min, Float32),
  `m_rows_sent_max` AggregateFunction(max, Float32),
  `m_rows_sent_p99` AggregateFunction(avg, Float32)
) ENGINE = {{ .aggregatingEngine }} PARTITION BY toYYYYMMDD(period_start)
ORDER BY
  (
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
    period_start
  );
