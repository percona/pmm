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
//
// Package main provides a ClickHouse migration runner for restoring clickhouse backups.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	defaultLastMigration = uint(21)
	defaultMigrationsDir = "file:///root/go/src/github.com/percona/pmm/qan-api2/migrations/sql"
)

func main() {
	lastMigrationFlag := flag.Uint("last-migration", defaultLastMigration, "Last migration number (e.g., 21)")
	userFlag := flag.String("user", "default", "ClickHouse username")
	passwordFlag := flag.String("password", "clickhouse", "ClickHouse password")
	migrationsDirFlag := flag.String("migrations-dir", defaultMigrationsDir, "Directory with migration files")
	flag.Parse()

	lastMigration := *lastMigrationFlag
	if lastMigration == 0 {
		log.Println("Usage: go run main.go --last-migration <number> [--user <user>] [--password <password>] [--migrations-dir <dir>]")
		os.Exit(1)
	}

	clickhouseDSN := fmt.Sprintf("clickhouse://localhost:9000?username=%s&password=%s&database=pmm", *userFlag, *passwordFlag)
	m, err := migrate.New(*migrationsDirFlag, clickhouseDSN)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	err = m.Migrate(lastMigration + 1)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations applied successfully.")
}
