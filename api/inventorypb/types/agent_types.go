package types

import "fmt"

// this list should be in sync with inventorypb/agents.pb.go
const (
	AgentTypePMMAgent                       = "PMM_AGENT"
	AgentTypeNodeExporter                   = "NODE_EXPORTER"
	AgentTypeMySQLdExporter                 = "MYSQLD_EXPORTER"
	AgentTypeMongoDBExporter                = "MONGODB_EXPORTER"
	AgentTypePostgresExporter               = "POSTGRES_EXPORTER"
	AgentTypeProxySQLExporter               = "PROXYSQL_EXPORTER"
	AgentTypeQANMySQLPerfSchemaAgent        = "QAN_MYSQL_PERFSCHEMA_AGENT"
	AgentTypeQANMySQLSlowlogAgent           = "QAN_MYSQL_SLOWLOG_AGENT"
	AgentTypeQANMongoDBProfilerAgent        = "QAN_MONGODB_PROFILER_AGENT"
	AgentTypeQANPostgreSQLPgStatementsAgent = "QAN_POSTGRESQL_PGSTATEMENTS_AGENT"
	AgentTypeRDSExporter                    = "RDS_EXPORTER"
)

// StoredAgentType represents Agent type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and Prometheus.
type StoredAgentType string

// Agent types (in the same order as in agents.proto).
const (
	StoredAgentTypePMMAgent                  StoredAgentType = "pmm-agent"
	StoredAgentTypeNodeExporter              StoredAgentType = "node_exporter"
	StoredAgentTypeMySQLdExporter            StoredAgentType = "mysqld_exporter"
	StoredAgentTypeMongoDBExporter           StoredAgentType = "mongodb_exporter"
	StoredAgentTypePostgresExporter          StoredAgentType = "postgres_exporter"
	StoredAgentTypeProxySQLExporter          StoredAgentType = "proxysql_exporter"
	StoredAgentTypeRDSExporter               StoredAgentType = "rds_exporter"
	StoredAgentTypeQANMySQLPerfSchema        StoredAgentType = "qan-mysql-perfschema-agent"
	StoredAgentTypeQANMySQLSlowlog           StoredAgentType = "qan-mysql-slowlog-agent"
	StoredAgentTypeQANMongoDBProfiler        StoredAgentType = "qan-mongodb-profiler-agent"
	StoredAgentTypeQANPostgreSQLPgStatements StoredAgentType = "qan-postgresql-pgstatements-agent"
)

var agentTypeStoredValues = map[string]StoredAgentType{
	AgentTypePMMAgent:                       StoredAgentTypePMMAgent,
	AgentTypeNodeExporter:                   StoredAgentTypeNodeExporter,
	AgentTypeMySQLdExporter:                 StoredAgentTypeMySQLdExporter,
	AgentTypeMongoDBExporter:                StoredAgentTypeMongoDBExporter,
	AgentTypePostgresExporter:               StoredAgentTypePostgresExporter,
	AgentTypeProxySQLExporter:               StoredAgentTypeProxySQLExporter,
	AgentTypeQANMySQLPerfSchemaAgent:        StoredAgentTypeRDSExporter,
	AgentTypeQANMySQLSlowlogAgent:           StoredAgentTypeQANMySQLPerfSchema,
	AgentTypeQANMongoDBProfilerAgent:        StoredAgentTypeQANMySQLSlowlog,
	AgentTypeQANPostgreSQLPgStatementsAgent: StoredAgentTypeQANMongoDBProfiler,
	AgentTypeRDSExporter:                    StoredAgentTypeQANPostgreSQLPgStatements,
}

// AgentTypeName returns the Agent type value as stored in databases:
func AgentTypeStoredValue(t string) StoredAgentType {
	res := agentTypeStoredValues[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Agent Type %s", t))
	}

	return res
}

// AgentTypeName returns human friendly agent type to be used in reports
func AgentTypeName(t string) string {
	return string(AgentTypeStoredValue(t))
}
