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
// Package main provides ClickHouse database connectivity, schema migrations,
// and partition management for the Query Analytics API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"          // register database/sql driver
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse" // register golang-migrate driver
	"github.com/jmoiron/sqlx"                                    // TODO: research alternatives. Ex.: https://github.com/go-reform/reform
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/migrations"
	"github.com/percona/pmm/utils/dsnutils"
)

const (
	databaseNotExistErrorCode = 81
)

// NewDB return updated db.
func NewDB(dsn string, maxIdleConns, maxOpenConns int, isCluster bool, clusterName string) *sqlx.DB {
	l := logrus.WithField("component", "db")
	// If ClickHouse is a cluster, wait until the cluster is ready.
	if isCluster {
		l.Info("PMM_CLICKHOUSE_IS_CLUSTER is set to 1")
		dsnURL, err := url.Parse(dsn)
		if err != nil {
			l.Fatalf("Error parsing DSN: %v", err)
		}
		dsnURL.Path = "/default"
		dsnDefault := dsnURL.String()
		l.Infof("DSN for cluster check: %s", dsnutils.RedactDSN(dsnDefault))

		for {
			isClusterReady, err := migrations.IsClickhouseClusterReady(dsnDefault, clusterName)
			if err != nil {
				l.Fatalf("Error checking ClickHouse cluster status: %v", err)
			}
			if isClusterReady {
				l.Info("ClickHouse cluster is ready")
				break
			}

			l.Info("Waiting for ClickHouse cluster to be ready... (system.clusters where remote_hosts > 0)")
			time.Sleep(1 * time.Second)
		}
	}

	l.Infof("New connection with DSN: %s", dsnutils.RedactDSN(dsn))
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		l.Errorf("Error connecting to ClickHouse: %v", err)
		exception, ok := errors.AsType[*clickhouse.Exception](err)
		if ok && exception.Code == databaseNotExistErrorCode {
			err = createDB(dsn, clusterName)
			if err != nil {
				l.Fatalf("Database wasn't created: %v", err)
			}
			l.Infof("Connecting again to %s", dsnutils.RedactDSN(dsn))
			db, err = sqlx.Connect("clickhouse", dsn)
			if err != nil {
				l.Fatalf("Connection: %v", err)
			}
		} else {
			l.Fatalf("Connection: %v", err)
		}
	}

	// TODO: find solution with better performance
	db.Mapper = reflectx.NewMapperTagFunc("json", strings.ToUpper, func(value string) string {
		if strings.Contains(value, ",") {
			return strings.Split(value, ",")[0]
		}
		return value
	})

	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetMaxOpenConns(maxOpenConns)

	data := map[string]any{
		"engine": migrations.GetEngine(isCluster),
	}
	err = migrations.Run(dsn, data, isCluster, clusterName)
	if err != nil {
		l.Fatalf("migrations: %v", err)
	}
	l.Info("Migrations applied successfully")

	return db
}

func createDB(dsn string, clusterName string) error {
	l := logrus.WithField("component", "db")
	l.Info("Creating database...")
	clickhouseURL, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	databaseName := strings.Replace(clickhouseURL.Path, "/", "", 1)
	clickhouseURL.Path = "/default"

	defaultDB, err := sqlx.Connect("clickhouse", clickhouseURL.String())
	if err != nil {
		return err
	}
	defer defaultDB.Close() //nolint:errcheck

	sql := "CREATE DATABASE IF NOT EXISTS " + databaseName
	if clusterName != "" {
		l.Infof("Using ClickHouse cluster name: %s", clusterName)
		sql += " ON CLUSTER \"" + clusterName + "\""
		sql += " ENGINE = Replicated('/clickhouse/databases/{uuid}', '{shard}', '{replica}')"
	} else {
		sql += " ENGINE = Atomic"
	}

	result, err := defaultDB.Exec(sql)
	if err != nil {
		l.Infof("Result: %v", result)
		return err
	}
	l.Infof("Database %s created using sql: %s", databaseName, sql)

	// The qan-api2 will exit after creating the database, it'll be restarted by supervisor
	return nil
}

// DropOldPartition drops number of days old partitions of pmm.metrics in ClickHouse.
//
// One unusable partition does not strand the rest, so every failure is collected and returned
// together. Errors are returned rather than logged: the caller decides how loudly to report
// them and when to try again.
//
// Every statement is bound to ctx so that a ClickHouse call blocking here cannot hold up
// shutdown.
func DropOldPartition(ctx context.Context, db *sqlx.DB, dbName string, days uint) error {
	l := logrus.WithField("component", "db")
	partitions := []string{}
	const query = `
		SELECT DISTINCT partition
		FROM system.parts
		WHERE database = ?
			AND table = 'metrics'
			AND visible = 1
			AND match(partition, '^[0-9]{8}$')
			AND toUInt32(partition) < toYYYYMMDD(now() - toIntervalDay(?))
		ORDER BY partition
	`
	err := db.SelectContext(
		ctx,
		&partitions,
		query,
		dbName,
		days,
	)
	if err != nil {
		return fmt.Errorf("failed to select partitions older than %d days: %w", days, err)
	}

	var errs []error
	dropped := 0
	for _, part := range partitions {
		_, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s.metrics DROP PARTITION %s`, dbName, part))
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to drop partition %s of %s.metrics: %w", part, dbName, err))
			continue
		}
		dropped++
	}

	// Logged even when nothing was old enough, so a healthy deployment leaves evidence that
	// retention is running at all.
	l.Infof("Data retention applied to %s.metrics: dropped %d of %d partitions older than %d days.",
		dbName, dropped, len(partitions), days)

	return errors.Join(errs...)
}
