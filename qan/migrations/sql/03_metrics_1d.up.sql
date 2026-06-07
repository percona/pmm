-- metrics_1d: daily rollup, identical structure to metrics_1h, longer retention.
CREATE TABLE IF NOT EXISTS metrics_1d AS metrics_1h
ENGINE = {{ .AggregatingMergeTree }}
PARTITION BY toYYYYMM(period_start)
ORDER BY (service_id, period_start, `database`, `schema`, cmd_type, queryid)
TTL period_start + INTERVAL 730 DAY
SETTINGS index_granularity = 8192;
