// Copyright (C) 2023 Percona LLC
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
// Package models provides the data models for the managed package.
//

// Package models provides functionality for handling database models and related tasks.
//
//nolint:lll
package models

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/utils/encryption"
)

const (
	// PMMServerPostgreSQLServiceName is a special Service Name representing PMM Server's PostgreSQL Service.
	PMMServerPostgreSQLServiceName = "pmm-server-postgresql"
	// - minPGVersion stands for minimal required PostgreSQL server version for PMM Server.
	minPGVersion float64 = 14
	// DefaultPostgreSQLAddr represent default local PostgreSQL database server address.
	DefaultPostgreSQLAddr = "127.0.0.1:5432"
	// PMMServerPostgreSQLNodeName is a special Node Name representing PMM Server's External PostgreSQL Node.
	PMMServerPostgreSQLNodeName = "pmm-server-db"

	// DisableSSLMode represent disable PostgreSQL ssl mode.
	DisableSSLMode string = "disable"
	// RequireSSLMode represent require PostgreSQL ssl mode.
	RequireSSLMode string = "require"
	// VerifyCaSSLMode represent verify-ca PostgreSQL ssl mode.
	VerifyCaSSLMode string = "verify-ca"
	// VerifyFullSSLMode represent verify-full PostgreSQL ssl mode.
	VerifyFullSSLMode string = "verify-full"
)

