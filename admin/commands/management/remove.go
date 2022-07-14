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

package management

import (
	"github.com/percona/pmm/admin/commands"
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

type RemoveCommand struct {
	ServiceType string `arg:"" enum:"${serviceTypesEnum}" help:"Service type, one of: ${serviceTypesEnum}"`
	ServiceName string `arg:"" default:"" help:"Service name"`
	ServiceID   string `help:"Service ID"`
}

func (cmd *RemoveCommand) RunCmd() (commands.Result, error) {
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
