package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/percona/pmm/qan-api2/migrations"
)

func main() {
	data := map[string]map[string]any{
		"01_init.up.sql": {"engine": "MergeTree"},
	}
	wd, err := os.Getwd()
	if err != nil {
		// handle error
	}
	sqlPath := filepath.Join(wd, "migrations", "sql")
	if err := migrations.GenerateTestSetupMigrations(data, sqlPath); err != nil {
		log.Fatalf("Failed to generate migrations: %v", err)
	}
	log.Println("Migrations generated.")
}
