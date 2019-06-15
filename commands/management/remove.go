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

package management

import (
	"fmt"
	"strings"

	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/service"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
)

var (
	serviceTypes = map[string]string{
		"mysql":      service.RemoveServiceBodyServiceTypeMYSQLSERVICE,
		"mongodb":    service.RemoveServiceBodyServiceTypeMONGODBSERVICE,
		"postgresql": service.RemoveServiceBodyServiceTypePOSTGRESQLSERVICE,
		"proxysql":   service.RemoveServiceBodyServiceTypePROXYSQLSERVICE,
	}
	serviceTypeKeys = []string{"mysql", "mongodb", "postgresql", "proxysql"}
)

var removeServiceGenericResultT = commands.ParseTemplate(`
Service removed.
`)

type removeServiceResult struct{}

func (res *removeServiceResult) Result() {}

func (res *removeServiceResult) String() string {
	return commands.RenderTemplate(removeServiceGenericResultT, res)
}

type removeMySQLCommand struct {
	ServiceType string
	ServiceName string
	ServiceID   string
}

func (cmd *removeMySQLCommand) Run() (commands.Result, error) {
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

	return new(removeServiceResult), nil
}

func (cmd *removeMySQLCommand) serviceType() *string {
	if val, ok := serviceTypes[cmd.ServiceType]; ok {
		return &val
	}
	return nil
}

// register command
var (
	Remove  = new(removeMySQLCommand)
	RemoveC = kingpin.Command("remove", "Remove Service from monitoring")
)

func init() {
	serviceTypeHelp := fmt.Sprintf("Service type, one of: %s", strings.Join(serviceTypeKeys, ", "))
	RemoveC.Arg("service-type", serviceTypeHelp).Required().EnumVar(&Remove.ServiceType, serviceTypeKeys...)
	RemoveC.Arg("service-name", "Service name").Required().StringVar(&Remove.ServiceName)

	RemoveC.Flag("service-id", "Service ID").StringVar(&Remove.ServiceID)
}