// DefaultAgentEncryptionColumnsV3 since 3.0.0 contains all tables and it's columns to be encrypted in PMM Server DB.
var DefaultAgentEncryptionColumnsV3 = []encryption.Table{
	{
		Name:        "agents",
		Identifiers: []string{"agent_id"},
		Columns: []encryption.Column{
			{Name: "username"},
			{Name: "password"},
			{Name: "agent_password"},
			{Name: "aws_options", CustomHandler: EncryptAWSOptionsHandler},
			{Name: "azure_options", CustomHandler: EncryptAzureOptionsHandler},
			{Name: "mongo_options", CustomHandler: EncryptMongoDBOptionsHandler},
			{Name: "mysql_options", CustomHandler: EncryptMySQLOptionsHandler},
			{Name: "postgresql_options", CustomHandler: EncryptPostgreSQLOptionsHandler},
		},
	},
}

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
	64: {
		`UPDATE artifacts SET data_model = 'logical'`,
	},
	65: {
		`CREATE TABLE user_flags (
			id INTEGER NOT NULL,
			tour_done BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
		)`,
	},
	66: {
		`UPDATE settings SET settings = settings #- '{ia, enabled}';`,
		`UPDATE settings SET settings = settings - 'ia' || jsonb_build_object('alerting', settings->'ia');`,
		`UPDATE ia_rules SET disabled = TRUE`,
	},
	67: {
		`UPDATE agents
		SET log_level = 'error'
		WHERE log_level = 'fatal'
		AND agent_type IN (
			'node_exporter',
			'mysqld_exporter',
			'postgres_exporter'
		);`,
	},
	68: {
		`ALTER TABLE agents
			ADD COLUMN max_query_length INTEGER NOT NULL DEFAULT 0`,

		`ALTER TABLE agents
			ALTER COLUMN max_query_length DROP DEFAULT`,
	},
	69: {
		`ALTER TABLE backup_locations
			DROP COLUMN pmm_server_config`,
	},
	70: {
		`ALTER TABLE restore_history
			ADD COLUMN pitr_timestamp TIMESTAMP`,
	},
	71: {
		`ALTER TABLE backup_locations
			RENAME COLUMN pmm_client_config TO filesystem_config`,
	},
	72: {
		`ALTER TABLE user_flags
			ADD COLUMN alerting_tour_done BOOLEAN NOT NULL DEFAULT false`,
	},
	73: {
		`CREATE TABLE roles (
			id SERIAL PRIMARY KEY,
			title VARCHAR NOT NULL UNIQUE,
			filter TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		CREATE TABLE user_roles (
			user_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (user_id, role_id)
		);

		CREATE INDEX role_id_index ON user_roles (role_id);

		WITH rows AS (
			INSERT INTO roles
			(title, filter, created_at, updated_at)
			VALUES
			('Full access', '', NOW(), NOW())
			RETURNING id
		), settings_id AS (
			UPDATE settings SET settings['default_role_id'] = (SELECT to_jsonb(id) FROM rows)
		)

		INSERT INTO user_roles
		(user_id, role_id, created_at, updated_at)
		SELECT u.id, (SELECT id FROM rows), NOW(), NOW() FROM user_flags u;`,
	},
	74: {
		`UPDATE scheduled_tasks
			SET "data" = jsonb_set("data", array["type", 'name'], to_jsonb("data"->"type"->>'name' || '-pmm-renamed-' || gen_random_uuid()))
			WHERE "data"->"type"->>'name' IN (SELECT "data"->"type"->>'name' nm FROM scheduled_tasks GROUP BY nm HAVING COUNT(*) > 1);
		CREATE UNIQUE INDEX scheduled_tasks_data_name_idx ON scheduled_tasks(("data"->"type"->>'name'))`,
	},
	75: {
		`ALTER TABLE kubernetes_clusters
            ADD COLUMN ready BOOLEAN NOT NULL DEFAULT false`,
	},
	76: {
		`ALTER TABLE roles
		ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
	},
	77: {
		`UPDATE scheduled_tasks
			SET data = jsonb_set(data, '{mongodb_backup, cluster_name}', to_jsonb((SELECT cluster FROM services WHERE services.service_id = data->'mongodb_backup'->>'service_id')))
			WHERE type = 'mongodb_backup'`,
	},
	78: {
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
        WHERE service_type = 'mongodb';`,
	},
	79: {
		`CREATE TABLE onboarding_system_tips (
			 id INTEGER PRIMARY KEY,
			 is_completed BOOLEAN NOT NULL,

			 created_at TIMESTAMP NOT NULL,
			 updated_at TIMESTAMP NOT NULL
		);

		INSERT INTO onboarding_system_tips(
			id, is_completed, created_at, updated_at
		) VALUES
			(1, false, current_timestamp, current_timestamp),
			(2, false, current_timestamp, current_timestamp),
			(3, false, current_timestamp, current_timestamp);

		CREATE TABLE onboarding_user_tips (
		   id SERIAL PRIMARY KEY,
		   tip_id INTEGER NOT NULL,
		   user_id INTEGER NOT NULL,
		   is_completed BOOLEAN NOT NULL,

		   created_at TIMESTAMP NOT NULL,
		   updated_at TIMESTAMP NOT NULL,
		   UNIQUE (user_id, tip_id)
		);
		`,
	},
	80: {
		`ALTER TABLE kubernetes_clusters ADD COLUMN postgresql JSONB`,
		`ALTER TABLE kubernetes_clusters ADD COLUMN pgbouncer JSONB`,
		`ALTER TABLE kubernetes_clusters ADD COLUMN pgbackrest JSONB`,
	},
	81: {
		`ALTER TABLE artifacts
		ADD COLUMN is_sharded_cluster BOOLEAN NOT NULL DEFAULT FALSE`,
	},
	82: {
		`ALTER TABLE artifacts
    		ADD COLUMN folder VARCHAR NOT NULL DEFAULT '',
			ADD COLUMN metadata_list JSONB;

		UPDATE scheduled_tasks
		SET data = jsonb_set(data, '{mongodb_backup, folder}', data->'mongodb_backup'->'name')
		WHERE type = 'mongodb_backup';`,
	},
	83: {
		`DROP TABLE IF EXISTS onboarding_system_tips`,
		`DROP TABLE IF EXISTS onboarding_user_tips`,
	},
	84: {
		`ALTER TABLE agents
		ADD COLUMN comments_parsing_disabled BOOLEAN NOT NULL DEFAULT TRUE`,

		`ALTER TABLE agents
		ALTER COLUMN comments_parsing_disabled DROP DEFAULT`,
	},
	85: {
		`ALTER TABLE services ADD COLUMN version VARCHAR`,
	},
	86: {
		`ALTER TABLE agents
		ADD COLUMN expose_exporter BOOLEAN NOT NULL DEFAULT TRUE;`,

		`ALTER TABLE agents
		ALTER COLUMN expose_exporter DROP DEFAULT`,
	},
	87: {
		`CREATE TABLE dumps (
			id VARCHAR NOT NULL,
			status VARCHAR NOT NULL CHECK (status <> ''),
			service_names VARCHAR[],
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			export_qan BOOLEAN NOT NULL,
			ignore_load BOOLEAN NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY (id)
			)`,

		`CREATE TABLE dump_logs (
			dump_id VARCHAR NOT NULL,
			chunk_id INTEGER NOT NULL,
			data TEXT NOT NULL,
			last_chunk BOOLEAN NOT NULL,
			FOREIGN KEY (dump_id) REFERENCES dumps (id) ON DELETE CASCADE,
			PRIMARY KEY (dump_id, chunk_id)
		)`,
	},
	88: {
		`ALTER TABLE agents ADD COLUMN metrics_resolutions JSONB`,
	},
	100: {
		`DROP TABLE kubernetes_clusters`,
	},
	101: {
		`DROP TABLE IF EXISTS ia_channels`,
		`DROP TABLE IF EXISTS ia_rules`,
		`ALTER TABLE ia_templates RENAME TO alert_rule_templates`,
		`UPDATE settings SET settings = settings #- '{alerting, email_settings}';`,
		`UPDATE settings SET settings = settings #- '{alerting, slack_settings}';`,
	},
	102: {
		`UPDATE settings SET settings = settings - 'alert_manager_url'`,
	},
	103: {
		`UPDATE settings SET settings = jsonb_insert(settings, '{alerting,enabled}', to_jsonb(NOT ((settings#>'{alerting,disabled}')::boolean))) WHERE (settings#>'{alerting,disabled}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{alerting, disabled}';`,

		`UPDATE settings SET settings = settings || jsonb_set(settings, '{updates,enabled}', to_jsonb( NOT ((settings#>'{updates,disabled}')::boolean))) WHERE (settings#>'{updates,disabled}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{updates, disabled}';`,

		`UPDATE settings SET settings = settings || jsonb_set(settings, '{telemetry,enabled}', to_jsonb( NOT ((settings#>'{telemetry,disabled}')::boolean))) WHERE (settings#>'{telemetry,disabled}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{telemetry, disabled}';`,

		`UPDATE settings SET settings = settings || jsonb_set(settings, '{backup_management,enabled}', to_jsonb( NOT ((settings#>'{backup_management,disabled}')::boolean))) WHERE (settings#>'{backup_management,disabled}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{backup_management, disabled}';`,

		`UPDATE settings SET settings = settings || jsonb_set(settings, '{sass,enabled}', to_jsonb( NOT ((settings#>'{sass,stt_disabled}')::boolean))) WHERE (settings#>'{sass,stt_disabled}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{sass, stt_disabled}';`,
	},
	104: {
		`UPDATE settings SET settings = settings || jsonb_set(settings, '{sass,disabled_advisors}', settings#>'{sass,disabled_stt_checks}') WHERE (settings#>'{sass,disabled_stt_checks}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{sass,disabled_stt_checks}';`,

		`UPDATE settings SET settings = settings || jsonb_set(settings, '{sass,advisor_run_intervals}', settings#>'{sass,stt_check_intervals}') WHERE (settings#>'{sass,disabled_stt_checks}') IS NOT NULL`,
		`UPDATE settings SET settings = settings #- '{sass,stt_check_intervals}';`,
	},
	105: {
		`ALTER TABLE agents DROP CONSTRAINT agents_node_id_fkey;`,
		`ALTER TABLE agents DROP CONSTRAINT agents_pmm_agent_id_fkey;`,
		`ALTER TABLE agents DROP CONSTRAINT agents_runs_on_node_id_fkey;`,
		`ALTER TABLE agents DROP CONSTRAINT agents_service_id_fkey;`,
		`ALTER TABLE artifacts DROP CONSTRAINT artifacts_location_id_fkey;`,
		`ALTER TABLE dump_logs DROP CONSTRAINT dump_logs_dump_id_fkey;`,
		`ALTER TABLE job_logs DROP CONSTRAINT job_logs_job_id_fkey;`,
		`ALTER TABLE restore_history DROP CONSTRAINT restore_history_artifact_id_fkey;`,
		`ALTER TABLE restore_history DROP CONSTRAINT restore_history_service_id_fkey;`,
		`ALTER TABLE service_software_versions DROP CONSTRAINT service_software_versions_service_id_fkey;`,
		`ALTER TABLE services DROP CONSTRAINT services_node_id_fkey;`,

		`UPDATE action_results SET id = SUBSTRING(id, 12) WHERE id LIKE '/action_id/%';`,
		`UPDATE action_results SET pmm_agent_id = SUBSTRING(pmm_agent_id, 11) WHERE pmm_agent_id LIKE '/agent_id/%';`,

		`UPDATE agents SET agent_id = SUBSTRING(agent_id, 11) WHERE agent_id LIKE '/agent_id/%';`,
		`UPDATE agents SET pmm_agent_id = SUBSTRING(pmm_agent_id, 11) WHERE pmm_agent_id LIKE '/agent_id/%';`,
		`UPDATE agents SET runs_on_node_id = SUBSTRING(runs_on_node_id, 10) WHERE runs_on_node_id LIKE '/node_id/%';`,
		`UPDATE agents SET node_id = SUBSTRING(node_id, 10) WHERE node_id LIKE '/node_id/%';`,
		`UPDATE agents SET service_id = SUBSTRING(service_id, 13) WHERE service_id LIKE '/service_id/%';`,

		`UPDATE artifacts SET id = SUBSTRING(id, 14) WHERE id LIKE '/artifact_id/%';`,
		`UPDATE artifacts SET location_id = SUBSTRING(location_id, 14) WHERE location_id LIKE '/location_id/%';`,
		`UPDATE artifacts SET service_id = SUBSTRING(service_id, 13) WHERE service_id LIKE '/service_id/%';`,
		`UPDATE artifacts SET schedule_id = SUBSTRING(schedule_id, 20) WHERE schedule_id LIKE '/scheduled_task_id/%';`,

		`UPDATE backup_locations SET id = SUBSTRING(id, 14) WHERE id LIKE '/location_id/%';`,

		`UPDATE job_logs SET job_id = SUBSTRING(job_id, 9) WHERE job_id LIKE '/job_id/%';`,

		`UPDATE jobs SET id = SUBSTRING(id, 9) WHERE id LIKE '/job_id/%';`,
		`UPDATE jobs SET pmm_agent_id = SUBSTRING(pmm_agent_id, 11) WHERE pmm_agent_id LIKE '/agent_id/%';`,

		`UPDATE nodes SET node_id = SUBSTRING(node_id, 10) WHERE node_id LIKE '/node_id/%';`,
		`UPDATE nodes SET machine_id = SUBSTRING(machine_id, 13) WHERE machine_id LIKE '/machine_id/%';`,

		`UPDATE restore_history SET id = SUBSTRING(id, 13) WHERE id LIKE '/restore_id/%';`,
		`UPDATE restore_history SET artifact_id = SUBSTRING(artifact_id, 14) WHERE artifact_id LIKE '/artifact_id/%';`,
		`UPDATE restore_history SET service_id = SUBSTRING(service_id, 13) WHERE service_id LIKE '/service_id/%';`,

		`UPDATE scheduled_tasks SET id = SUBSTRING(id, 20) WHERE id LIKE '/scheduled_task_id/%';`,

		`UPDATE service_software_versions SET service_id = SUBSTRING(service_id, 13) WHERE service_id LIKE '/service_id/%';`,

		`UPDATE services SET service_id = SUBSTRING(service_id, 13) WHERE service_id LIKE '/service_id/%';`,
		`UPDATE services SET node_id = SUBSTRING(node_id, 10) WHERE node_id LIKE '/node_id/%';`,

		`ALTER TABLE agents ADD CONSTRAINT agents_node_id_fkey FOREIGN KEY (node_id) REFERENCES nodes (node_id);`,
		`ALTER TABLE agents ADD CONSTRAINT agents_pmm_agent_id_fkey FOREIGN KEY (pmm_agent_id) REFERENCES agents (agent_id);`,
		`ALTER TABLE agents ADD CONSTRAINT agents_runs_on_node_id_fkey FOREIGN KEY (runs_on_node_id) REFERENCES nodes (node_id);`,
		`ALTER TABLE agents ADD CONSTRAINT agents_service_id_fkey FOREIGN KEY (service_id) REFERENCES services (service_id);`,
		`ALTER TABLE artifacts ADD CONSTRAINT artifacts_location_id_fkey FOREIGN KEY (location_id) REFERENCES backup_locations (id);`,
		`ALTER TABLE dump_logs ADD CONSTRAINT dump_logs_dump_id_fkey FOREIGN KEY (dump_id) REFERENCES dumps (id) ON DELETE CASCADE;`,
		`ALTER TABLE job_logs ADD CONSTRAINT job_logs_job_id_fkey FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE;`,
		`ALTER TABLE restore_history ADD CONSTRAINT restore_history_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES artifacts (id);`,
		`ALTER TABLE restore_history ADD CONSTRAINT restore_history_service_id_fkey FOREIGN KEY (service_id) REFERENCES services (service_id);`,
		`ALTER TABLE service_software_versions ADD CONSTRAINT service_software_versions_service_id_fkey FOREIGN KEY (service_id) REFERENCES services (service_id) ON DELETE CASCADE;`,
		`ALTER TABLE services ADD CONSTRAINT services_node_id_fkey FOREIGN KEY (node_id) REFERENCES nodes (node_id);`,
	},
	106: {
		`ALTER TABLE user_flags
			ADD COLUMN snoozed_pmm_version VARCHAR NOT NULL DEFAULT ''`,
	},
	107: {
		`ALTER TABLE agents ADD COLUMN exporter_options JSONB`,
		`UPDATE agents SET exporter_options = '{}'::jsonb`,
		`ALTER TABLE agents ADD COLUMN qan_options JSONB`,
		`UPDATE agents SET qan_options = '{}'::jsonb`,
		`ALTER TABLE agents ADD COLUMN aws_options JSONB`,
		`UPDATE agents SET aws_options = '{}'::jsonb`,

		`ALTER TABLE agents ALTER COLUMN azure_options TYPE JSONB USING to_jsonb(azure_options)`,
		`UPDATE agents SET azure_options = '{}'::jsonb WHERE azure_options IS NULL`,
		`ALTER TABLE agents ALTER COLUMN mysql_options TYPE JSONB USING to_jsonb(mysql_options)`,
		`UPDATE agents SET mysql_options = '{}'::jsonb WHERE mysql_options IS NULL`,

		`ALTER TABLE agents RENAME COLUMN mongo_db_tls_options TO mongo_options`,
		`UPDATE agents SET mongo_options = '{}'::jsonb WHERE mongo_options IS NULL`,

		`UPDATE agents SET postgresql_options = '{}'::jsonb WHERE postgresql_options IS NULL`,

		`UPDATE agents SET exporter_options['expose_exporter'] = to_jsonb(expose_exporter)`,
		`UPDATE agents SET exporter_options['push_metrics'] = to_jsonb(push_metrics)`,
		`UPDATE agents SET exporter_options['disabled_collectors'] = to_jsonb(disabled_collectors)`,
		`UPDATE agents SET exporter_options['metrics_resolutions'] = to_jsonb(metrics_resolutions)`,
		`UPDATE agents SET exporter_options['metrics_path'] = to_jsonb(metrics_path)`,
		`UPDATE agents SET exporter_options['metrics_scheme'] = to_jsonb(metrics_scheme)`,

		`ALTER TABLE agents DROP COLUMN expose_exporter`,
		`ALTER TABLE agents DROP COLUMN push_metrics`,
		`ALTER TABLE agents DROP COLUMN disabled_collectors`,
		`ALTER TABLE agents DROP COLUMN metrics_resolutions`,
		`ALTER TABLE agents DROP COLUMN metrics_path`,
		`ALTER TABLE agents DROP COLUMN metrics_scheme`,

		`UPDATE agents SET qan_options['max_query_length'] = to_jsonb(max_query_length)`,
		`UPDATE agents SET qan_options['max_query_log_size'] = to_jsonb(max_query_log_size)`,
		`UPDATE agents SET qan_options['query_examples_disabled'] = to_jsonb(query_examples_disabled)`,
		`UPDATE agents SET qan_options['comments_parsing_disabled'] = to_jsonb(comments_parsing_disabled)`,
		`ALTER TABLE agents DROP COLUMN max_query_length`,
		`ALTER TABLE agents DROP COLUMN max_query_log_size`,
		`ALTER TABLE agents DROP COLUMN query_examples_disabled`,
		`ALTER TABLE agents DROP COLUMN comments_parsing_disabled`,

		`UPDATE agents SET aws_options['aws_access_key'] = to_jsonb(aws_access_key);`,
		`UPDATE agents SET aws_options['aws_secret_key'] = to_jsonb(aws_secret_key);`,
		`UPDATE agents SET aws_options['rds_basic_metrics_disabled'] = to_jsonb(rds_basic_metrics_disabled);`,
		`UPDATE agents SET aws_options['rds_enhanced_metrics_disabled'] = to_jsonb(rds_enhanced_metrics_disabled);`,
		`ALTER TABLE agents DROP COLUMN aws_access_key`,
		`ALTER TABLE agents DROP COLUMN aws_secret_key`,
		`ALTER TABLE agents DROP COLUMN rds_basic_metrics_disabled`,
		`ALTER TABLE agents DROP COLUMN rds_enhanced_metrics_disabled`,

		`UPDATE agents SET mysql_options['table_count'] = to_jsonb(table_count);`,
		`UPDATE agents SET mysql_options['table_count_tablestats_group_limit'] = to_jsonb(table_count_tablestats_group_limit);`,
		`ALTER TABLE agents DROP COLUMN table_count`,
		`ALTER TABLE agents DROP COLUMN table_count_tablestats_group_limit`,
	},
	108: {
		`ALTER TABLE user_flags
			ADD COLUMN snoozed_api_keys_migration BOOLEAN NOT NULL DEFAULT false`,
	},
	109: {
		`ALTER TABLE agents ADD COLUMN valkey_options JSONB`,
		`UPDATE agents SET valkey_options = '{}'::jsonb`,
	},
}

