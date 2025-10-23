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

var changeAgentQANMySQLSlowlogAgentResultT = commands.ParseTemplate(`
QAN MySQL SlowLog agent configuration updated.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

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

type changeAgentQANMySQLSlowlogAgentResult struct {
	Agent   *agents.ChangeAgentOKBodyQANMysqlSlowlogAgent `json:"qan_mysql_slowlog_agent"`
	Changes []string                                      `json:"changes,omitempty"`
}

func (res *changeAgentQANMySQLSlowlogAgentResult) Result() {}

func (res *changeAgentQANMySQLSlowlogAgentResult) String() string {
	return commands.RenderTemplate(changeAgentQANMySQLSlowlogAgentResultT, res)
}

// ChangeAgentQANMySQLSlowlogAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentQANMySQLSlowlogAgentCommand struct {
	// Embedded flags
	flags.CommentsParsingChangeFlags
	flags.LogLevelFatalChangeFlags

	AgentID string `arg:"" help:"QAN MySQL SlowLog Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"Username for MySQL connection"`
	Password *string `help:"Password for MySQL connection"`

	// TLS options
	TLS           *bool   `help:"Use TLS for database connections"`
	TLSSkipVerify *bool   `help:"Skip TLS certificate and hostname validation"`
	TLSCaFile     *string `help:"TLS CA certificate file"`
	TLSCertFile   *string `help:"TLS certificate file"`
	TLSKeyFile    *string `help:"TLS certificate key file"`

	// QAN specific options
	MaxSlowlogFileSize   *string `help:"Maximum size of slow log file in bytes (default: 1GiB; 0: no limit)"`
	MaxQueryLength       *int32  `help:"Maximum query length for QAN (default: server-defined; -1: no limit)"`
	DisableQueryExamples *bool   `help:"Disable query examples"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`
}

// RunCmd executes the ChangeAgentQANMySQLSlowlogAgentCommand and returns the result.
func (cmd *ChangeAgentQANMySQLSlowlogAgentCommand) RunCmd() (commands.Result, error) {
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

	body := &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgent{
		Enable:                 cmd.Enable,
		Username:               cmd.Username,
		Password:               cmd.Password,
		TLS:                    cmd.TLS,
		TLSSkipVerify:          cmd.TLSSkipVerify,
		TLSCa:                  tlsCa,
		TLSCert:                tlsCert,
		TLSKey:                 tlsKey,
		MaxSlowlogFileSize:     cmd.MaxSlowlogFileSize,
		MaxQueryLength:         cmd.MaxQueryLength,
		DisableQueryExamples:   cmd.DisableQueryExamples,
		DisableCommentsParsing: cmd.CommentsParsingDisabled(),
		LogLevel:               convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyQANMysqlSlowlogAgentCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			QANMysqlSlowlogAgent: body,
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
		changes = append(changes, "updated TLS certificate key")
	}
	if cmd.MaxSlowlogFileSize != nil {
		changes = append(changes, "changed max slowlog file size to "+*cmd.MaxSlowlogFileSize)
	}
	if cmd.MaxQueryLength != nil {
		changes = append(changes, fmt.Sprintf("changed max query length to %d", *cmd.MaxQueryLength))
	}
	if cmd.DisableQueryExamples != nil {
		if *cmd.DisableQueryExamples {
			changes = append(changes, "disabled query examples")
		} else {
			changes = append(changes, "enabled query examples")
		}
	}
	if cmd.CommentsParsing != nil {
		if *cmd.CommentsParsingDisabled() {
			changes = append(changes, "disabled comments parsing")
		} else {
			changes = append(changes, "enabled comments parsing")
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

	return &changeAgentQANMySQLSlowlogAgentResult{
		Agent:   resp.Payload.QANMysqlSlowlogAgent,
		Changes: changes,
	}, nil
}
