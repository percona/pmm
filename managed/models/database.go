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
// Package models provides the data models for the managed package and
// the schema-migration machinery used by pmm-managed at startup.
//
//nolint:lll
package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/env"
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
	// DefaultSnoozeDuration represents duration for which an update is snoozed (default = 7 days).
	DefaultSnoozeDuration time.Duration = 7 * 24 * time.Hour
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
	{
		// ADRE LLM model list; api_key is the LLM provider key. Identified by the unique VARCHAR name
		// so the BIGINT id is never scanned by the column encryptor.
		Name:        "adre_models",
		Identifiers: []string{"name"},
		Columns: []encryption.Column{
			{Name: "api_key"},
		},
	},
	{
		// ADRE provisioning singleton: minted/integration secrets. Identified by the BOOLEAN id
		// (supported by prepareRowPointers as an identifier-only type).
		Name:        "adre_provisioning",
		Identifiers: []string{"id"},
		Columns: []encryption.Column{
			{Name: "holmes_api_key"},
			{Name: "pmm_sa_token"},
			{Name: "servicenow_api_key"},
			{Name: "servicenow_client_token"},
			{Name: "slack_bot_token"},
			{Name: "slack_app_token"},
			{Name: "alert_webhook_secret"},
		},
	},
}

// databaseSchema maps schema version from schema_migrations table (id column) to a slice of DDL queries.
//
//nolint:lll
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
		`ALTER TABLE user_flags DROP COLUMN snoozed_api_keys_migration`,
	},
	110: {
		`ALTER TABLE nodes ADD COLUMN instance_id VARCHAR NOT NULL DEFAULT ''`,
		`UPDATE nodes SET instance_id = address WHERE instance_id = ''`,
	},
	111: {
		`ALTER TABLE agents ADD COLUMN valkey_options JSONB`,
		`UPDATE agents SET valkey_options = '{}'::jsonb`,
	},
	112: {
		`UPDATE agents SET disabled = true WHERE agent_type = 'qan-postgresql-pgstatements-agent' AND service_id = (SELECT service_id FROM services WHERE service_name = 'pmm-server-postgresql' LIMIT 1);`,
	},
	113: {
		// Reset product tour for new navigation
		`UPDATE user_flags SET tour_done = false;

		ALTER TABLE user_flags
			ADD COLUMN snoozed_at TIMESTAMP,
			ADD COLUMN snooze_count INTEGER NOT NULL DEFAULT 0;

		UPDATE settings
			SET settings = settings || '{"updates": {"snooze_duration": ` + strconv.FormatInt(DefaultSnoozeDuration.Nanoseconds(), 10) + `}}'
			WHERE settings->'updates' IS NULL
			OR settings->'updates'->'snooze_duration' IS NULL`,
	},
	114: {
		`ALTER TABLE agents ADD COLUMN environment_variables TEXT`,
	},
	115: {
		`ALTER TABLE agents ADD COLUMN is_connected BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE nodes ADD COLUMN is_pmm_server_node BOOLEAN NOT NULL DEFAULT false`,
	},
	116: {
		`ALTER TABLE agents ADD COLUMN rta_options JSONB`,
		`UPDATE agents SET rta_options = '{}'::jsonb`,
	},
	127: {
		`CREATE TABLE log_parser_presets (
			id VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			description TEXT,
			operator_yaml TEXT NOT NULL,
			built_in BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (id),
			UNIQUE (name)
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'00000000-0000-4000-8000-000000000001',
			'mysql_error',
			'MySQL 8 error log format (timestamp thread_id [Subsystem] [CODE] [Component] message)',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z) (?P<thread_id>\d+) \[(?P<subsystem>[^\]]+)\] \[(?P<code>[^\]]+)\] \[(?P<component>[^\]]+)\] (?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006-01-02T15:04:05.000000Z'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.subsystem
  preset: none
  mapping:
    System: info
    Warning: warn
    Error: error
- type: move
  from: attributes.message
  to: body
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
	},
	128: {
		// Server log presets: used by pmm-agent on the server (and optionally elsewhere) for nginx, grafana, pmm-managed, pmm-agent, postgres.
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'nginx_access',
			'nginx_access',
			'Nginx access log (logfmt: time=..., host=..., status=...)',
			$yaml$- type: key_value_parser
  parse_from: body
  parse_to: attributes
  pair_delimiter: " "
  key_value_delimiter: "="
- type: time_parser
  parse_from: attributes.time
  layout: '2006-01-02T15:04:05Z07:00'
  layout_type: gotime
- type: add
  field: attributes.level
  value: 'EXPR(int(attributes.status) >= 500 ? "error" : (int(attributes.status) >= 400 ? "warn" : "info"))'
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    info: info
    warn: warn
    error: error
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'nginx_error',
			'nginx_error',
			'Nginx error log (timestamp [level] pid#tid: message)',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(?P<level>\w+)\] (?P<pid>\d+)#(?P<tid>\d+): (?P<message>.*?)(?:, client: (?P<client>[^,]+))?(?:, server: (?P<server>[^,]+))?(?:, request: "(?P<request>[^"]*)")?(?:, host: "(?P<host>[^"]*)")?.*'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006/01/02 15:04:05'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    debug: debug
    info: info
    notice: info
    warn: warn
    error: error
    crit: fatal
    alert: fatal
    emerg: fatal
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'grafana',
			'grafana',
			'Grafana log (logfmt, time with 9 fractional digits)',
			$yaml$- type: key_value_parser
  parse_from: body
  parse_to: attributes
  pair_delimiter: " "
  key_value_delimiter: "="
- type: time_parser
  parse_from: attributes.t
  layout: '2006-01-02T15:04:05.000000000Z07:00'
  layout_type: gotime
  on_error: drop
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    debug: debug
    info: info
    warn: warn
    error: error
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'pmm_managed',
			'pmm_managed',
			'PMM-managed log (logfmt)',
			$yaml$- type: key_value_parser
  parse_from: body
  parse_to: attributes
  pair_delimiter: " "
  key_value_delimiter: "="
- type: time_parser
  parse_from: attributes.time
  layout: '2006-01-02T15:04:05.000Z07:00'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    debug: debug
    info: info
    warning: warn
    warn: warn
    error: error
    fatal: fatal
    panic: fatal
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'pmm_agent',
			'pmm_agent',
			'PMM-agent log (logfmt)',
			$yaml$- type: key_value_parser
  parse_from: body
  parse_to: attributes
  pair_delimiter: " "
  key_value_delimiter: "="
- type: time_parser
  parse_from: attributes.time
  layout: '2006-01-02T15:04:05.000Z07:00'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    debug: debug
    info: info
    warning: warn
    warn: warn
    error: error
    fatal: fatal
    panic: fatal
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'postgres',
			'postgres',
			'PostgreSQL log (timestamp UTC [pid] level: message)',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+) UTC \[(?P<pid>\d+)\] (?P<level>\w+):\s*(?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006-01-02 15:04:05.000 UTC'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    debug: debug
    info: info
    notice: info
    warning: warn
    warn: warn
    error: error
    fatal: fatal
    panic: fatal
    LOG: info
    STATEMENT: info
- type: move
  from: attributes.message
  to: body
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
	},
	129: {
		// Additional PMM server log presets: clickhouse, otel-collector, supervisord.
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'clickhouse_server',
			'clickhouse_server',
			'ClickHouse server log (timestamp [pid] {} <Level> message)',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}\.\d+) .*? <(?P<level>\w+)> (?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006.01.02 15:04:05.000000'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    Trace: debug
    Debug: debug
    Information: info
    Warning: warn
    Error: error
- type: move
  from: attributes.message
  to: body
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'otel_collector',
			'otel_collector',
			'OTEL Collector log (tab-separated: timestamp, level, message)',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>[\d\-T:Z\.]+)\t(?P<level>\w+)\t(?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006-01-02T15:04:05.000Z'
  layout_type: gotime
  on_error: drop
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    debug: debug
    info: info
    warn: warn
    error: error
- type: move
  from: attributes.message
  to: body
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'supervisord',
			'supervisord',
			'Supervisord log (timestamp,ms LEVEL message)',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d+) (?P<level>\w+) (?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006-01-02 15:04:05,000'
  layout_type: gotime
- type: severity_parser
  parse_from: attributes.level
  preset: none
  mapping:
    INFO: info
    WARN: warn
    ERROR: error
    CRIT: fatal
- type: move
  from: attributes.message
  to: body
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
	},
	130: {
		// PMM AI Investigations: persistent incident reports with blocks, comments, chat.
		`CREATE TABLE investigations (
			id VARCHAR NOT NULL,
			title VARCHAR NOT NULL,
			status VARCHAR NOT NULL,
			severity VARCHAR NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			created_by VARCHAR NOT NULL DEFAULT '',
			time_from TIMESTAMPTZ NOT NULL,
			time_to TIMESTAMPTZ NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			summary_detailed TEXT NOT NULL DEFAULT '',
			root_cause_summary TEXT NOT NULL DEFAULT '',
			resolution_summary TEXT NOT NULL DEFAULT '',
			source_type VARCHAR NOT NULL DEFAULT 'manual',
			source_ref VARCHAR NOT NULL DEFAULT '',
			tags JSONB,
			config JSONB,
			PRIMARY KEY (id)
		)`,
		`CREATE TABLE investigation_blocks (
			id VARCHAR NOT NULL,
			investigation_id VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			title VARCHAR NOT NULL DEFAULT '',
			position INTEGER NOT NULL,
			config_json JSONB,
			data_json JSONB,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			created_by VARCHAR NOT NULL DEFAULT '',
			updated_by VARCHAR NOT NULL DEFAULT '',
			PRIMARY KEY (id),
			FOREIGN KEY (investigation_id) REFERENCES investigations (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE investigation_artifacts (
			id VARCHAR NOT NULL,
			investigation_id VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			uri_or_blob_ref TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			metadata_json JSONB,
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (id),
			FOREIGN KEY (investigation_id) REFERENCES investigations (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE investigation_messages (
			id VARCHAR NOT NULL,
			investigation_id VARCHAR NOT NULL,
			role VARCHAR NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			tool_name VARCHAR NOT NULL DEFAULT '',
			tool_result_json JSONB,
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (id),
			FOREIGN KEY (investigation_id) REFERENCES investigations (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE investigation_comments (
			id VARCHAR NOT NULL,
			investigation_id VARCHAR NOT NULL,
			block_id VARCHAR,
			anchor_json JSONB,
			author VARCHAR NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (id),
			FOREIGN KEY (investigation_id) REFERENCES investigations (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE investigation_timeline_events (
			id VARCHAR NOT NULL,
			investigation_id VARCHAR NOT NULL,
			event_time TIMESTAMPTZ NOT NULL,
			type VARCHAR NOT NULL DEFAULT '',
			title VARCHAR NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			source VARCHAR NOT NULL DEFAULT '',
			metadata_json JSONB,
			PRIMARY KEY (id),
			FOREIGN KEY (investigation_id) REFERENCES investigations (id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_investigation_blocks_investigation_id ON investigation_blocks (investigation_id)`,
		`CREATE INDEX idx_investigation_artifacts_investigation_id ON investigation_artifacts (investigation_id)`,
		`CREATE INDEX idx_investigation_messages_investigation_id ON investigation_messages (investigation_id)`,
		`CREATE INDEX idx_investigation_comments_investigation_id ON investigation_comments (investigation_id)`,
		`CREATE INDEX idx_investigation_timeline_events_investigation_id ON investigation_timeline_events (investigation_id)`,
	},
	131: {
		`ALTER TABLE investigations ADD COLUMN IF NOT EXISTS servicenow_ticket_id VARCHAR NOT NULL DEFAULT ''`,
	},
	132: {
		`ALTER TABLE investigations ADD COLUMN IF NOT EXISTS servicenow_ticket_number VARCHAR NOT NULL DEFAULT ''`,
	},
	133: {
		`CREATE TABLE IF NOT EXISTS qan_insights_cache (
			id VARCHAR NOT NULL,
			query_id VARCHAR NOT NULL,
			service_id VARCHAR NOT NULL,
			fingerprint VARCHAR NOT NULL DEFAULT '',
			time_from VARCHAR NOT NULL DEFAULT '',
			time_to VARCHAR NOT NULL DEFAULT '',
			analysis TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_qan_insights_cache_query_service ON qan_insights_cache (query_id, service_id)`,
	},
	134: {
		// Syslog / journal-style line: ISO8601 host tag[pid]: message (mysqld[8794], (mysqled)[9049], systemd[1], CRON[123], …).
		`INSERT INTO log_parser_presets (id, name, description, operator_yaml, built_in, created_at, updated_at) VALUES (
			'syslog_mysql_systemd',
			'syslog_mysql_systemd',
			'Syslog/journal line: ISO8601 host tag[pid]: message (e.g. mysqld[8794], systemd[1], (mysqled)[9049], CRON[1])',
			$yaml$- type: regex_parser
  regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2}))\s+(?P<host>\S+)\s+(?P<tag>\S+):\s*(?P<message>.*)$'
  parse_from: body
  parse_to: attributes
- type: time_parser
  parse_from: attributes.timestamp
  layout: '2006-01-02T15:04:05.999999999Z07:00'
  layout_type: gotime
  on_error: send
- type: move
  from: attributes.message
  to: body
$yaml$,
			true,
			NOW(),
			NOW()
		)`,
	},
	135: {
		`DROP TABLE IF EXISTS percona_sso_details`,
	},
	118: {
		`ALTER TABLE dumps ADD COLUMN encrypted boolean NOT NULL DEFAULT false`,
		`UPDATE dumps SET encrypted = false`,
	},
	136: {
		`CREATE TABLE adre_conversations (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			title VARCHAR(50) NOT NULL DEFAULT 'New chat',
			created_by VARCHAR NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
		)`,
		`CREATE INDEX idx_adre_conversations_created_by_last_msg ON adre_conversations (created_by, last_message_at DESC)`,
		`CREATE TABLE adre_messages (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			conversation_id BIGINT NOT NULL REFERENCES adre_conversations(id) ON DELETE CASCADE,
			role VARCHAR NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			tool_name VARCHAR NOT NULL DEFAULT '',
			tool_result_json JSONB,
			model VARCHAR NOT NULL DEFAULT '',
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			content_tsv tsvector GENERATED ALWAYS AS (
				to_tsvector('simple', coalesce(content,'') || ' ' || coalesce(tool_result_json::text,''))
			) STORED
		)`,
		`CREATE INDEX idx_adre_messages_conv_created ON adre_messages (conversation_id, created_at)`,
		`CREATE INDEX idx_adre_messages_content_tsv ON adre_messages USING gin (content_tsv)`,
	},
	137: {
		`CREATE TABLE holmes_usage_events (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			feature VARCHAR NOT NULL,
			feature_ref VARCHAR NOT NULL DEFAULT '',
			adre_conversation_id BIGINT,
			investigation_id VARCHAR NOT NULL DEFAULT '',
			model VARCHAR NOT NULL DEFAULT '',
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER,
			cached_tokens INTEGER,
			total_cost NUMERIC(14, 8),
			cost_prompt NUMERIC(14, 8),
			cost_completion NUMERIC(14, 8),
			cost_cached NUMERIC(14, 8),
			latency_ms INTEGER,
			triggered_by VARCHAR NOT NULL DEFAULT '',
			stream BOOLEAN NOT NULL DEFAULT FALSE,
			metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
		)`,
		`CREATE INDEX idx_holmes_usage_created_at ON holmes_usage_events (created_at DESC)`,
		`CREATE INDEX idx_holmes_usage_feature_created ON holmes_usage_events (feature, created_at DESC)`,
		`CREATE INDEX idx_holmes_usage_investigation ON holmes_usage_events (investigation_id) WHERE investigation_id <> ''`,
		`CREATE INDEX idx_holmes_usage_conversation ON holmes_usage_events (adre_conversation_id) WHERE adre_conversation_id IS NOT NULL`,
		`CREATE INDEX idx_holmes_usage_model ON holmes_usage_events (model, created_at DESC)`,
		`ALTER TABLE adre_messages ADD COLUMN IF NOT EXISTS cached_tokens INTEGER`,
		`ALTER TABLE adre_messages ADD COLUMN IF NOT EXISTS total_cost NUMERIC(14, 8)`,
		`ALTER TABLE adre_messages ADD COLUMN IF NOT EXISTS usage_event_id BIGINT`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS model VARCHAR NOT NULL DEFAULT ''`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS completion_tokens INTEGER`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS total_tokens INTEGER`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS cached_tokens INTEGER`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS total_cost NUMERIC(14, 8)`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS usage_event_id BIGINT`,
		`ALTER TABLE investigation_messages ADD COLUMN IF NOT EXISTS holmes_feature VARCHAR NOT NULL DEFAULT ''`,
		`ALTER TABLE investigations ADD COLUMN IF NOT EXISTS holmes_total_tokens BIGINT NOT NULL DEFAULT 0`,
		`ALTER TABLE investigations ADD COLUMN IF NOT EXISTS holmes_total_cost NUMERIC(14, 8) NOT NULL DEFAULT 0`,
		`ALTER TABLE investigations ADD COLUMN IF NOT EXISTS holmes_call_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS model VARCHAR NOT NULL DEFAULT ''`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS completion_tokens INTEGER`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS total_tokens INTEGER`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS cached_tokens INTEGER`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS total_cost NUMERIC(14, 8)`,
		`ALTER TABLE qan_insights_cache ADD COLUMN IF NOT EXISTS usage_event_id BIGINT`,
	},
	138: {
		// ADRE/HolmesGPT deployment config managed by PMM and rendered to the shared config dir.
		// Secret columns (adre_models.api_key, adre_provisioning.{holmes_api_key, pmm_sa_token,
		// servicenow_api_key, servicenow_client_token, slack_bot_token, slack_app_token}) are
		// encrypted at rest via DefaultAgentEncryptionColumnsV3 and masked on the API.
		// Singleton tables use a BOOLEAN id with a CHECK.
		`CREATE TABLE adre_holmes_config (
			id BOOLEAN PRIMARY KEY DEFAULT TRUE,
			config_yaml TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_by VARCHAR NOT NULL DEFAULT '',
			CONSTRAINT adre_holmes_config_singleton CHECK (id)
		)`,
		// The default chat/fast model is the config.yaml model: / fast_model: (what HolmesGPT honors);
		// this table only defines the available models rendered into model_list.yaml.
		`CREATE TABLE adre_models (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name VARCHAR NOT NULL UNIQUE,
			litellm_model VARCHAR NOT NULL,
			api_base VARCHAR NOT NULL DEFAULT '',
			api_key VARCHAR NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE adre_skills (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name VARCHAR NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			source VARCHAR NOT NULL DEFAULT 'user',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_by VARCHAR NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE adre_provisioning (
			id BOOLEAN PRIMARY KEY DEFAULT TRUE,
			holmes_api_key VARCHAR NOT NULL DEFAULT '',
			pmm_sa_token VARCHAR NOT NULL DEFAULT '',
			servicenow_api_key VARCHAR NOT NULL DEFAULT '',
			servicenow_client_token VARCHAR NOT NULL DEFAULT '',
			slack_bot_token VARCHAR NOT NULL DEFAULT '',
			slack_app_token VARCHAR NOT NULL DEFAULT '',
			pmm_sa_id INTEGER NOT NULL DEFAULT 0,
			pmm_url VARCHAR NOT NULL DEFAULT '',
			last_render_at TIMESTAMPTZ,
			render_status VARCHAR NOT NULL DEFAULT '',
			restart_required BOOLEAN NOT NULL DEFAULT FALSE,
			CONSTRAINT adre_provisioning_singleton CHECK (id)
		)`,
		`CREATE TABLE adre_config_audit (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			actor VARCHAR NOT NULL DEFAULT '',
			action VARCHAR NOT NULL,
			target VARCHAR NOT NULL DEFAULT '',
			at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			diff TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX idx_adre_config_audit_at ON adre_config_audit (at DESC)`,
	},
	139: {
		// Per-model extra LiteLLM params (YAML) for local/self-hosted models (temperature, num_ctx, etc.).
		`ALTER TABLE adre_models ADD COLUMN IF NOT EXISTS extra_params TEXT NOT NULL DEFAULT ''`,
	},
	140: {
		// F5 auto-investigate redesign: authoritative alert fingerprint on investigations, enabling
		// episode-scoped idempotency (one active investigation per firing alert). The partial unique
		// index coalesces concurrent claims for the same firing alert.
		`ALTER TABLE investigations ADD COLUMN IF NOT EXISTS alert_fingerprint VARCHAR NOT NULL DEFAULT ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS investigations_active_alert ` +
			`ON investigations (alert_fingerprint) ` +
			`WHERE alert_fingerprint <> '' AND status IN ('open','in_progress','investigating','running')`,
		// Supports the hourly auto-investigate cap count (created_by + created_at window) without a
		// sequential scan as the investigations table grows.
		`CREATE INDEX IF NOT EXISTS investigations_auto_created_at ` +
			`ON investigations (created_at) WHERE created_by = 'auto-investigate'`,
	},
	141: {
		// F5 auto-investigate webhook: shared secret PMM verifies on the Grafana alert webhook
		// (stored encrypted at rest, like the other adre_provisioning secrets).
		`ALTER TABLE adre_provisioning ADD COLUMN IF NOT EXISTS alert_webhook_secret VARCHAR NOT NULL DEFAULT ''`,
	},
	142: {
		// Script-based MySQL backups dispatched to DB nodes via Nomad.
		// Persistent, versioned config that renders the XtraBackup payload YAML.
		`CREATE TABLE backup_script_configs (
			id VARCHAR NOT NULL,
			name VARCHAR NOT NULL CHECK (name <> ''),
			service_id VARCHAR NOT NULL,
			node_name VARCHAR NOT NULL,
			backup_dir VARCHAR NOT NULL,
			compress BOOLEAN NOT NULL,
			compression_algorithm VARCHAR NOT NULL,
			copies INTEGER NOT NULL,
			replica_info BOOLEAN NOT NULL,
			xtrabackup_binary VARCHAR NOT NULL,
			rendered_yaml TEXT NOT NULL,
			config_version INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (id),
			UNIQUE (service_id, name)
		)`,
		// Catalog of dispatched runs, keyed by the cross-store backup_run_id.
		`CREATE TABLE backup_script_runs (
			id VARCHAR NOT NULL,
			config_id VARCHAR NOT NULL,
			service_id VARCHAR NOT NULL,
			node_name VARCHAR NOT NULL,
			nomad_job_id VARCHAR NOT NULL,
			status VARCHAR NOT NULL,
			backup_dir VARCHAR NOT NULL,
			size_bytes BIGINT NOT NULL,
			error VARCHAR NOT NULL,
			manifest JSONB,
			started_at TIMESTAMP NOT NULL,
			finished_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (id)
		)`,
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
		return nil, fmt.Errorf("failed to create a connection pool to PostgreSQL: %w", err)
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(5)  //nolint:mnd
	db.SetMaxOpenConns(10) //nolint:mnd

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
	HANodeID         string
	HAPeers          []string
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
	var pErr *pq.Error
	if errors.As(errCV, &pErr) && (pErr.Code == "28000" || pErr.Code == "28P01") {
		// 28000: invalid_authorization_specification (role does not exist, e.g. with trust auth)
		// 28P01: invalid_password - with password-based auth (md5/scram-sha-256), PostgreSQL returns this
		//        even when the role doesn't exist at all, to prevent user enumeration.
		// See https://www.postgresql.org/docs/current/errcodes-appendix.html
		//
		// In HA mode the external PostgreSQL must be pre-provisioned; auto-provisioning via the
		// embedded superuser password file is not available and must not be attempted.
		if params.HANodeID != "" {
			return nil, fmt.Errorf("cannot auto-provision database in HA mode: %w", errCV)
		}
		err := initWithRoot(params)
		if err != nil {
			return nil, err
		}
		errCV = checkVersion(ctx, db)
	}

	if errCV != nil {
		return nil, errCV
	}

	err := migrateDB(db, params)
	if err != nil {
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

	// Fold the just-processed columns into the tracked set, preserving already-tracked columns.
	// Replacing the set wholesale with only this run's columns would drop already-encrypted columns
	// (e.g. agents) when a new table is added to DefaultAgentEncryptionColumnsV3, causing them to be
	// re-encrypted (corrupted) on the next startup.
	_, err = UpdateSettings(tx, &ChangeSettingsParams{
		EncryptedItems: mergeEncryptedItems(settings.EncryptedItems, prepared, expectedState),
	})
	if err != nil {
		return err
	}

	return nil
}

// mergeEncryptedItems folds the just-processed columns into the tracked encrypted-columns set: on
// encrypt it adds them, on decrypt it removes them. The result is sorted. Keeping the existing entries
// (rather than replacing the set with only this run's columns) is what makes adding a table to
// DefaultAgentEncryptionColumnsV3 safe — otherwise already-encrypted columns would be dropped from the
// set and re-encrypted (corrupted) on a later run.
func mergeEncryptedItems(current, prepared []string, encrypt bool) []string {
	set := make(map[string]bool, len(current))
	for _, v := range current {
		set[v] = true
	}
	for _, key := range prepared {
		if encrypt {
			set[key] = true
		} else {
			delete(set, key)
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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

// initWithRoot tries to create the user and the database.
func initWithRoot(params SetupDBParams) error {
	if params.Logf != nil {
		params.Logf("Creating database %s and role %s", params.Name, params.Username)
	}

	// Read postgres password from the secure file
	passwordFile := "/srv/.postgres_password" //nolint:gosec
	passwordBytes, err := os.ReadFile(passwordFile)
	if err != nil {
		return fmt.Errorf("failed to read postgres password from %s: %w", passwordFile, err)
	}

	// we use postgres user for creating database
	db, err := OpenDB(SetupDBParams{Address: params.Address, Username: "postgres", Password: string(passwordBytes)})
	if err != nil {
		return fmt.Errorf("failed to open the database: %w", err)
	}
	defer db.Close() //nolint:errcheck

	var countDatabases int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_database WHERE datname = $1`, params.Name).Scan(&countDatabases)
	if err != nil {
		return fmt.Errorf("failed to select records from the database: %w", err)
	}

	if countDatabases == 0 {
		_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, params.Name))
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", params.Name, err)
		}
	}

	var countRoles int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_roles WHERE rolname=$1`, params.Username).Scan(&countRoles)
	if err != nil {
		return fmt.Errorf("failed to select records from the database: %w", err)
	}

	if countRoles == 0 {
		_, err = db.Exec(fmt.Sprintf(`CREATE USER "%s" LOGIN PASSWORD '%s'`, params.Username, params.Password))
		if err != nil {
			return fmt.Errorf("failed to create user %s: %w", params.Username, err)
		}

		_, err = db.Exec(`GRANT ALL PRIVILEGES ON DATABASE $1 TO $2`, params.Name, params.Username)
		if err != nil {
			return fmt.Errorf("failed to grant privileges to user %s on database %s: %w", params.Username, params.Name, err)
		}
	} else {
		// Role exists but authentication failed (e.g. pg_hba.conf switched from trust to
		// scram-sha-256 during an upgrade, leaving the role with no usable password hash).
		// initWithRoot is only ever called after a 28000/28P01 auth error, so resetting the
		// password to the currently configured value is OK.
		_, err = db.Exec(fmt.Sprintf(`ALTER USER "%s" WITH PASSWORD '%s'`, params.Username, params.Password))
		if err != nil {
			return fmt.Errorf("failed to update password for user %s: %w", params.Username, err)
		}
	}
	return nil
}

// migrateDB runs PostgreSQL database migrations.
func migrateDB(db *reform.DB, params SetupDBParams) error {
	var currentVersion int
	errDB := db.QueryRow("SELECT id FROM schema_migrations ORDER BY id DESC LIMIT 1").Scan(&currentVersion)
	// undefined_table (see https://www.postgresql.org/docs/current/errcodes-appendix.html)
	var pErr *pq.Error
	if errors.As(errDB, &pErr) && pErr.Code == "42P01" {
		errDB = nil
	}
	if errDB != nil {
		return errDB
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
				_, err := tx.Exec(q)
				if err != nil {
					return fmt.Errorf("failed to execute statement:\n%s: %w", q, err)
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
		err = SaveSettings(tx, s)
		if err != nil {
			return err
		}

		if params.HANodeID != "" {
			err = setupPMMServerHAAgents(tx.Querier, params)
		} else {
			err = setupPMMServerAgents(tx.Querier, params)
		}
		if err != nil {
			return err
		}

		return nil
	})
}

type agentConfig struct {
	ID string `yaml:"id"`
}

func runPMMAgentSetupHA(pmmAgentID string) error {
	args := []string{
		"setup",
		"--config-file", AgentConfigFilePath,
		"--server-address", "127.0.0.1:8443",
		"--id", pmmAgentID,
		"--skip-registration",
		"--server-insecure-tls",
	}
	cmd := exec.CommandContext(context.Background(), "pmm-agent", args...) //nolint:gosec
	logrus.Debugf("Running: pmm-agent %s", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting up pmm-agent: %w: %s", err, output)
	}
	return nil
}

func setupPMMServerHAAgents(q *reform.Querier, params SetupDBParams) error {
	agentID := uuid.New().String()
	nodeID := uuid.New().String()

	// create PMM Server Node and associated Agents in HA mode
	logrus.Infof("Setting up PMM Server agents in HA mode, Node ID: %s", params.HANodeID)

	file, err := os.Open(AgentConfigFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	var agentConfig agentConfig
	err = yaml.NewDecoder(file).Decode(&agentConfig)
	if err != nil {
		return fmt.Errorf("could not parse agent config file %s: %w", AgentConfigFilePath, err)
	}

	if agentConfig.ID == "" {
		return fmt.Errorf("the agent ID is empty in config file %s", AgentConfigFilePath)
	}

	if agentConfig.ID != PMMServerAgentID {
		// check if the agent with such ID already exists
		agent, err := FindAgentByID(q, agentConfig.ID)
		if err == nil {
			logrus.Infof("PMM Agent with ID %s already exists, skipping creation", agentConfig.ID)
			PMMServerAgentID = agentConfig.ID
			PMMServerNodeID = pointer.Get(agent.RunsOnNodeID)
			return nil
		}
	}

	labels := map[string]string{
		"cluster":     "pmm",
		"environment": "pmm",
	}

	// Shared inventory DB may already contain this HA node (e.g. replaced pod with a fresh PVC while
	// the hostname / HANodeID is unchanged). Re-use the existing node and pmm-agent IDs so local
	// pmm-agent setup matches inventory; otherwise createNodeWithID fails with AlreadyExists.
	existingNode, err := FindNodeByName(q, params.HANodeID)
	if err == nil { //nolint:nestif
		pmmAgents, ferr := FindPMMAgentsRunningOnNode(q, existingNode.NodeID)
		if ferr != nil {
			return ferr
		}
		if len(pmmAgents) == 0 {
			return fmt.Errorf("node %q (ID %s) exists in inventory but has no pmm-agent", params.HANodeID, existingNode.NodeID)
		}
		if len(pmmAgents) > 1 {
			logrus.Warnf("Multiple pmm-agents on HA node %q; using %s", params.HANodeID, pmmAgents[0].AgentID)
		}
		existingPmmAgentID := pmmAgents[0].AgentID

		if err := runPMMAgentSetupHA(existingPmmAgentID); err != nil { //nolint:noinlineerr
			return err
		}

		nodeExporters, err := FindAgents(q, AgentFilters{
			PMMAgentID: existingPmmAgentID,
			AgentType:  new(NodeExporterType),
		})
		if err != nil {
			return err
		}
		if len(nodeExporters) == 0 {
			if _, err := CreateNodeExporter(q, existingPmmAgentID, labels, false, false, []string{}, nil, ""); err != nil { //nolint:noinlineerr
				return err
			}
		}

		PMMServerAgentID = existingPmmAgentID
		logrus.Infof("Set PMMServerAgentID to (adopted existing HA inventory): %s", PMMServerAgentID)
		PMMServerNodeID = existingNode.NodeID
		logrus.Infof("Set PMMServerNodeID to (adopted existing HA inventory): %s", PMMServerNodeID)
		return nil
	}
	if status.Code(err) != codes.NotFound {
		return err
	}

	if err := runPMMAgentSetupHA(agentID); err != nil { //nolint:noinlineerr
		return err
	}

	node, err := createNodeWithID(q, nodeID, GenericNodeType, &CreateNodeParams{
		NodeName:        params.HANodeID,
		Address:         LocalhostAddr,
		CustomLabels:    labels,
		IsPMMServerNode: true,
	})
	if err != nil {
		logrus.Errorf("Failed to create a node with ID %s: %s", nodeID, err)
		return err
	}

	agent, err := createPMMAgentWithID(q, agentID, node.NodeID, labels)
	if err != nil {
		return err
	}

	_, err = CreateNodeExporter(q, agent.AgentID, labels, false, false, []string{}, nil, "")
	if err != nil {
		return err
	}

	// set PMMServerAgentID and PMMServerNodeID to generated values in HA setup
	PMMServerAgentID = agent.AgentID
	logrus.Infof("Set PMMServerAgentID to: %s", PMMServerAgentID)
	PMMServerNodeID = node.NodeID
	logrus.Infof("Set PMMServerNodeID to: %s", PMMServerNodeID)

	return nil
}

func setupPMMServerAgents(q *reform.Querier, params SetupDBParams) error {
	// create PMM Server Node and associated Agents
	node, err := createNodeWithID(q, PMMServerNodeID, GenericNodeType, &CreateNodeParams{
		NodeName:        "pmm-server",
		Address:         LocalhostAddr,
		IsPMMServerNode: true,
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			// this fixture was already added previously
			return nil
		}
		return err
	}

	_, err = createPMMAgentWithID(q, PMMServerAgentID, node.NodeID, nil)
	if err != nil {
		return err
	}
	_, err = CreateNodeExporter(q, PMMServerAgentID, nil, false, false, []string{}, nil, "")
	if err != nil {
		return err
	}

	address, port, err := parsePGAddress(params.Address)
	if err != nil {
		return err
	}
	if params.Address != DefaultPostgreSQLAddr {
		node, err = CreateNode(q, RemoteNodeType, &CreateNodeParams{
			NodeName: PMMServerPostgreSQLNodeName,
			Address:  address,
		})
		if err != nil {
			return err
		}
	} else {
		// Using postgres database in order to get metrics from entrypoint extension setup for QAN.
		params.Name = ""
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

	// PMM-6659: QAN's PgStatMonitorAgent agent running on PMM Server is disabled by default.
	// It can be enabled by setting PMM_ENABLE_INTERNAL_PG_QAN=1
	// We rely on just the environment variable here since we run this set up before loading the server settings.
	ap.Disabled = !env.GetBool(env.EnableInternalPgQAN)
	_, err = CreateAgent(q, QANPostgreSQLPgStatementsAgentType, ap)
	if err != nil {
		return err
	}

	return nil
}

// parsePGAddress parses PostgreSQL address into address:port; if no port specified returns default port number.
func parsePGAddress(address string) (string, uint16, error) {
	if !strings.Contains(address, ":") {
		return address, 5432, nil //nolint:mnd
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
