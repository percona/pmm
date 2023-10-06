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

// this list should be in sync with inventorypb/services.pb.go.
const (
	ServiceTypeMySQLService      = "MYSQL_SERVICE"
	ServiceTypeMongoDBService    = "MONGODB_SERVICE"
	ServiceTypePostgreSQLService = "POSTGRESQL_SERVICE"
	ServiceTypeProxySQLService   = "PROXYSQL_SERVICE"
	ServiceTypeHAProxyService    = "HAPROXY_SERVICE"
	ServiceTypeExternalService   = "EXTERNAL_SERVICE"
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
