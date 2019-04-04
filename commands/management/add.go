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

// Package management provides management commands.
package management

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/percona/pmm/api/managementpb/json/client"
	mysql "github.com/percona/pmm/api/managementpb/json/client/my_sql"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
)

var addMySQLResultT = template.Must(template.New("").Parse(strings.TrimSpace(`
TODO
{{ . }}
`)))

type addResult struct {
	Service *mysql.AddOKBodyService `json:"service"`
}

func (res *addResult) Result() {}

func (res *addResult) String() string {
	return commands.RenderTemplate(addMySQLResultT, res)
}

// addCommand implements `pmm-admin add mysql` command.
type addCommand struct {
	Service  string
	Username string
	Password string
}

// Run implements Command interface.
func (cmd *addCommand) Run() (commands.Result, error) {
	// TODO get NodeID from local pmm-agent

	// TODO get or create MySQL service for this Node via pmm-managed

	switch cmd.Service {
	case "mysql":
		params := &mysql.AddParams{
			Body: mysql.AddBody{
				Username: cmd.Username,
				Password: cmd.Password,
			},
			Context: commands.Ctx,
		}
		resp, err := client.Default.MySQL.Add(params)
		if err != nil {
			return nil, err
		}

		return &addResult{
			Service: resp.Payload.Service,
		}, nil

	default:
		return nil, fmt.Errorf("Unexpected Service %q.", cmd.Service)
	}
}

// register commands
var (
	Add  = new(addCommand)
	AddC = kingpin.Command("add", "TODO")
)

func init() {
	AddC.Arg("service-type", "TODO").EnumVar(&Add.Service, "mysql")
}
