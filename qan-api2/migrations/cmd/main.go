package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/percona/pmm/qan-api2/migrations"
)

func main() {
	data := map[string]map[string]any{
		"01_init.up.sql": {"engine": "MergeTree", "cluster": ""},
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	sqlPath := filepath.Join(wd, "migrations", "sql")
	if err := migrations.GenerateMigrations(data, sqlPath); err != nil {
		log.Fatalf("Failed to generate migrations: %v", err)
	}
	log.Println("Migrations generated.")
}
