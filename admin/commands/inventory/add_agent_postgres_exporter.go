// pmm-admin
// Copyright 2019 Percona LLC
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
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var addAgentPostgresExporterResultT = commands.ParseTemplate(`
Postgres Exporter added.
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

type addAgentPostgresExporterResult struct {
	Agent *agents.AddPostgresExporterOKBodyPostgresExporter `json:"postgres_exporter"`
}

func (res *addAgentPostgresExporterResult) Result() {}

func (res *addAgentPostgresExporterResult) String() string {
	return commands.RenderTemplate(addAgentPostgresExporterResultT, res)
}

func (cmd *PostgresExporterCmd) RunCmd() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	var tlsCa, tlsCert, tlsKey string
	if cmd.TLS {
		tlsCa, err = commands.ReadFile(cmd.TLSCAFile)
		if err != nil {
			return nil, err
		}

		tlsCert, err = commands.ReadFile(cmd.TLSCertFile)
		if err != nil {
			return nil, err
		}

		tlsKey, err = commands.ReadFile(cmd.TLSKeyFile)
		if err != nil {
			return nil, err
		}
	}

	params := &agents.AddPostgresExporterParams{
		Body: agents.AddPostgresExporterBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			AgentPassword:       cmd.AgentPassword,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			PushMetrics:         cmd.PushMetrics,
			DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),

			TLS:           cmd.TLS,
			TLSSkipVerify: cmd.TLSSkipVerify,
			TLSCa:         tlsCa,
			TLSCert:       tlsCert,
			TLSKey:        tlsKey,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddPostgresExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentPostgresExporterResult{
		Agent: resp.Payload.PostgresExporter,
	}, nil
}
