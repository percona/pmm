// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package models

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// FIXME Re-add created_at/updated_at: https://jira.percona.com/browse/PMM-3350

// databaseSchema maps schema version from schema_migrations table (id column) to a slice of DDL queries.
//
// Initial AUTO_INCREMENT values are spaced to prevent programming errors, or at least make them more visible.
// It does not imply that one can have at most 1000 nodes, etc.
var databaseSchema = [][]string{
	1: {
		`CREATE TABLE schema_migrations (
			id INT NOT NULL,
			PRIMARY KEY (id)
		)`,

		`CREATE TABLE nodes (
			-- common
			node_id VARCHAR(255) NOT NULL,
			node_type VARCHAR(255) NOT NULL,
			node_name VARCHAR(255) NOT NULL,
			machine_id VARCHAR(255),
			custom_labels TEXT,
			address VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			-- updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			-- Generic
			distro VARCHAR(255),
			distro_version VARCHAR(255),

			-- Container
			docker_container_id VARCHAR(255),
			docker_container_name VARCHAR(255),

			-- RemoteAmazonRDS
			-- RDS instance is stored in address
			region VARCHAR(255),

			PRIMARY KEY (node_id),
			UNIQUE (node_name),
			UNIQUE (machine_id),
			UNIQUE (docker_container_id),
			UNIQUE (address, region)
		)`,

		fmt.Sprintf(`INSERT INTO nodes (node_id, node_type,	node_name) VALUES ('%s', '%s', 'PMM Server')`, PMMServerNodeID, GenericNodeType), //nolint:gosec

		`CREATE TABLE services (
			-- common
			service_id VARCHAR(255) NOT NULL,
			service_type VARCHAR(255) NOT NULL,
			service_name VARCHAR(255) NOT NULL,
			node_id VARCHAR(255) NOT NULL,
			custom_labels TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			-- updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			-- MySQL
			address VARCHAR(255),
			port SMALLINT UNSIGNED,

			PRIMARY KEY (service_id),
			UNIQUE (service_name),
			FOREIGN KEY (node_id) REFERENCES nodes (node_id)
		)`,

		`CREATE TABLE agents (
			-- common
			agent_id VARCHAR(255) NOT NULL,
			agent_type VARCHAR(255) NOT NULL,
			runs_on_node_id VARCHAR(255),
			pmm_agent_id VARCHAR(255),
			custom_labels TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			-- updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			-- state
			status VARCHAR(255) NOT NULL,
			listen_port SMALLINT UNSIGNED,
			version VARCHAR(255),

			-- Credentials to access service
			username VARCHAR(255),
			password VARCHAR(255),
			metrics_url VARCHAR(255),

			PRIMARY KEY (agent_id),
			FOREIGN KEY (runs_on_node_id) REFERENCES nodes (node_id)
		)`,

		`CREATE TABLE agent_nodes (
			agent_id VARCHAR(255) NOT NULL,
			node_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

			FOREIGN KEY (agent_id) REFERENCES agents (agent_id),
			FOREIGN KEY (node_id) REFERENCES nodes (node_id),
			UNIQUE (agent_id, node_id)
		)`,

		`CREATE TABLE agent_services (
			agent_id VARCHAR(255) NOT NULL,
			service_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

			FOREIGN KEY (agent_id) REFERENCES agents (agent_id),
			FOREIGN KEY (service_id) REFERENCES services (service_id),
			UNIQUE (agent_id, service_id)
		)`,
	},
	2: {
		`
		-- MongoDBExporter
		ALTER TABLE agents ADD connection_string VARCHAR(255) AFTER password
		`,
	},
}

// OpenDB opens connection to MySQL database and runs migrations.
func OpenDB(name, username, password string, logf reform.Printf) (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = username
	cfg.Passwd = password
	cfg.DBName = name

	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"

	// required for reform
	cfg.ClientFoundRows = true
	cfg.ParseTime = true

	dsn := cfg.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		db.SetMaxIdleConns(10)
		db.SetMaxOpenConns(10)
		db.SetConnMaxLifetime(0)
		err = db.Ping()
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to MySQL.")
	}

	if name == "" {
		return db, nil
	}

	latestVersion := len(databaseSchema) - 1 // skip item 0
	var currentVersion int
	err = db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	if myErr, ok := err.(*mysql.MySQLError); ok && myErr.Number == 0x47a { // 1046 table doesn't exist
		err = nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current version.")
	}
	logf("Current database schema version: %d. Latest version: %d.", currentVersion, latestVersion)

	for version := currentVersion + 1; version <= latestVersion; version++ {
		logf("Migrating database to schema version %d ...", version)
		queries := databaseSchema[version]
		queries = append(queries, fmt.Sprintf(`INSERT INTO schema_migrations (id) VALUES (%d)`, version))
		for _, q := range queries {
			q = strings.TrimSpace(q)
			logf("\n%s\n", q)
			if _, err = db.Exec(q); err != nil {
				return nil, errors.Wrapf(err, "Failed to execute statement:\n%s.", q)
			}
		}
	}

	return db, nil
}

// postgresDatabaseSchema maps schema version from schema_migrations table (id column) to a slice of DDL queries.
var postgresDatabaseSchema = [][]string{
	1: {
		`CREATE TABLE schema_migrations (
			id INT PRIMARY KEY
		)`,

		`CREATE TABLE telemetry (
  			uuid VARCHAR PRIMARY KEY,
  			created_at TIMESTAMP NOT NULL
		)`,
	},
}

// OpenPostgresDB opens connection to PostgreSQL database and runs migrations.
func OpenPostgresDB(name, username, password string, logf reform.Printf) (*sql.DB, error) {
	q := make(url.Values)
	q.Set("sslmode", "disable")

	address := "127.0.0.1:5432"
	uri := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(username, password),
		Host:     address,
		Path:     name,
		RawQuery: q.Encode(),
	}
	if uri.Path == "" {
		uri.Path = "postgres"
	}
	dsn := uri.String()

	db, err := sql.Open("postgres", dsn)
	if err == nil {
		db.SetMaxIdleConns(10)
		db.SetMaxOpenConns(10)
		db.SetConnMaxLifetime(0)
		err = db.Ping()
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to PostgreSQL.")
	}

	if name == "" {
		return db, nil
	}

	latestVersion := len(postgresDatabaseSchema) - 1 // skip item 0
	var currentVersion int
	err = db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	if pErr, ok := err.(*pq.Error); ok && pErr.Code == "42P01" { // undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
		err = nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get current version.")
	}
	logf("Current database schema version: %d. Latest version: %d.", currentVersion, latestVersion)

	for version := currentVersion + 1; version <= latestVersion; version++ {
		logf("Migrating database to schema version %d ...", version)
		queries := postgresDatabaseSchema[version]
		queries = append(queries, fmt.Sprintf(`INSERT INTO schema_migrations (id) VALUES (%d)`, version))
		for _, q := range queries {
			q = strings.TrimSpace(q)
			logf("\n%s\n", q)
			if _, err = db.Exec(q); err != nil {
				return nil, errors.Wrapf(err, "Failed to execute statement:\n%s.", q)
			}
		}
	}

	return db, nil
}
