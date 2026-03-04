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

var changeAgentRTAMongoDBAgentResultT = commands.ParseTemplate(`
Real-Time Analytics MongoDB agent configuration updated.
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

type changeAgentRTAMongoDBAgentResult struct {
	Agent   *agents.ChangeAgentOKBodyRtaMongodbAgent `json:"rta_mongodb_agent"`
	Changes []string                                 `json:"changes,omitempty"`
}

func (res *changeAgentRTAMongoDBAgentResult) Result() {}

func (res *changeAgentRTAMongoDBAgentResult) String() string {
	return commands.RenderTemplate(changeAgentRTAMongoDBAgentResultT, res)
}

// ChangeAgentRTAMongoDBAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentRTAMongoDBAgentCommand struct {
	// Embedded flags
	flags.LogLevelFatalChangeFlags

	AgentID string `arg:"" help:"Real-Time Analytics MongoDB Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"MongoDB username for scraping metrics"`
	Password *string `help:"MongoDB password for scraping metrics"`

	SkipConnectionCheck bool `help:"Skip connection check"`

	// TLS options
	TLS                           *bool   `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 *bool   `help:"Skip TLS certificate verification"`
	TLSCertificateKeyFile         *string `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword *string `help:"Password for certificate"`
	TLSCaFile                     *string `help:"Path to certificate authority file"`

	// MongoDB specific options
	AuthenticationMechanism *string `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`

	// RTA specific options
	CollectInterval *time.Duration `placeholder:"DURATION" help:"Query collect interval (default: server-defined 2s)"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the ChangeAgentRTAMongoDBAgentCommand and returns the result.
func (cmd *ChangeAgentRTAMongoDBAgentCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseKeyValuePair(cmd.CustomLabels)

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

	body := &agents.ChangeAgentParamsBodyRtaMongodbAgent{
		Enable:                        cmd.Enable,
		Username:                      cmd.Username,
		Password:                      cmd.Password,
		TLS:                           cmd.TLS,
		TLSSkipVerify:                 cmd.TLSSkipVerify,
		TLSCertificateKey:             tlsCertificateKey,
		TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
		TLSCa:                         tlsCa,
		AuthenticationMechanism:       cmd.AuthenticationMechanism,
		LogLevel:                      convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyRtaMongodbAgentCustomLabels{
			Values: *customLabels,
		}
	}

	if cmd.CollectInterval != nil {
		body.RtaOptions = &agents.ChangeAgentParamsBodyRtaMongodbAgentRtaOptions{
			CollectInterval: cmd.CollectInterval.String(),
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			RtaMongodbAgent: body,
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
		changes = append(changes, "changed authentication mechanism to "+*cmd.AuthenticationMechanism)
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

	return &changeAgentRTAMongoDBAgentResult{
		Agent:   resp.Payload.RtaMongodbAgent,
		Changes: changes,
	}, nil
}