// ^^^ Avoid default values in schema definition. ^^^
// Go's zero values and non-zero default values in database do play nicely together in INSERTs and UPDATEs.

// OpenDB returns configured connection pool for PostgreSQL.
// OpenDB just validates its arguments without creating a connection to the database.
func OpenDB(params SetupDBParams) (*sql.DB, error) {
	q := make(url.Values)
	if params.SSLMode == "" {
		params.SSLMode = DisableSSLMode
	}

	q.Set("sslmode", params.SSLMode)
	if params.SSLMode != DisableSSLMode {
		q.Set("sslrootcert", params.SSLCAPath)
		q.Set("sslcert", params.SSLCertPath)
		q.Set("sslkey", params.SSLKeyPath)
	}

	uri := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(params.Username, params.Password),
		Host:     params.Address,
		Path:     params.Name,
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
	SSLMode          string
	SSLCAPath        string
	SSLKeyPath       string
	SSLCertPath      string
	SetupFixtures    SetupFixturesMode
	MigrationVersion *int
}

// SetupDB checks minimal required PostgreSQL version and runs database migrations. Optionally creates database and adds initial data.
func SetupDB(ctx context.Context, sqlDB *sql.DB, params SetupDBParams) (*reform.DB, error) {
	var logger reform.Logger
	if params.Logf != nil {
		logger = reform.NewPrintfLogger(params.Logf)
	}

	db := reform.NewDB(sqlDB, postgresql.Dialect, logger)
	errCV := checkVersion(ctx, db)
	if pErr, ok := errCV.(*pq.Error); ok && pErr.Code == "28000" { //nolint:errorlint
		// invalid_authorization_specification	(see https://www.postgresql.org/docs/current/errcodes-appendix.html)
		if err := initWithRoot(params); err != nil {
			return nil, errors.Wrapf(err, "couldn't connect to database with provided credentials. Tried to create user and database. Error: %s", errCV)
		}
		errCV = checkVersion(ctx, db)
	}

	if errCV != nil {
		return nil, errCV
	}

	if err := migrateDB(db, params); err != nil {
		return nil, err
	}

	return db, nil
}

