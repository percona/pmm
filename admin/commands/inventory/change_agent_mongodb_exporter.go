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
	"strings"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var changeAgentMongodbExporterResultT = commands.ParseTemplate(`
MongoDB Exporter agent configuration updated.
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

type changeAgentMongodbExporterResult struct {
	Agent   *agents.ChangeAgentOKBodyMongodbExporter `json:"mongodb_exporter"`
	Changes []string                                 `json:"changes,omitempty"`
}

func (res *changeAgentMongodbExporterResult) Result() {}

func (res *changeAgentMongodbExporterResult) String() string {
	return commands.RenderTemplate(changeAgentMongodbExporterResultT, res)
}

// ChangeAgentMongodbExporterCommand is used by Kong for CLI flags and commands.
type ChangeAgentMongodbExporterCommand struct {
	AgentID string `arg:"" help:"MongoDB Exporter Agent ID"`

	// NOTE: Only provided flags will be changed, others will remain unchanged

	// Basic options
	Enable        *bool   `help:"Enable or disable the agent"`
	Username      *string `help:"Username for MongoDB connection"`
	Password      *string `help:"Password for MongoDB connection"`
	AgentPassword *string `help:"Custom password for agent /metrics endpoint"`

	// TLS options
	TLS                           *bool   `help:"Use TLS for database connections"`
	TLSSkipVerify                 *bool   `help:"Skip TLS certificate and hostname validation"`
	TLSCertificateKeyFile         *string `help:"TLS certificate key file"`
	TLSCertificateKeyFilePassword *string `help:"TLS certificate key file password"`
	TLSCaFile                     *string `help:"TLS CA certificate file"`

	// MongoDB specific options
	AuthenticationMechanism *string `help:"Authentication mechanism for MongoDB"`
	AuthenticationDatabase  *string `help:"Authentication database for MongoDB"`
	StatsCollections        *string `help:"List of comma-separated collection names to collect"`
	CollectionsLimit        *int32  `help:"Collections limit"`

	// Exporter options
	DisableCollectors []string `help:"List of collector names to disable"`
	ExposeExporter    *bool    `help:"Expose the exporter process on all public interfaces"`
	PushMetrics       *bool    `help:"Enable push metrics with vmagent"`

	// Custom labels
	CustomLabels *map[string]string `mapsep:"," help:"Custom user-assigned labels"`

	// Log level
	flags.LogLevelFatalChangeFlags
}

// RunCmd executes the ChangeAgentMongodbExporterCommand and returns the result.
func (cmd *ChangeAgentMongodbExporterCommand) RunCmd() (commands.Result, error) {
	var changes []string

	// Parse custom labels if provided
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	// Parse stats collections if provided
	var statsCollections []string
	if cmd.StatsCollections != nil && *cmd.StatsCollections != "" {
		statsCollections = strings.Split(*cmd.StatsCollections, ",")
		for i, collection := range statsCollections {
			statsCollections[i] = strings.TrimSpace(collection)
		}
	}

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

	body := &agents.ChangeAgentParamsBodyMongodbExporter{
		Enable:                        cmd.Enable,
		Username:                      cmd.Username,
		Password:                      cmd.Password,
		AgentPassword:                 cmd.AgentPassword,
		TLS:                           cmd.TLS,
		TLSSkipVerify:                 cmd.TLSSkipVerify,
		TLSCertificateKey:             tlsCertificateKey,
		TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
		TLSCa:                         tlsCa,
		AuthenticationMechanism:       cmd.AuthenticationMechanism,
		AuthenticationDatabase:        cmd.AuthenticationDatabase,
		StatsCollections:              statsCollections,
		CollectionsLimit:              cmd.CollectionsLimit,
		DisableCollectors:             cmd.DisableCollectors,
		ExposeExporter:                cmd.ExposeExporter,
		EnablePushMetrics:             cmd.PushMetrics,
		LogLevel:                      convertLogLevelPtr(cmd.LogLevel),
	}

	if customLabels != nil {
		body.CustomLabels = &agents.ChangeAgentParamsBodyMongodbExporterCustomLabels{
			Values: *customLabels,
		}
	}

	params := &agents.ChangeAgentParams{
		AgentID: cmd.AgentID,
		Body: agents.ChangeAgentBody{
			MongodbExporter: body,
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
	if cmd.StatsCollections != nil {
		changes = append(changes, fmt.Sprintf("updated stats collections: %s", *cmd.StatsCollections))
	}
	if cmd.CollectionsLimit != nil {
		changes = append(changes, fmt.Sprintf("changed collections limit to %d", *cmd.CollectionsLimit))
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
		if len(*customLabels) > 0 {
			changes = append(changes, "updated custom labels")
		} else {
			changes = append(changes, "custom labels are removed")
		}
	}

	return &changeAgentMongodbExporterResult{
		Agent:   resp.Payload.MongodbExporter,
		Changes: changes,
	}, nil
}
