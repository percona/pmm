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
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/api/inventorypb/types"
	"github.com/percona/pmm/version"
)

var statusResultT = ParseTemplate(`
Agent ID: {{ .PMMAgentStatus.AgentID }}
Node ID : {{ .PMMAgentStatus.NodeID }}

PMM Server:
	URL    : {{ .PMMAgentStatus.ServerURL }}
	Version: {{ .PMMAgentStatus.ServerVersion }}

PMM Client:
	Connected        : {{ .PMMAgentStatus.Connected }}{{ if .PMMAgentStatus.Connected }}
	Time drift       : {{ .PMMAgentStatus.ServerClockDrift }}
	Latency          : {{ .PMMAgentStatus.ServerLatency }}{{ end }}
	pmm-admin version: {{ .PMMVersion }}
	pmm-agent version: {{ .PMMAgentStatus.AgentVersion }}
Agents:
{{ range .PMMAgentStatus.Agents }}	{{ .AgentID }} {{ .AgentType | $.HumanReadableAgentType }} {{ .Status | $.NiceAgentStatus }} {{ .Port }}
{{ end }}
`)

type statusResult struct {
	PMMAgentStatus *agentlocal.Status `json:"pmm_agent_status"`
	PMMVersion     string             `json:"pmm_admin_version"`
}

func (res *statusResult) HumanReadableAgentType(agentType string) string {
	return types.AgentTypeName(agentType)
}

func (res *statusResult) NiceAgentStatus(status string) string {
	return strings.Title(strings.ToLower(status))
}

func (res *statusResult) Result() {}

func (res *statusResult) String() string {
	return RenderTemplate(statusResultT, res)
}

func newStatusResult(status *agentlocal.Status) *statusResult {
	// hide username and password from PMM Server URL - if we have it at all
	if u, err := url.Parse(status.ServerURL); err == nil {
		u.User = nil
		status.ServerURL = u.String()
	}

	pmmVersion := version.PMMVersion
	if pmmVersion == "" {
		pmmVersion = "unknown"
	}

	return &statusResult{
		PMMAgentStatus: status,
		PMMVersion:     pmmVersion,
	}
}

type statusCommand struct {
	timeout time.Duration
}

func (cmd *statusCommand) Run() (Result, error) {
	// Unlike list, this command uses only local pmm-agent status.
	// It does not use PMM Server APIs.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	var status *agentlocal.Status

	var err error

	for {
		status, err = agentlocal.GetStatus(agentlocal.RequestNetworkInfo)
		if err == nil {
			break
		}

		select {
		case <-timeoutCtx.Done():
			if err == agentlocal.ErrNotSetUp { //nolint:errorlint,goerr113
				return nil, errors.Errorf("Failed to get PMM Agent status from local pmm-agent: %s.\n"+
					"Please run `pmm-admin config` with --server-url flag.", err)
			}

			return nil, errors.Errorf("Failed to get PMM Agent status from local pmm-agent: %s.", err) //nolint:golint
		default:
			time.Sleep(1 * time.Second)
		}
	}

	return newStatusResult(status), nil
}

// register command
var (
	Status  statusCommand
	StatusC = kingpin.Command("status", "Show information about local pmm-agent")
)

func init() {
	StatusC.Flag("wait", "Time to wait for a successful response from pmm-agent").DurationVar(&Status.timeout)
}
