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
	"fmt"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var changeAgentExternalExporterResultT = commands.ParseTemplate(`
External Exporter agent configuration updated.
Agent ID        : {{ .Agent.AgentID }}
Runs on node ID : {{ .Agent.RunsOnNodeID }}
Service ID      : {{ .Agent.ServiceID }}
Username        : {{ .Agent.Username }}
Scheme          : {{ .Agent.Scheme }}
Metrics path    : {{ .Agent.MetricsPath }}
Listen port     : {{ .Agent.ListenPort }}

Disabled        : {{ .Agent.Disabled }}
Custom labels   : {{ formatCustomLabels .Agent.CustomLabels }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentExternalExporterResult struct {
	Agent   *agents.ChangeAgentOKBodyExternalExporter `json:"external_exporter"`
	Changes []string                                  `json:"changes,omitempty"`
}

func (res *changeAgentExternalExporterResult) Result() {}

func (res *changeAgentExternalExporterResult) String() string {
	return commands.RenderTemplate(changeAgentExternalExporterResultT, res)
}

// ChangeAgentExternalExporterCommand is used by Kong for CLI flags and commands.
type ChangeAgentExternalExporterCommand struct {
	AgentID string `arg:"" help:"External Exporter Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"Username for the external exporter"`

	// External-specific options
	ListenPort    *int64  `help:"Listen port for the external exporter"`
	MetricsScheme *string `help:"Metrics scheme (http or https)"`
	MetricsPath   *string `help:"Metrics path"`
	PushMetrics   *bool   `help:"Enable push metrics with vmagent"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the ChangeAgentExternalExporterCommand and returns the result.
func (cmd *ChangeAgentExternalExporterCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	body := &agents.ChangeAgentParamsBodyExternalExporter{
		Enable:            cmd.Enable,
		Username:          cmd.Username,
		ListenPort:        cmd.ListenPort,
		Scheme:            cmd.MetricsScheme,
		MetricsPath:       cmd.MetricsPath,
		EnablePushMetrics: cmd.PushMetrics,
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyExternalExporterCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			ExternalExporter: body,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.ChangeAgent(params)
	if err != nil {
		return nil, err
	}

	// Track changes
	if cmd.Enable != nil {
		if *cmd.Enable {
			changes = append(changes, "enabled agent")
		} else {
			changes = append(changes, "disabled agent")
		}
	}
	if cmd.Username != nil {
		changes = append(changes, "updated username")
	}
	if cmd.ListenPort != nil {
		changes = append(changes, fmt.Sprintf("changed listen port to %d", *cmd.ListenPort))
	}
	if cmd.MetricsScheme != nil {
		changes = append(changes, fmt.Sprintf("changed metrics scheme to %s", *cmd.MetricsScheme))
	}
	if cmd.MetricsPath != nil {
		changes = append(changes, fmt.Sprintf("changed metrics path to %s", *cmd.MetricsPath))
	}
	if cmd.PushMetrics != nil {
		if *cmd.PushMetrics {
			changes = append(changes, "enabled push metrics")
		} else {
			changes = append(changes, "disabled push metrics")
		}
	}
	if customLabels != nil {
		if len(*customLabels) > 0 {
			changes = append(changes, "updated custom labels")
		} else {
			changes = append(changes, "custom labels are removed")
		}
	}

	return &changeAgentExternalExporterResult{
		Agent:   resp.Payload.ExternalExporter,
		Changes: changes,
	}, nil
}
