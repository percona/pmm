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

var addServiceMongoDBResultT = commands.ParseTemplate(`
MongoDB Service added.
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

type addServiceMongoDBResult struct {
	Service *services.AddMongoDBServiceOKBodyMongodb `json:"mongodb"`
}

func (res *addServiceMongoDBResult) Result() {}

func (res *addServiceMongoDBResult) String() string {
	return commands.RenderTemplate(addServiceMongoDBResultT, res)
}

type addServiceMongoDBCommand struct {
	ServiceName    string
	NodeID         string
	Address        string
	Port           int64
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
}

func (cmd *addServiceMongoDBCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &services.AddMongoDBServiceParams{
		Body: services.AddMongoDBServiceBody{
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

	resp, err := client.Default.Services.AddMongoDBService(params)
	if err != nil {
		return nil, err
	}
	return &addServiceMongoDBResult{
		Service: resp.Payload.Mongodb,
	}, nil
}

// register command
var (
	AddServiceMongoDB  = new(addServiceMongoDBCommand)
	AddServiceMongoDBC = addServiceC.Command("mongodb", "Add MongoDB service to inventory.")
)

func init() {
	AddServiceMongoDBC.Arg("name", "Service name.").StringVar(&AddServiceMongoDB.ServiceName)
	AddServiceMongoDBC.Arg("node-id", "Node ID.").StringVar(&AddServiceMongoDB.NodeID)
	AddServiceMongoDBC.Arg("address", "Address.").StringVar(&AddServiceMongoDB.Address)
	AddServiceMongoDBC.Arg("port", "Port.").Int64Var(&AddServiceMongoDB.Port)

	AddServiceMongoDBC.Flag("environment", "Environment name.").StringVar(&AddServiceMongoDB.Environment)
	AddServiceMongoDBC.Flag("cluster", "Cluster name.").StringVar(&AddServiceMongoDB.Cluster)
	AddServiceMongoDBC.Flag("replication-set", "Replication set name.").StringVar(&AddServiceMongoDB.ReplicationSet)
	AddServiceMongoDBC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddServiceMongoDB.CustomLabels)
}
