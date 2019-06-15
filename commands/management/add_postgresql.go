// pmm-admin
// Copyright (C) 2018 Percona LLC
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

package management

import (
	"fmt"
	"net"
	"os"
	"strconv"

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
	AddressPort    string
	ServiceName    string
	Username       string
	Password       string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string

	SkipConnectionCheck bool
}

func (cmd *addPostgreSQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return nil, err
	}

	host, portS, err := net.SplitHostPort(cmd.AddressPort)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, err
	}

	params := &postgresql.AddPostgreSQLParams{
		Body: postgresql.AddPostgreSQLBody{
			NodeID:         status.NodeID,
			ServiceName:    cmd.ServiceName,
			Address:        host,
			Port:           int64(port),
			PMMAgentID:     status.AgentID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			Username:       cmd.Username,
			Password:       cmd.Password,

			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
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
	AddPostgreSQLC.Arg("address", "PostgreSQL address and port (default: 127.0.0.1:5432)").Default("127.0.0.1:5432").StringVar(&AddPostgreSQL.AddressPort)

	hostname, _ := os.Hostname()
	serviceName := hostname + "-postgresql"
	serviceNameHelp := fmt.Sprintf("Service name (autodetected default: %s)", serviceName)
	AddPostgreSQLC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddPostgreSQL.ServiceName)

	AddPostgreSQLC.Flag("username", "PostgreSQL username").Default("postgres").StringVar(&AddPostgreSQL.Username)
	AddPostgreSQLC.Flag("password", "PostgreSQL password").StringVar(&AddPostgreSQL.Password)

	AddPostgreSQLC.Flag("environment", "Environment name").StringVar(&AddPostgreSQL.Environment)
	AddPostgreSQLC.Flag("cluster", "Cluster name").StringVar(&AddPostgreSQL.Cluster)
	AddPostgreSQLC.Flag("replication-set", "Replication set name").StringVar(&AddPostgreSQL.ReplicationSet)
	AddPostgreSQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddPostgreSQL.CustomLabels)

	AddPostgreSQLC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddPostgreSQL.SkipConnectionCheck)
}
