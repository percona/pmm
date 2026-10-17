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

// Package otel contains helpers for managing the ClickHouse schema used by OTEL logs.
package otel

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register ClickHouse driver
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/envvars"
)

const (
	defaultClickhouseAddr     = "127.0.0.1:9000"
	defaultClickhouseUser     = "default"
	defaultClickhousePassword = "clickhouse"
)

// EnsureOtelSchema creates the ClickHouse database `otel` and table `otel.logs` if they do not exist.
// DSN must be a ClickHouse connection string (e.g. "tcp://user:password@host:port/default"). RetentionDays is used for the TTL of otel.logs.
func EnsureOtelSchema(ctx context.Context, dsn string, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 7
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
	logrus.Debug("OTEL schema: database otel ensured")

	tableEngine := TableEngine()
	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS otel.logs
(
    Timestamp DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    TimestampTime DateTime DEFAULT toDateTime(Timestamp),
    TraceId String CODEC(ZSTD(1)),
    SpanId String CODEC(ZSTD(1)),
    TraceFlags UInt32 CODEC(ZSTD(1)),
    SeverityText LowCardinality(String) CODEC(ZSTD(1)),
    SeverityNumber Int32 CODEC(ZSTD(1)),
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
    INDEX idx_body Body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 8,
    INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_log_attr_key mapKeys(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_log_attr_value mapValues(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = %s
PARTITION BY toDate(TimestampTime)
PRIMARY KEY (ServiceName, TimestampTime)
ORDER BY (ServiceName, TimestampTime, Timestamp, TraceId)
TTL TimestampTime + toIntervalDay(%d)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1`, tableEngine, retentionDays)

	if _, err := db.ExecContext(ctx, createTable); err != nil { //nolint:noinlineerr
		return fmt.Errorf("create table otel.logs: %w", err)
	}
	logrus.Debug("OTEL schema: table otel.logs ensured")
	return nil
}

// EnsureOtelSchemaFromEnv creates the OTEL schema using PMM ClickHouse env vars (same as pmm-managed main).
// Uses PMM_CLICKHOUSE_ADDR, PMM_CLICKHOUSE_USER, PMM_CLICKHOUSE_PASSWORD. Safe to call when OTEL is disabled (no-op). Logs errors but does not fail the caller.
func EnsureOtelSchemaFromEnv(ctx context.Context, retentionDays int) {
	addr := envvars.GetEnv("PMM_CLICKHOUSE_ADDR", defaultClickhouseAddr)
	username := envvars.GetEnv("PMM_CLICKHOUSE_USER", defaultClickhouseUser)
	password := envvars.GetEnv("PMM_CLICKHOUSE_PASSWORD", defaultClickhousePassword)
	chURI := url.URL{
		Scheme: "tcp",
		User:   url.UserPassword(username, password),
		Host:   addr,
		Path:   "/default",
	}
	dsn := chURI.String()
	WaitForClickhouseClusterReady(ctx, dsn)
	err := EnsureOtelSchema(ctx, dsn, retentionDays)
	if err != nil {
		logrus.WithError(err).Warn("Failed to ensure OTEL ClickHouse schema")
	}
}
