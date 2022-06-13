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
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

var removeServiceResultT = commands.ParseTemplate(`
Service removed.
`)

type removeServiceResult struct{}

func (res *removeServiceResult) Result() {}

func (res *removeServiceResult) String() string {
	return commands.RenderTemplate(removeServiceResultT, res)
}

type removeServiceCommand struct {
	ServiceID string
	Force     bool
}

func (cmd *removeServiceCommand) Run() (commands.Result, error) {
	params := &services.RemoveServiceParams{
		Body: services.RemoveServiceBody{
			ServiceID: cmd.ServiceID,
			Force:     cmd.Force,
		},
		Context: commands.Ctx,
	}
	_, err := client.Default.Services.RemoveService(params)
	if err != nil {
		return nil, err
	}
	return &removeServiceResult{}, nil
}

// register command
var (
	RemoveService  removeServiceCommand
	RemoveServiceC = inventoryRemoveC.Command("service", "Remove service from inventory").Hide(hide)
)

func init() {
	RemoveServiceC.Arg("service-id", "Service ID").StringVar(&RemoveService.ServiceID)
	RemoveServiceC.Flag("force", "Remove service with all dependencies").BoolVar(&RemoveService.Force)
}
