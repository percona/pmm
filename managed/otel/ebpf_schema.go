// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

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
// create_schema on the collector remains false; PMM-managed owns DDL.
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

	if _, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS otel"); err != nil {
		return fmt.Errorf("create database otel: %w", err)
	}

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
    Duration UInt64 CODEC(ZSTD(1)),
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
    INDEX idx_span_attr_key mapKeys(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE = MergeTree
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SpanName, toDateTime(Timestamp))
TTL toDateTime(Timestamp) + toIntervalDay(%d)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, spanRetentionDays)

	if _, err := db.ExecContext(ctx, tracesDDL); err != nil {
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
) ENGINE = MergeTree
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
TTL toDateTime(TimeUnix) + toIntervalDay(%d)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, metricRetentionDays)

	if _, err := db.ExecContext(ctx, metricsSumDDL); err != nil {
		return fmt.Errorf("create otel.otel_metrics_sum: %w", err)
	}
	logrus.Debug("OTEL schema: table otel.otel_metrics_sum ensured")

	nodesDDL := `CREATE TABLE IF NOT EXISTS otel.service_map_nodes_1m
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
) ENGINE = MergeTree
PARTITION BY toDate(bucket)
ORDER BY (bucket, id)
TTL bucket + toIntervalDay(32)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`

	if _, err := db.ExecContext(ctx, nodesDDL); err != nil {
		return fmt.Errorf("create otel.service_map_nodes_1m: %w", err)
	}

	edgesDDL := `CREATE TABLE IF NOT EXISTS otel.service_map_edges_1m
(
    bucket DateTime,
    id String,
    source String,
    target String,
    mainstat String,
    secondarystat String,
    thickness Float64,
    pmm_node_id String
) ENGINE = MergeTree
PARTITION BY toDate(bucket)
ORDER BY (bucket, source, target)
TTL bucket + toIntervalDay(32)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`

	if _, err := db.ExecContext(ctx, edgesDDL); err != nil {
		return fmt.Errorf("create otel.service_map_edges_1m: %w", err)
	}

	logrus.Debug("OTEL schema: service map rollup tables ensured")
	return nil
}
