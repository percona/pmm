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

package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan/migrations"
)

// qanMigrationsTable keeps the new service's migration history separate from
// qan-api2's default schema_migrations in the shared pmm database.
const qanMigrationsTable = "qan_schema_migrations"

const dbConnectTimeout = 10 * time.Second

// NewDB opens a native ClickHouse connection, creates the database if needed,
// and applies the schema migrations.
func NewDB(addr, database, user, password string, maxIdle, maxOpen int, retentionDays uint) driver.Conn {
	l := logrus.WithField("component", "db")

	err := createDB(addr, database, user, password)
	if err != nil {
		l.Fatalf("Failed to create database %q: %v", database, err)
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr:         []string{addr},
		Auth:         clickhouse.Auth{Database: database, Username: user, Password: password},
		MaxIdleConns: maxIdle,
		MaxOpenConns: maxOpen,
		DialTimeout:  dbConnectTimeout,
	})
	if err != nil {
		l.Fatalf("Failed to open ClickHouse connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancel()
	err = conn.Ping(ctx)
	if err != nil {
		l.Fatalf("Failed to ping ClickHouse: %v", err)
	}

	err = migrations.Run(migrateDSN(addr, database, user, password))
	if err != nil {
		l.Fatalf("Failed to apply migrations: %v", err)
	}
	l.Info("Migrations applied successfully")

	err = applyRetention(conn, retentionDays)
	if err != nil {
		l.Fatalf("Failed to apply data retention: %v", err)
	}
	l.Infof("Data retention set to %d days.", retentionDays)

	return conn
}

// applyRetention sets each tier's TTL from the configured retention window via
// ALTER MODIFY TTL. --data-retention is the overall window (the daily tier); finer
// tiers keep their shorter operational caps but never exceed the configured value.
func applyRetention(conn driver.Conn, days uint) error {
	tiers := []struct {
		table   string
		ttlDays uint
	}{
		{"metrics_raw", min(uint(7), days)},
		{"metrics_1h", min(uint(90), days)},
		{"metrics_1d", days},
		{"metrics_by_endpoint_1h", min(uint(30), days)},
		{"dim_values", min(uint(90), days)},
		{"query_examples", min(uint(8), days)},
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancel()
	for _, t := range tiers {
		query := fmt.Sprintf(
			"ALTER TABLE %s MODIFY TTL period_start + INTERVAL %d DAY SETTINGS materialize_ttl_after_modify = 0",
			t.table, t.ttlDays,
		)
		err := conn.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("set TTL on %s: %w", t.table, err)
		}
	}
	return nil
}

// migrateDSN builds the golang-migrate clickhouse DSN with a dedicated migrations table.
func migrateDSN(addr, database, user, password string) string {
	u := url.URL{
		Scheme:   "clickhouse",
		User:     url.UserPassword(user, password),
		Host:     addr,
		Path:     "/" + database,
		RawQuery: "x-migrations-table=" + qanMigrationsTable,
	}
	return u.String()
}

func createDB(addr, database, user, password string) error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: "default", Username: user, Password: password},
	})
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancel()
	return conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database))
}
