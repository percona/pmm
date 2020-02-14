package inventorypb

import (
	"fmt"
)

var niceAgentTypes = map[int]string{
	// no invalid
	1:  "pmm-agent",
	2:  "node_exporter",
	3:  "mysqld_exporter",
	4:  "mongodb_exporter",
	5:  "postgres_exporter",
	6:  "proxysql_exporter",
	7:  "mysql-perfschema-agent",
	8:  "mysql-slowlog-agent",
	9:  "mongodb-profiler-agent",
	10: "postgresql-pgstatements-agent",
	11: "rds_exporter",
}

// AgentTypeName returns human friendly agent type to be used in reports
func AgentTypeName(t int) string {
	res := niceAgentTypes[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Agent Type %d", t))
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
