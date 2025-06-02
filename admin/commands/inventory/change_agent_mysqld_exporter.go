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

var changeAgentMysqldExporterResultT = commands.ParseTemplate(`
MySQL Exporter agent configuration updated.
Agent ID                     : {{ .Agent.AgentID }}
PMM-Agent ID                 : {{ .Agent.PMMAgentID }}
Service ID                   : {{ .Agent.ServiceID }}
Username                     : {{ .Agent.Username }}
Listen port                  : {{ .Agent.ListenPort }}
TLS enabled                  : {{ .Agent.TLS }}
Skip TLS verification        : {{ .Agent.TLSSkipVerify }}
Push metrics enabled         : {{ .Agent.PushMetricsEnabled }}
Expose exporter              : {{ .Agent.ExposeExporter }}

Disabled                     : {{ .Agent.Disabled }}
Custom labels                : {{ formatCustomLabels .Agent.CustomLabels }}
Process exec path            : {{ .Agent.ProcessExecPath }}
Log level                    : {{ formatLogLevel .Agent.LogLevel }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentMysqldExporterResult struct {
	Agent   *agents.ChangeAgentOKBodyMysqldExporter `json:"mysqld_exporter"`
	Changes []string                                `json:"changes,omitempty"`
}

func (res *changeAgentMysqldExporterResult) Result() {}

func (res *changeAgentMysqldExporterResult) String() string {
	return commands.RenderTemplate(changeAgentMysqldExporterResultT, res)
}

// ChangeAgentMysqldExporterCommand is used by Kong for CLI flags and commands.
type ChangeAgentMysqldExporterCommand struct {
	AgentID string `arg:"" help:"MySQL Exporter Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable        *bool   `help:"Enable or disable the agent"`
	Username      *string `help:"Username for MySQL connection"`
	Password      *string `help:"Password for MySQL connection"`
	AgentPassword *string `help:"Custom password for agent /metrics endpoint"`

	// TLS options
	TLS           *bool   `help:"Use TLS for database connections"`
	TLSSkipVerify *bool   `help:"Skip TLS certificate and hostname validation"`
	TLSCaFile     *string `help:"TLS CA certificate file"`
	TLSCertFile   *string `help:"TLS certificate file"`
	TLSKeyFile    *string `help:"TLS certificate key file"`

	// Exporter options
	TablestatsGroupTableLimit *int32   `help:"Tablestats group collectors by table limit"`
	DisableCollectors         []string `help:"List of collector names to disable"`
	ExposeExporter            *bool    `help:"Expose the exporter process on all public interfaces"`
	PushMetrics               *bool    `help:"Enable push metrics with vmagent"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Log level
	flags.LogLevelNoFatalChangeFlags
}

// RunCmd executes the ChangeAgentMysqldExporterCommand and returns the result.
func (cmd *ChangeAgentMysqldExporterCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	// Read TLS files if provided
	var tlsCa, tlsCert, tlsKey *string

	if cmd.TLSCaFile != nil {
		content, err := commands.ReadFile(*cmd.TLSCaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS CA file: %w", err)
		}
		tlsCa = &content
	}

	if cmd.TLSCertFile != nil {
		content, err := commands.ReadFile(*cmd.TLSCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS cert file: %w", err)
		}
		tlsCert = &content
	}

	if cmd.TLSKeyFile != nil {
		content, err := commands.ReadFile(*cmd.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS key file: %w", err)
		}
		tlsKey = &content
	}

	body := &agents.ChangeAgentParamsBodyMysqldExporter{
		Enable:                    cmd.Enable,
		Username:                  cmd.Username,
		Password:                  cmd.Password,
		TLS:                       cmd.TLS,
		TLSSkipVerify:             cmd.TLSSkipVerify,
		TLSCa:                     tlsCa,
		TLSCert:                   tlsCert,
		TLSKey:                    tlsKey,
		TablestatsGroupTableLimit: cmd.TablestatsGroupTableLimit,
		DisableCollectors:         cmd.DisableCollectors,
		AgentPassword:             cmd.AgentPassword,
		ExposeExporter:            cmd.ExposeExporter,
		EnablePushMetrics:         cmd.PushMetrics,
		LogLevel:                  convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyMysqldExporterCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			MysqldExporter: body,
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
	if cmd.TLSCaFile != nil {
		changes = append(changes, "updated TLS CA certificate")
	}
	if cmd.TLSCertFile != nil {
		changes = append(changes, "updated TLS certificate")
	}
	if cmd.TLSKeyFile != nil {
		changes = append(changes, "updated TLS certificate key")
	}
	if cmd.TablestatsGroupTableLimit != nil {
		changes = append(changes, fmt.Sprintf("changed tablestats group table limit to %d", *cmd.TablestatsGroupTableLimit))
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

	return &changeAgentMysqldExporterResult{
		Agent:   resp.Payload.MysqldExporter,
		Changes: changes,
	}, nil
}
