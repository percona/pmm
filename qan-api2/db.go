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
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"          // register database/sql driver
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse" // register golang-migrate driver
	"github.com/jmoiron/sqlx"                                    // TODO: research alternatives. Ex.: https://github.com/go-reform/reform
	"github.com/jmoiron/sqlx/reflectx"

	"github.com/percona/pmm/qan-api2/migrations"
)

const (
	databaseNotExistErrorCode = 81
)

// NewDB return updated db.
func NewDB(dsn string, maxIdleConns, maxOpenConns int) *sqlx.DB {
	// If the environment variable PMM_CLICKHOUSE_IS_CLUSTER is set to "1",
	// wait until the ClickHouse cluster is ready (i.e., remote_hosts > 0 in system.clusters).
	// This ensures the cluster is fully initialized before continuing.
	if os.Getenv("PMM_CLICKHOUSE_IS_CLUSTER") == "1" {
		isCluster, err := migrations.IsClickhouseCluster(dsn)
		if err != nil {
			log.Fatalf("Error checking ClickHouse cluster status: %v", err)
		}
		for !isCluster {
			log.Println("Waiting for ClickHouse cluster to be ready... (system.clusters remote_hosts > 0)")
			time.Sleep(1 * time.Second)
		}
	}

	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok && exception.Code == databaseNotExistErrorCode { //nolint:errorlint
			err = createDB(dsn)
			if err != nil {
				log.Fatalf("Database wasn't created: %v", err)
			}
			db, err = sqlx.Connect("clickhouse", dsn)
			if err != nil {
				log.Fatalf("Connection: %v", err)
			}
		} else {
			log.Fatalf("Connection: %v", err)
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

	data := map[string]map[string]any{
		"01_init.up.sql": {"engine": migrations.GetEngine(dsn)},
	}

	// If PMM_CLICKHOUSE_CLUSTER_NAME is set, use it in the migration to create the metrics table with the specified cluster.
	clusterName := os.Getenv("PMM_CLICKHOUSE_CLUSTER_NAME")
	if clusterName != "" {
		log.Printf("Using ClickHouse cluster name: %s", clusterName)
		data["01_init.up.sql"]["cluster"] = fmt.Sprintf("ON CLUSTER '%s' ", clusterName)
	}

	if err := migrations.Run(dsn, data); err != nil {
		log.Fatal("Migrations: ", err)
	}
	log.Println("Migrations applied.")
	return db
}

func createDB(dsn string) error {
	log.Println("Creating database")
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

	result, err := defaultDB.Exec(fmt.Sprintf(`CREATE DATABASE %s ENGINE = Atomic`, databaseName))
	if err != nil {
		log.Printf("Result: %v", result)
		return err
	}
	log.Println("Database was created")
	return nil
	// The qan-api2 will exit after creating the database, it'll be restarted by supervisor
}

// DropOldPartition drops number of days old partitions of pmm.metrics in ClickHouse.
func DropOldPartition(db *sqlx.DB, dbName string, days uint) {
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
		log.Printf("Select %d days old partitions of system.parts. Result: %v, Error: %v", days, partitions, err)
		return
	}
	for _, part := range partitions {
		result, err := db.Exec(fmt.Sprintf(`ALTER TABLE metrics DROP PARTITION %s`, part))
		log.Printf("Drop %s partitions of metrics. Result: %v, Error: %v", part, result, err)
	}
}
