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
	"fmt"
	"strings"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

var addAgentExternalExporterResultT = commands.ParseTemplate(`
External Exporter added.
Agent ID              : {{ .Agent.AgentID }}
Runs on node ID       : {{ .Agent.RunsOnNodeID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
Scheme                : {{ .Agent.Scheme }}
Metrics path          : {{ .Agent.MetricsPath }}
Listen port           : {{ .Agent.ListenPort }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ .Agent.CustomLabels }}
`)

type addAgentExternalExporterResult struct {
	Agent *agents.AddExternalExporterOKBodyExternalExporter `json:"external_exporter"`
}

func (res *addAgentExternalExporterResult) Result() {}

func (res *addAgentExternalExporterResult) String() string {
	return commands.RenderTemplate(addAgentExternalExporterResultT, res)
}

type addAgentExternalExporterCommand struct {
	RunsOnNodeID string
	ServiceID    string
	Username     string
	Password     string
	CustomLabels string
	Scheme       string
	MetricsPath  string
	ListenPort   int64
	PushMetrics  bool
}

func (cmd *addAgentExternalExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	if cmd.MetricsPath != "" && !strings.HasPrefix(cmd.MetricsPath, "/") {
		cmd.MetricsPath = fmt.Sprintf("/%s", cmd.MetricsPath)
	}

	params := &agents.AddExternalExporterParams{
		Body: agents.AddExternalExporterBody{
			RunsOnNodeID: cmd.RunsOnNodeID,
			ServiceID:    cmd.ServiceID,
			Username:     cmd.Username,
			Password:     cmd.Password,
			Scheme:       cmd.Scheme,
			MetricsPath:  cmd.MetricsPath,
			ListenPort:   cmd.ListenPort,
			CustomLabels: customLabels,
			PushMetrics:  cmd.PushMetrics,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddExternalExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentExternalExporterResult{
		Agent: resp.Payload.ExternalExporter,
	}, nil
}

// register command
var (
	AddAgentExternalExporter  addAgentExternalExporterCommand
	AddAgentExternalExporterC = addAgentC.Command("external", "Add external exporter to inventory").Hide(hide)
)

func init() {
	AddAgentExternalExporterC.Flag("runs-on-node-id", "Node identifier where this instance runs").Required().StringVar(&AddAgentExternalExporter.RunsOnNodeID)
	AddAgentExternalExporterC.Flag("service-id", "Service identifier").Required().StringVar(&AddAgentExternalExporter.ServiceID)
	AddAgentExternalExporterC.Flag("username", "HTTP Basic auth username for scraping metrics").StringVar(&AddAgentExternalExporter.Username)
	AddAgentExternalExporterC.Flag("password", "HTTP Basic auth password for scraping metrics").StringVar(&AddAgentExternalExporter.Password)
	AddAgentExternalExporterC.Flag("scheme", "Scheme to generate URI to exporter metrics endpoints (http, https)").StringVar(&AddAgentExternalExporter.Scheme)
	AddAgentExternalExporterC.Flag("metrics-path", "Path under which metrics are exposed, used to generate URI").StringVar(&AddAgentExternalExporter.MetricsPath)
	AddAgentExternalExporterC.Flag("listen-port", "Listen port for scraping metrics").Required().Int64Var(&AddAgentExternalExporter.ListenPort)
	AddAgentExternalExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentExternalExporter.CustomLabels)
	AddAgentExternalExporterC.Flag("push-metrics", "Enables push metrics model flow,"+
		" it will be sent to the server by an agent").BoolVar(&AddAgentExternalExporter.PushMetrics)
}
