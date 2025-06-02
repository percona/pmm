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

var changeAgentAzureDatabaseExporterResultT = commands.ParseTemplate(`
Azure Database Exporter agent configuration updated.
Agent ID                    : {{ .Agent.AgentID }}
PMM-Agent ID                : {{ .Agent.PMMAgentID }}
Node ID                     : {{ .Agent.NodeID }}
Azure Subscription ID      : {{ .Agent.AzureDatabaseSubscriptionID }}
Azure Resource Type        : {{ .Agent.AzureDatabaseResourceType }}
Listen port                 : {{ .Agent.ListenPort }}
Push metrics enabled        : {{ .Agent.PushMetricsEnabled }}

Disabled                    : {{ .Agent.Disabled }}
Custom labels               : {{ formatCustomLabels .Agent.CustomLabels }}
Process exec path           : {{ .Agent.ProcessExecPath }}
Log level                   : {{ formatLogLevel .Agent.LogLevel }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentAzureDatabaseExporterResult struct {
	Agent   *agents.ChangeAgentOKBodyAzureDatabaseExporter `json:"azure_database_exporter"`
	Changes []string                                       `json:"changes,omitempty"`
}

func (res *changeAgentAzureDatabaseExporterResult) Result() {}

func (res *changeAgentAzureDatabaseExporterResult) String() string {
	return commands.RenderTemplate(changeAgentAzureDatabaseExporterResultT, res)
}

// ChangeAgentAzureDatabaseExporterCommand is used by Kong for CLI flags and commands.
type ChangeAgentAzureDatabaseExporterCommand struct {
	AgentID string `arg:"" help:"Azure Database Exporter Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable *bool `help:"Enable or disable the agent"`

	// Azure credentials
	AzureClientID       *string `help:"Azure Client ID"`
	AzureClientSecret   *string `help:"Azure Client Secret"`
	AzureTenantID       *string `help:"Azure Tenant ID"`
	AzureSubscriptionID *string `help:"Azure Subscription ID"`
	AzureResourceGroup  *string `help:"Azure Resource Group"`

	// Exporter options
	PushMetrics *bool `help:"Enable push metrics with vmagent"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Log level
	flags.LogLevelFatalChangeFlags
}

// RunCmd executes the ChangeAgentAzureDatabaseExporterCommand and returns the result.
func (cmd *ChangeAgentAzureDatabaseExporterCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	body := &agents.ChangeAgentParamsBodyAzureDatabaseExporter{
		Enable:              cmd.Enable,
		AzureClientID:       cmd.AzureClientID,
		AzureClientSecret:   cmd.AzureClientSecret,
		AzureTenantID:       cmd.AzureTenantID,
		AzureSubscriptionID: cmd.AzureSubscriptionID,
		AzureResourceGroup:  cmd.AzureResourceGroup,
		EnablePushMetrics:   cmd.PushMetrics,
		LogLevel:            convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyAzureDatabaseExporterCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			AzureDatabaseExporter: body,
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
	if cmd.AzureClientID != nil {
		changes = append(changes, "updated azure_client_id")
	}
	if cmd.AzureClientSecret != nil {
		changes = append(changes, "updated azure_client_secret")
	}
	if cmd.AzureTenantID != nil {
		changes = append(changes, "updated azure_tenant_id")
	}
	if cmd.AzureSubscriptionID != nil {
		changes = append(changes, "updated azure_subscription_id")
	}
	if cmd.AzureResourceGroup != nil {
		changes = append(changes, "updated azure_resource_group")
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

	return &changeAgentAzureDatabaseExporterResult{
		Agent:   resp.Payload.AzureDatabaseExporter,
		Changes: changes,
	}, nil
}
