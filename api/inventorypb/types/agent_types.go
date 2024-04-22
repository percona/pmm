// Copyright (C) 2024 Percona LLC
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

// this list should be in sync with inventorypb/agents.pb.go.
const (
	AgentTypePMMAgent                        = "PMM_AGENT"
	AgentTypeVMAgent                         = "VM_AGENT"
	AgentTypeNodeExporter                    = "NODE_EXPORTER"
	AgentTypeMySQLdExporter                  = "MYSQLD_EXPORTER"
	AgentTypeMongoDBExporter                 = "MONGODB_EXPORTER"
	AgentTypePostgresExporter                = "POSTGRES_EXPORTER"
	AgentTypeProxySQLExporter                = "PROXYSQL_EXPORTER"
	AgentTypeQANMySQLPerfSchemaAgent         = "QAN_MYSQL_PERFSCHEMA_AGENT"
	AgentTypeQANMySQLSlowlogAgent            = "QAN_MYSQL_SLOWLOG_AGENT"
	AgentTypeQANMongoDBProfilerAgent         = "QAN_MONGODB_PROFILER_AGENT"
	AgentTypeQANPostgreSQLPgStatementsAgent  = "QAN_POSTGRESQL_PGSTATEMENTS_AGENT"
	AgentTypeQANPostgreSQLPgStatMonitorAgent = "QAN_POSTGRESQL_PGSTATMONITOR_AGENT"
	AgentTypeRDSExporter                     = "RDS_EXPORTER"
	AgentTypeExternalExporter                = "EXTERNAL_EXPORTER"
	AgentTypeAzureDatabaseExporter           = "AZURE_DATABASE_EXPORTER"
)

var agentTypeNames = map[string]string{
	// no invalid
	AgentTypePMMAgent:                        "pmm_agent",
	AgentTypeVMAgent:                         "vmagent",
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
