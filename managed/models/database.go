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

	19: {
		`ALTER TABLE agents
			ADD COLUMN push_metrics BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE agents
			ALTER COLUMN push_metrics DROP DEFAULT`,
	},

	20: {
		`ALTER TABLE agents DROP CONSTRAINT runs_on_node_id_only_for_pmm_agent_and_external`,
	},

	21: {
		`ALTER TABLE agents
			ADD CONSTRAINT runs_on_node_id_only_for_pmm_agent
            CHECK (((runs_on_node_id IS NULL) <> (agent_type='` + string(PMMAgentType) + `'))  OR (agent_type='` + string(ExternalExporterType) + `'))`,
	},

	22: {
		`CREATE TABLE ia_channels (
			id VARCHAR NOT NULL,
			summary VARCHAR NOT NULL,
			type VARCHAR NOT NULL,

			email_config JSONB,
			pagerduty_config JSONB,
			slack_config JSONB,
			webhook_config JSONB,

			disabled BOOLEAN NOT NULL,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},

	23: {
		`CREATE TABLE ia_templates (
			name VARCHAR NOT NULL,
			version INTEGER NOT NULL,
			summary VARCHAR NOT NULL,
			tiers JSONB NOT NULL,
			expr VARCHAR NOT NULL,
			params JSONB,
			"for" BIGINT,
			severity VARCHAR NOT NULL,
			labels TEXT,
			annotations TEXT,
			source VARCHAR NOT NULL,
			yaml TEXT NOT NULL,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (name)
		)`,
	},

	24: {
		`CREATE TABLE ia_rules (
			id VARCHAR NOT NULL,
			template_name VARCHAR NOT NULL,
			summary VARCHAR NOT NULL,
			disabled BOOLEAN NOT NULL,
			params JSONB,
			"for" BIGINT,
			severity VARCHAR NOT NULL,
			custom_labels TEXT,
			filters JSONB,
			channel_ids JSONB NOT NULL,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},
	25: {
		`ALTER TABLE agents ADD COLUMN mongo_db_tls_options JSONB`,
	},
	26: {
		`ALTER TABLE ia_rules ALTER COLUMN channel_ids DROP NOT NULL`,
	},
	27: {
		`CREATE TABLE backup_locations (
			id VARCHAR NOT NULL,
			name VARCHAR NOT NULL CHECK (name <> ''),
			description VARCHAR NOT NULL,
			type VARCHAR NOT NULL CHECK (type <> ''),
			s3_config JSONB,
			pmm_server_config JSONB,
			pmm_client_config JSONB,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id),
			UNIQUE (name)
		)`,
	},
	28: {
		`ALTER TABLE agents ADD COLUMN disabled_collectors VARCHAR[]`,
	},
	29: {
		`CREATE TABLE artifacts (
			id VARCHAR NOT NULL,
			name VARCHAR NOT NULL CHECK (name <> ''),
			vendor VARCHAR NOT NULL CHECK (vendor <> ''),
			location_id VARCHAR NOT NULL CHECK (location_id <> ''),
			service_id VARCHAR NOT NULL CHECK (service_id <> ''),
			data_model VARCHAR NOT NULL CHECK (data_model <> ''),
			status VARCHAR NOT NULL CHECK (status <> ''),
			created_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},
	30: {
		`CREATE TABLE job_results (
			id VARCHAR NOT NULL,
			pmm_agent_id VARCHAR CHECK (pmm_agent_id <> ''),
			type VARCHAR NOT NULL,
			done BOOLEAN NOT NULL,
			error VARCHAR NOT NULL,
			result JSONB,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},
	31: {
		`ALTER TABLE agents
			ADD COLUMN azure_options VARCHAR`,
	},
	32: {
		`CREATE TABLE check_settings (
			name VARCHAR NOT NULL,
			interval VARCHAR NOT NULL,
			PRIMARY KEY (name)
		)`,
	},
	33: {
		`ALTER TABLE kubernetes_clusters ADD COLUMN pxc JSONB`,
		`ALTER TABLE kubernetes_clusters ADD COLUMN proxysql JSONB`,
		`ALTER TABLE kubernetes_clusters ADD COLUMN mongod JSONB`,
	},
	34: {
		`ALTER TABLE kubernetes_clusters ADD COLUMN haproxy JSONB`,
	},
	35: {
		`CREATE TABLE restore_history (
			id VARCHAR NOT NULL,
			artifact_id VARCHAR NOT NULL CHECK (artifact_id <> ''),
			service_id VARCHAR NOT NULL CHECK (service_id <> ''),
			status VARCHAR NOT NULL CHECK (status <> ''),
			started_at TIMESTAMP NOT NULL,
			finished_at TIMESTAMP,

			PRIMARY KEY (id),
			FOREIGN KEY (artifact_id) REFERENCES artifacts (id),
			FOREIGN KEY (service_id) REFERENCES services (service_id)
		)`,
	},
	36: {
		`ALTER TABLE agents
		ADD COLUMN mysql_options VARCHAR`,
	},
	37: {
		`ALTER TABLE agents ALTER COLUMN max_query_log_size TYPE BIGINT`,
	},
	38: {
		`DELETE FROM artifacts a
			WHERE NOT EXISTS (
				SELECT FROM backup_locations
   				WHERE id = a.location_id
   			)`,
		`ALTER TABLE artifacts ADD FOREIGN KEY (location_id) REFERENCES backup_locations (id)`,
		`ALTER TABLE artifacts DROP CONSTRAINT artifacts_service_id_check`,
	},
	39: {
		`CREATE TABLE scheduled_tasks (
			id VARCHAR NOT NULL,
			cron_expression VARCHAR NOT NULL CHECK (cron_expression <> ''),
			type VARCHAR NOT NULL CHECK (type <> ''),
			start_at TIMESTAMP,
			last_run TIMESTAMP,
			next_run TIMESTAMP,
			data JSONB,
			disabled BOOLEAN,
			running BOOLEAN,
			error VARCHAR,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},
	40: {
		`ALTER TABLE artifacts
      ADD COLUMN type VARCHAR NOT NULL CHECK (type <> '') DEFAULT 'on_demand',
      ADD COLUMN schedule_id VARCHAR`,
		`ALTER TABLE artifacts ALTER COLUMN type DROP DEFAULT`,
	},
	41: {
		`ALTER TABLE agents ADD COLUMN postgresql_options JSONB`,
	},
	42: {
		`ALTER TABLE agents
		ADD COLUMN agent_password VARCHAR CHECK (agent_password <> '')`,
	},
	43: {
		`UPDATE artifacts SET schedule_id = '' WHERE schedule_id IS NULL`,
		`ALTER TABLE artifacts ALTER COLUMN schedule_id SET NOT NULL`,
	},
	44: {
		`CREATE TABLE service_software_versions (
			service_id VARCHAR NOT NULL CHECK (service_id <> ''),
			service_type VARCHAR NOT NULL CHECK (service_type <> ''),
			software_versions JSONB,
			next_check_at TIMESTAMP,

			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (service_id),
			FOREIGN KEY (service_id) REFERENCES services (service_id) ON DELETE CASCADE
		);`,
		`INSERT INTO service_software_versions(
			service_id,
			service_type,
			software_versions,
			next_check_at,
			created_at,
			updated_at
		)
		SELECT
			service_id,
			service_type,
			'[]' AS software_versions,
			(NOW() AT TIME ZONE 'utc') AS next_check_at,
			(NOW() AT TIME ZONE 'utc') AS created_at,
			(NOW() AT TIME ZONE 'utc') AS updated_at
		FROM services
        WHERE service_type = 'mysql';`,
	},
	45: {
		`ALTER TABLE artifacts
			ADD COLUMN updated_at TIMESTAMP`,
		`UPDATE artifacts SET updated_at = created_at`,
		`ALTER TABLE artifacts ALTER COLUMN updated_at SET NOT NULL`,
		`ALTER TABLE job_results RENAME TO jobs`,
		`ALTER TABLE jobs
			ADD COLUMN data JSONB,
			ADD COLUMN retries INTEGER,
			ADD COLUMN interval BIGINT,
			ADD COLUMN timeout BIGINT
		`,
	},
	46: {
		`ALTER TABLE artifacts ADD COLUMN db_version VARCHAR NOT NULL DEFAULT ''`,
		`ALTER TABLE artifacts ALTER COLUMN db_version DROP DEFAULT`,
	},
	47: {
		`CREATE TABLE job_logs (
			job_id VARCHAR NOT NULL,
			chunk_id INTEGER NOT NULL,
			data TEXT NOT NULL,
			last_chunk BOOLEAN NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE,
			PRIMARY KEY (job_id, chunk_id)
		)`,
	},
	48: {
		`ALTER TABLE artifacts
      ADD COLUMN mode VARCHAR NOT NULL CHECK (mode <> '') DEFAULT 'snapshot'`,
		`ALTER TABLE artifacts ALTER COLUMN mode DROP DEFAULT`,
		`UPDATE scheduled_tasks set data = jsonb_set(data::jsonb, '{mysql_backup, data_model}', '"physical"') WHERE type = 'mysql_backup'`,
		`UPDATE scheduled_tasks set data = jsonb_set(data::jsonb, '{mysql_backup, mode}', '"snapshot"') WHERE type = 'mysql_backup'`,
		`UPDATE scheduled_tasks set data = jsonb_set(data::jsonb, '{mongodb_backup, data_model}', '"logical"') WHERE type = 'mongodb_backup'`,
		`UPDATE scheduled_tasks set data = jsonb_set(data::jsonb, '{mongodb_backup, mode}', '"snapshot"') WHERE type = 'mongodb_backup'`,
		`UPDATE jobs SET data = jsonb_set(data::jsonb, '{mongo_db_backup, mode}', '"snapshot"') WHERE type = 'mongodb_backup'`,
		`UPDATE jobs SET data = data - 'mongo_db_backup' || jsonb_build_object('mongodb_backup', data->'mongo_db_backup') WHERE type = 'mongodb_backup';`,
		`UPDATE jobs SET data = data - 'mongo_db_restore_backup' || jsonb_build_object('mongodb_restore_backup', data->'mongo_db_restore_backup') WHERE type = 'mongodb_restore_backup';`,
	},
	49: {
		`CREATE TABLE percona_sso_details (
			client_id VARCHAR NOT NULL,
			client_secret VARCHAR NOT NULL,
			issuer_url VARCHAR NOT NULL,
			scope VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
	},
	50: {
		`INSERT INTO job_logs(
			job_id,
			chunk_id,
			data,
			last_chunk
		)
        SELECT
            id AS job_id,
            0 AS chunk_id,
            '' AS data,
            TRUE AS last_chunk
        FROM jobs j
			WHERE type = 'mongodb_backup' AND NOT EXISTS (
				SELECT FROM job_logs
				WHERE job_id = j.id
			);`,
	},
	51: {
		`ALTER TABLE services
			ADD COLUMN database_name VARCHAR NOT NULL DEFAULT ''`,
	},
	52: {
		`UPDATE services SET database_name = 'postgresql' 
			WHERE service_type = 'postgresql' and database_name = ''`,
	},
	53: {
		`UPDATE services SET database_name = 'postgres' 
			WHERE service_type = 'postgresql' and database_name = 'postgresql'`,
	},
	54: {
		`ALTER TABLE percona_sso_details
			ADD COLUMN access_token VARCHAR`,
	},
	55: {
		`DELETE FROM ia_rules`,
		`ALTER TABLE ia_rules
			RENAME COLUMN params TO params_values`,
		`ALTER TABLE ia_rules
			ADD COLUMN name VARCHAR NOT NULL,
			ADD COLUMN expr_template VARCHAR NOT NULL,
			ADD COLUMN params_definitions JSONB,
			ADD COLUMN default_for BIGINT,
			ADD COLUMN default_severity VARCHAR NOT NULL,
			ADD COLUMN labels TEXT,
			ADD COLUMN annotations TEXT`,
	},
	56: {
		`ALTER TABLE ia_templates
			DROP COLUMN tiers`,
	},
	57: {
		`ALTER TABLE percona_sso_details
			ADD COLUMN organization_id VARCHAR`,
	},
	58: {
		`UPDATE agents SET mongo_db_tls_options = jsonb_set(mongo_db_tls_options, '{stats_collections}', to_jsonb(string_to_array(mongo_db_tls_options->>'stats_collections', ',')))
			WHERE 'mongo_db_tls_options' is not null AND jsonb_typeof(mongo_db_tls_options->'stats_collections') = 'string'`,
	},
	59: {
		`DELETE FROM percona_sso_details WHERE organization_id IS NULL`,
	},
	60: {
		`ALTER TABLE percona_sso_details
			RENAME COLUMN client_id TO pmm_managed_client_id;
		ALTER TABLE percona_sso_details
			RENAME COLUMN client_secret TO pmm_managed_client_secret;
		ALTER TABLE percona_sso_details
			ADD COLUMN grafana_client_id VARCHAR NOT NULL,
			ADD COLUMN pmm_server_name VARCHAR NOT NULL,
			ALTER COLUMN organization_id SET NOT NULL`,
	},
	61: {
		`UPDATE settings SET settings = settings #- '{sass, stt_enabled}';`,
	},
	62: {
		`ALTER TABLE agents
		ADD COLUMN process_exec_path TEXT`,
	},
	63: {
		`ALTER TABLE agents
			ADD COLUMN log_level VARCHAR`,
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
	Address          string
	Name             string
	Username         string
	Password         string
	SetupFixtures    SetupFixturesMode
	MigrationVersion *int
}

// SetupDB runs PostgreSQL database migrations and optionally creates database and adds initial data.
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
	errDB := db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)

	if pErr, ok := errDB.(*pq.Error); ok && pErr.Code == "28000" {
		// invalid_authorization_specification	(see https://www.postgresql.org/docs/current/errcodes-appendix.html)
		databaseName := params.Name
		roleName := params.Username

		if params.Logf != nil {
			params.Logf("Creating database %s and role %s", databaseName, roleName)
		}
		// we use empty password/db and postgres user for creating database
		db, err := OpenDB(params.Address, "", "postgres", "")
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer db.Close() //nolint:errcheck

		var countDatabases int
		err = db.QueryRow(`SELECT COUNT(*) FROM pg_database WHERE datname = $1`, databaseName).Scan(&countDatabases)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if countDatabases == 0 {
			_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, databaseName))
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		var countRoles int
		err = db.QueryRow(`SELECT COUNT(*) FROM pg_roles WHERE rolname=$1`, roleName).Scan(&countRoles)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if countRoles == 0 {
			_, err = db.Exec(fmt.Sprintf(`CREATE USER "%s" LOGIN PASSWORD '%s'`, roleName, params.Password))
			if err != nil {
				return nil, errors.WithStack(err)
			}

			_, err = db.Exec(`GRANT ALL PRIVILEGES ON DATABASE $1 TO $2`, databaseName, roleName)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
		errDB = db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	}
	if pErr, ok := errDB.(*pq.Error); ok && pErr.Code == "42P01" { // undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
		errDB = nil
	}

	if errDB != nil {
		return nil, errors.WithStack(errDB)
	}
	if params.Logf != nil {
		params.Logf("Current database schema version: %d. Latest version: %d.", currentVersion, latestVersion)
	}

	// rollback all migrations if one of them fails; PostgreSQL supports DDL transactions
	err := db.InTransaction(func(tx *reform.TX) error {
		for version := currentVersion + 1; version <= latestVersion; version++ {
			if params.Logf != nil {
				params.Logf("Migrating database to schema version %d ...", version)
			}

			queries := databaseSchema[version]
			queries = append(queries, fmt.Sprintf(`INSERT INTO schema_migrations (id) VALUES (%d)`, version))
			for _, q := range queries {
				q = strings.TrimSpace(q)
				if _, err := tx.Exec(q); err != nil {
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
	if _, err = CreateNodeExporter(q, PMMServerAgentID, nil, false, []string{}, nil); err != nil {
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
