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

var changeAgentProxysqlExporterResultT = commands.ParseTemplate(`
ProxySQL Exporter agent configuration updated.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Listen port           : {{ .Agent.ListenPort }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}
Push metrics enabled  : {{ .Agent.PushMetricsEnabled }}
Expose exporter       : {{ .Agent.ExposeExporter }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ formatCustomLabels .Agent.CustomLabels }}
Process exec path     : {{ .Agent.ProcessExecPath }}
Log level             : {{ formatLogLevel .Agent.LogLevel }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentProxysqlExporterResult struct {
	Agent   *agents.ChangeAgentOKBodyProxysqlExporter `json:"proxysql_exporter"`
	Changes []string                                  `json:"changes,omitempty"`
}

func (res *changeAgentProxysqlExporterResult) Result() {}

func (res *changeAgentProxysqlExporterResult) String() string {
	return commands.RenderTemplate(changeAgentProxysqlExporterResultT, res)
}

// ChangeAgentProxysqlExporterCommand is used by Kong for CLI flags and commands.
type ChangeAgentProxysqlExporterCommand struct {
	AgentID string `arg:"" help:"ProxySQL Exporter Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable        *bool   `help:"Enable or disable the agent"`
	Username      *string `help:"Username for ProxySQL connection"`
	Password      *string `help:"Password for ProxySQL connection"`
	AgentPassword *string `help:"Custom password for agent /metrics endpoint"`

	// TLS options
	TLS           *bool `help:"Use TLS for database connections"`
	TLSSkipVerify *bool `help:"Skip TLS certificate and hostname validation"`

	// Exporter options
	DisableCollectors []string `help:"List of collector names to disable"`
	ExposeExporter    *bool    `help:"Expose the exporter process on all public interfaces"`
	PushMetrics       *bool    `help:"Enable push metrics with vmagent"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Log level
	flags.LogLevelFatalChangeFlags
}

// RunCmd executes the ChangeAgentProxysqlExporterCommand and returns the result.
func (cmd *ChangeAgentProxysqlExporterCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	body := &agents.ChangeAgentParamsBodyProxysqlExporter{
		Enable:            cmd.Enable,
		Username:          cmd.Username,
		Password:          cmd.Password,
		TLS:               cmd.TLS,
		TLSSkipVerify:     cmd.TLSSkipVerify,
		DisableCollectors: cmd.DisableCollectors,
		AgentPassword:     cmd.AgentPassword,
		ExposeExporter:    cmd.ExposeExporter,
		EnablePushMetrics: cmd.PushMetrics,
		LogLevel:          convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyProxysqlExporterCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			ProxysqlExporter: body,
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
	if cmd.Password != nil {
		changes = append(changes, "updated password")
	}
	if cmd.AgentPassword != nil {
		changes = append(changes, "updated agent password")
	}
	if cmd.TLS != nil {
		if *cmd.TLS {
			changes = append(changes, "enabled TLS")
		} else {
			changes = append(changes, "disabled TLS")
		}
	}
	if cmd.TLSSkipVerify != nil {
		if *cmd.TLSSkipVerify {
			changes = append(changes, "enabled TLS skip verification")
		} else {
			changes = append(changes, "disabled TLS skip verification")
		}
	}
	if cmd.DisableCollectors != nil {
		changes = append(changes, fmt.Sprintf("updated disabled collectors: %v", cmd.DisableCollectors))
	}
	if cmd.ExposeExporter != nil {
		if *cmd.ExposeExporter {
			changes = append(changes, "enabled expose exporter")
		} else {
			changes = append(changes, "disabled expose exporter")
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
		if len(*customLabels) != 0 {
			changes = append(changes, "updated custom labels")
		} else {
			changes = append(changes, "custom labels are removed")
		}
	}

	return &changeAgentProxysqlExporterResult{
		Agent:   resp.Payload.ProxysqlExporter,
		Changes: changes,
	}, nil
}
