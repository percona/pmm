-- query_examples: bulky example/plan strings kept out of the hot path, own TTL.
CREATE TABLE IF NOT EXISTS query_examples (
  queryid          LowCardinality(String),
  service_id       LowCardinality(String),
  period_start     DateTime,
  example          String,
  example_metrics  String,
  query_plan       String,
  planid           String,
  plan_summary     String,
  is_truncated     UInt8,
  example_type     LowCardinality(String)
) ENGINE = {{ .MergeTree }}
PARTITION BY toYYYYMMDD(period_start)
ORDER BY (queryid, service_id, period_start)
TTL period_start + INTERVAL 8 DAY
SETTINGS index_granularity = 8192;
