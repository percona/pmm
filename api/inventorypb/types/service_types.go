package types

import "fmt"

const (
	ServiceTypeMySQLService      = "MYSQL_SERVICE"
	ServiceTypeMongoDBService    = "MONGODB_SERVICE"
	ServiceTypePostgreSQLService = "POSTGRESQL_SERVICE"
	ServiceTypeProxySQLService   = "PROXYSQL_SERVICE"
)

var serviceTypeNames = map[string]string{
	ServiceTypeMySQLService:      "mysql-service",
	ServiceTypeMongoDBService:    "mongodb-service",
	ServiceTypePostgreSQLService: "postgresql-service",
	ServiceTypeProxySQLService:   "proxysql-service",
}

// ServiceTypeName returns human friendly agent type to be used in reports
func ServiceTypeName(t string) string {
	res := serviceTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Service Type %s", t))
	}

	return res
}
