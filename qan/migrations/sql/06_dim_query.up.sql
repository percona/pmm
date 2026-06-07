-- dim_query: queryid -> identity (fingerprint, tables, explain fingerprint, seen
-- window). AggregatingMergeTree with anyLast/min/max keeps one row per queryid.
CREATE TABLE IF NOT EXISTS dim_query (
  queryid              LowCardinality(String),
  fingerprint          SimpleAggregateFunction(anyLast, String),
  `tables`             SimpleAggregateFunction(anyLast, Array(String)),
  explain_fingerprint  SimpleAggregateFunction(anyLast, String),
  placeholders_count   SimpleAggregateFunction(anyLast, UInt32),
  first_seen           SimpleAggregateFunction(min, DateTime),
  last_seen            SimpleAggregateFunction(max, DateTime)
) ENGINE = {{ .AggregatingMergeTree }}
ORDER BY queryid
SETTINGS index_granularity = 8192;
