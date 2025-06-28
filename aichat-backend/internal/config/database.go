package config

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

var dbLogger = logrus.WithField("component", "database-config")

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// OpenDatabase opens a database connection and returns a sql.DB instance
func (c *DatabaseConfig) OpenDatabase() (*sql.DB, error) {
	dbLogger.Debug("Opening database connection")
	maskedDSN := maskPassword(c.DSN)
	dbLogger.WithField("dsn", maskedDSN).Debug("Using DSN for connection")

	// Open database connection using DSN
	sqlDB, err := sql.Open("postgres", c.DSN)
	if err != nil {
		dbLogger.WithError(err).Error("Failed to open database connection")
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	dbLogger.Debug("Database connection opened successfully")

	// Configure connection pool
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(0)
	dbLogger.Debug("Connection pool configured (max_open=10, max_idle=5)")

	// Test connection
	dbLogger.Debug("Testing database connection")
	if err := sqlDB.Ping(); err != nil {
		dbLogger.WithError(err).Error("Database ping failed")
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	dbLogger.Info("Database connection established successfully")

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
