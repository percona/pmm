// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package otel

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register ClickHouse driver
	"github.com/sirupsen/logrus"
)

// EnsureOtelTracesMetricsAndServiceMapTables creates ClickHouse tables used by the OTEL ClickHouse exporter
// for traces and sum metrics, plus Phase 1 service map rollup targets. DDL aligns with
// opentelemetry-collector-contrib clickhouseexporter (MergeTree, column names).
// Create_schema on the collector remains false; PMM-managed owns DDL.
func EnsureOtelTracesMetricsAndServiceMapTables(ctx context.Context, dsn string, spanRetentionDays, metricRetentionDays int) error {
	if spanRetentionDays <= 0 {
		spanRetentionDays = 7
	}
	if metricRetentionDays <= 0 {
		metricRetentionDays = 90
	}
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	defer db.Close() //nolint:errcheck

	db.SetConnMaxLifetime(0)

	if err := ensureOtelDatabase(ctx, db); err != nil { //nolint:noinlineerr
		return err
	}

	tableEngine := TableEngine()
	tracesDDL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.otel_traces
(
    Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
    TraceId String CODEC(ZSTD(1)),
    SpanId String CODEC(ZSTD(1)),
    ParentSpanId String CODEC(ZSTD(1)),
    TraceState String CODEC(ZSTD(1)),
    SpanName LowCardinality(String) CODEC(ZSTD(1)),
    SpanKind LowCardinality(String) CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    SpanAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    Duration Int64 CODEC(ZSTD(1)),
    StatusCode LowCardinality(String) CODEC(ZSTD(1)),
    StatusMessage String CODEC(ZSTD(1)),
    Events Nested (
        Timestamp DateTime64(9),
        Name LowCardinality(String),
        Attributes Map(LowCardinality(String), String)
    ) CODEC(ZSTD(1)),
    Links Nested (
        TraceId String,
        SpanId String,
        TraceState String,
        Attributes Map(LowCardinality(String), String)
    ) CODEC(ZSTD(1)),
    INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_span_attr_key mapKeys(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_span_attr_value mapValues(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_duration Duration TYPE minmax GRANULARITY 1
) ENGINE = %s
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SpanName, toDateTime(Timestamp), TraceId)
TTL toDateTime(Timestamp) + toIntervalDay(%d)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, tableEngine, spanRetentionDays)

	if _, err := db.ExecContext(ctx, tracesDDL); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.otel_traces: %w", err)
	}
	logrus.Debug("OTEL schema: table otel.otel_traces ensured")

	metricsSumDDL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.otel_metrics_sum
(
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    Value Float64 CODEC(ZSTD(1)),
    Flags UInt32 CODEC(ZSTD(1)),
    Exemplars Nested (
        FilteredAttributes Map(LowCardinality(String), String),
        TimeUnix DateTime64(9),
        Value Float64,
        SpanId String,
        TraceId String
    ) CODEC(ZSTD(1)),
    AggregationTemporality Int32 CODEC(ZSTD(1)),
    IsMonotonic Boolean CODEC(Delta, ZSTD(1)),
    INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_attr_key mapKeys(Attributes) TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE = %s
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
TTL toDateTime(TimeUnix) + toIntervalDay(%d)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, tableEngine, metricRetentionDays)

	if _, err := db.ExecContext(ctx, metricsSumDDL); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.otel_metrics_sum: %w", err)
	}
	logrus.Debug("OTEL schema: table otel.otel_metrics_sum ensured")

	nodesDDL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.service_map_nodes_1m
(
    bucket DateTime,
    id String,
    title String,
    subtitle String,
    mainstat String,
    secondarystat String,
    color String,
    pmm_node_id String,
    pmm_agent_id String
) ENGINE = %s
PARTITION BY toDate(bucket)
ORDER BY (bucket, id)
TTL bucket + toIntervalDay(32)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, tableEngine)

	if _, err := db.ExecContext(ctx, nodesDDL); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.service_map_nodes_1m: %w", err)
	}

	edgesDDL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.service_map_edges_1m
