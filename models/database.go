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

	"github.com/AlekSi/pointer"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

// PMMServerPostgreSQLServiceName is a special Service Name representing PMM Server's PostgreSQL Service.
const PMMServerPostgreSQLServiceName = "pmm-server-postgresql"

// databaseSchema maps schema version from schema_migrations table (id column) to a slice of DDL queries.
var databaseSchema = [][]string{
	1: {
		`CREATE TABLE schema_migrations (
			id INTEGER NOT NULL,
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

	2: {
		`CREATE TABLE settings (
			settings JSONB
		)`,
		`INSERT INTO settings (settings) VALUES ('{}')`,
	},

	3: {
		`ALTER TABLE agents
			ADD COLUMN tls BOOLEAN NOT NULL DEFAULT false,
			ADD COLUMN tls_skip_verify BOOLEAN NOT NULL DEFAULT false`,

		`ALTER TABLE agents
			ALTER COLUMN tls DROP DEFAULT,
			ALTER COLUMN tls_skip_verify DROP DEFAULT`,
	},

	4: {
		`ALTER TABLE agents
			ADD COLUMN query_examples_disabled BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN max_query_log_size INTEGER NOT NULL DEFAULT 0`,

		`ALTER TABLE agents
			ALTER COLUMN query_examples_disabled DROP DEFAULT,
			ALTER COLUMN max_query_log_size DROP DEFAULT`,
	},

	5: {
		// e'\n' to treat \n as a newline, not as two characters
		`UPDATE nodes SET machine_id = trim(e'\n' from machine_id) WHERE machine_id IS NOT NULL`,
	},

	6: {
		`ALTER TABLE agents
			ADD COLUMN table_count INTEGER`,
	},

	7: {
		`ALTER TABLE agents
			ADD COLUMN node_id VARCHAR CHECK (node_id <> ''),
			ADD COLUMN service_id VARCHAR CHECK (service_id <> '')`,
		`UPDATE agents SET node_id=agent_nodes.node_id
			FROM agent_nodes
			WHERE agent_nodes.agent_id = agents.agent_id`,
		`UPDATE agents SET service_id=agent_services.service_id
			FROM agent_services
			WHERE agent_services.agent_id = agents.agent_id`,

		`DROP TABLE agent_nodes, agent_services`,

		`ALTER TABLE agents
			ADD CONSTRAINT node_id_or_service_id_or_pmm_agent_id CHECK (
				(CASE WHEN node_id IS NULL THEN 0 ELSE 1 END) +
  				(CASE WHEN service_id IS NULL THEN 0 ELSE 1 END) +
  				(CASE WHEN pmm_agent_id IS NOT NULL THEN 0 ELSE 1 END) = 1),
			ADD FOREIGN KEY (service_id) REFERENCES services(service_id),
			ADD FOREIGN KEY (node_id) REFERENCES nodes(node_id)`,
	},

	8: {
		// default to 1000 for soft migration from 2.1
		`ALTER TABLE agents
			ADD COLUMN table_count_tablestats_group_limit INTEGER NOT NULL DEFAULT 1000`,

		`ALTER TABLE agents
			ALTER COLUMN table_count_tablestats_group_limit DROP DEFAULT`,
	},

	9: {
		`ALTER TABLE agents
			ADD COLUMN aws_access_key VARCHAR,
			ADD COLUMN aws_secret_key VARCHAR`,
	},

	10: {
		// update 5/5/60 to 5/10/60 for 2.4 only if defaults were not changed
		`UPDATE settings SET
			settings = settings || '{"metrics_resolutions":{"hr": 5000000000, "mr": 10000000000, "lr": 60000000000}}'
			WHERE settings->'metrics_resolutions'->>'hr' = '5000000000'
			AND settings->'metrics_resolutions'->>'mr' = '5000000000'
			AND settings->'metrics_resolutions'->>'lr' = '60000000000'`,
	},

	11: {
		`ALTER TABLE services
			ADD COLUMN socket VARCHAR CONSTRAINT address_socket_check CHECK (
				(address IS NOT NULL AND socket IS NULL) OR (address IS NULL AND socket IS NOT NULL)
			)`,

		`ALTER TABLE services
			ADD CONSTRAINT address_port_check CHECK (
				(address IS NULL AND port IS NULL) OR (address IS NOT NULL AND port IS NOT NULL)
			),
			ADD CONSTRAINT port_check CHECK (
				port IS NULL OR (port > 0 AND port < 65535)
			)`,
	},

	12: {
		`ALTER TABLE agents
			ADD COLUMN rds_basic_metrics_disabled BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN rds_enhanced_metrics_disabled BOOLEAN NOT NULL DEFAULT FALSE`,

		`ALTER TABLE agents
			ALTER COLUMN rds_basic_metrics_disabled DROP DEFAULT,
			ALTER COLUMN rds_enhanced_metrics_disabled DROP DEFAULT`,
	},

	13: {
		`ALTER TABLE services
			DROP CONSTRAINT address_socket_check`,

		`ALTER TABLE services
			ADD CONSTRAINT address_socket_check CHECK (
				(address IS NOT NULL AND socket IS NULL) OR (address IS NULL AND socket IS NOT NULL) OR (address IS NULL AND socket IS NULL)
			)`,
	},

	14: {
		`ALTER TABLE agents
			DROP CONSTRAINT node_id_or_service_id_or_pmm_agent_id,
			DROP CONSTRAINT runs_on_node_id_only_for_pmm_agent,
			DROP CONSTRAINT agents_metrics_url_check`,
		`ALTER TABLE agents
			ADD CONSTRAINT node_id_or_service_id_for_non_pmm_agent CHECK (
				(node_id IS NULL) <> (service_id IS NULL) OR (agent_type = '` + string(PMMAgentType) + `')),
			ADD CONSTRAINT runs_on_node_id_only_for_pmm_agent_and_external
				CHECK ((runs_on_node_id IS NULL) <> (agent_type='` + string(PMMAgentType) + `' OR agent_type='` + string(ExternalExporterType) + `' ))`,
		`ALTER TABLE agents RENAME COLUMN metrics_url TO metrics_path`,
		`ALTER TABLE agents
			ADD CONSTRAINT agents_metrics_path_check CHECK (metrics_path <> '')`,
		`ALTER TABLE agents ADD COLUMN metrics_scheme VARCHAR`,
	},

	15: {
		// query action results are binary data
		`ALTER TABLE action_results
			DROP COLUMN output,
			ADD COLUMN output bytea`,
	},

	16: {
		`ALTER TABLE services
			DROP CONSTRAINT port_check`,

		`ALTER TABLE services
			ADD CONSTRAINT port_check CHECK (
				port IS NULL OR (port > 0 AND port < 65536)
			)`,
	},

	17: {
		`CREATE TABLE kubernetes_clusters (
			-- common
			id VARCHAR NOT NULL,
			kubernetes_cluster_name VARCHAR NOT NULL CHECK (kubernetes_cluster_name <> ''),
			kube_config TEXT NOT NULL CHECK (kube_config <> ''),
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id),
			UNIQUE (kubernetes_cluster_name)
		)`,
	},

	18: {
		`ALTER TABLE services
			ADD COLUMN external_group VARCHAR NOT NULL DEFAULT ''`,

		`UPDATE services SET external_group = 'external' WHERE service_type = '` + string(ExternalServiceType) + `'`,

		`ALTER TABLE services
			ALTER COLUMN external_group DROP DEFAULT`,

		// Only service with type external can have non empty value of group.
		`ALTER TABLE services
			ADD CONSTRAINT services_external_group_check CHECK (
				(service_type <> '` + string(ExternalServiceType) + `' AND external_group = '')
				OR
				(service_type = '` + string(ExternalServiceType) + `' AND external_group <> '')
			)`,
	},
}

// ^^^ Avoid default values in schema definition. ^^^
// aleksi: Go's zero values and non-zero default values in database do play nicely together in INSERTs and UPDATEs.

// OpenDB returns configured connection pool for PostgreSQL.
func OpenDB(address, name, username, password string) (*sql.DB, error) {
	q := make(url.Values)
	q.Set("sslmode", "disable")

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
		return nil, errors.Wrap(err, "failed to create a connection pool to PostgreSQL")
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)

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
	Logf             reform.Printf
	Username         string
	Password         string
	SetupFixtures    SetupFixturesMode
	MigrationVersion *int
}

// SetupDB runs PostgreSQL database migrations and optionally adds initial data.
func SetupDB(sqlDB *sql.DB, params *SetupDBParams) (*reform.DB, error) {
	var logger reform.Logger
	if params.Logf != nil {
		logger = reform.NewPrintfLogger(params.Logf)
	}
	db := reform.NewDB(sqlDB, postgresql.Dialect, logger)

	latestVersion := len(databaseSchema) - 1 // skip item 0
	if params.MigrationVersion != nil {
		latestVersion = *params.MigrationVersion
	}
	var currentVersion int
	err := db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	if pErr, ok := err.(*pq.Error); ok && pErr.Code == "42P01" { // undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
		err = nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if params.Logf != nil {
		params.Logf("Current database schema version: %d. Latest version: %d.", currentVersion, latestVersion)
	}

	// rollback all migrations if one of them fails; PostgreSQL supports DDL transactions
	err = db.InTransaction(func(tx *reform.TX) error {
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

		// fill settings with defaults
		s, err := GetSettings(tx)
		if err != nil {
			return err
		}
		if err = SaveSettings(tx, s); err != nil {
			return err
		}

		if err = setupFixture1(tx.Querier, params.Username, params.Password); err != nil {
			return err
		}
		if err = setupFixture2(tx.Querier, params.Username, params.Password); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func setupFixture1(q *reform.Querier, username, password string) error {
	// create PMM Server Node and associated Agents
	node, err := createNodeWithID(q, PMMServerNodeID, GenericNodeType, &CreateNodeParams{
		NodeName: "pmm-server",
		Address:  "127.0.0.1",
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
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
		ServiceName: PMMServerPostgreSQLServiceName,
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
