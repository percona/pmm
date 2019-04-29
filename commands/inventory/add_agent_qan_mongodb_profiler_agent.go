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
	"github.com/percona/pmm/api/inventorypb/json/client/agents"

	"github.com/percona/pmm-admin/commands"
)

var addAgentQANMongoDBProfilerAgentResultT = commands.ParseTemplate(`
QAN MongoDB profiler agent added.
Agent ID     : {{ .Agent.AgentID }}
PMM-Agent ID : {{ .Agent.PMMAgentID }}
Service ID   : {{ .Agent.ServiceID }}
Username     : {{ .Agent.Username }}
Password     : {{ .Agent.Password }}

Status       : {{ .Agent.Status }}
Disabled     : {{ .Agent.Disabled }}
Custom labels: {{ .Agent.CustomLabels }}
`)

type addAgentQANMongoDBProfilerAgentResult struct {
	Agent *agents.AddQANMongoDBProfilerAgentOKBodyQANMongodbProfilerAgent `json:"qan_mongodb_profiler_agent"`
}

func (res *addAgentQANMongoDBProfilerAgentResult) Result() {}

func (res *addAgentQANMongoDBProfilerAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANMongoDBProfilerAgentResultT, res)
}

type addAgentQANMongoDBProfilerAgentCommand struct {
	PMMAgentID   string
	ServiceID    string
	Username     string
	Password     string
	CustomLabels string
}

func (cmd *addAgentQANMongoDBProfilerAgentCommand) Run() (commands.Result, error) {
	customLabels, err := parseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddQANMongoDBProfilerAgentParams{
		Body: agents.AddQANMongoDBProfilerAgentBody{
			PMMAgentID:   cmd.PMMAgentID,
			ServiceID:    cmd.ServiceID,
			Username:     cmd.Username,
			Password:     cmd.Password,
			CustomLabels: customLabels,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddQANMongoDBProfilerAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentQANMongoDBProfilerAgentResult{
		Agent: resp.Payload.QANMongodbProfilerAgent,
	}, nil
}

// register command
var (
	AddAgentQANMongoDBProfilerAgent  = new(addAgentQANMongoDBProfilerAgentCommand)
	AddAgentQANMongoDBProfilerAgentC = addAgentC.Command("qan-mongodb-profiler-agent", "add QAN MongoDB profiler agent to inventory.")
)

func init() {
	AddAgentQANMongoDBProfilerAgentC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance.").StringVar(&AddAgentQANMongoDBProfilerAgent.PMMAgentID)
	AddAgentQANMongoDBProfilerAgentC.Arg("service-id", "Service identifier.").StringVar(&AddAgentQANMongoDBProfilerAgent.ServiceID)
	AddAgentQANMongoDBProfilerAgentC.Arg("username", "MongoDB username for scraping metrics.").
		StringVar(&AddAgentQANMongoDBProfilerAgent.Username)
	AddAgentQANMongoDBProfilerAgentC.Flag("password", "MongoDB password for scraping metrics.").StringVar(&AddAgentQANMongoDBProfilerAgent.Password)
	AddAgentQANMongoDBProfilerAgentC.Flag("custom-labels", "Custom user-assigned labels.").StringVar(&AddAgentQANMongoDBProfilerAgent.CustomLabels)
}
