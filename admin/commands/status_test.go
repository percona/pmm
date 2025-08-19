// Copyright (C) 2023 Percona LLC
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
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/agentlocal"
)

func TestStatus(t *testing.T) {
	res := newStatusResult(&agentlocal.Status{
		AgentID:       "pmm-server",
		NodeID:        "pmm-server",
		NodeName:      "pmm-server-node",
		ServerURL:     "https://username:password@address/",
		ServerVersion: "2.4.0",
		AgentVersion:  "2.5.1",
		Agents: []agentlocal.AgentStatus{{
			AgentID:   "1afe233f-b319-4645-be6c-a1e05d4a545b",
			AgentType: "AGENT_TYPE_NODE_EXPORTER",
			Status:    "RUNNING",
			Port:      3310,
		}, {
			AgentID:   "2c7c0e04-6eef-411d-bcce-51e138e771cc",
			AgentType: "AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT",
			Status:    "RUNNING",
		}, {
			AgentID:   "ef134bec-a9ff-4e7f-802a-f387a75c0180",
			AgentType: "AGENT_TYPE_QAN_POSTGRESQL_PGSTATMONITOR_AGENT",
			Status:    "RUNNING",
		}, {
			AgentID:   "4824ac2b-3f1f-4e9b-90d1-3f56b891bb8b",
			AgentType: "AGENT_TYPE_POSTGRES_EXPORTER",
			Status:    "RUNNING",
			Port:      5432,
		}},
	})

	expected := strings.TrimSpace(`
Agent ID : pmm-server
Node ID  : pmm-server
Node name: pmm-server-node

PMM Server:
	URL    : https://address/
	Version: 2.4.0

PMM Client:
	Connected        : false
	Connection uptime: 0
	pmm-admin version: unknown
	pmm-agent version: 2.5.1
Agents:
	1afe233f-b319-4645-be6c-a1e05d4a545b node_exporter                  Running        3310
	2c7c0e04-6eef-411d-bcce-51e138e771cc postgresql_pgstatements_agent  Running        0
	ef134bec-a9ff-4e7f-802a-f387a75c0180 postgresql_pgstatmonitor_agent Running        0
	4824ac2b-3f1f-4e9b-90d1-3f56b891bb8b postgres_exporter              Running        5432
	`) + "\n"
	assert.Equal(t, expected, res.String())
}

func TestStatusJSON(t *testing.T) {
	res := newStatusResult(&agentlocal.Status{
		ServerURL: "https://username:password@address/",
	})
	b, err := json.MarshalIndent(res, "", "  ")
	require.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)

	// String, not JSON object with Scheme, Host, etc. Username and password are stripped.
	m = m["pmm_agent_status"].(map[string]interface{})
	assert.Equal(t, "https://address/", m["server_url"])
}
