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

var addServicePostgreSQLResultT = commands.ParseTemplate(`
PostgreSQL Service added.
Service ID   : {{ .Service.ServiceID }}
Service name : {{ .Service.ServiceName }}
Node ID      : {{ .Service.NodeID }}
Address      : {{ .Service.Address }}
Port         : {{ .Service.Port }}
Custom labels: {{ .Service.CustomLabels }}

Environment    : {{ .Service.Environment }}
`)

type addServicePostgreSQLResult struct {
	Service *services.AddPostgreSQLServiceOKBodyPostgresql `json:"postgresql"`
}

func (res *addServicePostgreSQLResult) Result() {}

func (res *addServicePostgreSQLResult) String() string {
	return commands.RenderTemplate(addServicePostgreSQLResultT, res)
}

type addServicePostgreSQLCommand struct {
	ServiceName  string
	NodeID       string
	Address      string
	Port         int64
	CustomLabels string
	Environment  string
}

func (cmd *addServicePostgreSQLCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &services.AddPostgreSQLServiceParams{
		Body: services.AddPostgreSQLServiceBody{
			ServiceName:  cmd.ServiceName,
			NodeID:       cmd.NodeID,
			Address:      cmd.Address,
			Port:         cmd.Port,
			CustomLabels: customLabels,
			Environment:  cmd.Environment,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Services.AddPostgreSQLService(params)
	if err != nil {
		return nil, err
	}
	return &addServicePostgreSQLResult{
		Service: resp.Payload.Postgresql,
	}, nil
}

// register command
var (
	AddServicePostgreSQL  = new(addServicePostgreSQLCommand)
	AddServicePostgreSQLC = addServiceC.Command("postgresql", "Add PostgreSQL service to inventory.")
)

func init() {
	AddServicePostgreSQLC.Arg("name", "Service name").StringVar(&AddServicePostgreSQL.ServiceName)
	AddServicePostgreSQLC.Arg("node-id", "Node ID").StringVar(&AddServicePostgreSQL.NodeID)
	AddServicePostgreSQLC.Arg("address", "Address.").StringVar(&AddServicePostgreSQL.Address)
	AddServicePostgreSQLC.Arg("port", "Port.").Int64Var(&AddServicePostgreSQL.Port)

	AddServicePostgreSQLC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddServicePostgreSQL.CustomLabels)
	AddServicePostgreSQLC.Flag("environment", "Environment name.").StringVar(&AddServicePostgreSQL.Environment)
}
