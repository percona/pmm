package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	lastMigrationFlag := flag.Int("last-migration", 21, "Last migration number (e.g., 21)")
	userFlag := flag.String("user", "default", "ClickHouse username")
	passwordFlag := flag.String("password", "clickhouse", "ClickHouse password")
	flag.Parse()

	user := *userFlag
	password := *passwordFlag
	lastMigration := *lastMigrationFlag
	if lastMigration == 0 {
		log.Println("Usage: go run main.go --last-migration <number> [--user <user>] [--password <password>]")
		os.Exit(1)
	}
	migrationDir := "file:///root/go/src/github.com/percona/pmm/qan-api2/migrations/sql"

	clickhouseDSN := fmt.Sprintf("clickhouse://localhost:9000?username=%s&password=%s&database=pmm", user, password)
	m, err := migrate.New(migrationDir, clickhouseDSN)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	if err := m.Migrate(uint(lastMigration + 1)); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations applied successfully.")
}
