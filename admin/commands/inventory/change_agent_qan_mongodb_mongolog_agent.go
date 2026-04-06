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

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var changeAgentQANMongoDBMongologAgentResultT = commands.ParseTemplate(`
QAN MongoDB Mongolog agent configuration updated.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}
Max query length      : {{ .Agent.MaxQueryLength }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ formatCustomLabels .Agent.CustomLabels }}
Log level             : {{ formatLogLevel .Agent.LogLevel }}

{{- if .Changes}}
Configuration changes applied:
{{- range .Changes}}
  - {{ . }}
{{- end}}
{{- end}}
`)

type changeAgentQANMongoDBMongologAgentResult struct {
	Agent   *agents.ChangeAgentOKBodyQANMongodbMongologAgent `json:"qan_mongodb_mongolog_agent"`
	Changes []string                                         `json:"changes,omitempty"`
}

func (res *changeAgentQANMongoDBMongologAgentResult) Result() {}

func (res *changeAgentQANMongoDBMongologAgentResult) String() string {
	return commands.RenderTemplate(changeAgentQANMongoDBMongologAgentResultT, res)
}

// ChangeAgentQANMongoDBMongologAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentQANMongoDBMongologAgentCommand struct {
	// Embedded flags
	flags.LogLevelFatalChangeFlags

	AgentID string `arg:"" help:"QAN MongoDB Mongolog Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"MongoDB username for scraping metrics"`
	Password *string `help:"MongoDB password for scraping metrics"`

	// TLS options
	TLS                           *bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 *bool   `help:"Skip TLS certificate verification"`
	TLSCertificateKeyFile         *string `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword *string `help:"Password for certificate"`
	TLSCaFile                     *string `help:"Path to certificate authority file"`

	// Limit query length in QAN (default: server-defined; -1: no limit).
	MaxQueryLength *int32 `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`

	// MongoDB specific options
	AuthenticationMechanism *string `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
	AuthenticationDatabase  *string `help:"Authentication database."`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the ChangeAgentQANMongoDBMongologAgentCommand and returns the result.
func (cmd *ChangeAgentQANMongoDBMongologAgentCommand) RunCmd() (commands.Result, error) {
	var changes []string

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

	body := &agents.ChangeAgentParamsBodyQANMongodbMongologAgent{
		Enable:                        cmd.Enable,
		Username:                      cmd.Username,
		Password:                      cmd.Password,
		TLS:                           cmd.TLS,
		TLSSkipVerify:                 cmd.TLSSkipVerify,
		TLSCertificateKey:             tlsCertificateKey,
		TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
		TLSCa:                         tlsCa,
		MaxQueryLength:                cmd.MaxQueryLength,
		AuthenticationMechanism:       cmd.AuthenticationMechanism,
		AuthenticationDatabase:        cmd.AuthenticationDatabase,
		LogLevel:                      convertLogLevelPtr(cmd.LogLevel),
	}

	// Parse custom labels if provided
	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)
	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyQANMongodbMongologAgentCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			QANMongodbMongologAgent: body,
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
	if cmd.MaxQueryLength != nil {
		changes = append(changes, fmt.Sprintf("changed max query length to %d", *cmd.MaxQueryLength))
	}
	if cmd.AuthenticationMechanism != nil {
		changes = append(changes, "changed authentication mechanism to "+*cmd.AuthenticationMechanism)
	}
	if cmd.AuthenticationDatabase != nil {
		changes = append(changes, "changed authentication database to "+*cmd.AuthenticationDatabase)
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

	return &changeAgentQANMongoDBMongologAgentResult{
		Agent:   resp.Payload.QANMongodbMongologAgent,
		Changes: changes,
	}, nil
}
