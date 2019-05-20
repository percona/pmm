// pmm-admin
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package commands

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/agentlocal"
)

var statusResultT = ParseTemplate(`
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

type statusResult struct {
	Status *agentlocal.Status `json:"status"`
}

func (res *statusResult) Result() {}

func (res *statusResult) String() string {
	return RenderTemplate(statusResultT, res)
}

type statusCommand struct {
}

func (cmd *statusCommand) Run() (Result, error) {
	// Unlike list, this command uses only local pmm-agent status.
	// It does not use PMM Server APIs.

	status, err := agentlocal.GetStatus(agentlocal.RequestNetworkInfo)
	if err != nil {
		return nil, err
	}

	return &statusResult{
		Status: status,
	}, nil
}

// register command
var (
	Status  = new(statusCommand)
	StatusC = kingpin.Command("status", "Show information about local pmm-agent.")
)
