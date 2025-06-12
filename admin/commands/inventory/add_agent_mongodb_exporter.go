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
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addAgentMongodbExporterResultT = commands.ParseTemplate(`
MongoDB Exporter added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Listen port           : {{ .Agent.ListenPort }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentMongodbExporterResult struct {
	Agent *agents.AddAgentOKBodyMongodbExporter `json:"mongodb_exporter"`
}

func (res *addAgentMongodbExporterResult) Result() {}

func (res *addAgentMongodbExporterResult) String() string {
	return commands.RenderTemplate(addAgentMongodbExporterResultT, res)
}

// AddAgentMongodbExporterCommand is used by Kong for CLI flags and commands.
type AddAgentMongodbExporterCommand struct {
	PMMAgentID                    string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                     string            `arg:"" help:"Service identifier"`
	Username                      string            `arg:"" optional:"" help:"MongoDB username for scraping metrics"`
	Password                      string            `help:"MongoDB password for scraping metrics"`
	AgentPassword                 string            `help:"Custom password for /metrics endpoint"`
	CustomLabels                  map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool              `help:"Skip connection check"`
	TLS                           bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool              `help:"Skip TLS certificate verification"`
	TLSCertificateKeyFile         string            `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string            `help:"Password for certificate"`
	TLSCaFile                     string            `help:"Path to certificate authority file"`
	AuthenticationMechanism       string            `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`
	PushMetrics                   bool              `help:"Enables push metrics model flow, it will be sent to the server by an agent"`
	DisableCollectors             []string          `help:"Comma-separated list of collector names to exclude from exporter"`
	StatsCollections              []string          `help:"Collections for collstats & indexstats"`
	CollectionsLimit              int32             `name:"max-collections-limit" placeholder:"number" help:"Disable collstats & indexstats if there are more than <n> collections"` //nolint:lll

	flags.LogLevelFatalFlags
}

// RunCmd executes the AddAgentMongodbExporterCommand and returns the result.
func (cmd *AddAgentMongodbExporterCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseCustomLabels(cmd.CustomLabels)

	tlsCertificateKey, err := commands.ReadFile(cmd.TLSCertificateKeyFile)
	if err != nil {
		return nil, err
	}
	tlsCa, err := commands.ReadFile(cmd.TLSCaFile)
	if err != nil {
		return nil, err
	}

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			MongodbExporter: &agents.AddAgentParamsBodyMongodbExporter{
				PMMAgentID:                    cmd.PMMAgentID,
				ServiceID:                     cmd.ServiceID,
				Username:                      cmd.Username,
				Password:                      cmd.Password,
				AgentPassword:                 cmd.AgentPassword,
				CustomLabels:                  customLabels,
				SkipConnectionCheck:           cmd.SkipConnectionCheck,
				TLS:                           cmd.TLS,
				TLSSkipVerify:                 cmd.TLSSkipVerify,
				TLSCertificateKey:             tlsCertificateKey,
				TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
				TLSCa:                         tlsCa,
				AuthenticationMechanism:       cmd.AuthenticationMechanism,
				PushMetrics:                   cmd.PushMetrics,
				DisableCollectors:             commands.ParseDisableCollectors(cmd.DisableCollectors),
				StatsCollections:              commands.ParseDisableCollectors(cmd.StatsCollections),
				CollectionsLimit:              cmd.CollectionsLimit,
				LogLevel:                      cmd.LogLevelFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentMongodbExporterResult{
		Agent: resp.Payload.MongodbExporter,
	}, nil
}
