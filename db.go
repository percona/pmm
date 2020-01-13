// qan-api2
// Copyright (C) 2019 Percona LLC
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
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/clickhouse" // register golang-migrate driver
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/jmoiron/sqlx" // TODO: research alternatives. Ex.: https://github.com/go-reform/reform
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/kshvakov/clickhouse" // register database/sql driver

	"github.com/percona/qan-api2/migrations"
)

// NewDB return updated db.
func NewDB(dsn string, maxIdleConns, maxOpenConns int) *sqlx.DB {
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
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

func runMigrations(dsn string) error {
	s := bindata.Resource(migrations.AssetNames(), migrations.Asset)

	d, err := bindata.WithInstance(s)
	if err != nil {
		return err
	}
	log.Println("dsn: ", dsn)
	m, err := migrate.NewWithSourceInstance("go-bindata", d, dsn)
	if err != nil {
		return err
	}

	// run up to the latest migration
	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}
	return err
}

// DropOldPartition drops number of days old partitions of pmm.metrics in ClickHouse.
func DropOldPartition(db *sqlx.DB, days uint) {
	partitions := []string{}
	const query = `
		SELECT DISTINCT partition
		FROM system.parts
		WHERE toUInt32(partition) < toYYYYMMDD(now() - toIntervalDay(?)) ORDER BY partition
	`
	err := db.Select(
		&partitions,
		query,
		days,
	)
	if err != nil {
		log.Printf("Select %d days old partitions of system.parts. Result: %v, Error: %v", days, partitions, err)
		return
	}
	for _, part := range partitions {
		result, err := db.Exec(fmt.Sprintf(`ALTER TABLE metrics DROP PARTITION %s`, part))
		log.Printf("Drop %s partitions of metrics. Result: %v, Error: %v", part, result, err)
	}
}
