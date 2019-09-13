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

	"github.com/percona/pmm-managed/models"
)

// mysqldExporterConfig returns desired configuration of mysqld_exporter process.
func mysqldExporterConfig(service *models.Service, exporter *models.Agent) *agentpb.SetStateRequest_AgentProcess {
	tdp := templateDelimsPair(
		pointer.GetString(service.Address),
		pointer.GetString(exporter.Username),
		pointer.GetString(exporter.Password),
		pointer.GetString(exporter.MetricsURL),
	)

	args := []string{
		// LR
		"--collect.binlog_size",
		"--collect.engine_tokudb_status",
		"--collect.global_variables",
		"--collect.heartbeat",
		"--collect.info_schema.clientstats",
		"--collect.info_schema.innodb_tablespaces",
		"--collect.info_schema.userstats",
		"--collect.perf_schema.eventsstatements",
		"--collect.perf_schema.file_instances",
		"--collect.custom_query.lr",

		// LR: disabled due to https://jira.percona.com/browse/PMM-4610
		// TODO https://jira.percona.com/browse/PMM-4535
		"--no-collect.auto_increment.columns",
		"--no-collect.info_schema.tables",
		"--no-collect.info_schema.tablestats",
		"--no-collect.perf_schema.indexiowaits",
		"--no-collect.perf_schema.tableiowaits",

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

		// MR: disabled due to https://jira.percona.com/browse/PMM-4610
		// TODO https://jira.percona.com/browse/PMM-4535
		"--no-collect.perf_schema.tablelocks",

		// HR
		"--collect.global_status",
		"--collect.info_schema.innodb_metrics",
		"--collect.custom_query.hr",
		"--collect.standard.go",
		"--collect.standard.process",

		"--collect.custom_query.lr.directory=/usr/local/percona/pmm2/collectors/custom-queries/mysql/low-resolution",
		"--collect.custom_query.mr.directory=/usr/local/percona/pmm2/collectors/custom-queries/mysql/medium-resolution",
		"--collect.custom_query.hr.directory=/usr/local/percona/pmm2/collectors/custom-queries/mysql/high-resolution",

		"--exporter.max-idle-conns=3",
		"--exporter.max-open-conns=3",
		"--exporter.conn-max-lifetime=55s",
		"--exporter.global-conn-pool",
		"--web.listen-address=:" + tdp.left + " .listen_port " + tdp.right,
	}

	if pointer.GetString(exporter.MetricsURL) != "" {
		args = append(args, "--web.telemetry-path="+*exporter.MetricsURL)
	}

	sort.Strings(args)

	return &agentpb.SetStateRequest_AgentProcess{
		Type:               agentpb.Type_MYSQLD_EXPORTER,
		TemplateLeftDelim:  tdp.left,
		TemplateRightDelim: tdp.right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, time.Second, "")),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.AgentID),
		},
	}
}

// qanMySQLPerfSchemaAgentConfig returns desired configuration of qan-mysql-perfschema built-in agent.
func qanMySQLPerfSchemaAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 agentpb.Type_QAN_MYSQL_PERFSCHEMA_AGENT,
		Dsn:                  agent.DSN(service, time.Second, ""),
		DisableQueryExamples: agent.QueryExamplesDisabled,
	}
}

// qanMySQLSlowlogAgentConfig returns desired configuration of qan-mysql-slowlog built-in agent.
func qanMySQLSlowlogAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type:                 agentpb.Type_QAN_MYSQL_SLOWLOG_AGENT,
		Dsn:                  agent.DSN(service, time.Second, ""),
		DisableQueryExamples: agent.QueryExamplesDisabled,
		MaxQueryLogSize:      agent.MaxQueryLogSize,
	}
}
