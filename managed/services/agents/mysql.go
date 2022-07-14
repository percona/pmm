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

// mysqldExporterConfig returns desired configuration of mysqld_exporter process.
func mysqldExporterConfig(service *models.Service, exporter *models.Agent, redactMode redactMode, pmmAgentVersion *version.Parsed) *agentpb.SetStateRequest_AgentProcess {
	tdp := exporter.TemplateDelimiters(service)

	args := []string{
		// LR
		"--collect.binlog_size",
		"--collect.engine_tokudb_status",
		"--collect.global_variables",
		"--collect.heartbeat",
		"--collect.info_schema.clientstats",
		"--collect.info_schema.userstats",
		"--collect.perf_schema.eventsstatements",
		"--collect.custom_query.lr",

		// MR
		"--collect.engine_innodb_status",
		"--collect.info_schema.innodb_cmp",
		"--collect.info_schema.innodb_cmpmem",
		"--collect.info_schema.processlist",
		"--collect.info_schema.query_response_time",
		"--collect.perf_schema.eventswaits",
		"--collect.perf_schema.file_events",
		"--collect.slave_status",
		"--collect.custom_query.mr",

		// HR
		"--collect.global_status",
		"--collect.info_schema.innodb_metrics",
		"--collect.custom_query.hr",
		"--collect.standard.go",
		"--collect.standard.process",

		"--collect.custom_query.lr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/mysql/low-resolution",
		"--collect.custom_query.mr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/mysql/medium-resolution",
		"--collect.custom_query.hr.directory=" + pathsBase(pmmAgentVersion, tdp.Left, tdp.Right) + "/collectors/custom-queries/mysql/high-resolution",

		"--exporter.max-idle-conns=3",
		"--exporter.max-open-conns=3",
		"--exporter.conn-max-lifetime=55s",
		"--exporter.global-conn-pool",
		"--web.listen-address=:" + tdp.Left + " .listen_port " + tdp.Right,
	}

	if exporter.IsMySQLTablestatsGroupEnabled() {
		// keep in sync with Prometheus scrape configs generator
		tablestatsGroup := []string{
			// LR
			"--collect.info_schema.innodb_tablespaces",
			"--collect.auto_increment.columns",
			"--collect.info_schema.tables",
			"--collect.info_schema.tablestats",
			"--collect.perf_schema.indexiowaits",
			"--collect.perf_schema.tableiowaits",
			"--collect.perf_schema.file_instances",

			// MR
			"--collect.perf_schema.tablelocks",
		}
		args = append(args, tablestatsGroup...)
	}

	args = collectors.FilterOutCollectors("--collect.", args, exporter.DisabledCollectors)

	if pointer.GetString(exporter.MetricsPath) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsPath)
	}

	files := exporter.Files()
	if files != nil {
		for k := range files {
			switch k {
			case "tlsCa":
				args = append(args, "--mysql.ssl-ca-file="+tdp.Left+" .TextFiles.tlsCa "+tdp.Right)
			case "tlsCert":
				args = append(args, "--mysql.ssl-cert-file="+tdp.Left+" .TextFiles.tlsCert "+tdp.Right)
			case "tlsKey":
				args = append(args, "--mysql.ssl-key-file="+tdp.Left+" .TextFiles.tlsKey "+tdp.Right)
			default:
				continue
			}
		}

		if exporter.TLSSkipVerify {
			args = append(args, "--mysql.ssl-skip-verify")
		}
	}

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion)

	sort.Strings(args)

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_MYSQLD_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, time.Second, "", nil)),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.GetAgentPassword()),
		},
		TextFiles: exporter.Files(),
	}
	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res
}

// qanMySQLPerfSchemaAgentConfig returns desired configuration of qan-mysql-perfschema built-in agent.
func qanMySQLPerfSchemaAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
		Dsn:                  agent.DSN(service, time.Second, "", nil),
		DisableQueryExamples: agent.QueryExamplesDisabled,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
		TlsSkipVerify: agent.TLSSkipVerify,
	}
}

// qanMySQLSlowlogAgentConfig returns desired configuration of qan-mysql-slowlog built-in agent.
func qanMySQLSlowlogAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	tdp := agent.TemplateDelimiters(service)
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 inventorypb.AgentType_QAN_MYSQL_SLOWLOG_AGENT,
		Dsn:                  agent.DSN(service, time.Second, "", nil),
		DisableQueryExamples: agent.QueryExamplesDisabled,
		MaxQueryLogSize:      agent.MaxQueryLogSize,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
		TlsSkipVerify: agent.TLSSkipVerify,
	}
}
