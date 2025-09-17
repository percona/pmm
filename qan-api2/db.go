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
	"bytes"
	"embed"
	"fmt"
	iofs "io/fs"
	"log"
	"net/url"
	"strings"
	"text/template"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2" // register database/sql driver
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse" // register golang-migrate driver
	"github.com/jmoiron/sqlx"                                    // TODO: research alternatives. Ex.: https://github.com/go-reform/reform
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

const (
	databaseNotExistErrorCode = 81
	databaseEngineSimple      = "MergeTree"
	databaseEngineCluster     = "ReplicatedMergeTree('/clickhouse/tables/%d/metrics', '%d')"
)

// NewDB return updated db.
func NewDB(dsn string, maxIdleConns, maxOpenConns int) *sqlx.DB {
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

	if err := runMigrations(dsn); err != nil {
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

func getEngine(dsn string) string {
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		return databaseEngineSimple
	}
	defer db.Close()

	rows, err := db.Queryx("SELECT shard_num, replica_num FROM system.clusters WHERE cluster = 'default';")
	if err != nil {
		return databaseEngineSimple
	}
	defer rows.Close()

	if rows.Next() {
		var shardNum int
		var replicaNum int
		if err := rows.Scan(&shardNum, &replicaNum); err != nil {
			return databaseEngineSimple
		}

		return fmt.Sprintf(databaseEngineCluster, shardNum, replicaNum)
	}
	return databaseEngineSimple
}

//go:embed migrations/sql/*.sql
var fs embed.FS

func runMigrations(dsn string) error {
	dynamic := map[string]map[string]interface{}{
		"01_init.up.sql": {"engine": getEngine(dsn)},
	}

	entries, err := iofs.ReadDir(fs, "migrations/sql")
	if err != nil {
		return err
	}

	var migrations []memMigration
	for i, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := fs.ReadFile("migrations/sql/" + entry.Name())
		if err != nil {
			return err
		}

		migration := memMigration{
			Version:    uint(i + 1),
			Identifier: entry.Name(),
		}
		if dynamic[entry.Name()] == nil {
			migration.Up = string(content)
			migrations = append(migrations, migration)
			continue
		}

		var buf bytes.Buffer
		tmpl, err := template.New(entry.Name()).Parse(string(content))
		if err != nil {
			return err
		}
		if err := tmpl.Execute(&buf, dynamic[entry.Name()]); err != nil {
			return err
		}
		migration.Up = buf.String()
		migrations = append(migrations, migration)
	}

	src := newDynamicMigrations(migrations)
	m, err := migrate.NewWithSourceInstance("dynamic", src, dsn)
	if err != nil {
		return err
	}

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	return err
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
