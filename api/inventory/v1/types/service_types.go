// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package types

import "fmt"

// this list should be in sync with inventory/services.pb.go.
const (
	ServiceTypeMySQLService      = "SERVICE_TYPE_MYSQL_SERVICE"
	ServiceTypeMongoDBService    = "SERVICE_TYPE_MONGODB_SERVICE"
	ServiceTypePostgreSQLService = "SERVICE_TYPE_POSTGRESQL_SERVICE"
	ServiceTypeProxySQLService   = "SERVICE_TYPE_PROXYSQL_SERVICE"
	ServiceTypeHAProxyService    = "SERVICE_TYPE_HAPROXY_SERVICE"
	ServiceTypeExternalService   = "SERVICE_TYPE_EXTERNAL_SERVICE"
)

var serviceTypeNames = map[string]string{
	// no invalid
	ServiceTypeMySQLService:      "MySQL",
	ServiceTypeMongoDBService:    "MongoDB",
	ServiceTypePostgreSQLService: "PostgreSQL",
	ServiceTypeProxySQLService:   "ProxySQL",
	ServiceTypeHAProxyService:    "HAProxy",
	ServiceTypeExternalService:   "External",
}

// ServiceTypeName returns human friendly service type to be used in reports.
func ServiceTypeName(t string) string {
	res := serviceTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Service Type %s", t))
	}

	return res
}
