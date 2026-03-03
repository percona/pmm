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

package management

import (
	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addOtelResultT = commands.ParseTemplate(`
OTEL Collector added.
Agent ID     : {{ .Agent.AgentID }}
PMM-Agent ID : {{ .Agent.PMMAgentID }}
Status       : {{ .Agent.Status }}
Disabled     : {{ .Agent.Disabled }}
`)

type addOtelResult struct {
	Agent *agents.AddAgentOKBodyOtelCollector `json:"otel_collector"`
}

func (res *addOtelResult) Result() {}

func (res *addOtelResult) String() string {
	return commands.RenderTemplate(addOtelResultT, res)
}

// AddOtelCommand is used by Kong for CLI flags and commands.
type AddOtelCommand struct {
	PMMAgentID   string            `help:"Node ID where pmm-agent runs (default is autodetected)"`
	LogFilePaths []string          `name:"log-file-paths" help:"Comma-separated list of log file paths to collect (e.g. /var/log/mysql/error.log)"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd runs the command for AddOtelCommand.
func (cmd *AddOtelCommand) RunCmd() (commands.Result, error) {
	pmmAgentID := cmd.PMMAgentID
	if pmmAgentID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		pmmAgentID = status.AgentID
	}

	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			OtelCollector: &agents.AddAgentParamsBodyOtelCollector{
				PMMAgentID:   pmmAgentID,
				CustomLabels: customLabels,
				LogFilePaths: cmd.LogFilePaths,
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addOtelResult{
		Agent: resp.Payload.OtelCollector,
	}, nil
}
