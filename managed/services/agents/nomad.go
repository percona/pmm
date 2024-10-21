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
	_ "embed"
	"strings"
	"text/template"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

//go:embed nomad_client.hcl
var nomadConfigTemplate string

func nomadClientConfig(node *models.Node, exporter *models.Agent, agentVersion *version.Parsed) (*agentv1.SetStateRequest_AgentProcess, error) {
	// TODO:
	// list tls certificates
	// command to start nomad client
	// generate configuration file for nomad client
	// nomad agent -client -config <tmp path>/nomad-client.hcl
	args := []string{
		"agent",
		"-client",
		"-config",
		"{{ .TextFiles.nomadConfigPlaceholder }}",
	}

	tdp := models.TemplateDelimsPair(
		append(args, pointer.GetString(exporter.MetricsPath))...,
	)

	config, err := generateNomadClientConfig(node, exporter, tdp)
	if err != nil {
		return nil, err
	}
	pathsToCerts := "/srv/nomad/certs"

	caCert := ""
	certFile := ""
	keyFile := ""
	params := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles: map[string]string{
			"nomadConfigPlaceholder": config,
			"caCert":                 caCert,
			"certFile":               certFile,
			"keyFile":                keyFile,
		},
	}

	return params, nil
}

func generateNomadClientConfig(node *models.Node, exporter *models.Agent, tdp models.DelimiterPair) (string, error) {
	logLevel := "info"
	if exporter.LogLevel != nil {
		logLevel = *exporter.LogLevel
	}

	nomadConfigParams := map[string]string{
		"NodeName":         node.NodeName,
		"NodeID":           node.NodeID,
		"PMMServerAddress": tdp.Left + "server_host" + tdp.Right + ":4647",
		"NodeAddress":      node.Address,
		"CaFile":           "{{ .TextFiles.caCert }}",
		"CertFile":         "{{ .TextFiles.certFile }}",
		"KeyFile":          "{{ .TextFiles.keyFile }}",
		"DataDir":          tdp.Left + "nomad_data_dir" + tdp.Right,
		"LogLevel":         strings.ToUpper(logLevel),
	}

	var configBuffer bytes.Buffer
	if tmpl, err := template.New("nomadConfig").Parse(nomadConfigTemplate); err != nil {
		return "", errors.Wrap(err, "Failed to parse nomad config template")
	} else if err = tmpl.Execute(&configBuffer, nomadConfigParams); err != nil {
		return "", errors.Wrap(err, "Failed to execute nomad config template")
	}
	return configBuffer.String(), nil
}
