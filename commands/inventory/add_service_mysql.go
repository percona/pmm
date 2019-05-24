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

var addServiceMySQLResultT = commands.ParseTemplate(`
MySQL Service added.
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

type addServiceMySQLResult struct {
	Service *services.AddMySQLServiceOKBodyMysql `json:"mysql"`
}

func (res *addServiceMySQLResult) Result() {}

func (res *addServiceMySQLResult) String() string {
	return commands.RenderTemplate(addServiceMySQLResultT, res)
}

type addServiceMySQLCommand struct {
	ServiceName    string
	NodeID         string
	Address        string
	Port           int64
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
}

func (cmd *addServiceMySQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &services.AddMySQLServiceParams{
		Body: services.AddMySQLServiceBody{
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

	resp, err := client.Default.Services.AddMySQLService(params)
	if err != nil {
		return nil, err
	}
	return &addServiceMySQLResult{
		Service: resp.Payload.Mysql,
	}, nil
}

// register command
var (
	AddServiceMySQL  = new(addServiceMySQLCommand)
	AddServiceMySQLC = addServiceC.Command("mysql", "Add MySQL service to inventory.")
)

func init() {
	AddServiceMySQLC.Arg("name", "Service name.").StringVar(&AddServiceMySQL.ServiceName)
	AddServiceMySQLC.Arg("node-id", "Node ID.").StringVar(&AddServiceMySQL.NodeID)
	AddServiceMySQLC.Arg("address", "Address.").StringVar(&AddServiceMySQL.Address)
	AddServiceMySQLC.Arg("port", "Port.").Int64Var(&AddServiceMySQL.Port)

	AddServiceMySQLC.Flag("environment", "Environment name.").StringVar(&AddServiceMySQL.Environment)
	AddServiceMySQLC.Flag("cluster", "Cluster name.").StringVar(&AddServiceMySQL.Cluster)
	AddServiceMySQLC.Flag("replication-set", "Replication set name.").StringVar(&AddServiceMySQL.ReplicationSet)
	AddServiceMySQLC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddServiceMySQL.CustomLabels)
}
