-- dim_values: precomputed per-dimension value list with weights. Backs the
-- Filters panel so it never scans the fact tables (fixes PMM-15113).
CREATE TABLE IF NOT EXISTS dim_values (
  dimension     LowCardinality(String),
  value         String,
  period_start  DateTime,
  weight        Float64
) ENGINE = {{ .SummingMergeTree }}
PARTITION BY toYYYYMMDD(period_start)
ORDER BY (dimension, value, period_start)
TTL period_start + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
