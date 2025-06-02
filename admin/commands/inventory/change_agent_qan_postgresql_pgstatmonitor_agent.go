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

var changeAgentQANPostgreSQLPgStatMonitorAgentResultT = commands.ParseTemplate(`
QAN PostgreSQL PgStatMonitor agent configuration updated.
Agent ID                     : {{ .Agent.AgentID }}
PMM-Agent ID                 : {{ .Agent.PMMAgentID }}
Service ID                   : {{ .Agent.ServiceID }}
Username                     : {{ .Agent.Username }}
TLS enabled                  : {{ .Agent.TLS }}
Skip TLS verification        : {{ .Agent.TLSSkipVerify }}
Max query length             : {{ .Agent.MaxQueryLength }}
Query examples disabled      : {{ .Agent.QueryExamplesDisabled }}
Disable comments parsing     : {{ .Agent.DisableCommentsParsing }}

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

type changeAgentQANPostgreSQLPgStatMonitorAgentResult struct {
	Agent   *agents.ChangeAgentOKBodyQANPostgresqlPgstatmonitorAgent `json:"qan_postgresql_pgstatmonitor_agent"`
	Changes []string                                                 `json:"changes,omitempty"`
}

func (res *changeAgentQANPostgreSQLPgStatMonitorAgentResult) Result() {}

func (res *changeAgentQANPostgreSQLPgStatMonitorAgentResult) String() string {
	return commands.RenderTemplate(changeAgentQANPostgreSQLPgStatMonitorAgentResultT, res)
}

// ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand is used by Kong for CLI flags and commands.
type ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand struct {
	AgentID string `arg:"" help:"QAN PostgreSQL PgStatMonitor Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable   *bool   `help:"Enable or disable the agent"`
	Username *string `help:"Username for PostgreSQL connection"`
	Password *string `help:"Password for PostgreSQL connection"`

	// TLS options
	TLS           *bool   `help:"Use TLS for database connections"`
	TLSSkipVerify *bool   `help:"Skip TLS certificate and hostname validation"`
	TLSCaFile     *string `help:"TLS CA certificate file"`
	TLSCertFile   *string `help:"TLS certificate file"`
	TLSKeyFile    *string `help:"TLS certificate key file"`

	// QAN specific options
	MaxQueryLength       *int32 `help:"Maximum query length for QAN (default: server-defined; -1: no limit)"`
	DisableQueryExamples *bool  `help:"Disable query examples"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	flags.CommentsParsingChangeFlags
	flags.LogLevelFatalChangeFlags
}

// RunCmd executes the ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand and returns the result.
func (cmd *ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
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

	body := &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgent{
		Enable:                 cmd.Enable,
		Username:               cmd.Username,
		Password:               cmd.Password,
		TLS:                    cmd.TLS,
		TLSSkipVerify:          cmd.TLSSkipVerify,
		TLSCa:                  tlsCa,
		TLSCert:                tlsCert,
		TLSKey:                 tlsKey,
		MaxQueryLength:         cmd.MaxQueryLength,
		DisableQueryExamples:   cmd.DisableQueryExamples,
		DisableCommentsParsing: cmd.CommentsParsingChangeFlags.CommentsParsingDisabled(),
		LogLevel:               convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyQANPostgresqlPgstatmonitorAgentCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			QANPostgresqlPgstatmonitorAgent: body,
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
	if cmd.CommentsParsingChangeFlags.CommentsParsing != nil {
		if *cmd.CommentsParsingChangeFlags.CommentsParsingDisabled() {
			changes = append(changes, "disabled comments parsing")
		} else {
			changes = append(changes, "enabled comments parsing")
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

	return &changeAgentQANPostgreSQLPgStatMonitorAgentResult{
		Agent:   resp.Payload.QANPostgresqlPgstatmonitorAgent,
		Changes: changes,
	}, nil
}
