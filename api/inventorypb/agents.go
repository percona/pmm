package inventorypb

import (
	"fmt"
)

// agentTypeNames is the human readable list of agent names to be used in reports and
// commands like list or status
var agentTypeNames = map[string]string{
	// no invalid
	"PMM_AGENT":                         "pmm_agent",
	"NODE_EXPORTER":                     "node_exporter",
	"MYSQLD_EXPORTER":                   "mysqld_exporter",
	"MONGODB_EXPORTER":                  "mongodb_exporter",
	"POSTGRES_EXPORTER":                 "postgres_exporter",
	"PROXYSQL_EXPORTER":                 "proxysql_exporter",
	"QAN_MYSQL_PERFSCHEMA_AGENT":        "mysql_perfschema_agent",
	"QAN_MYSQL_SLOWLOG_AGENT":           "mysql_slowlog_agent",
	"QAN_MONGODB_PROFILER_AGENT":        "mongodb_profiler_agent",
	"QAN_POSTGRESQL_PGSTATEMENTS_AGENT": "postgresql_pgstatements_agent",
	"RDS_EXPORTER":                      "rds_exporter",
}

// AgentTypeName returns human friendly agent type to be used in reports
func AgentTypeName(t string) string {
	res := agentTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Agent Type %s", t))
	}

	return res
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
