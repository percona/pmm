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
	"net"
	"strconv"

	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	"github.com/percona/pmm/api/inventorypb/types"

	"github.com/percona/pmm-admin/commands"
)

var listServicesResultT = commands.ParseTemplate(`
Services list.

{{ printf "%-13s" "Service type" }} {{ printf "%-20s" "Service name" }} {{ printf "%-17s" "Address and Port" }} {{ "Service ID" }}
{{ range .Services }}
{{- printf "%-13s" .HumanReadableServiceType }} {{ printf "%-20s" .ServiceName }} {{ printf "%-17s" .AddressPort }} {{ .ServiceID }}
{{ end }}
`)

var acceptableServiceTypes = map[string][]string{
	types.ServiceTypeMySQLService:      {types.ServiceTypeName(types.ServiceTypeMySQLService)},
	types.ServiceTypeMongoDBService:    {types.ServiceTypeName(types.ServiceTypeMongoDBService)},
	types.ServiceTypePostgreSQLService: {types.ServiceTypeName(types.ServiceTypePostgreSQLService)},
	types.ServiceTypeProxySQLService:   {types.ServiceTypeName(types.ServiceTypeProxySQLService)},
	types.ServiceTypeExternalService:   {types.ServiceTypeName(types.ServiceTypeExternalService)},
}

type listResultService struct {
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	AddressPort string `json:"address_port"`
}

func (s listResultService) HumanReadableServiceType() string {
	return types.ServiceTypeName(s.ServiceType)
}

type listServicesResult struct {
	Services []listResultService `json:"services"`
}

func (res *listServicesResult) Result() {}

func (res *listServicesResult) String() string {
	return commands.RenderTemplate(listServicesResultT, res)
}

type listServicesCommand struct {
	filters     services.ListServicesBody
	ServiceType string
}

func (cmd *listServicesCommand) Run() (commands.Result, error) {
	serviceType, err := formatTypeValue(acceptableServiceTypes, cmd.ServiceType)
	if err != nil {
		return nil, err
	}

	cmd.filters.ServiceType = serviceType

	params := &services.ListServicesParams{
		Body:    cmd.filters,
		Context: commands.Ctx,
	}
	result, err := client.Default.Services.ListServices(params)
	if err != nil {
		return nil, err
	}

	var servicesList []listResultService
	for _, s := range result.Payload.Mysql {
		addressPort := net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10))
		if s.Socket != "" {
			addressPort = s.Socket
		}
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeMySQLService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: addressPort,
		})
	}
	for _, s := range result.Payload.Mongodb {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeMongoDBService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range result.Payload.Postgresql {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypePostgreSQLService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range result.Payload.Proxysql {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeProxySQLService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: net.JoinHostPort(s.Address, strconv.FormatInt(s.Port, 10)),
		})
	}
	for _, s := range result.Payload.External {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeExternalService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
		})
	}

	return &listServicesResult{
		Services: servicesList,
	}, nil
}

// register command
var (
	ListServices  = new(listServicesCommand)
	ListServicesC = inventoryListC.Command("services", "Show services in inventory").Hide(hide)
)

func init() {
	ListServicesC.Flag("node-id", "Filter by Node identifier").StringVar(&ListServices.filters.NodeID)
	ListServicesC.Flag("service-type", "Filter by Service type").StringVar(&ListServices.ServiceType)
}
