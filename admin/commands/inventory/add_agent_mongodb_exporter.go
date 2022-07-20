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

type addAgentMongodbExporterCommand struct {
	PMMAgentID                    string
	ServiceID                     string
	Username                      string
	Password                      string
	AgentPassword                 string
	CustomLabels                  string
	SkipConnectionCheck           bool
	TLS                           bool
	TLSSkipVerify                 bool
	TLSCertificateKeyFile         string
	TLSCertificateKeyFilePassword string
	TLSCaFile                     string
	AuthenticationMechanism       string
	PushMetrics                   bool
	DisableCollectors             string

	StatsCollections string
	CollectionsLimit int32
}

func (cmd *addAgentMongodbExporterCommand) Run() (commands.Result, error) {
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
			LogLevel:                      &addExporterLogLevel,
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

// register command
var (
	AddAgentMongodbExporter  addAgentMongodbExporterCommand
	AddAgentMongodbExporterC = addAgentC.Command("mongodb-exporter", "Add mongodb_exporter to inventory").Hide(hide)
)

func init() {
	AddAgentMongodbExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentMongodbExporter.PMMAgentID)
	AddAgentMongodbExporterC.Arg("service-id", "Service identifier").Required().StringVar(&AddAgentMongodbExporter.ServiceID)
	AddAgentMongodbExporterC.Arg("username", "MongoDB username for scraping metrics").StringVar(&AddAgentMongodbExporter.Username)
	AddAgentMongodbExporterC.Flag("password", "MongoDB password for scraping metrics").StringVar(&AddAgentMongodbExporter.Password)
	AddAgentMongodbExporterC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&AddAgentMongodbExporter.AgentPassword)
	AddAgentMongodbExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentMongodbExporter.CustomLabels)
	AddAgentMongodbExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentMongodbExporter.SkipConnectionCheck)
	AddAgentMongodbExporterC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddAgentMongodbExporter.TLS)
	AddAgentMongodbExporterC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddAgentMongodbExporter.TLSSkipVerify)
	AddAgentMongodbExporterC.Flag("tls-certificate-key-file", "Path to TLS certificate PEM file").StringVar(&AddAgentMongodbExporter.TLSCertificateKeyFile)
	AddAgentMongodbExporterC.Flag("tls-certificate-key-file-password", "Password for certificate").StringVar(&AddAgentMongodbExporter.TLSCertificateKeyFilePassword)
	AddAgentMongodbExporterC.Flag("tls-ca-file", "Path to certificate authority file").StringVar(&AddAgentMongodbExporter.TLSCaFile)
	AddAgentMongodbExporterC.Flag("authentication-mechanism", "Authentication mechanism. Default is empty. Use MONGODB-X509 for ssl certificates").
		StringVar(&AddAgentMongodbExporter.AuthenticationMechanism)
	AddAgentMongodbExporterC.Flag("push-metrics", "Enables push metrics model flow,"+
		" it will be sent to the server by an agent").BoolVar(&AddAgentMongodbExporter.PushMetrics)

	AddAgentMongodbExporterC.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter").StringVar(
		&AddAgentMongodbExporter.DisableCollectors)
	AddAgentMongodbExporterC.Flag("stats-collections", "Collections for collstats & indexstats").StringVar(&AddAgentMongodbExporter.StatsCollections)
	AddAgentMongodbExporterC.Flag("max-collections-limit", "Disable collstats & indexstats if there are more than <n> collections").
		Int32Var(&AddAgentMongodbExporter.CollectionsLimit)
	addExporterGlobalFlags(AddAgentMongodbExporterC)
}
