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
	"strings"

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
	PMMAgentID    string            `help:"Node ID where pmm-agent runs (default is autodetected)"`
	LogFilePaths  []string          `name:"log-file-paths" help:"Comma-separated list of log file paths to collect (e.g. /var/log/mysql/error.log). Used with --parser-preset or as raw if --log-sources not set."`
	LogSources    string            `name:"log-sources" help:"Comma-separated path:preset pairs (e.g. /var/log/mysql/error.log:mysql_error,/other.log:raw). Preset 'raw' means no parsing. Available presets: mysql_error, nginx_access, nginx_error, grafana, pmm_managed, pmm_agent, postgres, raw. Overrides --log-file-paths and --parser-preset."`
	ParserPreset  string            `name:"parser-preset" help:"Parser preset for all paths from --log-file-paths. Available presets: mysql_error, nginx_access, nginx_error, grafana, pmm_managed, pmm_agent, postgres, raw. Ignored if --log-sources is set."`
	CustomLabels  map[string]string `mapsep:"," help:"Custom user-assigned labels"`
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

	body := &agents.AddAgentParamsBodyOtelCollector{
		PMMAgentID:   pmmAgentID,
		CustomLabels: customLabels,
	}
	if cmd.LogSources != "" {
		// Parse path:preset pairs.
		for _, pair := range strings.Split(cmd.LogSources, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			path := pair
			preset := "raw"
			if idx := strings.Index(pair, ":"); idx >= 0 {
				path = strings.TrimSpace(pair[:idx])
				preset = strings.TrimSpace(pair[idx+1:])
				if preset == "" {
					preset = "raw"
				}
			}
			if path != "" {
				body.LogSources = append(body.LogSources, &agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0{
					Path:   path,
					Preset: preset,
				})
			}
		}
	} else if len(cmd.LogFilePaths) != 0 {
		if cmd.ParserPreset != "" {
			for _, p := range cmd.LogFilePaths {
				p = strings.TrimSpace(p)
				if p != "" {
					body.LogSources = append(body.LogSources, &agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0{
						Path:   p,
						Preset: cmd.ParserPreset,
					})
				}
			}
		} else {
			body.LogFilePaths = cmd.LogFilePaths
		}
	}

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			OtelCollector: body,
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
