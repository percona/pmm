package inventorypb

import (
	"fmt"
)

var niceAgentTypes = map[AgentType]string{
	// no invalid
	AgentType_PMM_AGENT:                         "pmm-agent",
	AgentType_NODE_EXPORTER:                     "node_exporter",
	AgentType_MYSQLD_EXPORTER:                   "mysqld_exporter",
	AgentType_MONGODB_EXPORTER:                  "mongodb_exporter",
	AgentType_POSTGRES_EXPORTER:                 "postgres_exporter",
	AgentType_PROXYSQL_EXPORTER:                 "proxysql_exporter",
	AgentType_QAN_MYSQL_PERFSCHEMA_AGENT:        "mysql-perfschema-agent",
	AgentType_QAN_MYSQL_SLOWLOG_AGENT:           "mysql-slowlog-agent",
	AgentType_QAN_MONGODB_PROFILER_AGENT:        "mongodb-profiler-agent",
	AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT: "postgresql-pgstatements-agent",
	AgentType_RDS_EXPORTER:                      "rds_exporter",
}

func NiceAgentType(t AgentType) string {
	res := niceAgentTypes[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Agent Type %s", t.String()))
	}
	return res
}

func NiceAgentTypeFromString(s string) string {
	t := AgentType(AgentType_value[s])
	return NiceAgentType(t)
}

//go-sumtype:decl Agent

// Agent is a common interface for all types of Agents.
type Agent interface {
	sealedAgent() //nolint:unused
}

// in order of AgentType enum

func (*PMMAgent) sealedAgent()                       {}
func (*NodeExporter) sealedAgent()                   {}
func (*MySQLdExporter) sealedAgent()                 {}
func (*MongoDBExporter) sealedAgent()                {}
func (*PostgresExporter) sealedAgent()               {}
func (*ProxySQLExporter) sealedAgent()               {}
func (*QANMySQLPerfSchemaAgent) sealedAgent()        {}
func (*QANMySQLSlowlogAgent) sealedAgent()           {}
func (*QANMongoDBProfilerAgent) sealedAgent()        {}
func (*QANPostgreSQLPgStatementsAgent) sealedAgent() {}
func (*RDSExporter) sealedAgent()                    {}
