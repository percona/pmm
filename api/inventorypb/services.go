// Copyright (C) 2019 Percona LLC
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

package inventorypb

//go-sumtype:decl Service

// Service is a common interface for all types of Services.
type Service interface {
	sealedService()
}

// in order of ServiceType enum

func (*MySQLService) sealedService()      {}
func (*MongoDBService) sealedService()    {}
func (*PostgreSQLService) sealedService() {}
func (*ProxySQLService) sealedService()   {}
func (*HAProxyService) sealedService()    {}
func (*ExternalService) sealedService()   {}
