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
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

var initialCurrentTime = Now().Format(time.RFC3339)

// databaseSchema maps schema version from schema_migrations table (id column) to a slice of DDL queries.
var databaseSchema = [][]string{
	1: {
		`CREATE TABLE schema_migrations (
			id INT NOT NULL,
			PRIMARY KEY (id)
		)`,

		`CREATE TABLE nodes (
			-- common
			node_id VARCHAR NOT NULL,
			node_type VARCHAR NOT NULL CHECK (node_type <> ''),
			node_name VARCHAR NOT NULL CHECK (node_name <> ''),
			machine_id VARCHAR CHECK (machine_id <> ''),
			distro VARCHAR NOT NULL,
			node_model VARCHAR NOT NULL,
			az VARCHAR NOT NULL,
			custom_labels TEXT,
			address VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			-- Container
			container_id VARCHAR CHECK (container_id <> ''),
			container_name VARCHAR CHECK (container_name <> ''),

			-- RemoteAmazonRDS
			-- RDS instance is stored in address
			region VARCHAR CHECK (region <> ''),

			PRIMARY KEY (node_id),
			UNIQUE (node_name),
			UNIQUE (container_id),
			UNIQUE (address, region)
		)`,
		`CREATE UNIQUE INDEX nodes_machine_id_generic_key
			ON nodes (machine_id)
			WHERE node_type = 'generic'
		`,

		fmt.Sprintf(`INSERT INTO nodes (node_id, node_type,	node_name, distro, node_model, az, address, created_at, updated_at) `+ //nolint:gosec
			`VALUES ('%s', '%s', 'PMM Server', 'Linux', '', '', '', '%s', '%s')`, //nolint:gosec
			PMMServerNodeID, GenericNodeType, initialCurrentTime, initialCurrentTime), //nolint:gosec

		`CREATE TABLE services (
			-- common
			service_id VARCHAR NOT NULL,
			service_type VARCHAR NOT NULL CHECK (service_type <> ''),
			service_name VARCHAR NOT NULL CHECK (service_name <> ''),
			node_id VARCHAR NOT NULL CHECK (node_id <> ''),
			environment VARCHAR NOT NULL,
			cluster VARCHAR NOT NULL,
			replication_set VARCHAR NOT NULL,
			custom_labels TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			address VARCHAR(255) CHECK (address <> ''),
			port INTEGER,

			PRIMARY KEY (service_id),
			UNIQUE (service_name),
			FOREIGN KEY (node_id) REFERENCES nodes (node_id)
		)`,

		`CREATE TABLE agents (
			-- common
			agent_id VARCHAR NOT NULL,
			agent_type VARCHAR NOT NULL CHECK (agent_type <> ''),
			runs_on_node_id VARCHAR CHECK (runs_on_node_id <> ''),
			pmm_agent_id VARCHAR CHECK (pmm_agent_id <> ''),
			custom_labels TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			-- state
			disabled BOOLEAN NOT NULL,
			status VARCHAR NOT NULL,
			listen_port INTEGER,
			version VARCHAR CHECK (version <> ''),

			-- Credentials to access service
			username VARCHAR CHECK (username <> ''),
			password VARCHAR CHECK (password <> ''),
			metrics_url VARCHAR CHECK (metrics_url <> ''),

			PRIMARY KEY (agent_id),
			FOREIGN KEY (runs_on_node_id) REFERENCES nodes (node_id),
			FOREIGN KEY (pmm_agent_id) REFERENCES agents (agent_id),
			CONSTRAINT runs_on_node_id_xor_pmm_agent_id CHECK ((runs_on_node_id IS NULL) <> (pmm_agent_id IS NULL)),
			CONSTRAINT runs_on_node_id_only_for_pmm_agent CHECK ((runs_on_node_id IS NULL) <> (agent_type='` + string(PMMAgentType) + `'))
		)`,

		`CREATE TABLE agent_nodes (
			agent_id VARCHAR NOT NULL,
			node_id VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,

			FOREIGN KEY (agent_id) REFERENCES agents (agent_id),
			FOREIGN KEY (node_id) REFERENCES nodes (node_id),
			UNIQUE (agent_id, node_id)
		)`,

		`CREATE TABLE agent_services (
			agent_id VARCHAR NOT NULL,
			service_id VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,

			FOREIGN KEY (agent_id) REFERENCES agents (agent_id),
			FOREIGN KEY (service_id) REFERENCES services (service_id),
			UNIQUE (agent_id, service_id)
		)`,

		`CREATE TABLE telemetry (
  			uuid VARCHAR PRIMARY KEY,
  			created_at TIMESTAMP NOT NULL
		)`,

		`CREATE TABLE action_results (
			id VARCHAR NOT NULL,
			pmm_agent_id VARCHAR CHECK (pmm_agent_id <> ''),
			done BOOLEAN NOT NULL,
			error VARCHAR NOT NULL,
			output TEXT NOT NULL,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},
}

// OpenDB opens connection to PostgreSQL database and runs migrations.
func OpenDB(name, username, password string, logf reform.Printf) (*sql.DB, error) {
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

	latestVersion := len(databaseSchema) - 1 // skip item 0
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