// EncryptDB encrypts a set of columns in a specific database and table.
func EncryptDB(tx *reform.TX, database string, itemsToEncrypt []encryption.Table) error {
	return dbEncryption(tx, database, itemsToEncrypt, encryption.EncryptItems, true)
}

// DecryptDB decrypts a set of columns in a specific database and table.
func DecryptDB(tx *reform.TX, database string, itemsToEncrypt []encryption.Table) error {
	return dbEncryption(tx, database, itemsToEncrypt, encryption.DecryptItems, false)
}

func dbEncryption(tx *reform.TX, database string, items []encryption.Table,
	encryptionHandler func(tx *reform.TX, tables []encryption.Table) error,
	expectedState bool,
) error {
	if len(items) == 0 {
		return nil
	}

	settings, err := GetSettings(tx)
	if err != nil {
		return err
	}
	currentColumns := make(map[string]bool)
	for _, v := range settings.EncryptedItems {
		currentColumns[v] = true
	}

	tables := []encryption.Table{}
	prepared := []string{}
	for _, table := range items {
		columns := []encryption.Column{}
		for _, column := range table.Columns {
			dbTableColumn := fmt.Sprintf("%s.%s.%s", database, table.Name, column.Name)
			if currentColumns[dbTableColumn] == expectedState {
				continue
			}

			columns = append(columns, column)
			prepared = append(prepared, dbTableColumn)
		}
		if len(columns) == 0 {
			continue
		}

		table.Columns = columns
		tables = append(tables, table)
	}
	if len(tables) == 0 {
		return nil
	}

	err = encryptionHandler(tx, tables)
	if err != nil {
		return err
	}

	encryptedItems := []string{}
	if expectedState {
		encryptedItems = prepared
	}

	_, err = UpdateSettings(tx, &ChangeSettingsParams{
		EncryptedItems: encryptedItems,
	})
	if err != nil {
		return err
	}

	return nil
}

