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
	Agent *agents.AddMongoDBExporterOKBodyMongodbExporter `json:"mongodb_exporter"`
}

func (res *addAgentMongodbExporterResult) Result() {}

func (res *addAgentMongodbExporterResult) String() string {
	return commands.RenderTemplate(addAgentMongodbExporterResultT, res)
}

func (cmd *MongoDBExporterCommand) RunCmd() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	tlsCertificateKey, err := commands.ReadFile(cmd.TLSCertificateKeyFile)
	if err != nil {
		return nil, err
	}
	tlsCa, err := commands.ReadFile(cmd.TLSCaFile)
	if err != nil {
		return nil, err
	}

	params := &agents.AddMongoDBExporterParams{
		Body: agents.AddMongoDBExporterBody{
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
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddMongoDBExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentMongodbExporterResult{
		Agent: resp.Payload.MongodbExporter,
	}, nil
}
