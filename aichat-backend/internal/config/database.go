package config

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// GetDatabaseConfig returns database configuration from environment variables
func GetDatabaseConfig() *DatabaseConfig {
	// Try to get DSN from environment variable first
	if dsn := os.Getenv("AICHAT_DATABASE_URL"); dsn != "" {
		return &DatabaseConfig{
			DSN: dsn,
		}
	}

	// Fallback to building DSN from individual environment variables for backward compatibility
	host := getEnvOrDefault("AICHAT_DB_HOST", "127.0.0.1")
	port := getEnvIntOrDefault("AICHAT_DB_PORT", 5432)
	database := getEnvOrDefault("AICHAT_DB_NAME", "ai_chat")
	username := getEnvOrDefault("AICHAT_DB_USERNAME", "ai_chat_user")
	password := getEnvOrDefault("AICHAT_DB_PASSWORD", "ai_chat_secure_password")
	sslMode := getEnvOrDefault("AICHAT_DB_SSL_MODE", "disable")

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

	return &DatabaseConfig{
		DSN: uri.String(),
	}
}

// OpenDatabase opens a database connection and returns a sql.DB instance
func (c *DatabaseConfig) OpenDatabase() (*sql.DB, error) {
	// Open database connection using DSN
	sqlDB, err := sql.Open("postgres", c.DSN)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database connection")
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(0)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, errors.Wrap(err, "failed to ping database")
	}

	return sqlDB, nil
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
