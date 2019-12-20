// pmm-admin
// Copyright 2019 Percona LLC
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

package management

import (
	"fmt"
	"os"
	"strings"

	"github.com/percona/pmm/api/managementpb/json/client"
	postgresql "github.com/percona/pmm/api/managementpb/json/client/postgre_sql"

	"github.com/percona/pmm-admin/agentlocal"
	"github.com/percona/pmm-admin/commands"
)

var addPostgreSQLResultT = commands.ParseTemplate(`
PostgreSQL Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addPostgreSQLResult struct {
	Service *postgresql.AddPostgreSQLOKBodyService `json:"service"`
}

func (res *addPostgreSQLResult) Result() {}

func (res *addPostgreSQLResult) String() string {
	return commands.RenderTemplate(addPostgreSQLResultT, res)
}

type addPostgreSQLCommand struct {
	Address        string
	NodeID         string
	PMMAgentID     string
	ServiceName    string
	Username       string
	Password       string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string

	QuerySource string

	SkipConnectionCheck bool
	TLS                 bool
	TLSSkipVerify       bool
}

func (cmd *addPostgreSQLCommand) GetServiceName() string {
	return cmd.ServiceName
}

func (cmd *addPostgreSQLCommand) GetAddress() string {
	return cmd.Address
}

func (cmd *addPostgreSQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.PMMAgentID == "" || cmd.NodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		if cmd.PMMAgentID == "" {
			cmd.PMMAgentID = status.AgentID
		}
		if cmd.NodeID == "" {
			cmd.NodeID = status.NodeID
		}
	}

	serviceName, host, port, err := processGlobalAddFlags(cmd)
	if err != nil {
		return nil, err
	}

	var usePgStatements bool
	switch cmd.QuerySource {
	case "pgstatements":
		usePgStatements = true
	case "none":
		// nothing
	}

	params := &postgresql.AddPostgreSQLParams{
		Body: postgresql.AddPostgreSQLBody{
			NodeID:         cmd.NodeID,
			ServiceName:    serviceName,
			Address:        host,
			Port:           int64(port),
			PMMAgentID:     cmd.PMMAgentID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			Username:       cmd.Username,
			Password:       cmd.Password,
			CustomLabels:   customLabels,

			QANPostgresqlPgstatementsAgent: usePgStatements,

			SkipConnectionCheck: cmd.SkipConnectionCheck,
			TLS:                 cmd.TLS,
			TLSSkipVerify:       cmd.TLSSkipVerify,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.PostgreSQL.AddPostgreSQL(params)
	if err != nil {
		return nil, err
	}

	return &addPostgreSQLResult{
		Service: resp.Payload.Service,
	}, nil
}

// register command
var (
	AddPostgreSQL  = new(addPostgreSQLCommand)
	AddPostgreSQLC = AddC.Command("postgresql", "Add PostgreSQL to monitoring")
)

func init() {
	hostname, _ := os.Hostname()
	serviceName := hostname + "-postgresql"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddPostgreSQLC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddPostgreSQL.ServiceName)

	AddPostgreSQLC.Arg("address", "PostgreSQL address and port (default: 127.0.0.1:5432)").Default("127.0.0.1:5432").StringVar(&AddPostgreSQL.Address)

	AddPostgreSQLC.Flag("node-id", "Node ID (default is autodetected)").StringVar(&AddPostgreSQL.NodeID)
	AddPostgreSQLC.Flag("pmm-agent-id", "The pmm-agent identifier which runs this instance (default is autodetected)").StringVar(&AddPostgreSQL.PMMAgentID)

	AddPostgreSQLC.Flag("username", "PostgreSQL username").Default("postgres").StringVar(&AddPostgreSQL.Username)
	AddPostgreSQLC.Flag("password", "PostgreSQL password").StringVar(&AddPostgreSQL.Password)

	querySources := []string{"pgstatements", "none"} // TODO add "auto"
	querySourceHelp := fmt.Sprintf("Source of SQL queries, one of: %s (default: %s)", strings.Join(querySources, ", "), querySources[0])
	AddPostgreSQLC.Flag("query-source", querySourceHelp).Default(querySources[0]).EnumVar(&AddPostgreSQL.QuerySource, querySources...)

	AddPostgreSQLC.Flag("environment", "Environment name").StringVar(&AddPostgreSQL.Environment)
	AddPostgreSQLC.Flag("cluster", "Cluster name").StringVar(&AddPostgreSQL.Cluster)
	AddPostgreSQLC.Flag("replication-set", "Replication set name").StringVar(&AddPostgreSQL.ReplicationSet)
	AddPostgreSQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddPostgreSQL.CustomLabels)

	AddPostgreSQLC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddPostgreSQL.SkipConnectionCheck)
	AddPostgreSQLC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddPostgreSQL.TLS)
	AddPostgreSQLC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddPostgreSQL.TLSSkipVerify)
}
