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

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// clickhouseExporterConfig returns the desired configuration of the clickhouse_exporter process.
func clickhouseExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
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
		DialTimeout: exporter.EffectiveDialTimeout(),
	}

	args = append(args, "--clickhouse.dsn="+exporter.DSN(service, dnsParams, nil, pmmAgentVersion))
	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion, false)
	sort.Strings(args)

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_CLICKHOUSE_EXPORTER,
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

// qanClickHouseQueryLogAgentConfig returns desired configuration of qan-clickhouse-querylog-agent built-in agent.
func qanClickHouseQueryLogAgentConfig(service *models.Service, agent *models.Agent, pmmAgentVersion *version.Parsed) *agentv1.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	dnsParams := models.DSNParams{
		DialTimeout: agent.EffectiveDialTimeout(),
		Database:    service.DatabaseName,
	}

	return &agentv1.SetStateRequest_BuiltinAgent{
		Type:                   inventoryv1.AgentType_AGENT_TYPE_QAN_CLICKHOUSE_QUERYLOG_AGENT,
		Dsn:                    agent.DSN(service, dnsParams, nil, pmmAgentVersion),
		MaxQueryLength:         agent.QANOptions.MaxQueryLength,
		DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
		TextFiles: &agentv1.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}
