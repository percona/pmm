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
// Package main.
package main

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse" // register golang-migrate driver
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/migrations"
	"github.com/percona/pmm/utils/dsnutils"
)

const (
	databaseNotExistErrorCode = 81
)

// NewDB creates a new ClickHouse database connection using TLS certificate authentication.
func NewDB(addr, database, username string, tlsConfig *tls.Config, maxIdleConns, maxOpenConns int, isCluster bool, clusterName string, migrationDSN string) *sqlx.DB {
	l := logrus.WithField("component", "db")
	// If ClickHouse is a cluster, wait until the cluster is ready.
	if isCluster {
		l.Info("PMM_CLICKHOUSE_IS_CLUSTER is set to 1")
		dsnURL, err := url.Parse(migrationDSN)
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

	// Connect using TLS cert-based auth via clickhouse.Connector
	opts := &clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
		},
		TLS: tlsConfig,
	}

	sqlDB := sql.OpenDB(clickhouse.Connector(opts))
	db := sqlx.NewDb(sqlDB, "clickhouse")

	// Verify connection
	if err := db.Ping(); err != nil {
		l.Errorf("Error connecting to ClickHouse: %v", err)
		var exception *clickhouse.Exception
		if errors.As(err, &exception) && exception.Code == databaseNotExistErrorCode {
			err = createDB(migrationDSN, clusterName)
			if err != nil {
				l.Fatalf("Database wasn't created: %v", err)
			}
			// Reconnect after creating database
			sqlDB = sql.OpenDB(clickhouse.Connector(opts))
			db = sqlx.NewDb(sqlDB, "clickhouse")
			if err := db.Ping(); err != nil {
				l.Fatalf("Connection after DB creation: %v", err)
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
	if err := migrations.Run(migrationDSN, data, isCluster, clusterName); err != nil {
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

	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", databaseName)
	if clusterName != "" {
		l.Infof("Using ClickHouse cluster name: %s", clusterName)
		sql = fmt.Sprintf("%s ON CLUSTER \"%s\"", sql, clusterName)
		sql = fmt.Sprintf("%s ENGINE = Replicated('/clickhouse/databases/{uuid}', '{shard}', '{replica}')", sql)
	} else {
		sql = fmt.Sprintf("%s ENGINE = Atomic", sql)
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
func DropOldPartition(db *sqlx.DB, dbName string, days uint) {
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
	err := db.Select(
		&partitions,
		query,
		dbName,
		days,
	)
	if err != nil {
		l.Infof("Select %d days old partitions of system.parts. Result: %v, Error: %v", days, partitions, err)
		return
	}
	for _, part := range partitions {
		result, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s.metrics DROP PARTITION %s`, dbName, part))
		l.Infof("Drop partition %s of %s.metrics. Result: %v, Error: %v", part, dbName, result, err)
	}
}
