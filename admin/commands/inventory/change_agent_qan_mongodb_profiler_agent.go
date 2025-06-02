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

var changeAgentQANMongoDBProfilerAgentResultT = commands.ParseTemplate(`
QAN MongoDB Profiler agent configuration updated.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}
Max query length      : {{ .Agent.MaxQueryLength }}

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

type changeAgentQANMongoDBProfilerAgentResult struct {
	Agent   *agents.ChangeAgentOKBodyQANMongodbProfilerAgent `json:"qan_mongodb_profiler_agent"`
	Changes []string                                         `json:"changes,omitempty"`
}

func (res *changeAgentQANMongoDBProfilerAgentResult) Result() {}

func (res *changeAgentQANMongoDBProfilerAgentResult) String() string {
	return commands.RenderTemplate(changeAgentQANMongoDBProfilerAgentResultT, res)
}

// ChangeAgentQANMongoDBProfilerAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentQANMongoDBProfilerAgentCommand struct {
	AgentID string `arg:"" help:"QAN MongoDB Profiler Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"Username for MongoDB connection"`
	Password *string `help:"Password for MongoDB connection"`

	// TLS options
	TLS                           *bool   `help:"Use TLS for database connections"`
	TLSSkipVerify                 *bool   `help:"Skip TLS certificate and hostname validation"`
	TLSCertificateKeyFile         *string `help:"TLS certificate key file"`
	TLSCertificateKeyFilePassword *string `help:"TLS certificate key file password"`
	TLSCaFile                     *string `help:"TLS CA certificate file"`

	// MongoDB specific options
	AuthenticationMechanism *string `help:"Authentication mechanism for MongoDB"`
	AuthenticationDatabase  *string `help:"Authentication database for MongoDB"`

	// QAN specific options
	MaxQueryLength *int32 `help:"Maximum query length for QAN (default: server-defined; -1: no limit)"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Log level
	flags.LogLevelFatalChangeFlags
}

// RunCmd executes the ChangeAgentQANMongoDBProfilerAgentCommand and returns the result.
func (cmd *ChangeAgentQANMongoDBProfilerAgentCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	// Read TLS files if provided
	var tlsCertificateKey, tlsCa *string

	if cmd.TLSCertificateKeyFile != nil {
		content, err := commands.ReadFile(*cmd.TLSCertificateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS certificate key file: %w", err)
		}
		tlsCertificateKey = &content
	}

	if cmd.TLSCaFile != nil {
		content, err := commands.ReadFile(*cmd.TLSCaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS CA file: %w", err)
		}
		tlsCa = &content
	}

	body := &agents.ChangeAgentParamsBodyQANMongodbProfilerAgent{
		Enable:                        cmd.Enable,
		Username:                      cmd.Username,
		Password:                      cmd.Password,
		TLS:                           cmd.TLS,
		TLSSkipVerify:                 cmd.TLSSkipVerify,
		TLSCertificateKey:             tlsCertificateKey,
		TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
		TLSCa:                         tlsCa,
		AuthenticationMechanism:       cmd.AuthenticationMechanism,
		AuthenticationDatabase:        cmd.AuthenticationDatabase,
		MaxQueryLength:                cmd.MaxQueryLength,
		LogLevel:                      convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil && len(*customLabels) > 0 {
		body.CustomLabels = &agents.ChangeAgentParamsBodyQANMongodbProfilerAgentCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			QANMongodbProfilerAgent: body,
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
	if cmd.TLSCertificateKeyFile != nil {
		changes = append(changes, "updated TLS certificate key")
	}
	if cmd.TLSCertificateKeyFilePassword != nil {
		changes = append(changes, "updated TLS certificate key password")
	}
	if cmd.TLSCaFile != nil {
		changes = append(changes, "updated TLS CA certificate")
	}
	if cmd.AuthenticationMechanism != nil {
		changes = append(changes, fmt.Sprintf("changed authentication mechanism to %s", *cmd.AuthenticationMechanism))
	}
	if cmd.AuthenticationDatabase != nil {
		changes = append(changes, fmt.Sprintf("changed authentication database to %s", *cmd.AuthenticationDatabase))
	}
	if cmd.MaxQueryLength != nil {
		changes = append(changes, fmt.Sprintf("changed max query length to %d", *cmd.MaxQueryLength))
	}
	if cmd.LogLevel != nil {
		changes = append(changes, fmt.Sprintf("changed log level to %s", *cmd.LogLevel))
	}
	if customLabels != nil && len(*customLabels) > 0 {
		changes = append(changes, "updated custom labels")
	}

	return &changeAgentQANMongoDBProfilerAgentResult{
		Agent:   resp.Payload.QANMongodbProfilerAgent,
		Changes: changes,
	}, nil
}
