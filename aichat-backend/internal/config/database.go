package config

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// GetDatabaseConfig returns database configuration from environment variables
func GetDatabaseConfig() *DatabaseConfig {
	log.Printf("🗄️  Database Config: Loading database configuration...")

	// Try to get DSN from environment variable first
	if dsn := os.Getenv("AICHAT_DATABASE_URL"); dsn != "" {
		log.Printf("🗄️  Database Config: Using AICHAT_DATABASE_URL environment variable")
		// Mask password in log
		maskedDSN := maskPassword(dsn)
		log.Printf("🗄️  Database Config: DSN: %s", maskedDSN)
		return &DatabaseConfig{
			DSN: dsn,
		}
	}

	log.Printf("🗄️  Database Config: AICHAT_DATABASE_URL not found, building from individual env vars")

	// Fallback to building DSN from individual environment variables for backward compatibility
	host := getEnvOrDefault("AICHAT_DB_HOST", "127.0.0.1")
	port := getEnvIntOrDefault("AICHAT_DB_PORT", 5432)
	database := getEnvOrDefault("AICHAT_DB_NAME", "ai_chat")
	username := getEnvOrDefault("AICHAT_DB_USERNAME", "ai_chat_user")
	password := getEnvOrDefault("AICHAT_DB_PASSWORD", "ai_chat_secure_password")
	sslMode := getEnvOrDefault("AICHAT_DB_SSL_MODE", "disable")

	log.Printf("🗄️  Database Config: host=%s, port=%d, database=%s, username=%s, sslmode=%s",
		host, port, database, username, sslMode)

	// Build DSN from individual components
	q := make(url.Values)
	q.Set("sslmode", sslMode)

	uri := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(username, password),
		Host:     fmt.Sprintf("%s:%d", host, port),
		Path:     database,
		RawQuery: q.Encode(),
	}

	dsn := uri.String()
	maskedDSN := maskPassword(dsn)
	log.Printf("🗄️  Database Config: Built DSN: %s", maskedDSN)

	return &DatabaseConfig{
		DSN: dsn,
	}
}

// OpenDatabase opens a database connection and returns a sql.DB instance
func (c *DatabaseConfig) OpenDatabase() (*sql.DB, error) {
	log.Printf("🗄️  Database Config: Opening database connection...")
	maskedDSN := maskPassword(c.DSN)
	log.Printf("🗄️  Database Config: Using DSN: %s", maskedDSN)

	// Open database connection using DSN
	sqlDB, err := sql.Open("postgres", c.DSN)
	if err != nil {
		log.Printf("❌ Database Config: Failed to open database connection: %v", err)
		return nil, errors.Wrap(err, "failed to open database connection")
	}
	log.Printf("✅ Database Config: Database connection opened successfully")

	// Configure connection pool
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(0)
	log.Printf("🗄️  Database Config: Connection pool configured (max_open=10, max_idle=5)")

	// Test connection
	log.Printf("🗄️  Database Config: Testing database connection...")
	if err := sqlDB.Ping(); err != nil {
		log.Printf("❌ Database Config: Database ping failed: %v", err)
		sqlDB.Close()
		return nil, errors.Wrap(err, "failed to ping database")
	}
	log.Printf("✅ Database Config: Database ping successful")

	return sqlDB, nil
}

// maskPassword masks the password in a DSN for logging
func maskPassword(dsn string) string {
	if strings.Contains(dsn, "@") {
		// For postgres://user:pass@host format
		parts := strings.Split(dsn, "@")
		if len(parts) == 2 {
			userPart := parts[0]
			if strings.Contains(userPart, ":") {
				userPass := strings.Split(userPart, ":")
				if len(userPass) >= 2 {
					return userPass[0] + ":***@" + parts[1]
				}
			}
		}
	}
	return dsn // Return as-is if we can't parse it
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
