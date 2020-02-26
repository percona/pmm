package types

import "fmt"

// this list should be in sync with inventorypb/services.pb.go
const (
	ServiceTypeMySQLService      = "MYSQL_SERVICE"
	ServiceTypeMongoDBService    = "MONGODB_SERVICE"
	ServiceTypePostgreSQLService = "POSTGRESQL_SERVICE"
	ServiceTypeProxySQLService   = "PROXYSQL_SERVICE"
)

var serviceTypeNames = map[string]string{
	// no invalid
	ServiceTypeMySQLService:      "MySQL",
	ServiceTypeMongoDBService:    "MongoDB",
	ServiceTypePostgreSQLService: "PostgreSQL",
	ServiceTypeProxySQLService:   "ProxySQL",
}

// ServiceTypeName returns human friendly service type to be used in reports
func ServiceTypeName(t string) string {
	res := serviceTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Service Type %s", t))
	}

	return res
}
