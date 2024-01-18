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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

var addExternalServiceResultT = commands.ParseTemplate(`
External Service added.
Service ID     : {{ .Service.ServiceID }}
Service name   : {{ .Service.ServiceName }}
Node ID        : {{ .Service.NodeID }}
Environment    : {{ .Service.Environment }}
Cluster name   : {{ .Service.Cluster }}
Replication set: {{ .Service.ReplicationSet }}
Custom labels  : {{ .Service.CustomLabels }}
Group          : {{ .Service.Group }}
`)

type addServiceExternalResult struct {
	Service *services.AddExternalServiceOKBodyExternal `json:"external"`
}

func (res *addServiceExternalResult) Result() {}

func (res *addServiceExternalResult) String() string {
	return commands.RenderTemplate(addExternalServiceResultT, res)
}

// AddServiceExternalCommand is used by Kong for CLI flags and commands.
type AddServiceExternalCommand struct {
	ServiceName    string            `name:"name" required:"" help:"External service name. Required"`
	NodeID         string            `required:"" help:"External service node ID. Required"`
	Environment    string            `help:"Environment name"`
	Cluster        string            `help:"Cluster name"`
	ReplicationSet string            `help:"Replication set name"`
	CustomLabels   map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	Group          string            `help:"Group name of external service"`
}

// RunCmd executes the AddServiceExternalCommand and returns the result.
func (cmd *AddServiceExternalCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	params := &services.AddExternalServiceParams{
		Body: services.AddExternalServiceBody{
			ServiceName:    cmd.ServiceName,
			NodeID:         cmd.NodeID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			CustomLabels:   customLabels,
			Group:          cmd.Group,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Services.AddExternalService(params)
	if err != nil {
		return nil, err
	}
	return &addServiceExternalResult{
		Service: resp.Payload.External,
	}, nil
}
