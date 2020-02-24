package types

import "fmt"

const (
	AgentTypePmmAgent                       = "PMM_AGENT"
	AgentTypeNodeExporter                   = "NODE_EXPORTER"
	AgentTypeMysqldExporter                 = "MYSQLD_EXPORTER"
	AgentTypeMongodbExporter                = "MONGODB_EXPORTER"
	AgentTypePostgresExporter               = "POSTGRES_EXPORTER"
	AgentTypeProxysqlExporter               = "PROXYSQL_EXPORTER"
	AgentTypeQanMysqlPerfschemaAgent        = "QAN_MYSQL_PERFSCHEMA_AGENT"
	AgentTypeQanMysqlSlowlogAgent           = "QAN_MYSQL_SLOWLOG_AGENT"
	AgentTypeQanMongodbProfilerAgent        = "QAN_MONGODB_PROFILER_AGENT"
	AgentTypeQanPostgresqlPgstatementsAgent = "QAN_POSTGRESQL_PGSTATEMENTS_AGENT"
	AgentTypeRdsExporter                    = "RDS_EXPORTER"
)

// agentTypeNames is the human readable list of agent names to be used in reports and
// commands like list or status
var agentTypeNames = map[string]string{
	// no invalid
	AgentTypePmmAgent:                       "pmm_agent",
	AgentTypeNodeExporter:                   "node_exporter",
	AgentTypeMysqldExporter:                 "mysqld_exporter",
	AgentTypeMongodbExporter:                "mongodb_exporter",
	AgentTypePostgresExporter:               "postgres_exporter",
	AgentTypeProxysqlExporter:               "proxysql_exporter",
	AgentTypeQanMysqlPerfschemaAgent:        "mysql_perfschema_agent",
	AgentTypeQanMysqlSlowlogAgent:           "mysql_slowlog_agent",
	AgentTypeQanMongodbProfilerAgent:        "mongodb_profiler_agent",
	AgentTypeQanPostgresqlPgstatementsAgent: "postgresql_pgstatements_agent",
	AgentTypeRdsExporter:                    "rds_exporter",
}

// AgentTypeName returns human friendly agent type to be used in reports
func AgentTypeName(t string) string {
	res := agentTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Agent Type %s", t))
	}

	return res
}
