// Copyright (C) 2023 Percona LLC
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

package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestGenerateNomadAgentConfig(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		node := &models.Node{
			NodeName: "node-name",
			NodeID:   "node-id",
			Address:  "node-address",
		}
		agent := &models.Agent{
			PMMAgentID: pointer.ToString("agent-id"),
			AgentType:  models.NomadAgentType,
			LogLevel:   pointer.To("debug"),
		}
		tdp := models.TemplateDelimsPair()
		config, err := generateNomadAgentConfig(node, agent, tdp)
		require.NoError(t, err)
		expected := `log_level = "DEBUG"

disable_update_check = true
data_dir = "{{ .nomad_data_dir }}" # it shall be persistent
region = "global"
datacenter = "PMM Deployment"
name = "PMM Agent node-name"

ui {
  enabled = false
}

ports {
  # Bind HTTP interface to this port
  http = "{{ .listen_port }}"
  # Bind RPC interface to this port
  rpc  = "4649"
}

addresses {
  http = "127.0.0.1"
  rpc = "127.0.0.1"
}

advertise {
  # 127.0.0.1 is not applicable here
  http = "node-address" # filled by PMM Server
  rpc = "node-address"  # filled by PMM Server
}

client {
  enabled = true
  cpu_total_compute = 1000

  servers = ["{{ .server_host }}:4647"] # filled by PMM Server

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec,storage,podman,containerd"
    "driver.allowlist" = "raw_exec"
  }

  # optional labels assigned to Nomad Client, can be the same as PMM Agent's.
  meta {
    pmm-agent = "1"
    agent_type = "nomad-agent"
    node_id = "node-id"
    node_name = "node-name"
  }
}

server {
  enabled = false
}

tls {
  http = true
  rpc  = true
  ca_file   = "{{ .TextFiles.caCert }}" # filled by PMM Agent
  cert_file = "{{ .TextFiles.certFile }}" # filled by PMM Agent
  key_file  = "{{ .TextFiles.keyFile }}" # filled by PMM Agent

  verify_server_hostname = true
}

# Enabled plugins
plugin "raw_exec" {
  config {
      enabled = true
  }
}
`
		assert.Equal(t, expected, config)
	})
}
