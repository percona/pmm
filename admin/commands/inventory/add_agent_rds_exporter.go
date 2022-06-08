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

var addAgentRDSExporterResultT = commands.ParseTemplate(`
RDS Exporter added.
Agent ID                  : {{ .Agent.AgentID }}
PMM-Agent ID              : {{ .Agent.PMMAgentID }}
Node ID                   : {{ .Agent.NodeID }}
Listen port               : {{ .Agent.ListenPort }}

Status                    : {{ .Agent.Status }}
Disabled                  : {{ .Agent.Disabled }}
Basic metrics disabled    : {{ .Agent.BasicMetricsDisabled }}
Enhanced metrics disabled : {{ .Agent.EnhancedMetricsDisabled }}
Custom labels             : {{ .Agent.CustomLabels }}
`)

type addAgentRDSExporterResult struct {
	Agent *agents.AddRDSExporterOKBodyRDSExporter `json:"rds_exporter"`
}

func (res *addAgentRDSExporterResult) Result() {}

func (res *addAgentRDSExporterResult) String() string {
	return commands.RenderTemplate(addAgentRDSExporterResultT, res)
}

type addAgentRDSExporterCommand struct {
	PMMAgentID             string
	NodeID                 string
	AWSAccessKey           string
	AWSSecretKey           string
	CustomLabels           string
	SkipConnectionCheck    bool
	DisableBasicMetrics    bool
	DisableEnhancedMetrics bool
	PushMetrics            bool
}

func (cmd *addAgentRDSExporterCommand) Run() (commands.Result, error) {
	customLabels, err := commands.ParseCustomLabels(cmd.CustomLabels)
	if err != nil {
		return nil, err
	}

	params := &agents.AddRDSExporterParams{
		Body: agents.AddRDSExporterBody{
			PMMAgentID:             cmd.PMMAgentID,
			NodeID:                 cmd.NodeID,
			AWSAccessKey:           cmd.AWSAccessKey,
			AWSSecretKey:           cmd.AWSSecretKey,
			CustomLabels:           customLabels,
			SkipConnectionCheck:    cmd.SkipConnectionCheck,
			DisableBasicMetrics:    cmd.DisableBasicMetrics,
			DisableEnhancedMetrics: cmd.DisableEnhancedMetrics,
			PushMetrics:            cmd.PushMetrics,
		},
		Context: commands.Ctx,
	}

	resp, err := client.Default.Agents.AddRDSExporter(params)
	if err != nil {
		return nil, err
	}
	return &addAgentRDSExporterResult{
		Agent: resp.Payload.RDSExporter,
	}, nil
}

// register command
var (
	AddAgentRDSExporter  addAgentRDSExporterCommand
	AddAgentRDSExporterC = addAgentC.Command("rds-exporter", "Add rds_exporter to inventory").Hide(hide)
)

func init() {
	AddAgentRDSExporterC.Arg("pmm-agent-id", "The pmm-agent identifier which runs this instance").Required().StringVar(&AddAgentRDSExporter.PMMAgentID)
	AddAgentRDSExporterC.Arg("node-id", "Node identifier").Required().StringVar(&AddAgentRDSExporter.NodeID)
	AddAgentRDSExporterC.Flag("aws-access-key", "AWS Access Key ID").StringVar(&AddAgentRDSExporter.AWSAccessKey)
	AddAgentRDSExporterC.Flag("aws-secret-key", "AWS Secret Access Key").StringVar(&AddAgentRDSExporter.AWSSecretKey)
	AddAgentRDSExporterC.Flag("custom-labels", "Custom user-assigned labels").StringVar(&AddAgentRDSExporter.CustomLabels)
	AddAgentRDSExporterC.Flag("skip-connection-check", "Skip connection check").BoolVar(&AddAgentRDSExporter.SkipConnectionCheck)
	AddAgentRDSExporterC.Flag("disable-basic-metrics", "Disable basic metrics").BoolVar(&AddAgentRDSExporter.DisableBasicMetrics)
	AddAgentRDSExporterC.Flag("disable-enhanced-metrics", "Disable enhanced metrics").BoolVar(&AddAgentRDSExporter.DisableEnhancedMetrics)
	AddAgentRDSExporterC.Flag("push-metrics", "Enables push metrics model flow,"+
		" it will be sent to the server by an agent").BoolVar(&AddAgentRDSExporter.PushMetrics)
}