// checkVersion checks minimal required PostgreSQL server version.
func checkVersion(ctx context.Context, db reform.DBTXContext) error {
	PGVersion, err := GetPostgreSQLVersion(ctx, db)
	if err != nil {
		return err
	}

	if PGVersion.Float() < minPGVersion {
		return fmt.Errorf("unsupported PMM Server PostgreSQL server version: %s. Please upgrade to version %.1f or newer", PGVersion, minPGVersion)
	}
	return nil
}

// initWithRoot tries to create given user and database under default postgres role.
func initWithRoot(params SetupDBParams) error {
	if params.Logf != nil {
		params.Logf("Creating database %s and role %s", params.Name, params.Username)
	}
	// we use empty password/db and postgres user for creating database
	db, err := OpenDB(SetupDBParams{Address: params.Address, Username: "postgres"})
	if err != nil {
		return errors.WithStack(err)
	}
	defer db.Close() //nolint:errcheck

	var countDatabases int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_database WHERE datname = $1`, params.Name).Scan(&countDatabases)
	if err != nil {
		return errors.WithStack(err)
	}

	if countDatabases == 0 {
		_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, params.Name))
		if err != nil {
			return errors.WithStack(err)
		}
	}

	var countRoles int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_roles WHERE rolname=$1`, params.Username).Scan(&countRoles)
	if err != nil {
		return errors.WithStack(err)
	}

	if countRoles == 0 {
		_, err = db.Exec(fmt.Sprintf(`CREATE USER "%s" LOGIN PASSWORD '%s'`, params.Username, params.Password))
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = db.Exec(`GRANT ALL PRIVILEGES ON DATABASE $1 TO $2`, params.Name, params.Username)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// migrateDB runs PostgreSQL database migrations.
func migrateDB(db *reform.DB, params SetupDBParams) error {
	var currentVersion int
	errDB := db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	// undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
	if pErr, ok := errDB.(*pq.Error); ok && pErr.Code == "42P01" { //nolint:errorlint
		errDB = nil
	}
	if errDB != nil {
		return errors.WithStack(errDB)
	}

	latestVersion := len(databaseSchema) - 1 // skip item 0
	if params.MigrationVersion != nil {
		latestVersion = *params.MigrationVersion
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
				if _, err := tx.Exec(q); err != nil {
					return errors.Wrapf(err, "failed to execute statement:\n%s", q)
				}
			}
		}

		if params.SetupFixtures == SkipFixtures {
			return nil
		}

		err := EncryptDB(tx, params.Name, DefaultAgentEncryptionColumnsV3)
		if err != nil {
			return err
		}

		// fill settings with defaults
		s, err := GetSettings(tx)
		if err != nil {
			return err
		}
		if err = SaveSettings(tx, s); err != nil {
			return err
		}

		err = setupPMMServerAgents(tx.Querier, params)
		if err != nil {
			return err
		}

		return nil
	})
}

