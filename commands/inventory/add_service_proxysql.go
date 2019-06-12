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

package inventory

import (
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"

	"github.com/percona/pmm-admin/commands"
)

var addServiceProxySQLResultT = commands.ParseTemplate(`
ProxySQL Service added.
Service ID     : {{ .Service.ServiceID }}
Service name   : {{ .Service.ServiceName }}
Node ID        : {{ .Service.NodeID }}
Address        : {{ .Service.Address }}
Port           : {{ .Service.Port }}
Environment    : {{ .Service.Environment }}
Cluster name   : {{ .Service.Cluster }}
Replication set: {{ .Service.ReplicationSet }}
Custom labels  : {{ .Service.CustomLabels }}
`)

type addServiceProxySQLResult struct {
	Service *services.AddProxySQLServiceOKBodyProxysql `json:"proxysql"`
}

func (res *addServiceProxySQLResult) Result() {}

func (res *addServiceProxySQLResult) String() string {
	return commands.RenderTemplate(addServiceProxySQLResultT, res)
}

type addServiceProxySQLCommand struct {
	ServiceName    string
	NodeID         string
	Address        string
	Port           int64
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
}

func (cmd *addServiceProxySQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &services.AddProxySQLServiceParams{
		Body: services.AddProxySQLServiceBody{
			ServiceName:    cmd.ServiceName,
			NodeID:         cmd.NodeID,
			Address:        cmd.Address,
			Port:           cmd.Port,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			CustomLabels:   customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Services.AddProxySQLService(params)
	if err != nil {
		return nil, err
	}
	return &addServiceProxySQLResult{
		Service: resp.Payload.Proxysql,
	}, nil
}

// register command
var (
	AddServiceProxySQL  = new(addServiceProxySQLCommand)
	AddServiceProxySQLC = addServiceC.Command("proxysql", "Add ProxySQL service to inventory")
)

func init() {
	AddServiceProxySQLC.Arg("name", "Service name").StringVar(&AddServiceProxySQL.ServiceName)
	AddServiceProxySQLC.Arg("node-id", "Node ID").StringVar(&AddServiceProxySQL.NodeID)
	AddServiceProxySQLC.Arg("address", "Address").StringVar(&AddServiceProxySQL.Address)
	AddServiceProxySQLC.Arg("port", "Port").Default("6032").Int64Var(&AddServiceProxySQL.Port)

	AddServiceProxySQLC.Flag("environment", "Environment name").StringVar(&AddServiceProxySQL.Environment)
	AddServiceProxySQLC.Flag("cluster", "Cluster name").StringVar(&AddServiceProxySQL.Cluster)
	AddServiceProxySQLC.Flag("replication-set", "Replication set name").StringVar(&AddServiceProxySQL.ReplicationSet)
	AddServiceProxySQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddServiceProxySQL.CustomLabels)
}
