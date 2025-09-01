// Copyright (C) 2023 Percona LLC
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
	"github.com/percona/pmm/api/inventory/v1/json/client"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
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
	Service *services.AddServiceOKBodyMysql `json:"mysql"`
}

func (res *addServiceMySQLResult) Result() {}

func (res *addServiceMySQLResult) String() string {
	return commands.RenderTemplate(addServiceMySQLResultT, res)
}

// AddServiceMySQLCommand is used by Kong for CLI flags and commands.
type AddServiceMySQLCommand struct {
	ServiceName    string            `arg:"" optional:"" name:"name" help:"Service name"`
	NodeID         string            `arg:"" optional:"" help:"Node ID"`
	Address        string            `arg:"" optional:"" help:"Address"`
	Port           int64             `arg:"" optional:"" help:"Port"`
	Socket         string            `help:"Path to MySQL socket"`
	Environment    string            `help:"Environment name"`
	Cluster        string            `help:"Cluster name"`
	ReplicationSet string            `help:"Replication set name"`
	CustomLabels   map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd runs the command for AddServiceMySQLCommand.
func (cmd *AddServiceMySQLCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)
	params := &services.AddServiceParams{
		Body: services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
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
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.ServicesService.AddService(params)
	if err != nil {
		return nil, err
	}

	return &addServiceMySQLResult{
		Service: resp.Payload.Mysql,
	}, nil
}
