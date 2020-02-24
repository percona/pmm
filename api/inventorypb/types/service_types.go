package types

import "fmt"

const (
	AgentTypeMysqlService      = "MYSQL_SERVICE"
	AgentTypeMongodbService    = "MONGODB_SERVICE"
	AgentTypePostgresqlService = "POSTGRESQL_SERVICE"
	AgentTypeProxysqlService   = "PROXYSQL_SERVICE"
)

var serviceTypeNames = map[string]string{
	AgentTypeMysqlService:      "mysql-service",
	AgentTypeMongodbService:    "mongodb-service",
	AgentTypePostgresqlService: "postgresql-service",
	AgentTypeProxysqlService:   "proxysql-service",
}

// ServiceTypeName returns human friendly agent type to be used in reports
func ServiceTypeName(t string) string {
	res := serviceTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Service Type %s", t))
	}

	return res
}
