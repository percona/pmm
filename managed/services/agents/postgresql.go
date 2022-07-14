// Copyright (C) 2017 Percona LLC
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
	"fmt"
	"sort"
	"time"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

var postgresExporterAutodiscoveryVersion = version.MustParse("2.15.99")

// postgresExporterConfig returns desired configuration of postgres_exporter process.
func postgresExporterConfig(service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) *agentpb.SetStateRequest_AgentProcess {
	if service.DatabaseName == "" {
		panic("database name not set")
	}

	tdp := exporter.TemplateDelimiters(service)

	args := []string{
		// LR
		"--collect.custom_query.lr",

		// MR
		"--collect.custom_query.mr",

		// HR
		"--collect.custom_query.hr",

		"--collect.custom_query.lr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/postgresql/low-resolution",
		"--collect.custom_query.mr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/postgresql/medium-resolution",
		"--collect.custom_query.hr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/postgresql/high-resolution",
		"--web.listen-address=:" + tdp.Left + " .listen_port " + tdp.Right,
	}

	if !pmmAgentVersion.Less(postgresExporterAutodiscoveryVersion) {
		args = append(args,
			"--auto-discover-databases",
			"--exclude-databases=template0,template1,postgres,cloudsqladmin,pmm-managed-dev,azure_maintenance")
	}

	if pointer.GetString(exporter.MetricsPath) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsPath)
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.DisabledCollectors)

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion)

	sort.Strings(args)

	timeout := 1 * time.Second
	if exporter.AzureOptions != nil {
		timeout = 5 * time.Second
	}

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, timeout, service.DatabaseName, nil)),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.GetAgentPassword()),
		},
		TextFiles: exporter.Files(),
	}
	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res
}

// qanPostgreSQLPgStatementsAgentConfig returns desired configuration of qan-mongodb-profiler-agent built-in agent.
func qanPostgreSQLPgStatementsAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type: inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
		Dsn:  agent.DSN(service, 5*time.Second, service.DatabaseName, nil),
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}

// qanPostgreSQLPgStatMonitorAgentConfig returns desired configuration of qan-mongodb-profiler-agent built-in agent.
func qanPostgreSQLPgStatMonitorAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
		Dsn:                  agent.DSN(service, time.Second, service.DatabaseName, nil),
		DisableQueryExamples: agent.QueryExamplesDisabled,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
	}
}
