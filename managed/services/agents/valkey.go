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
	"sort"
	"time"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// valkeyExporterConfig returns the desired configuration of the valkey_exporter process.
func valkeyExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) *agentv1.SetStateRequest_AgentProcess {
	listenAddress := getExporterListenAddress(node, exporter)
	tdp := exporter.TemplateDelimiters(service)
	args := []string{
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
	}

	if exporter.ExporterOptions.MetricsPath != "" {
		args = append(args, "--web.telemetry-path="+exporter.ExporterOptions.MetricsPath)
	}

	dnsParams := models.DSNParams{
		DialTimeout: 3 * time.Second,
	}
	args = append(args, "--redis.addr="+exporter.DSN(service, dnsParams, nil, pmmAgentVersion))
	sort.Strings(args)

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles:          exporter.Files(),
	}
	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res
}
