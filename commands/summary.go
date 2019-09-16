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

package commands

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/agentlocal"
)

var summaryResultT = ParseTemplate(`
Agent ID: {{ .Status.AgentID }}
Node ID : {{ .Status.NodeID }}

PMM Server:
	URL    : {{ .Status.ServerURL }}
	Version: {{ .Status.ServerVersion }}

PMM-agent:
	Connected : {{ .Status.Connected }}{{ if .Status.Connected }}
	Time drift: {{ .Status.ServerClockDrift }}
	Latency   : {{ .Status.ServerLatency }}
{{ end }}
Agents:
{{ range .Status.Agents }}	{{ .AgentID }} {{ .AgentType }} {{ .Status }}
{{ end }}
`)

type summaryResult struct {
	PMMAgentStatus *agentlocal.Status `json:"pmm_agent_status"`
}

func (res *summaryResult) Result() {}

func (res *summaryResult) String() string {
	return RenderTemplate(summaryResultT, res)
}

type summaryCommand struct {
}

func (cmd *summaryCommand) Run() (Result, error) {
	status, err := agentlocal.GetStatus(agentlocal.RequestNetworkInfo)
	if err != nil {
		return nil, err
	}

	return &summaryResult{
		PMMAgentStatus: status,
	}, nil
}

// register command
var (
	Summary  = new(summaryCommand)
	SummaryC = kingpin.Command("summary", "")
	StatusC  = kingpin.Command("status", "").Hidden() // TODO remove it https://jira.percona.com/browse/PMM-4704
)

func init() {
	// TODO add flag to skip .tar.gz/zip generation
}
