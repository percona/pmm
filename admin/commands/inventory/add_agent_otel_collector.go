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
	"strings"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addAgentOTELCollectorResultT = commands.ParseTemplate(`
OTEL Collector added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Listen port           : {{ .Agent.ListenPort }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}

Logs Collection       : {{ if .Agent.LogsConfig }}{{ if .Agent.LogsConfig.Enabled }}Enabled{{ else }}Disabled{{ end }}{{ else }}Disabled{{ end }}
{{ if .Agent.LogsConfig }}{{ if .Agent.LogsConfig.LogSources }}Log Sources:
{{ range .Agent.LogsConfig.LogSources }}  - {{ .ServiceName }}: {{ .Path }}
{{ end }}{{ end }}{{ end }}
`)

type addAgentOTELCollectorResult struct {
	Agent *agents.AddAgentOKBodyOtelCollector `json:"otel_collector"`
}

func (res *addAgentOTELCollectorResult) Result() {}

func (res *addAgentOTELCollectorResult) String() string {
	return commands.RenderTemplate(addAgentOTELCollectorResultT, res)
}

// AddAgentOTELCollectorCommand is used by Kong for CLI flags and commands.
// OTEL Collector collects logs (and future: traces, profiles, eBPF) from database nodes.
type AddAgentOTELCollectorCommand struct {
	PMMAgentID   string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Logs collection settings
	EnableLogs     bool     `help:"Enable logs collection"`
	LogSources     []string `help:"Log sources in format 'service_name:path:parser_type' (e.g., 'mysql:/var/log/mysql/error.log:regex')"`
	EnableSyslog   bool     `help:"Enable syslog receiver"`
	SyslogPort     uint32   `default:"514" help:"Port for syslog receiver"`
	EnableJournald bool     `help:"Enable journald receiver for systemd logs"`
	JournaldUnits  []string `help:"Systemd units to collect logs from (e.g., 'mysqld', 'postgresql')"`
}

// RunCmd executes the AddAgentOTELCollectorCommand and returns the result.
func (cmd *AddAgentOTELCollectorCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)

	// Parse log sources from CLI format
	var logSources []*agents.AddAgentParamsBodyOtelCollectorLogsConfigLogSourcesItems0
	for _, src := range cmd.LogSources {
		parts := strings.SplitN(src, ":", 3)
		if len(parts) >= 2 {
			logSource := &agents.AddAgentParamsBodyOtelCollectorLogsConfigLogSourcesItems0{
				ServiceName: parts[0],
				Path:        parts[1],
			}
			if len(parts) >= 3 {
				logSource.ParserType = parts[2]
			} else {
				logSource.ParserType = "raw" // default parser
			}
			logSources = append(logSources, logSource)
		}
	}

	// Build logs config
	var logsConfig *agents.AddAgentParamsBodyOtelCollectorLogsConfig
	if cmd.EnableLogs || len(logSources) > 0 || cmd.EnableSyslog || cmd.EnableJournald {
		logsConfig = &agents.AddAgentParamsBodyOtelCollectorLogsConfig{
			Enabled:        cmd.EnableLogs || len(logSources) > 0,
			LogSources:     logSources,
			EnableSyslog:   cmd.EnableSyslog,
			SyslogPort:     cmd.SyslogPort,
			EnableJournald: cmd.EnableJournald,
			JournaldUnits:  cmd.JournaldUnits,
		}
	}

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			OtelCollector: &agents.AddAgentParamsBodyOtelCollector{
				PMMAgentID:   cmd.PMMAgentID,
				CustomLabels: customLabels,
				LogsConfig:   logsConfig,
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentOTELCollectorResult{
		Agent: resp.Payload.OtelCollector,
	}, nil
}

