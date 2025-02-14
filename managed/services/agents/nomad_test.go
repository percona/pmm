package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestGenerateNomadClientConfig(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		node := &models.Node{
			NodeName: "node-name",
			NodeID:   "node-id",
			Address:  "node-address",
		}
		agent := &models.Agent{
			PMMAgentID: pointer.ToString("agent-id"),
			AgentType:  models.NomadClientType,
			LogLevel:   pointer.To("debug"),
		}
		tdp := models.TemplateDelimsPair()
		config, err := generateNomadClientConfig(node, agent, tdp)
		require.NoError(t, err)
		expected := `log_level = "DEBUG"

disable_update_check = true
data_dir = "{{nomad_data_dir}}" # it shall be persistent
region = "global"
datacenter = "PMM Deployment"
name = "PMM Agent node-name"

ui {
  enabled = false
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

  servers = ["{{server_host}}:4647"] # filled by PMM Server

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec"
    "driver.allowlist" = "raw_exec"
  }

  # optional lables set to Nomad Client, may be the same as for PMM Agent.
  meta {
    pmm-agent = "1"
    agent_type = "nomad-client"
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
