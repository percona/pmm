// pmm-managed
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

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/collectors"
)

// postgresExporterConfig returns desired configuration of postgres_exporter process.
func postgresExporterConfig(service *models.Service, exporter *models.Agent, redactMode redactMode) *agentpb.SetStateRequest_AgentProcess {
	tdp := exporter.TemplateDelimiters(service)

	args := []string{
		// LR
		"--collect.custom_query.lr",

		// MR
		"--collect.custom_query.mr",

		// HR
		"--collect.custom_query.hr",

		"--collect.custom_query.lr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/low-resolution",
		"--collect.custom_query.mr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/medium-resolution",
		"--collect.custom_query.hr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/high-resolution",

		"--auto-discover-databases",
		"--exclude-databases=template0,template1,postgres",
		"--web.listen-address=:" + tdp.Left + " .listen_port " + tdp.Right,
	}

	if pointer.GetString(exporter.MetricsPath) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsPath)
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.DisabledCollectors)

	sort.Strings(args)

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, time.Second, "postgres", nil)),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.AgentID),
		},
	}
	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res
}

// qanPostgreSQLPgStatementsAgentConfig returns desired configuration of qan-mongodb-profiler-agent built-in agent.
func qanPostgreSQLPgStatementsAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type: inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
		Dsn:  agent.DSN(service, time.Second, "postgres", nil),
	}
}

// qanPostgreSQLPgStatMonitorAgentConfig returns desired configuration of qan-mongodb-profiler-agent built-in agent.
func qanPostgreSQLPgStatMonitorAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
		Dsn:                  agent.DSN(service, time.Second, "postgres", nil),
		DisableQueryExamples: agent.QueryExamplesDisabled,
	}
}
