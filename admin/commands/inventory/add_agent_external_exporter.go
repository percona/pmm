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

package inventory

import (
	"fmt"
	"strings"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var addAgentExternalExporterResultT = commands.ParseTemplate(`
External Exporter added.
Agent ID              : {{ .Agent.AgentID }}
Runs on node ID       : {{ .Agent.RunsOnNodeID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Scheme                : {{ .Agent.Scheme }}
Metrics path          : {{ .Agent.MetricsPath }}
Listen port           : {{ .Agent.ListenPort }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentExternalExporterResult struct {
	Agent *agents.AddExternalExporterOKBodyExternalExporter `json:"external_exporter"`
}

func (res *addAgentExternalExporterResult) Result() {}

func (res *addAgentExternalExporterResult) String() string {
	return commands.RenderTemplate(addAgentExternalExporterResultT, res)
}

func (cmd *ExternalExporterCmd) RunCmd() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	params := &agents.AddExternalExporterParams{
		Body: agents.AddExternalExporterBody{
			RunsOnNodeID: cmd.RunsOnNodeID,
			ServiceID:    cmd.ServiceID,
			Username:     cmd.Username,
			Password:     cmd.Password,
			Scheme:       cmd.Scheme,
			MetricsPath:  cmd.MetricsPath,
			ListenPort:   cmd.ListenPort,
			CustomLabels: customLabels,
			PushMetrics:  cmd.PushMetrics,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddExternalExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentExternalExporterResult{
		Agent: resp.Payload.ExternalExporter,
	}, nil
}
