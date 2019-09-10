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

var addServicePostgreSQLResultT = commands.ParseTemplate(`
PostgreSQL Service added.
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

type addServicePostgreSQLResult struct {
	Service *services.AddPostgreSQLServiceOKBodyPostgresql `json:"postgresql"`
}

func (res *addServicePostgreSQLResult) Result() {}

func (res *addServicePostgreSQLResult) String() string {
	return commands.RenderTemplate(addServicePostgreSQLResultT, res)
}

type addServicePostgreSQLCommand struct {
	ServiceName    string
	NodeID         string
	Address        string
	Port           int64
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
}

func (cmd *addServicePostgreSQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &services.AddPostgreSQLServiceParams{
		Body: services.AddPostgreSQLServiceBody{
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

	resp, err := client.Default.Services.AddPostgreSQLService(params)
	if err != nil {
		return nil, err
	}
	return &addServicePostgreSQLResult{
		Service: resp.Payload.Postgresql,
	}, nil
}

// register command
var (
	AddServicePostgreSQL  = new(addServicePostgreSQLCommand)
	AddServicePostgreSQLC = addServiceC.Command("postgresql", "Add PostgreSQL service to inventory")
)

func init() {
	AddServicePostgreSQLC.Arg("name", "Service name").StringVar(&AddServicePostgreSQL.ServiceName)
	AddServicePostgreSQLC.Arg("node-id", "Node ID").StringVar(&AddServicePostgreSQL.NodeID)
	AddServicePostgreSQLC.Arg("address", "Address").StringVar(&AddServicePostgreSQL.Address)
	AddServicePostgreSQLC.Arg("port", "Port").Int64Var(&AddServicePostgreSQL.Port)

	AddServicePostgreSQLC.Flag("environment", "Environment name").StringVar(&AddServicePostgreSQL.Environment)
	AddServicePostgreSQLC.Flag("cluster", "Cluster name").StringVar(&AddServicePostgreSQL.Cluster)
	AddServicePostgreSQLC.Flag("replication-set", "Replication set name").StringVar(&AddServicePostgreSQL.ReplicationSet)
	AddServicePostgreSQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddServicePostgreSQL.CustomLabels)
}
