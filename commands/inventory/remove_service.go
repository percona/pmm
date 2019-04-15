// pmm-admin
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package inventory

import (
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"

	"github.com/percona/pmm-admin/commands"
)

var removeServiceGenericResultT = commands.ParseTemplate(`
Service removed.
`)

type removeServiceResult struct{}

func (res *removeServiceResult) Result() {}

func (res *removeServiceResult) String() string {
	return commands.RenderTemplate(removeServiceGenericResultT, res)
}

type removeServiceCommand struct {
	ServiceID string
}

func (cmd *removeServiceCommand) Run() (commands.Result, error) {
	params := &services.RemoveServiceParams{
		Body: services.RemoveServiceBody{
			ServiceID: cmd.ServiceID,
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
	RemoveService  = new(removeServiceCommand)
	RemoveServiceC = inventoryRemoveC.Command("service", "Remove service from inventory.")
)

func init() {
	RemoveServiceC.Arg("service-id", "Service ID").StringVar(&RemoveService.ServiceID)
}
