// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package otel

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register ClickHouse driver
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/envvars"
)

const defaultClickhouseAddr = "127.0.0.1:9000"

// EnsureOtelSchema creates the ClickHouse database `otel` and table `otel.logs` if they do not exist.
// clickhouseAddr should be "host:port" (e.g. "127.0.0.1:9000"). retentionDays is used for the TTL of otel.logs.
func EnsureOtelSchema(ctx context.Context, clickhouseAddr string, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 7
	}
	dsn := fmt.Sprintf("tcp://%s/default", clickhouseAddr)
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	defer db.Close() //nolint:errcheck

	db.SetConnMaxLifetime(0)

	if _, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS otel"); err != nil {
		return fmt.Errorf("create database otel: %w", err)
	}
	logrus.Debug("OTEL schema: database otel ensured")

	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.logs
(
    Timestamp DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    TimestampTime DateTime DEFAULT toDateTime(Timestamp),
    TraceId String CODEC(ZSTD(1)),
    SpanId String CODEC(ZSTD(1)),
    TraceFlags UInt8,
    SeverityText LowCardinality(String) CODEC(ZSTD(1)),
    SeverityNumber UInt8,
    ServiceName LowCardinality(String) CODEC(ZSTD(1)),
    Body String CODEC(ZSTD(1)),
    ResourceSchemaUrl LowCardinality(String) CODEC(ZSTD(1)),
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeSchemaUrl LowCardinality(String) CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion LowCardinality(String) CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    LogAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_body Body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 8
)
ENGINE = MergeTree
PARTITION BY toDate(TimestampTime)
PRIMARY KEY (ServiceName, TimestampTime)
ORDER BY (ServiceName, TimestampTime, Timestamp)
TTL TimestampTime + toIntervalDay(%d)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, retentionDays)

	if _, err := db.ExecContext(ctx, createTable); err != nil {
		return fmt.Errorf("create table otel.logs: %w", err)
	}
	logrus.Debug("OTEL schema: table otel.logs ensured")
	return nil
}

// EnsureOtelSchemaFromEnv creates the OTEL schema using PMM_CLICKHOUSE_ADDR env and settings retention.
// Safe to call when OTEL is disabled (no-op). Logs errors but does not fail the caller.
func EnsureOtelSchemaFromEnv(ctx context.Context, retentionDays int) {
	addr := envvars.GetEnv("PMM_CLICKHOUSE_ADDR", defaultClickhouseAddr)
	if err := EnsureOtelSchema(ctx, addr, retentionDays); err != nil {
		logrus.WithError(err).Warn("Failed to ensure OTEL ClickHouse schema")
	}
}
