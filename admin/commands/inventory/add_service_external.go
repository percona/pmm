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

type addExternalServiceResult struct {
	Service *services.AddExternalServiceOKBodyExternal `json:"external"`
}

func (res *addExternalServiceResult) Result() {}

func (res *addExternalServiceResult) String() string {
	return commands.RenderTemplate(addExternalServiceResultT, res)
}

type addExternalServiceCommand struct {
	ServiceName    string
	NodeID         string
	Environment    string
	Cluster        string
	ReplicationSet string
	CustomLabels   string
	Group          string
}

func (cmd *addExternalServiceCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

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
	return &addExternalServiceResult{
		Service: resp.Payload.External,
	}, nil
}

// register command
var (
	AddExternalService  = new(addExternalServiceCommand)
	AddExternalServiceC = addServiceC.Command("external", "Add an external service to inventory").Hide(hide)
)

func init() {
	AddExternalServiceC.Flag("name", "External service name. Required").Required().StringVar(&AddExternalService.ServiceName)
	AddExternalServiceC.Flag("node-id", "External service node ID. Required").Required().StringVar(&AddExternalService.NodeID)
	AddExternalServiceC.Flag("environment", "Environment name").StringVar(&AddExternalService.Environment)
	AddExternalServiceC.Flag("cluster", "Cluster name").StringVar(&AddExternalService.Cluster)
	AddExternalServiceC.Flag("replication-set", "Replication set name").StringVar(&AddExternalService.ReplicationSet)
	AddExternalServiceC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddExternalService.CustomLabels)
	AddExternalServiceC.Flag("group", "Group name of external service").StringVar(&AddExternalService.Group)
}