(
    bucket DateTime,
    id String,
    source String,
    target String,
    mainstat String,
    secondarystat String,
    thickness Float64,
    pmm_node_id String
) ENGINE = %s
PARTITION BY toDate(bucket)
ORDER BY (bucket, source, target)
TTL bucket + toIntervalDay(32)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, tableEngine)

	if _, err := db.ExecContext(ctx, edgesDDL); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.service_map_edges_1m: %w", err)
	}

	logrus.Debug("OTEL schema: service map rollup tables ensured")
	return nil
}

// EnsureOtelCorootHelperTables creates Coroot-style helper tables and materialized views for logs/traces facets.
func EnsureOtelCorootHelperTables(ctx context.Context, dsn string, logsRetentionDays, tracesRetentionDays int) error {
	if logsRetentionDays <= 0 {
		logsRetentionDays = 7
	}
	if tracesRetentionDays <= 0 {
		tracesRetentionDays = 7
	}
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	defer db.Close() //nolint:errcheck

	db.SetConnMaxLifetime(0)

	if err := ensureOtelDatabase(ctx, db); err != nil { //nolint:noinlineerr
		return err
	}

	replEngine := ReplacingTableEngine()
	logsHelper := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.logs_service_name_severity_text
(
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    SeverityText LowCardinality(String) CODEC(ZSTD(1)),
    LastSeen DateTime64(9) CODEC(Delta(8), ZSTD(1))
)
ENGINE = %s
PRIMARY KEY (ServiceName, SeverityText)
ORDER BY (ServiceName, SeverityText)
TTL toDateTime(LastSeen) + toIntervalDay(%d)`, replEngine, logsRetentionDays)
	if _, err := db.ExecContext(ctx, logsHelper); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.logs_service_name_severity_text: %w", err)
	}
	logsMV := `CREATE MATERIALIZED VIEW IF NOT EXISTS otel.logs_service_name_severity_text_mv
TO otel.logs_service_name_severity_text
AS
SELECT
    ServiceName,
    SeverityText,
    max(Timestamp) AS LastSeen
FROM otel.logs
GROUP BY ServiceName, SeverityText`
	if _, err := db.ExecContext(ctx, logsMV); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create logs_service_name_severity_text_mv: %w", err)
	}

	traceTSEngine := TableEngine()
	traceTS := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.otel_traces_trace_id_ts
(
    TraceId String CODEC(ZSTD(1)),
    Start DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    End DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    INDEX idx_trace_id TraceId TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = %s
ORDER BY (TraceId, toUnixTimestamp(Start))
TTL toDateTime(Start) + toIntervalDay(%d)
SETTINGS index_granularity = 8192`, traceTSEngine, tracesRetentionDays)
	if _, err := db.ExecContext(ctx, traceTS); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.otel_traces_trace_id_ts: %w", err)
	}
	traceTSMV := `CREATE MATERIALIZED VIEW IF NOT EXISTS otel.otel_traces_trace_id_ts_mv
TO otel.otel_traces_trace_id_ts
AS
SELECT
    TraceId,
    min(Timestamp) AS Start,
    max(Timestamp) AS End
FROM otel.otel_traces
WHERE TraceId != ''
GROUP BY TraceId`
	if _, err := db.ExecContext(ctx, traceTSMV); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel_traces_trace_id_ts_mv: %w", err)
	}

	traceSvc := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.otel_traces_service_name
(
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    LastSeen DateTime64(9) CODEC(Delta(8), ZSTD(1))
)
ENGINE = %s
PRIMARY KEY (ServiceName)
ORDER BY (ServiceName)
TTL toDateTime(LastSeen) + toIntervalDay(%d)`, replEngine, tracesRetentionDays)
	if _, err := db.ExecContext(ctx, traceSvc); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel.otel_traces_service_name: %w", err)
	}
	traceSvcMV := `CREATE MATERIALIZED VIEW IF NOT EXISTS otel.otel_traces_service_name_mv
TO otel.otel_traces_service_name
AS
SELECT
    ServiceName,
    max(Timestamp) AS LastSeen
FROM otel.otel_traces
GROUP BY ServiceName`
	if _, err := db.ExecContext(ctx, traceSvcMV); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create otel_traces_service_name_mv: %w", err)
	}

	logrus.Debug("OTEL schema: coroot helper tables ensured")
	return nil
}
