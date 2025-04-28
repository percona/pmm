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

var addAgentQANMongoDBMongologAgentResultT = commands.ParseTemplate(`
QAN MongoDB Mongolog agent added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Status                : {{ .Agent.Status }}
Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentQANMongoDBMongologAgentResult struct {
	Agent *agents.AddAgentOKBodyQANMongodbMongologAgent `json:"qan_mongodb_mongolog_agent"`
}

func (res *addAgentQANMongoDBMongologAgentResult) Result() {}

func (res *addAgentQANMongoDBMongologAgentResult) String() string {
	return commands.RenderTemplate(addAgentQANMongoDBMongologAgentResultT, res)
}

// addAgentQANMongoDBMongologAgentCommand is used by Kong for CLI flags and commands.
//
//nolint:lll
type AddAgentQANMongoDBMongologAgentCommand struct {
	PMMAgentID                    string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID                     string            `arg:"" help:"Service identifier"`
	Username                      string            `arg:"" optional:"" help:"MongoDB username for scraping metrics"`
	Password                      string            `help:"MongoDB password for scraping metrics"`
	CustomLabels                  map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck           bool              `help:"Skip connection check"`
	MaxQueryLength                int32             `placeholder:"NUMBER" help:"Limit query length in QAN (default: server-defined; -1: no limit)"`
	DisableQueryExamples          bool              `name:"disable-queryexamples" help:"Disable collection of query examples"`
	TLS                           bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify                 bool              `help:"Skip TLS certificates validation"`
	TLSCertificateKeyFile         string            `help:"Path to TLS certificate PEM file"`
	TLSCertificateKeyFilePassword string            `help:"Password for certificate"`
	TLSCaFile                     string            `help:"Path to certificate authority file"`
	AuthenticationMechanism       string            `help:"Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates"`

	flags.LogLevelFatalFlags
}

// RunCmd executes the AddAgentQANMongoDBMongologAgentCommand and returns the result.
func (cmd *AddAgentQANMongoDBMongologAgentCommand) RunCmd() (commands.Result, error) {
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
			QANMongodbMongologAgent: &agents.AddAgentParamsBodyQANMongodbMongologAgent{
				PMMAgentID:                    cmd.PMMAgentID,
				ServiceID:                     cmd.ServiceID,
				Username:                      cmd.Username,
				Password:                      cmd.Password,
				CustomLabels:                  customLabels,
				SkipConnectionCheck:           cmd.SkipConnectionCheck,
				MaxQueryLength:                cmd.MaxQueryLength,
				TLS:                           cmd.TLS,
				TLSSkipVerify:                 cmd.TLSSkipVerify,
				TLSCertificateKey:             tlsCertificateKey,
				TLSCertificateKeyFilePassword: cmd.TLSCertificateKeyFilePassword,
				TLSCa:                         tlsCa,
				AuthenticationMechanism:       cmd.AuthenticationMechanism,
				LogLevel:                      cmd.LogLevelFatalFlags.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	return &addAgentQANMongoDBMongologAgentResult{
		Agent: resp.Payload.QANMongodbMongologAgent,
	}, nil
}
