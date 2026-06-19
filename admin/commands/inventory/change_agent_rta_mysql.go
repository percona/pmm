// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"fmt"
	"time"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var changeAgentRTAMySQLAgentResultT = commands.ParseTemplate(`
Real-Time Analytics MySQL agent configuration updated.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ formatCustomLabels .Agent.CustomLabels }}
Collect interval      : {{ .Agent.RtaOptions.CollectInterval }}
Log level             : {{ formatLogLevel .Agent.LogLevel }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentRTAMySQLAgentResult struct {
	Agent   *agents.ChangeAgentOKBodyRtaMysqlAgent `json:"rta_mysql_agent"`
	Changes []string                               `json:"changes,omitempty"`
}

func (res *changeAgentRTAMySQLAgentResult) Result() {}

func (res *changeAgentRTAMySQLAgentResult) String() string {
	return commands.RenderTemplate(changeAgentRTAMySQLAgentResultT, res)
}

// ChangeAgentRTAMySQLAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentRTAMySQLAgentCommand struct {
	// Embedded flags
	flags.LogLevelFatalChangeFlags

	AgentID string `arg:"" help:"Real-Time Analytics MySQL Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"MySQL username for getting queries data"`
	Password *string `help:"MySQL password for getting queries data"`

	// TLS options
	TLS           *bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify *bool   `help:"Skip TLS certificate verification"`
	TLSCaFile     *string `help:"Path to certificate authority file"`
	TLSCertFile   *string `help:"Path to client certificate file"`
	TLSKeyFile    *string `help:"Path to client key file"`

	// RTA specific options
	CollectInterval *time.Duration `placeholder:"DURATION" help:"Query collect interval (default: server-defined 2s)"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the ChangeAgentRTAMySQLAgentCommand and returns the result.
func (cmd *ChangeAgentRTAMySQLAgentCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)

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
			return nil, fmt.Errorf("failed to read TLS certificate file: %w", err)
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

	body := &agents.ChangeAgentParamsBodyRtaMysqlAgent{
		Enable:        cmd.Enable,
		Username:      cmd.Username,
		Password:      cmd.Password,
		TLS:           cmd.TLS,
		TLSSkipVerify: cmd.TLSSkipVerify,
		TLSCa:         tlsCa,
		TLSCert:       tlsCert,
		TLSKey:        tlsKey,
		LogLevel:      convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyRtaMysqlAgentCustomLabels{
			Values: *customLabels,
		}
	}

	if cmd.CollectInterval != nil {
		body.RtaOptions = &agents.ChangeAgentParamsBodyRtaMysqlAgentRtaOptions{
			CollectInterval: cmd.CollectInterval.String(),
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			RtaMysqlAgent: body,
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
		changes = append(changes, "updated TLS key")
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

	if cmd.CollectInterval != nil {
		changes = append(changes, fmt.Sprintf("changed collect interval to %s", *cmd.CollectInterval))
	}

	return &changeAgentRTAMySQLAgentResult{
		Agent:   resp.Payload.RtaMysqlAgent,
		Changes: changes,
	}, nil
}
