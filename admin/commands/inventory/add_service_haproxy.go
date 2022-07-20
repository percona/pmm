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
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
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

type addHAProxyServiceResult struct {
	Service *services.AddHAProxyServiceOKBodyHaproxy `json:"haproxy"`
}

func (res *addHAProxyServiceResult) Result() {}

func (res *addHAProxyServiceResult) String() string {
	return commands.RenderTemplate(addHAProxyServiceResultT, res)
}

type addHAProxyServiceCommand struct {
	ServiceName    string
	NodeID         string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
}

func (cmd *addHAProxyServiceCommand) Run() (commands.Result, error) {
	isSupported, err := helpers.IsHAProxySupported()
	if !isSupported {
		return nil, err
	}

	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	params := &services.AddHAProxyServiceParams{
		Body: services.AddHAProxyServiceBody{
			ServiceName:    cmd.ServiceName,
			NodeID:         cmd.NodeID,
			Environment:    cmd.Environment,
			Cluster:        cmd.Cluster,
			ReplicationSet: cmd.ReplicationSet,
			CustomLabels:   customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Services.AddHAProxyService(params)
	if err != nil {
		return nil, err
	}
	return &addHAProxyServiceResult{
		Service: resp.Payload.Haproxy,
	}, nil
}

// register command
var (
	AddHAProxyService  addHAProxyServiceCommand
	AddHAProxyServiceC = addServiceC.Command("haproxy", "Add an haproxy service to the inventory").Hide(hide)
)

func init() {
	AddHAProxyServiceC.Arg("name", "HAProxy service name").StringVar(&AddHAProxyService.ServiceName)
	AddHAProxyServiceC.Arg("node-id", "HAProxy service node ID").StringVar(&AddHAProxyService.NodeID)
	AddHAProxyServiceC.Flag("environment", "Environment name like 'production' or 'qa'").
		PlaceHolder("prod").StringVar(&AddHAProxyService.Environment)
	AddHAProxyServiceC.Flag("cluster", "Cluster name").
		PlaceHolder("east-cluster").StringVar(&AddHAProxyService.Cluster)
	AddHAProxyServiceC.Flag("replication-set", "Replication set name").
		PlaceHolder("rs1").StringVar(&AddHAProxyService.ReplicationSet)
	AddHAProxyServiceC.Flag("custom-labels", "Custom user-assigned labels. Example: region=east,app=app1").StringVar(&AddHAProxyService.CustomLabels)
}
