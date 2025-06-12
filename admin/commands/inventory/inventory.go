// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package inventory provides inventory commands.
package inventory

import (
	"strings"

	"github.com/pkg/errors"
)

// InventoryCommand is used by Kong for CLI flags and commands.
type InventoryCommand struct { //nolint:revive
	List   ListCommand   `cmd:"" help:"List inventory commands"`
	Add    AddCommand    `cmd:"" help:"Add to inventory commands"`
	Remove RemoveCommand `cmd:"" help:"Remove from inventory commands"`
	Change ChangeCommand `cmd:"" help:"Change inventory commands"`
}

// ListCommand is used by Kong for CLI flags and commands.
type ListCommand struct {
	Agents   ListAgentsCommand   `cmd:"" help:"Show agents in inventory"`
	Nodes    ListNodesCommand    `cmd:"" help:"Show nodes in inventory"`
	Services ListServicesCommand `cmd:"" help:"Show services in inventory"`
}

// AddCommand is used by Kong for CLI flags and commands.
type AddCommand struct {
	Agent   AddAgentCommand   `cmd:"" help:"Add agent to inventory"`
	Node    AddNodeCommand    `cmd:"" help:"Add node to inventory"`
	Service AddServiceCommand `cmd:"" help:"Add service to inventory"`
}

// AddAgentCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddAgentCommand struct {
	ExternalExporter AddAgentExternalExporterCommand `cmd:"" name:"external" help:"Add external exporter to inventory"`
	MongodbExporter  AddAgentMongodbExporterCommand  `cmd:"" help:"Add mongodb_exporter to inventory"`
	MysqldExporter   AddAgentMysqldExporterCommand   `cmd:"" help:"Add mysqld_exporter to inventory"`
	NodeExporter     AddAgentNodeExporterCommand     `cmd:"" help:"Add Node exporter to inventory"`
	PMMAgent         AddPMMAgentCommand              `cmd:"" help:"Add PMM agent to inventory"`
	PostgresExporter AddAgentPostgresExporterCommand `cmd:"" help:"Add postgres_exporter to inventory"`
	ProxysqlExporter AddAgentProxysqlExporterCommand `cmd:"" help:"Add proxysql_exporter to inventory"`

	QANMongoDBProfilerAgent         AddAgentQANMongoDBProfilerAgentCommand         `cmd:"" name:"qan-mongodb-profiler-agent" help:"Add QAN MongoDB profiler agent to inventory"`
	QANMySQLPerfSchemaAgent         AddAgentQANMySQLPerfSchemaAgentCommand         `cmd:"" name:"qan-mysql-perfschema-agent" help:"Add QAN MySQL perf schema agent to inventory"`
	QANMySQLSlowlogAgent            AddAgentQANMySQLSlowlogAgentCommand            `cmd:"" name:"qan-mysql-slowlog-agent" help:"Add QAN MySQL slowlog agent to inventory"`
	QANPostgreSQLPgStatementsAgent  AddAgentQANPostgreSQLPgStatementsAgentCommand  `cmd:"" name:"qan-postgresql-pgstatements-agent" help:"Add QAN PostgreSQL Stat Statements Agent to inventory"`
	QANPostgreSQLPgStatMonitorAgent AddAgentQANPostgreSQLPgStatMonitorAgentCommand `cmd:"" name:"qan-postgresql-pgstatmonitor-agent" help:"Add QAN PostgreSQL Stat Monitor Agent to inventory"`

	RDSExporter AddAgentRDSExporterCommand `cmd:"" help:"Add rds_exporter to inventory"`
}

// AddNodeCommand is used by Kong for CLI flags and commands.
type AddNodeCommand struct {
	Container AddNodeContainerCommand `cmd:"" help:"Add container node to inventory"`
	Generic   AddNodeGenericCommand   `cmd:"" help:"Add generic node to inventory"`
	Remote    AddNodeRemoteCommand    `cmd:"" help:"Add Remote node to inventory"`
	RemoteRDS AddNodeRemoteRDSCommand `cmd:"" help:"Add Remote RDS node to inventory"`
}

// AddServiceCommand is used by Kong for CLI flags and commands.
type AddServiceCommand struct {
	External   AddServiceExternalCommand   `cmd:"" help:"Add an external service to inventory"`
	HAProxy    AddServiceHAProxyCommand    `cmd:"" name:"haproxy" help:"Add HAProxy service to inventory"`
	MongoDB    AddServiceMongoDBCommand    `cmd:"" name:"mongodb" help:"Add MongoDB service to inventory"`
	MySQL      AddServiceMySQLCommand      `cmd:"" name:"mysql" help:"Add MySQL service to inventory"`
	PostgreSQL AddServicePostgreSQLCommand `cmd:"" name:"postgresql" help:"Add PostgreSQL service to inventory"`
	ProxySQL   AddServiceProxySQLCommand   `cmd:"" name:"proxysql" help:"Add ProxySQL service to inventory"`
}

