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
	"bytes"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

var nomadConfigTemplate = `log_level = "{{ .LogLevel }}"

disable_update_check = true
data_dir = "{{ .DataDir }}" # it shall be persistent
region = "global"
datacenter = "PMM Deployment"
name = "PMM Agent {{ .NodeName }}"

ui {
  enabled = false
}

addresses {
  http = "127.0.0.1"
  rpc = "127.0.0.1"
}

advertise {
  # 127.0.0.1 is not applicable here
  http = "{{ .NodeAddress }}" # filled by PMM Server
  rpc = "{{ .NodeAddress }}"  # filled by PMM Server
}

client {
  enabled = true
  cpu_total_compute = 1000

  servers = ["{{ .PMMServerAddress }}"] # filled by PMM Server

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec,storage,podman,containerd"
    "driver.allowlist" = "raw_exec"
  }

  # optional labels assigned to Nomad Client, can be the same as PMM Agent's.
  meta {
    pmm-agent = "1"
  {{- range $key, $value := .Labels }}
    {{ $key }} = "{{ $value }}"
  {{- end }}
  }
}

server {
  enabled = false
}

tls {
  http = true
  rpc  = true
  ca_file   = "{{ .CaFile }}" # filled by PMM Agent
  cert_file = "{{ .CertFile }}" # filled by PMM Agent
  key_file  = "{{ .KeyFile }}" # filled by PMM Agent

  verify_server_hostname = true
}

# Enabled plugins
plugin "raw_exec" {
  config {
      enabled = true
  }
}
`

func nomadClientConfig(n nomad, node *models.Node, exporter *models.Agent) (*agentv1.SetStateRequest_AgentProcess, error) {
	args := []string{
		"agent",
		"-client",
		"-config",
		"{{ .TextFiles.nomadConfig }}",
	}

	tdp := models.TemplateDelimsPair()

	config, err := generateNomadClientConfig(node, exporter, tdp)
	if err != nil {
		return nil, err
	}

	caCert, err := n.GetCACert()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read CA certificate")
	}
	certFile, err := n.GetClientCert()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read client certificate")
	}
	keyFile, err := n.GetClientKey()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read client key")
	}
	params := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_NOMAD_AGENT,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles: map[string]string{
			"nomadConfig": config,
			"caCert":      caCert,
			"certFile":    certFile,
			"keyFile":     keyFile,
		},
	}

	return params, nil
}

func generateNomadClientConfig(node *models.Node, exporter *models.Agent, tdp models.DelimiterPair) (string, error) {
	logLevel := "info"
	if exporter.LogLevel != nil {
		logLevel = *exporter.LogLevel
	}
	labels, err := models.MergeLabels(node, nil, exporter)
	if err != nil {
		return "", errors.Wrap(err, "Failed to get unified labels")
	}

	nomadConfigParams := map[string]interface{}{
		"NodeName":         node.NodeName,
		"NodeID":           node.NodeID,
		"Labels":           labels,
		"PMMServerAddress": tdp.Left + " .server_host " + tdp.Right + ":4647",
		"NodeAddress":      node.Address,
		"CaFile":           tdp.Left + " .TextFiles.caCert " + tdp.Right,
		"CertFile":         tdp.Left + " .TextFiles.certFile " + tdp.Right,
		"KeyFile":          tdp.Left + " .TextFiles.keyFile " + tdp.Right,
		"DataDir":          tdp.Left + " .nomad_data_dir " + tdp.Right,
		"LogLevel":         strings.ToUpper(logLevel),
	}

	var configBuffer bytes.Buffer
	tmpl, err := template.New("nomadConfig").Parse(nomadConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse nomad config template")
	}

	err = tmpl.Execute(&configBuffer, nomadConfigParams)
	if err != nil {
		return "", errors.Wrap(err, "Failed to execute nomad config template")
	}
	return configBuffer.String(), nil
}
