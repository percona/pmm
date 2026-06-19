// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"time"

	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

var addAgentRTAMySQLAgentResultT = commands.ParseTemplate(`
Real-Time Analytics MySQL agent added.
Agent ID              : {{ .Agent.AgentID }}
PMM-Agent ID          : {{ .Agent.PMMAgentID }}
Service ID            : {{ .Agent.ServiceID }}
Username              : {{ .Agent.Username }}
TLS enabled           : {{ .Agent.TLS }}
Skip TLS verification : {{ .Agent.TLSSkipVerify }}

Disabled              : {{ .Agent.Disabled }}
Custom labels         : {{ formatCustomLabels .Agent.CustomLabels }}
Collect interval      : {{ .Agent.RtaOptions.CollectInterval }}
Log level             : {{ formatLogLevel .Agent.LogLevel }}
`)

type addAgentRTAMySQLAgentResult struct {
	Agent *agents.AddAgentOKBodyRtaMysqlAgent `json:"rta_mysql_agent"`
}

func (res *addAgentRTAMySQLAgentResult) Result() {}

func (res *addAgentRTAMySQLAgentResult) String() string {
	return commands.RenderTemplate(addAgentRTAMySQLAgentResultT, res)
}

// AddAgentRTAMySQLAgentCommand is used by Kong for CLI flags and commands.
type AddAgentRTAMySQLAgentCommand struct {
	PMMAgentID          string            `arg:"" help:"The pmm-agent identifier which runs this instance"`
	ServiceID           string            `arg:"" help:"Service identifier"`
	Username            string            `arg:"" optional:"" help:"MySQL username for getting queries data"`
	Password            string            `help:"MySQL password for getting queries data"`
	CustomLabels        map[string]string `mapsep:"," help:"Custom user-assigned labels"`
	SkipConnectionCheck bool              `help:"Skip connection check"`
	TLS                 bool              `help:"Use TLS to connect to the database"`
	TLSSkipVerify       bool              `help:"Skip TLS certificate verification"`
	TLSCaFile           string            `help:"Path to certificate authority file"`
	TLSCertFile         string            `help:"Path to client certificate file"`
	TLSKeyFile          string            `help:"Path to client key file"`
	CollectInterval     *time.Duration    `placeholder:"DURATION" help:"Query collect interval (default: server-defined 2s)"`

	flags.LogLevelFatalFlags
}

// RunCmd executes the AddAgentRTAMySQLAgentCommand and returns the result.
func (cmd *AddAgentRTAMySQLAgentCommand) RunCmd() (commands.Result, error) {
	customLabels := commands.ParseKeyValuePair(&cmd.CustomLabels)

	tlsCa, err := commands.ReadFile(cmd.TLSCaFile)
	if err != nil {
		return nil, err
	}

	tlsCert, err := commands.ReadFile(cmd.TLSCertFile)
	if err != nil {
		return nil, err
	}

	tlsKey, err := commands.ReadFile(cmd.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			RtaMysqlAgent: &agents.AddAgentParamsBodyRtaMysqlAgent{
				PMMAgentID:          cmd.PMMAgentID,
				ServiceID:           cmd.ServiceID,
				Username:            cmd.Username,
				Password:            cmd.Password,
				CustomLabels:        *customLabels,
				SkipConnectionCheck: cmd.SkipConnectionCheck,
				TLS:                 cmd.TLS,
				TLSSkipVerify:       cmd.TLSSkipVerify,
				TLSCa:               tlsCa,
				TLSCert:             tlsCert,
				TLSKey:              tlsKey,
				LogLevel:            cmd.LogLevel.EnumValue(),
			},
		},
		Context: commands.Ctx,
	}

	if cmd.CollectInterval != nil {
		params.Body.RtaMysqlAgent.RtaOptions = &agents.AddAgentParamsBodyRtaMysqlAgentRtaOptions{
			CollectInterval: cmd.CollectInterval.String(),
		}
	}

	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}

	return &addAgentRTAMySQLAgentResult{
		Agent: resp.Payload.RtaMysqlAgent,
	}, nil
}
