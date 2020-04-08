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
	if err := commands.ValidatePort(int(cmd.Port)); err != nil {
		return nil, err
	}
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
	AddServiceMongoDBC = addServiceC.Command("mongodb", "Add MongoDB service to inventory").Hide(hide)
)

func init() {
	AddServiceMongoDBC.Arg("name", "Service name").StringVar(&AddServiceMongoDB.ServiceName)
	AddServiceMongoDBC.Arg("node-id", "Node ID").StringVar(&AddServiceMongoDB.NodeID)
	AddServiceMongoDBC.Arg("address", "Address").StringVar(&AddServiceMongoDB.Address)
	AddServiceMongoDBC.Arg("port", "Port").Int64Var(&AddServiceMongoDB.Port)

	AddServiceMongoDBC.Flag("environment", "Environment name").StringVar(&AddServiceMongoDB.Environment)
	AddServiceMongoDBC.Flag("cluster", "Cluster name").StringVar(&AddServiceMongoDB.Cluster)
	AddServiceMongoDBC.Flag("replication-set", "Replication set name").StringVar(&AddServiceMongoDB.ReplicationSet)
	AddServiceMongoDBC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddServiceMongoDB.CustomLabels)
}
