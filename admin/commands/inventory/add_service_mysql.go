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
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

var addServiceMySQLResultT = commands.ParseTemplate(`
MySQL Service added.
Service ID     : {{ .Service.ServiceID }}
Service name   : {{ .Service.ServiceName }}
Node ID        : {{ .Service.NodeID }}
{{ if .Service.Socket -}}
Socket         : {{ .Service.Socket }}
{{- else -}}
Address        : {{ .Service.Address }}
Port           : {{ .Service.Port }}
{{- end }}
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
	Socket         string
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
			Socket:         cmd.Socket,
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
	AddServiceMySQL  addServiceMySQLCommand
	AddServiceMySQLC = addServiceC.Command("mysql", "Add MySQL service to inventory").Hide(hide)
)

func init() {
	AddServiceMySQLC.Arg("name", "Service name").StringVar(&AddServiceMySQL.ServiceName)
	AddServiceMySQLC.Arg("node-id", "Node ID").StringVar(&AddServiceMySQL.NodeID)
	AddServiceMySQLC.Arg("address", "Address").StringVar(&AddServiceMySQL.Address)
	AddServiceMySQLC.Arg("port", "Port").Int64Var(&AddServiceMySQL.Port)
	AddServiceMySQLC.Flag("socket", "Path to MySQL socket").StringVar(&AddServiceMySQL.Socket)

	AddServiceMySQLC.Flag("environment", "Environment name").StringVar(&AddServiceMySQL.Environment)
	AddServiceMySQLC.Flag("cluster", "Cluster name").StringVar(&AddServiceMySQL.Cluster)
	AddServiceMySQLC.Flag("replication-set", "Replication set name").StringVar(&AddServiceMySQL.ReplicationSet)
	AddServiceMySQLC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddServiceMySQL.CustomLabels)
}
