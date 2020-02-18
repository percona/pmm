package inventorypb

import fmt "fmt"

//go-sumtype:decl Service

// Service is a common interface for all types of Services.
type Service interface {
	sealedService() //nolint:unused
}

var serviceTypeNames = map[string]string{
	"MYSQL_SERVICE":      "mysql-service",
	"MONGODB_SERVICE":    "mongodb-service",
	"POSTGRESQL_SERVICE": "postgresql-service",
	"PROXYSQL_SERVICE":   "proxysql-service",
}

// ServiceTypeName returns human friendly agent type to be used in reports
func ServiceTypeName(t string) string {
	res := serviceTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Service Type %s", t))
	}

	return res
}

// in order of ServiceType enum

func (*MySQLService) sealedService()      {}
func (*MongoDBService) sealedService()    {}
func (*PostgreSQLService) sealedService() {}
func (*ProxySQLService) sealedService()   {}
