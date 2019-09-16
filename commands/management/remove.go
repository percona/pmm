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

package management

import (
	"fmt"
	"strings"

	"github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/service"
	"gopkg.in/alecthomas/kingpin.v2"

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
	if val, ok := allServiceTypes[cmd.ServiceType]; ok {
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
	serviceTypeHelp := fmt.Sprintf("Service type, one of: %s", strings.Join(allServiceTypesKeys, ", "))
	RemoveC.Arg("service-type", serviceTypeHelp).Required().EnumVar(&Remove.ServiceType, allServiceTypesKeys...)
	RemoveC.Arg("service-name", "Service name").Required().StringVar(&Remove.ServiceName)

	RemoveC.Flag("service-id", "Service ID").StringVar(&Remove.ServiceID)
}
