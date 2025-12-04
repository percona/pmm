-- OTEL logs tables for collecting logs from PMM clients
-- Supports logs from database servers (MySQL, PostgreSQL, MongoDB, etc.)
-- Designed to match OTEL log data model

-- Main logs table using MergeTree engine for efficient time-series queries
CREATE TABLE IF NOT EXISTS otel_logs
(
    -- Timestamp fields
    Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    ObservedTimestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    
    -- Trace context (for correlation with traces)
    TraceId String CODEC(ZSTD(1)),
    SpanId String CODEC(ZSTD(1)),
    TraceFlags UInt32 CODEC(ZSTD(1)),
    
    -- Severity
    SeverityText LowCardinality(String) CODEC(ZSTD(1)),
    SeverityNumber Int32 CODEC(ZSTD(1)),
    
    -- Log body (the actual log message)
    Body String CODEC(ZSTD(1)),
    
    -- Resource attributes (from the source)
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    
    -- Scope (instrumentation library info)
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    
    -- Log attributes
    LogAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    
    -- PMM-specific fields (extracted from attributes for faster queries)
    ServiceName LowCardinality(String) DEFAULT ResourceAttributes['service.name'] CODEC(ZSTD(1)),
    HostName LowCardinality(String) DEFAULT ResourceAttributes['host.name'] CODEC(ZSTD(1)),
    NodeId String DEFAULT ResourceAttributes['pmm_node_id'] CODEC(ZSTD(1)),
    AgentId String DEFAULT ResourceAttributes['pmm_agent_id'] CODEC(ZSTD(1)),
    
    -- Materialized columns for common queries
    Date Date DEFAULT toDate(Timestamp),
    
    -- Index for efficient queries
    INDEX idx_service_name ServiceName TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_host_name HostName TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_severity SeverityText TYPE set(100) GRANULARITY 1,
    INDEX idx_body Body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
    INDEX idx_node_id NodeId TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(Timestamp)
ORDER BY (ServiceName, HostName, Timestamp)
TTL Timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

-- Aggregation table for real-time log stats by service
CREATE TABLE IF NOT EXISTS logs_by_service_hourly
(
    Hour DateTime CODEC(Delta, ZSTD(1)),
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    HostName LowCardinality(String) CODEC(ZSTD(1)),
    SeverityText LowCardinality(String) CODEC(ZSTD(1)),
    LogCount UInt64 CODEC(ZSTD(1)),
    ErrorCount UInt64 CODEC(ZSTD(1)),
    WarnCount UInt64 CODEC(ZSTD(1))
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(Hour)
ORDER BY (Hour, ServiceName, HostName, SeverityText)
TTL Hour + INTERVAL 90 DAY DELETE;

-- Materialized view for automatic aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS logs_by_service_hourly_mv
TO logs_by_service_hourly
AS SELECT
    toStartOfHour(Timestamp) AS Hour,
    ServiceName,
    HostName,
    SeverityText,
    count() AS LogCount,
    countIf(SeverityText IN ('ERROR', 'FATAL', 'CRITICAL')) AS ErrorCount,
    countIf(SeverityText = 'WARN') AS WarnCount
FROM otel_logs
GROUP BY Hour, ServiceName, HostName, SeverityText;

