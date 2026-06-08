-- metrics_1h: hourly rollup. AggregatingMergeTree merges rows sharing the ORDER BY
-- key via per-column SimpleAggregateFunction (all plain values, no opaque state ->
-- stable across ClickHouse upgrades). The ORDER BY carries every breakdown
-- dimension so distinct groups are never collapsed.
CREATE TABLE IF NOT EXISTS metrics_1h (
  queryid          LowCardinality(String),
  service_id       LowCardinality(String),
  `database`       LowCardinality(String),
  `schema`         LowCardinality(String),
  cmd_type         LowCardinality(String),
  period_start     DateTime,
  service_name     SimpleAggregateFunction(anyLast, LowCardinality(String)),
  service_type     SimpleAggregateFunction(anyLast, LowCardinality(String)),
  cluster          SimpleAggregateFunction(anyLast, LowCardinality(String)),
  environment      SimpleAggregateFunction(anyLast, LowCardinality(String)),
  replication_set  SimpleAggregateFunction(anyLast, LowCardinality(String)),
  node_name        SimpleAggregateFunction(anyLast, LowCardinality(String)),
  az               SimpleAggregateFunction(anyLast, LowCardinality(String)),
  region           SimpleAggregateFunction(anyLast, LowCardinality(String)),
  container_name   SimpleAggregateFunction(anyLast, LowCardinality(String)),
  labels           SimpleAggregateFunction(anyLast, Map(String, String)),
  num_queries                SimpleAggregateFunction(sum, Float64),
  num_queries_with_errors    SimpleAggregateFunction(sum, Float64),
  num_queries_with_warnings  SimpleAggregateFunction(sum, Float64),
  m_query_time_sum     SimpleAggregateFunction(sum, Float64),
  m_query_time_cnt     SimpleAggregateFunction(sum, UInt64),
  m_query_time_min     SimpleAggregateFunction(min, Float32),
  m_query_time_max     SimpleAggregateFunction(max, Float32),
  m_query_time_sketch  SimpleAggregateFunction(sumMap, Map(UInt16, UInt64)),
  m_lock_time_sum      SimpleAggregateFunction(sum, Float64),
  m_lock_time_cnt      SimpleAggregateFunction(sum, UInt64),
  m_lock_time_min      SimpleAggregateFunction(min, Float32),
  m_lock_time_max      SimpleAggregateFunction(max, Float32),
  m_lock_time_sketch   SimpleAggregateFunction(sumMap, Map(UInt16, UInt64)),
  m_rows_sent_sum      SimpleAggregateFunction(sum, Float64),
  m_rows_sent_cnt      SimpleAggregateFunction(sum, UInt64),
  m_rows_sent_min      SimpleAggregateFunction(min, Float32),
  m_rows_sent_max      SimpleAggregateFunction(max, Float32),
  m_rows_examined_sum  SimpleAggregateFunction(sum, Float64),
  m_rows_examined_cnt  SimpleAggregateFunction(sum, UInt64),
  m_rows_examined_min  SimpleAggregateFunction(min, Float32),
  m_rows_examined_max  SimpleAggregateFunction(max, Float32),
  m_rows_affected_sum  SimpleAggregateFunction(sum, Float64),
  m_rows_affected_cnt  SimpleAggregateFunction(sum, UInt64),
  m_rows_affected_min  SimpleAggregateFunction(min, Float32),
  m_rows_affected_max  SimpleAggregateFunction(max, Float32),
  m_bytes_sent_sum     SimpleAggregateFunction(sum, Float64),
  m_bytes_sent_cnt     SimpleAggregateFunction(sum, UInt64),
  m_bytes_sent_min     SimpleAggregateFunction(min, Float32),
  m_bytes_sent_max     SimpleAggregateFunction(max, Float32),
  m_sum  SimpleAggregateFunction(sumMap, Map(String, Float64)),
  m_cnt  SimpleAggregateFunction(sumMap, Map(String, UInt64))
) ENGINE = {{ .AggregatingMergeTree }}
PARTITION BY toYYYYMMDD(period_start)
ORDER BY (service_id, period_start, `database`, `schema`, cmd_type, queryid)
TTL period_start + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
