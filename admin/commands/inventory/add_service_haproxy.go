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
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

var addHAProxyServiceResultT = commands.ParseTemplate(`
HAProxy Service added.
Service ID     : {{ .Service.ServiceID }}
Service name   : {{ .Service.ServiceName }}
Node ID        : {{ .Service.NodeID }}
Environment    : {{ .Service.Environment }}
Cluster name   : {{ .Service.Cluster }}
Replication set: {{ .Service.ReplicationSet }}
Custom labels  : {{ .Service.CustomLabels }}
`)

type addServiceHAProxyResult struct {
	Service *services.AddServiceOKBodyHaproxy `json:"haproxy"`
}

func (res *addServiceHAProxyResult) Result() {}

func (res *addServiceHAProxyResult) String() string {
	return commands.RenderTemplate(addHAProxyServiceResultT, res)
}

// AddServiceHAProxyCommand is used by Kong for CLI flags and commands.
type AddServiceHAProxyCommand struct {
	ServiceName    string            `arg:"" optional:"" name:"name" help:"HAProxy service name"`
	NodeID         string            `arg:"" optional:"" help:"HAProxy service node ID"`
	Environment    string            `placeholder:"prod" help:"Environment name like 'production' or 'qa'"`
	Cluster        string            `placeholder:"east-cluster" help:"Cluster name"`
	ReplicationSet string            `placeholder:"rs1" help:"Replication set name"`
	CustomLabels   map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the AddServiceHAProxyCommand and returns the result.
func (cmd *AddServiceHAProxyCommand) RunCmd() (commands.Result, error) {
	isSupported, err := helpers.IsHAProxySupported()
	if !isSupported {
		return nil, err
	}

	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)

	params := &services.AddServiceParams{
		Body: services.AddServiceBody{
			Haproxy: &services.AddServiceParamsBodyHaproxy{
				ServiceName:    cmd.ServiceName,
				NodeID:         cmd.NodeID,
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
	return &addServiceHAProxyResult{
		Service: resp.Payload.Haproxy,
	}, nil
}
