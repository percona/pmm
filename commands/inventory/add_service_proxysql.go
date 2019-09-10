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