// RemoveCommand is used by Kong for CLI flags and commands.
type RemoveCommand struct {
	Agent   RemoveAgentCommand   `cmd:"" help:"Remove agent from inventory"`
	Node    RemoveNodeCommand    `cmd:"" help:"Remove node from inventory"`
	Service RemoveServiceCommand `cmd:"" help:"Remove service from inventory"`
}

// ChangeCommand is used by Kong for CLI flags and commands.
type ChangeCommand struct {
	Agent ChangeAgentCommand `cmd:"" help:"Change agent configuration"`
}

// ChangeAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentCommand struct {
	NodeExporter          ChangeAgentNodeExporterCommand          `cmd:"" help:"Change node_exporter configuration (only passed flags will be changed)"`
	MysqldExporter        ChangeAgentMysqldExporterCommand        `cmd:"" help:"Change mysqld_exporter configuration (only passed flags will be changed)"`
	MongodbExporter       ChangeAgentMongodbExporterCommand       `cmd:"" help:"Change mongodb_exporter configuration (only passed flags will be changed)"`
	PostgresExporter      ChangeAgentPostgresExporterCommand      `cmd:"" help:"Change postgres_exporter configuration (only passed flags will be changed)"`
	ProxysqlExporter      ChangeAgentProxysqlExporterCommand      `cmd:"" help:"Change proxysql_exporter configuration (only passed flags will be changed)"`
	ExternalExporter      ChangeAgentExternalExporterCommand      `cmd:"" help:"Change external exporter configuration (only passed flags will be changed)"`
	RdsExporter           ChangeAgentRDSExporterCommand           `cmd:"" help:"Change rds_exporter configuration (only passed flags will be changed)"`
	AzureDatabaseExporter ChangeAgentAzureDatabaseExporterCommand `cmd:"" help:"Change azure_database_exporter configuration (only passed flags will be changed)"`
	ValkeyExporter        ChangeAgentValkeyExporterCommand        `cmd:"" help:"Change valkey_exporter configuration (only passed flags will be changed)"`
	NomadAgent            ChangeAgentNomadAgentCommand            `cmd:"" help:"Change nomad_agent configuration (only passed flags will be changed)"`

	QANMySQLPerfSchemaAgent         ChangeAgentQANMySQLPerfSchemaAgentCommand         `cmd:"" name:"qan-mysql-perfschema-agent" help:"Change QAN MySQL perf schema agent configuration (only passed flags will be changed)"`
	QANMySQLSlowlogAgent            ChangeAgentQANMySQLSlowlogAgentCommand            `cmd:"" name:"qan-mysql-slowlog-agent" help:"Change QAN MySQL slowlog agent configuration (only passed flags will be changed)"`
	QANMongoDBProfilerAgent         ChangeAgentQANMongoDBProfilerAgentCommand         `cmd:"" name:"qan-mongodb-profiler-agent" help:"Change QAN MongoDB profiler agent configuration (only passed flags will be changed)"`
	QANPostgreSQLPgStatementsAgent  ChangeAgentQANPostgreSQLPgStatementsAgentCommand  `cmd:"" name:"qan-postgresql-pgstatements-agent" help:"Change QAN PostgreSQL pgstatements agent configuration (only passed flags will be changed)"`
	QANPostgreSQLPgStatMonitorAgent ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand `cmd:"" name:"qan-postgresql-pgstatmonitor-agent" help:"Change QAN PostgreSQL pgstatmonitor agent configuration (only passed flags will be changed)"`
}

// formatTypeValue checks acceptable type value and variations contains input and returns type value.
// Values comparison is case-insensitive.
func formatTypeValue(acceptableTypeValues map[string][]string, input string) (*string, error) {
	if input == "" {
		return nil, nil //nolint:nilnil
	}

	for value, variations := range acceptableTypeValues {
		variations = append(variations, value)
		for _, variation := range variations {
			if strings.EqualFold(variation, input) {
				return &value, nil
			}
		}
	}
	return nil, errors.Errorf("unexpected type value %q", input)
}

// RunCmd is a stub that allows to display the InventoryCommand's help.
func (cmd *InventoryCommand) RunCmd() {}
