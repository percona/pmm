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
		ServerURL:     "https://username:password@address/",
		ServerVersion: "2.4.0",
		AgentVersion:  "2.5.1",
		Agents: []agentlocal.AgentStatus{{
			AgentID:   "/agent_id/1afe233f-b319-4645-be6c-a1e05d4a545b",
			AgentType: "NODE_EXPORTER",
			Status:    "RUNNING",
			Port:      3310,
		}, {
			AgentID:   "/agent_id/2c7c0e04-6eef-411d-bcce-51e138e771cc",
			AgentType: "QAN_POSTGRESQL_PGSTATEMENTS_AGENT",
			Status:    "RUNNING",
		}, {
			AgentID:   "/agent_id/4824ac2b-3f1f-4e9b-90d1-3f56b891bb8b",
			AgentType: "POSTGRES_EXPORTER",
			Status:    "RUNNING",
			Port:      5432,
		}},
	})

	expected := strings.TrimSpace(`
Agent ID: pmm-server
Node ID : pmm-server

PMM Server:
	URL    : https://address/
	Version: 2.4.0

PMM Client:
	Connected        : false
	pmm-admin version: unknown
	pmm-agent version: 2.5.1
Agents:
	/agent_id/1afe233f-b319-4645-be6c-a1e05d4a545b node_exporter Running 3310
	/agent_id/2c7c0e04-6eef-411d-bcce-51e138e771cc postgresql_pgstatements_agent Running 0
	/agent_id/4824ac2b-3f1f-4e9b-90d1-3f56b891bb8b postgres_exporter Running 5432
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
