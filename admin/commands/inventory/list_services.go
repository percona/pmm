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
	"fmt"
	"net"
	"strconv"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

var listServicesResultT = commands.ParseTemplate(`
Services list.

{{ printf "%-22s" "Service type" }} {{ printf "%-20s" "Service name" }} {{ printf "%-17s" "Address and Port" }} {{ "Service ID" }}
{{ range .Services }}
{{- printf "%-22s" .HumanReadableServiceType }} {{ printf "%-20s" .ServiceName }} {{ printf "%-17s" .AddressPort }} {{ .ServiceID }}
{{ end }}
`)

var acceptableServiceTypes = map[string][]string{
	types.ServiceTypeMySQLService:      {types.ServiceTypeName(types.ServiceTypeMySQLService)},
	types.ServiceTypeMongoDBService:    {types.ServiceTypeName(types.ServiceTypeMongoDBService)},
	types.ServiceTypePostgreSQLService: {types.ServiceTypeName(types.ServiceTypePostgreSQLService)},
	types.ServiceTypeValkeyService:     {types.ServiceTypeName(types.ServiceTypeValkeyService)},
	types.ServiceTypeProxySQLService:   {types.ServiceTypeName(types.ServiceTypeProxySQLService)},
	types.ServiceTypeHAProxyService:    {types.ServiceTypeName(types.ServiceTypeHAProxyService)},
	types.ServiceTypeExternalService:   {types.ServiceTypeName(types.ServiceTypeExternalService)},
}

type listResultService struct {
	ServiceType string `json:"service_type"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	AddressPort string `json:"address_port"`
	Group       string `json:"external_group"`
}

func (s listResultService) HumanReadableServiceType() string {
	serviceTypeName := types.ServiceTypeName(s.ServiceType)

	if s.ServiceType == types.ServiceTypeExternalService {
		return fmt.Sprintf("%s:%s", serviceTypeName, s.Group)
	}

	return serviceTypeName
}

type listServicesResult struct {
	Services []listResultService `json:"services"`
}

func (res *listServicesResult) Result() {}

func (res *listServicesResult) String() string {
	return commands.RenderTemplate(listServicesResultT, res)
}

// ListServicesCommand is used by Kong for CLI flags and commands.
type ListServicesCommand struct {
	NodeID        string `help:"Filter by Node identifier"`
	ServiceType   string `help:"Filter by Service type"`
	ExternalGroup string `help:"Filter by external group"`
}

// RunCmd executes the ListServicesCommand and returns the result.
func (cmd *ListServicesCommand) RunCmd() (commands.Result, error) {
	serviceType, err := formatTypeValue(acceptableServiceTypes, cmd.ServiceType)
	if err != nil {
		return nil, err
	}

	params := &services.ListServicesParams{
		NodeID:        pointer.ToString(cmd.NodeID),
		ExternalGroup: pointer.ToString(cmd.ExternalGroup),
		ServiceType:   serviceType,
		Context:       commands.Ctx,
	}
	result, err := client.Default.ServicesService.ListServices(params)
	if err != nil {
		return nil, err
	}

	getAddressPort := func(socket, address string, port int64) string {
		if socket != "" {
			return socket
		}
		return net.JoinHostPort(address, strconv.FormatInt(port, 10))
	}

	var servicesList []listResultService //nolint:prealloc
	for _, s := range result.Payload.Mysql {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeMySQLService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: getAddressPort(s.Socket, s.Address, s.Port),
		})
	}
	for _, s := range result.Payload.Mongodb {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeMongoDBService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: getAddressPort(s.Socket, s.Address, s.Port),
		})
	}
	for _, s := range result.Payload.Postgresql {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypePostgreSQLService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: getAddressPort(s.Socket, s.Address, s.Port),
		})
	}
	for _, s := range result.Payload.Valkey {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeValkeyService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: getAddressPort(s.Socket, s.Address, s.Port),
		})
	}
	for _, s := range result.Payload.Proxysql {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeProxySQLService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			AddressPort: getAddressPort(s.Socket, s.Address, s.Port),
		})
	}
	for _, s := range result.Payload.Haproxy {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeHAProxyService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
		})
	}
	for _, s := range result.Payload.External {
		servicesList = append(servicesList, listResultService{
			ServiceType: types.ServiceTypeExternalService,
			ServiceID:   s.ServiceID,
			ServiceName: s.ServiceName,
			Group:       s.Group,
		})
	}

	return &listServicesResult{
		Services: servicesList,
	}, nil
}
