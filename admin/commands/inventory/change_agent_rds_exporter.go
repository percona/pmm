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
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var changeAgentRDSExporterResultT = commands.ParseTemplate(`
RDS Exporter agent configuration updated.
Agent ID                   : {{ .Agent.AgentID }}
PMM-Agent ID               : {{ .Agent.PMMAgentID }}
Node ID                    : {{ .Agent.NodeID }}
Listen port                : {{ .Agent.ListenPort }}
Push metrics enabled       : {{ .Agent.PushMetricsEnabled }}

Disabled                   : {{ .Agent.Disabled }}
Basic metrics disabled     : {{ .Agent.BasicMetricsDisabled }}
Enhanced metrics disabled  : {{ .Agent.EnhancedMetricsDisabled }}
Custom labels              : {{ formatCustomLabels .Agent.CustomLabels }}
Process exec path          : {{ .Agent.ProcessExecPath }}
Log level                  : {{ formatLogLevel .Agent.LogLevel }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentRDSExporterResult struct {
	Agent   *agents.ChangeAgentOKBodyRDSExporter `json:"rds_exporter"`
	Changes []string                             `json:"changes,omitempty"`
}

func (res *changeAgentRDSExporterResult) Result() {}

func (res *changeAgentRDSExporterResult) String() string {
	return commands.RenderTemplate(changeAgentRDSExporterResultT, res)
}

// ChangeAgentRDSExporterCommand is used by Kong for CLI flags and commands.
type ChangeAgentRDSExporterCommand struct {
	AgentID string `arg:"" help:"RDS Exporter Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable *bool `help:"Enable or disable the agent"`

	// AWS credentials
	AWSAccessKey *string `help:"AWS access key"`
	AWSSecretKey *string `help:"AWS secret key"`

	// RDS-specific options
	DisableBasicMetrics    *bool `help:"Disable basic metrics"`
	DisableEnhancedMetrics *bool `help:"Disable enhanced metrics"`

	// Exporter options
	PushMetrics *bool `help:"Enable push metrics with vmagent"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Log level
	flags.LogLevelFatalChangeFlags
}

// RunCmd executes the ChangeAgentRDSExporterCommand and returns the result.
func (cmd *ChangeAgentRDSExporterCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	body := &agents.ChangeAgentParamsBodyRDSExporter{
		Enable:                 cmd.Enable,
		AWSAccessKey:           cmd.AWSAccessKey,
		AWSSecretKey:           cmd.AWSSecretKey,
		DisableBasicMetrics:    cmd.DisableBasicMetrics,
		DisableEnhancedMetrics: cmd.DisableEnhancedMetrics,
		EnablePushMetrics:      cmd.PushMetrics,
		LogLevel:               convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyRDSExporterCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			RDSExporter: body,
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
	if cmd.AWSAccessKey != nil {
		changes = append(changes, "updated AWS access key")
	}
	if cmd.AWSSecretKey != nil {
		changes = append(changes, "updated AWS secret key")
	}
	if cmd.DisableBasicMetrics != nil {
		if *cmd.DisableBasicMetrics {
			changes = append(changes, "disabled basic metrics")
		} else {
			changes = append(changes, "enabled basic metrics")
		}
	}
	if cmd.DisableEnhancedMetrics != nil {
		if *cmd.DisableEnhancedMetrics {
			changes = append(changes, "disabled enhanced metrics")
		} else {
			changes = append(changes, "enabled enhanced metrics")
		}
	}
	if cmd.PushMetrics != nil {
		if *cmd.PushMetrics {
			changes = append(changes, "enabled push metrics")
		} else {
			changes = append(changes, "disabled push metrics")
		}
	}
	if cmd.LogLevel != nil {
		changes = append(changes, fmt.Sprintf("changed log level to %s", *cmd.LogLevel))
	}
	if customLabels != nil {
		if len(*customLabels) > 0 {
			changes = append(changes, "updated custom labels")
		} else {
			changes = append(changes, "custom labels are removed")
		}
	}

	return &changeAgentRDSExporterResult{
		Agent:   resp.Payload.RDSExporter,
		Changes: changes,
	}, nil
}
