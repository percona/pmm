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
	start := flag.Int("start", 0, "Start migration number (e.g., 21)")
	clickhouseDSN := flag.String("dsn", "clickhouse://localhost:9000?username=default&password=clickhouse&database=pmm", "ClickHouse DSN")
	flag.Parse()

	// Read user and password from environment variables
	user := os.Getenv("CLICKHOUSE_USER")
	password := os.Getenv("CLICKHOUSE_PASSWORD")

	// Inject user and password into DSN if provided
	dsn := *clickhouseDSN
	if user != "" {
		// Replace or add username in DSN
		dsn = replaceDSNParam(dsn, "username", user)
	}
	if password != "" {
		// Replace or add password in DSN
		dsn = replaceDSNParam(dsn, "password", password)
	}

	if *start == 0 {
		fmt.Println("Usage: go run run_clickhouse_migrations.go --start <number> [--dsn <dsn>] (env: CLICKHOUSE_USER, CLICKHOUSE_PASSWORD)")
		os.Exit(1)
	}

	migrationDir := "file:///root/go/src/github.com/percona/pmm/qan-api2/migrations/sql"

	m, err := migrate.New(
		migrationDir,
		dsn,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Migrate(uint(*start + 1)); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migrations applied successfully.")

}

// replaceDSNParam replaces or adds a parameter in the ClickHouse DSN query string
func replaceDSNParam(dsn, param, value string) string {
	// Simple implementation: replace param in query string or add if missing
	// Only works for DSN with query params
	idx := len(dsn)
	q := "?"
	qIdx := indexOf(dsn, q)
	if qIdx != -1 {
		idx = qIdx
	}
	base := dsn
	query := ""
	if idx < len(dsn) {
		base = dsn[:idx]
		query = dsn[idx+1:]
	}
	params := parseQueryParams(query)
	params[param] = value
	return base + "?" + buildQueryParams(params)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func parseQueryParams(query string) map[string]string {
	params := make(map[string]string)
	for _, pair := range split(query, "&") {
		if pair == "" {
			continue
		}
		kv := split(pair, "=")
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
}

func buildQueryParams(params map[string]string) string {
	result := ""
	first := true
	for k, v := range params {
		if !first {
			result += "&"
		}
		result += k + "=" + v
		first = false
	}
	return result
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
}
