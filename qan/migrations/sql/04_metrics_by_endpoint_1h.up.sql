-- metrics_by_endpoint_1h: drill-down rollup keeping the high-cardinality
-- username/client_host dimensions that the base grain excludes. Shorter TTL.
CREATE TABLE IF NOT EXISTS metrics_by_endpoint_1h (
  queryid          LowCardinality(String),
  service_id       LowCardinality(String),
  `database`       LowCardinality(String),
  `schema`         LowCardinality(String),
  cmd_type         LowCardinality(String),
  username         LowCardinality(String),
  client_host      LowCardinality(String),
  period_start     DateTime,
  num_queries          SimpleAggregateFunction(sum, Float64),
  m_query_time_sum     SimpleAggregateFunction(sum, Float64),
  m_query_time_cnt     SimpleAggregateFunction(sum, UInt64),
  m_query_time_sketch  SimpleAggregateFunction(sumMap, Map(UInt16, UInt64))
) ENGINE = {{ .AggregatingMergeTree }}
PARTITION BY toYYYYMMDD(period_start)
ORDER BY (service_id, period_start, `database`, `schema`, cmd_type, username, client_host, queryid)
TTL period_start + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
