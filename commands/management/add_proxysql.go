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
	proxysql "github.com/percona/pmm/api/managementpb/json/client/proxy_sql"

	"github.com/percona/pmm-admin/agentlocal"
	"github.com/percona/pmm-admin/commands"
)

var addProxySQLResultT = commands.ParseTemplate(`
ProxySQL Service added.
Service ID  : {{ .Service.ServiceID }}
Service name: {{ .Service.ServiceName }}
`)

type addProxySQLResult struct {
	Service *proxysql.AddProxySQLOKBodyService `json:"service"`
}

func (res *addProxySQLResult) Result() {}

func (res *addProxySQLResult) String() string {
	return commands.RenderTemplate(addProxySQLResultT, res)
}

type addProxySQLCommand struct {
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

func (cmd *addProxySQLCommand) Run() (commands.Result, error) {
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

	params := &proxysql.AddProxySQLParams{
		Body: proxysql.AddProxySQLBody{
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
	resp, err := client.Default.ProxySQL.AddProxySQL(params)
	if err != nil {
		return nil, err
	}

	return &addProxySQLResult{
		Service: resp.Payload.Service,
	}, nil
}

// register command
var (
	AddProxySQL  = new(addProxySQLCommand)
	AddProxySQLC = AddC.Command("proxysql", "Add ProxySQL to monitoring")
)

func init() {
	AddProxySQLC.Arg("address", "ProxySQL address and port. Default: 127.0.0.1:3306").Default("127.0.0.1:6032").StringVar(&AddProxySQL.AddressPort)

	hostname, _ := os.Hostname()
	serviceName := hostname + "-proxysql"
	serviceNameHelp := fmt.Sprintf("Service name. Default: %s", serviceName)
	AddProxySQLC.Arg("name", serviceNameHelp).Default(serviceName).StringVar(&AddProxySQL.ServiceName)

	AddProxySQLC.Flag("username", "ProxySQL username").Default("admin").StringVar(&AddProxySQL.Username)
	AddProxySQLC.Flag("password", "ProxySQL password").Default("admin").StringVar(&AddProxySQL.Password)

	AddProxySQLC.Flag("environment", "Environment name").StringVar(&AddProxySQL.Environment)
	AddProxySQLC.Flag("cluster", "Cluster name").StringVar(&AddProxySQL.Cluster)
	AddProxySQLC.Flag("replication-set", "Replication set name").StringVar(&AddProxySQL.ReplicationSet)
	AddProxySQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddProxySQL.CustomLabels)

	AddProxySQLC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddProxySQL.SkipConnectionCheck)
}
