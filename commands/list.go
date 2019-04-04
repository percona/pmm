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

package commands

import (
	"strings"
	"text/template"

	"github.com/percona/pmm/api/inventory/json/client"
	"github.com/percona/pmm/api/inventory/json/client/agents"
	"gopkg.in/alecthomas/kingpin.v2"
)

var listResultT = template.Must(template.New("").Parse(strings.TrimSpace(`
TODO
{{ . }}
`)))

type listResult struct {
	Agents *agents.ListAgentsOKBody
}

func (res *listResult) Result() {}

func (res *listResult) String() string {
	return RenderTemplate(listResultT, res)
}

type listCommand struct {
	PMMAgentID string
}

func (cmd *listCommand) Run() (Result, error) {
	agents, err := client.Default.Agents.ListAgents(&agents.ListAgentsParams{
		Body: agents.ListAgentsBody{
			PMMAgentID: cmd.PMMAgentID,
		},
		Context: Ctx,
	})
	if err != nil {
		return nil, err
	}

	return &listResult{
		Agents: agents.Payload,
	}, nil
}

// register commands
var (
	List  = new(listCommand)
	ListC = kingpin.Command("list", "Show Agents statuses.")
)
