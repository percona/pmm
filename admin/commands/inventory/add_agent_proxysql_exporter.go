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

var addAgentProxysqlExporterResultT = commands.ParseTemplate(`
Proxysql Exporter added.
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

type addAgentProxysqlExporterResult struct {
	Agent *agents.AddProxySQLExporterOKBodyProxysqlExporter `json:"proxysql_exporter"`
}

func (res *addAgentProxysqlExporterResult) Result() {}

func (res *addAgentProxysqlExporterResult) String() string {
	return commands.RenderTemplate(addAgentProxysqlExporterResultT, res)
}

type addAgentProxysqlExporterCommand struct {
	PMMAgentID          string
	ServiceID           string
	Username            string
	Password            string
	AgentPassword       string
	CustomLabels        string
	SkipConnectionCheck bool
	TLS                 bool
	TLSSkipVerify       bool
	PushMetrics         bool
	DisableCollectors   string
}

func (cmd *addAgentProxysqlExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}
	params := &agents.AddProxySQLExporterParams{
		Body: agents.AddProxySQLExporterBody{
			PMMAgentID:          cmd.PMMAgentID,
			ServiceID:           cmd.ServiceID,
			Username:            cmd.Username,
			Password:            cmd.Password,
			AgentPassword:       cmd.AgentPassword,
			CustomLabels:        customLabels,
			SkipConnectionCheck: cmd.SkipConnectionCheck,
			TLS:                 cmd.TLS,
			TLSSkipVerify:       cmd.TLSSkipVerify,
			PushMetrics:         cmd.PushMetrics,
			DisableCollectors:   commands.ParseDisableCollectors(cmd.DisableCollectors),
			LogLevel:            &addExporterLogLevel,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddProxySQLExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentProxysqlExporterResult{
		Agent: resp.Payload.ProxysqlExporter,
	}, nil
}

// register command
var (
	AddAgentProxysqlExporter  addAgentProxysqlExporterCommand
	AddAgentProxysqlExporterC = addAgentC.Command("proxysql-exporter", "Add proxysql_exporter to inventory").Hide(hide)
)

func init() {
	AddAgentProxysqlExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentProxysqlExporter.PMMAgentID)
	AddAgentProxysqlExporterC.Arg("service-id", "Service identifier").Required().StringVar(&AddAgentProxysqlExporter.ServiceID)
	AddAgentProxysqlExporterC.Arg("username", "ProxySQL username for scraping metrics").Default("admin").StringVar(&AddAgentProxysqlExporter.Username)
	AddAgentProxysqlExporterC.Flag("password", "ProxySQL password for scraping metrics").Default("admin").StringVar(&AddAgentProxysqlExporter.Password)
	AddAgentProxysqlExporterC.Flag("agent-password", "Custom password for /metrics endpoint").StringVar(&AddAgentProxysqlExporter.AgentPassword)
	AddAgentProxysqlExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentProxysqlExporter.CustomLabels)
	AddAgentProxysqlExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentProxysqlExporter.SkipConnectionCheck)
	AddAgentProxysqlExporterC.Flag("tls", "Use TLS to connect to the database").BoolVar(&AddAgentProxysqlExporter.TLS)
	AddAgentProxysqlExporterC.Flag("tls-skip-verify", "Skip TLS certificates validation").BoolVar(&AddAgentProxysqlExporter.TLSSkipVerify)
	AddAgentProxysqlExporterC.Flag("push-metrics", "Enables push metrics model flow,"+
		" it will be sent to the server by an agent").BoolVar(&AddAgentProxysqlExporter.PushMetrics)
	AddAgentProxysqlExporterC.Flag("disable-collectors",
		"Comma-separated list of collector names to exclude from exporter").StringVar(&AddAgentProxysqlExporter.DisableCollectors)
	addExporterGlobalFlags(AddAgentProxysqlExporterC)
}
