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

	"github.com/AlekSi/pointer"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
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

// OpenDB returns configured connection pool for PostgreSQL.
func OpenDB(name, username, password string) (*sql.DB, error) {
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
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to PostgreSQL.")
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(0)
	return db, nil
}

// SetupFixturesMode defines if SetupDB adds initial data to the database or not.
type SetupFixturesMode int

const (
	// SetupFixtures adds initial data to the database.
	SetupFixtures SetupFixturesMode = iota
	// SkipFixtures skips adding initial data to the database. Useful for tests.
	SkipFixtures
)

// SetupDBParams represents SetupDB parameters.
type SetupDBParams struct {
	Logf          reform.Printf
	Username      string
	Password      string
	SetupFixtures SetupFixturesMode
}

// SetupDB runs PostgreSQL database migrations and optionally adds initial data.
func SetupDB(sqlDB *sql.DB, params *SetupDBParams) error {
	var logger reform.Logger
	if params.Logf != nil {
		logger = reform.NewPrintfLogger(params.Logf)
	}
	db := reform.NewDB(sqlDB, postgresql.Dialect, logger)

	latestVersion := len(databaseSchema) - 1 // skip item 0
	var currentVersion int
	err := db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	if pErr, ok := err.(*pq.Error); ok && pErr.Code == "42P01" { // undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
		err = nil
	}
	if err != nil {
		return errors.WithStack(err)
	}
	if params.Logf != nil {
		params.Logf("Current database schema version: %d. Latest version: %d.", currentVersion, latestVersion)
	}

	// rollback all migrations if one of them fails; PostgreSQL supports DDL transactions
	return db.InTransaction(func(tx *reform.TX) error {
		for version := currentVersion + 1; version <= latestVersion; version++ {
			if params.Logf != nil {
				params.Logf("Migrating database to schema version %d ...", version)
			}

			queries := databaseSchema[version]
			queries = append(queries, fmt.Sprintf(`INSERT INTO schema_migrations (id) VALUES (%d)`, version))
			for _, q := range queries {
				q = strings.TrimSpace(q)
				if _, err = tx.Exec(q); err != nil {
					return errors.Wrapf(err, "failed to execute statement:\n%s", q)
				}
			}
		}

		if params.SetupFixtures == SkipFixtures {
			return nil
		}

		if err = setupFixture1(tx.Querier, params.Username, params.Password); err != nil {
			return err
		}
		if err = setupFixture2(tx.Querier, params.Username, params.Password); err != nil {
			return err
		}
		return nil
	})
}

func setupFixture1(q *reform.Querier, username, password string) error {
	// create PMM Server Node and associated Agents
	node, err := createNodeWithID(q, PMMServerNodeID, GenericNodeType, &CreateNodeParams{
		NodeName: "PMM Server",
		Address:  "127.0.0.1",
	})
	if err != nil {
		if status.Code(err) != codes.AlreadyExists {
			// this fixture was already added previously
			return nil
		}
		return err
	}
	if _, err = createPMMAgentWithID(q, PMMServerAgentID, node.NodeID, nil); err != nil {
		return err
	}
	if _, err = CreateNodeExporter(q, PMMServerAgentID, nil); err != nil {
		return err
	}

	// create PostgreSQL Service and associated Agents
	service, err := AddNewService(q, PostgreSQLServiceType, &AddDBMSServiceParams{
		ServiceName: "PMM Server PostgreSQL",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(5432),
	})
	if err != nil {
		return err
	}
	_, err = CreateAgent(q, PostgresExporterType, &CreateAgentParams{
		PMMAgentID: PMMServerAgentID,
		ServiceID:  service.ServiceID,
		Username:   username,
		Password:   password,
	})
	if err != nil {
		return err
	}
	_, err = CreateAgent(q, QANPostgreSQLPgStatementsAgentType, &CreateAgentParams{
		PMMAgentID: PMMServerAgentID,
		ServiceID:  service.ServiceID,
		Username:   username,
		Password:   password,
	})
	if err != nil {
		return err
	}

	return nil
}

func setupFixture2(q *reform.Querier, username, password string) error {
	// TODO add clickhouse_exporter

	return nil
}
