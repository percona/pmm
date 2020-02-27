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
	storedAgentTypePMMAgent                  StoredAgentType = "pmm-agent"
	storedAgentTypeNodeExporter              StoredAgentType = "node_exporter"
	storedAgentTypeMySQLdExporter            StoredAgentType = "mysqld_exporter"
	storedAgentTypeMongoDBExporter           StoredAgentType = "mongodb_exporter"
	storedAgentTypePostgresExporter          StoredAgentType = "postgres_exporter"
	storedAgentTypeProxySQLExporter          StoredAgentType = "proxysql_exporter"
	storedAgentTypeRDSExporter               StoredAgentType = "rds_exporter"
	storedAgentTypeQANMySQLPerfSchema        StoredAgentType = "qan-mysql-perfschema-agent"
	storedAgentTypeQANMySQLSlowlog           StoredAgentType = "qan-mysql-slowlog-agent"
	storedAgentTypeQANMongoDBProfiler        StoredAgentType = "qan-mongodb-profiler-agent"
	storedAgentTypeQANPostgreSQLPgStatements StoredAgentType = "qan-postgresql-pgstatements-agent"
)

var agentTypeStoredValues = map[string]StoredAgentType{
	AgentTypePMMAgent:                       storedAgentTypePMMAgent,
	AgentTypeNodeExporter:                   storedAgentTypeNodeExporter,
	AgentTypeMySQLdExporter:                 storedAgentTypeMySQLdExporter,
	AgentTypeMongoDBExporter:                storedAgentTypeMongoDBExporter,
	AgentTypePostgresExporter:               storedAgentTypePostgresExporter,
	AgentTypeProxySQLExporter:               storedAgentTypeProxySQLExporter,
	AgentTypeQANMySQLPerfSchemaAgent:        storedAgentTypeRDSExporter,
	AgentTypeQANMySQLSlowlogAgent:           storedAgentTypeQANMySQLPerfSchema,
	AgentTypeQANMongoDBProfilerAgent:        storedAgentTypeQANMySQLSlowlog,
	AgentTypeQANPostgreSQLPgStatementsAgent: storedAgentTypeQANMongoDBProfiler,
	AgentTypeRDSExporter:                    storedAgentTypeQANPostgreSQLPgStatements,
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