func setupPMMServerAgents(q *reform.Querier, params SetupDBParams) error {
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
	if _, err = CreateNodeExporter(q, PMMServerAgentID, nil, false, false, []string{}, nil, ""); err != nil {
		return err
	}
	address, port, err := parsePGAddress(params.Address)
	if err != nil {
		return err
	}
	if params.Address != DefaultPostgreSQLAddr {
		if node, err = CreateNode(q, RemoteNodeType, &CreateNodeParams{
			NodeName: PMMServerPostgreSQLNodeName,
			Address:  address,
		}); err != nil {
			return err
		}
	} else {
		params.Name = "" // using postgres database in order to get metrics from entrypoint extension setup for QAN.
	}

	// create PostgreSQL Service and associated Agents
	service, err := AddNewService(q, PostgreSQLServiceType, &AddDBMSServiceParams{
		ServiceName: PMMServerPostgreSQLServiceName,
		NodeID:      node.NodeID,
		Database:    params.Name,
		Address:     &node.Address,
		Port:        &port,
	})
	if err != nil {
		return err
	}

	ap := &CreateAgentParams{
		PMMAgentID:    PMMServerAgentID,
		ServiceID:     service.ServiceID,
		TLS:           params.SSLMode != DisableSSLMode,
		TLSSkipVerify: params.SSLMode == DisableSSLMode || params.SSLMode == VerifyCaSSLMode,
		Username:      params.Username,
		Password:      params.Password,
		QANOptions: QANOptions{
			CommentsParsingDisabled: true,
		},
	}
	if ap.TLS {
		ap.PostgreSQLOptions = PostgreSQLOptions{}
		for path, field := range map[string]*string{
			params.SSLCAPath:   &ap.PostgreSQLOptions.SSLCa,
			params.SSLCertPath: &ap.PostgreSQLOptions.SSLCert,
			params.SSLKeyPath:  &ap.PostgreSQLOptions.SSLKey,
		} {
			if path == "" {
				continue
			}
			content, err := os.ReadFile(path) //nolint:gosec
			if err != nil {
				return err
			}
			*field = string(content)
		}
	}
	_, err = CreateAgent(q, PostgresExporterType, ap)
	if err != nil {
		return err
	}
	_, err = CreateAgent(q, QANPostgreSQLPgStatementsAgentType, ap)
	if err != nil {
		return err
	}
	return nil
}

// parsePGAddress parses PostgreSQL address into address:port; if no port specified returns default port number.
func parsePGAddress(address string) (string, uint16, error) {
	if !strings.Contains(address, ":") {
		return address, 5432, nil
	}
	address, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	parsedPort, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return "", 0, err
	}
	return address, uint16(parsedPort), nil
}
