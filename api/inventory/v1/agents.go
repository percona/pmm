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

package inventoryv1

//go-sumtype:decl Agent

// Agent is a common interface for all types of Agents.
type Agent interface {
	sealedAgent()
}

// Ordered the same as AgentType enum.

func (*PMMAgent) sealedAgent()                        {}
func (*VMAgent) sealedAgent()                         {}
func (*NomadAgent) sealedAgent()                      {}
func (*NodeExporter) sealedAgent()                    {}
func (*MySQLdExporter) sealedAgent()                  {}
func (*MongoDBExporter) sealedAgent()                 {}
func (*PostgresExporter) sealedAgent()                {}
func (*ProxySQLExporter) sealedAgent()                {}
func (*QANMySQLPerfSchemaAgent) sealedAgent()         {}
func (*QANMySQLSlowlogAgent) sealedAgent()            {}
func (*QANMongoDBProfilerAgent) sealedAgent()         {}
func (*QANMongoDBMongologAgent) sealedAgent()         {}
func (*QANPostgreSQLPgStatementsAgent) sealedAgent()  {}
func (*QANPostgreSQLPgStatMonitorAgent) sealedAgent() {}
func (*RDSExporter) sealedAgent()                     {}
func (*ExternalExporter) sealedAgent()                {}
func (*AzureDatabaseExporter) sealedAgent()           {}
func (*ValkeyExporter) sealedAgent()                  {}
