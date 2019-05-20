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
	"net"
	"strconv"

	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"

	"github.com/percona/pmm-admin/commands"
)

var listServicesResultT = commands.ParseTemplate(`
Services list.

{{ printf "%-13s" "Service type" }} {{ printf "%-20s" "Service name" }} {{ printf "%-17s" "Address and Port" }} {{ "Service ID" }}
{{ range .Services }}
{{- printf "%-13s" .ServiceType }} {{ printf "%-20s" .ServiceName }} {{ printf "%-17s" .AddressPort }} {{ .ServiceID }}
{{ end }}
`)

type listResultService struct {
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	AddressPort string `json:"address_port"`
}

type listServicesResult struct {
	Services []listResultService `json:"services"`
}

func (res *listServicesResult) Result() {}

func (res *listServicesResult) String() string {
	return commands.RenderTemplate(listServicesResultT, res)
}

type listServicesCommand struct {
}

func (cmd *listServicesCommand) Run() (commands.Result, error) {
	params := &services.ListServicesParams{
		Context: commands.Ctx,
	}
	result, err := client.Default.Services.ListServices(params)
	if err != nil {
		return nil, err
	}

	var services []listResultService
	for _, s := range result.Payload.Mysql {
		services = append(services, listResultService{
			ServiceType: "MySQL",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range result.Payload.Mongodb {
		services = append(services, listResultService{
			ServiceType: "MongoDB",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range result.Payload.Postgresql {
		services = append(services, listResultService{
			ServiceType: "PostgreSQL",
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}

	return &listServicesResult{
		Services: services,
	}, nil
}

// register command
var (
	ListServices  = new(listServicesCommand)
	ListServicesC = inventoryListC.Command("services", "Show services in inventory.")
)
