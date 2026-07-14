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

package management

import (
	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	inventoryClient "github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/service"
)

var removeServiceGenericResultT = commands.ParseTemplate(`
Service removed.
`)

type removeServiceResult struct{}

func (res *removeServiceResult) Result() {}

func (res *removeServiceResult) String() string {
	return commands.RenderTemplate(removeServiceGenericResultT, res)
}

// RemoveCommand is used by Kong for CLI flags and commands.
type RemoveCommand struct {
	ServiceType string `arg:"" enum:"${serviceTypesEnum}" help:"Service type, one of: ${serviceTypesEnum}"`
	ServiceName string `arg:"" default:"" help:"Service name"`
	ServiceID   string `help:"Service ID"`
}

// RunCmd runs the command for RemoveCommand.
func (cmd *RemoveCommand) RunCmd() (commands.Result, error) {
	if cmd.ServiceID == "" && cmd.ServiceName == "" {
		// Automatic service lookup during removal
		//
		// Get services and remove it automatically once it's only one
		// service registered
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}

		servicesRes, err := inventoryClient.Default.Services.ListServices(&services.ListServicesParams{
			Body: services.ListServicesBody{
				NodeID:      status.NodeID,
				ServiceType: cmd.serviceType(),
			},
			Context: commands.Ctx,
		})
		if err != nil {
			return nil, err
		}
		switch {
		case len(servicesRes.Payload.Mysql) == 1:
			cmd.ServiceID = servicesRes.Payload.Mysql[0].ServiceID
		case len(servicesRes.Payload.Mongodb) == 1:
			cmd.ServiceID = servicesRes.Payload.Mongodb[0].ServiceID
		case len(servicesRes.Payload.Postgresql) == 1:
			cmd.ServiceID = servicesRes.Payload.Postgresql[0].ServiceID
		case len(servicesRes.Payload.Proxysql) == 1:
			cmd.ServiceID = servicesRes.Payload.Proxysql[0].ServiceID
		case len(servicesRes.Payload.Haproxy) == 1:
			cmd.ServiceID = servicesRes.Payload.Haproxy[0].ServiceID
		case len(servicesRes.Payload.External) == 1:
			cmd.ServiceID = servicesRes.Payload.External[0].ServiceID
		}
		if cmd.ServiceID == "" {
			//nolint:revive,golint
			return nil, errors.New(`We could not find a service associated with the local node. Please provide "Service ID" or "Service name".`)
		}
	}

	params := &service.RemoveServiceParams{
		Body: service.RemoveServiceBody{
			ServiceID:   cmd.ServiceID,
			ServiceName: cmd.ServiceName,
			ServiceType: cmd.serviceType(),
		},
		Context: commands.Ctx,
	}
	_, err := client.Default.Service.RemoveService(params)
	if err != nil {
		return nil, err
	}

	return &removeServiceResult{}, nil
}

func (cmd *RemoveCommand) serviceType() *string {
	if val, ok := allServiceTypes[cmd.ServiceType]; ok {
		return &val
	}
	return nil
}
