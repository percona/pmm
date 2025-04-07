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

// Package types contains various entities types.
package types

import "fmt"

// this list should be in sync with inventory/agents.pb.go.
const (
	AgentTypePMMAgent                        = "AGENT_TYPE_PMM_AGENT"
	AgentTypeVMAgent                         = "AGENT_TYPE_VM_AGENT"
	AgentTypeNomadAgent                      = "AGENT_TYPE_NOMAD_AGENT"
	AgentTypeNodeExporter                    = "AGENT_TYPE_NODE_EXPORTER"
	AgentTypeMySQLdExporter                  = "AGENT_TYPE_MYSQLD_EXPORTER"
	AgentTypeMongoDBExporter                 = "AGENT_TYPE_MONGODB_EXPORTER"
	AgentTypePostgresExporter                = "AGENT_TYPE_POSTGRES_EXPORTER"
	AgentTypeProxySQLExporter                = "AGENT_TYPE_PROXYSQL_EXPORTER"
	AgentTypeQANMySQLPerfSchemaAgent         = "AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT"
	AgentTypeQANMySQLSlowlogAgent            = "AGENT_TYPE_QAN_MYSQL_SLOWLOG_AGENT"
	AgentTypeQANMongoDBProfilerAgent         = "AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT"
	AgentTypeQANMongoDBMongologAgent         = "AGENT_TYPE_QAN_MONGODB_MONGOLOG_AGENT"
	AgentTypeQANPostgreSQLPgStatementsAgent  = "AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT"
	AgentTypeQANPostgreSQLPgStatMonitorAgent = "AGENT_TYPE_QAN_POSTGRESQL_PGSTATMONITOR_AGENT"
	AgentTypeRDSExporter                     = "AGENT_TYPE_RDS_EXPORTER"
	AgentTypeExternalExporter                = "AGENT_TYPE_EXTERNAL_EXPORTER"
	AgentTypeAzureDatabaseExporter           = "AGENT_TYPE_AZURE_DATABASE_EXPORTER"
)

var agentTypeNames = map[string]string{
	// no invalid
	AgentTypePMMAgent:                        "pmm_agent",
	AgentTypeVMAgent:                         "vmagent",
	AgentTypeNomadAgent:                      "nomad_agent",
	AgentTypeNodeExporter:                    "node_exporter",
	AgentTypeMySQLdExporter:                  "mysqld_exporter",
	AgentTypeMongoDBExporter:                 "mongodb_exporter",
	AgentTypePostgresExporter:                "postgres_exporter",
	AgentTypeProxySQLExporter:                "proxysql_exporter",
	AgentTypeQANMySQLPerfSchemaAgent:         "mysql_perfschema_agent",
	AgentTypeQANMySQLSlowlogAgent:            "mysql_slowlog_agent",
	AgentTypeQANMongoDBProfilerAgent:         "mongodb_profiler_agent",
	AgentTypeQANPostgreSQLPgStatementsAgent:  "postgresql_pgstatements_agent",
	AgentTypeQANPostgreSQLPgStatMonitorAgent: "postgresql_pgstatmonitor_agent",
	AgentTypeRDSExporter:                     "rds_exporter",
	AgentTypeExternalExporter:                "external-exporter",
	AgentTypeAzureDatabaseExporter:           "azure_database_exporter",
}

// AgentTypeName returns human friendly agent type to be used in reports.
func AgentTypeName(t string) string {
	res := agentTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Agent Type %s", t))
	}

	return res
}
