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
)

const (
	databaseNotExistErrorCode = 81
)

// NewDB return updated db.
func NewDB(dsn string, maxIdleConns, maxOpenConns int, isCluster bool, clusterName string) *sqlx.DB {
	l := logrus.WithField("component", "db")
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		l.Fatalf("error parsing DSN: %v", err)
	}

	// If ClickHouse is a cluster, wait until the cluster is ready.
	if isCluster {
		l.Info("PMM_CLICKHOUSE_IS_CLUSTER is set to 1")
		dsnURL.Path = "/default"
		dsnDefault := dsnURL.String()
		l.Infof("DSN for cluster check: %s", dsnURL.Redacted())

		for {
			isClusterReady, err := migrations.IsClickhouseClusterReady(dsnDefault, clusterName)
			if err != nil {
				l.Fatalf("error checking ClickHouse cluster status: %v", err)
			}
			if isClusterReady {
				l.Info("ClickHouse cluster is ready")
				break
			}

			l.Info("waiting for ClickHouse cluster to be ready... (system.clusters where remote_hosts > 0)")
			time.Sleep(1 * time.Second)
		}
	}

	l.Infof("new connection with DSN: %s", dsnURL.Redacted())
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		l.Errorf("error connecting to clickhouse: %v", err)
		var exception *clickhouse.Exception
		if errors.As(err, &exception) && exception.Code == databaseNotExistErrorCode {
			l.Info("one of expected errors - database does not exist, creating")
			err = createDB(dsn, clusterName, l)
			if err != nil {
				l.Fatalf("database wasn't created: %v", err)
			}
			l.Infof("database created, connecting again %s", dsnURL.Redacted())
			db, err = sqlx.Connect("clickhouse", dsn)
			if err != nil {
				l.Fatalf("connection: %v", err)
			}
		} else {
			l.Fatalf("connection: %v", err)
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
		"engine": migrations.GetEngine(dsn, clusterName),
	}
	if clusterName != "" {
		l.Infof("Using ClickHouse cluster name: %s", clusterName)
		data["cluster"] = fmt.Sprintf("ON CLUSTER %s", clusterName)
	}
	if err := migrations.Run(dsn, data, isCluster, clusterName); err != nil {
		l.Fatalf("migrations: %v", err)
	}
	l.Info("migrations applied")

	return db
}

func createDB(dsn string, clusterName string, l *logrus.Entry) error {
	l.Info("Creating database")
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

	sql := fmt.Sprintf("CREATE DATABASE %s", databaseName)
	if clusterName != "" {
		l.Infof("Using ClickHouse cluster name: %s", clusterName)
		sql = fmt.Sprintf("%s ON CLUSTER \"%s\"", sql, clusterName)
	}
	sql = fmt.Sprintf("%s ENGINE = Atomic", sql)

	result, err := defaultDB.Exec(sql)
	if err != nil {
		l.Infof("Result: %v", result)
		return err
	}
	l.Info("Database was created")

	// The qan-api2 will exit after creating the database, it'll be restarted by supervisor
	return nil
}

// DropOldPartition drops number of days old partitions of pmm.metrics in ClickHouse.
func DropOldPartition(db *sqlx.DB, dbName string, days uint, l *logrus.Entry) {
	partitions := []string{}
	const query = `
		SELECT DISTINCT partition
		FROM system.parts
		WHERE toUInt32(partition) < toYYYYMMDD(now() - toIntervalDay(?)) AND database = ? and visible = 1 ORDER BY partition
	`
	err := db.Select(
		&partitions,
		query,
		days,
		dbName)
	if err != nil {
		l.Infof("Select %d days old partitions of system.parts. Result: %v, Error: %v", days, partitions, err)
		return
	}
	for _, part := range partitions {
		result, err := db.Exec(fmt.Sprintf(`ALTER TABLE metrics DROP PARTITION %s`, part))
		l.Infof("Drop %s partitions of metrics. Result: %v, Error: %v", part, result, err)
	}
}
