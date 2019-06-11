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
		"-collect.binlog_size",
		"-collect.global_status",
		"-collect.global_variables",
		"-collect.info_schema.innodb_cmp",
		"-collect.info_schema.innodb_cmpmem",
		"-collect.info_schema.innodb_metrics",
		"-collect.info_schema.processlist",
		"-collect.info_schema.query_response_time",
		"-collect.info_schema.userstats",
		"-collect.perf_schema.eventswaits",
		"-collect.perf_schema.file_events",
		"-collect.slave_status",
		"-web.listen-address=:" + tdp.left + " .listen_port " + tdp.right,
	}

	// TODO Make it configurable.
	args = append(args, "-collect.auto_increment.columns")
	args = append(args, "-collect.info_schema.tables")
	args = append(args, "-collect.info_schema.tablestats")
	args = append(args, "-collect.perf_schema.indexiowaits")
	args = append(args, "-collect.perf_schema.tableiowaits")
	args = append(args, "-collect.perf_schema.tablelocks")
	args = append(args, "-collect.custom_query=false")

	if pointer.GetString(exporter.MetricsURL) != "" {
		args = append(args, "-web.telemetry-path="+*exporter.MetricsURL)
	}

	sort.Strings(args)

	return &agentpb.SetStateRequest_AgentProcess{
		Type:               agentpb.Type_MYSQLD_EXPORTER,
		TemplateLeftDelim:  tdp.left,
		TemplateRightDelim: tdp.right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, time.Second, "")),
		},
	}
}

// qanMySQLPerfSchemaAgentConfig returns desired configuration of qan-mysql-perfschema built-in agent.
func qanMySQLPerfSchemaAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type: agentpb.Type_QAN_MYSQL_PERFSCHEMA_AGENT,
		Dsn:  agent.DSN(service, time.Second, ""),
	}
}

// qanMySQLSlowlogAgentConfig returns desired configuration of qan-mysql-slowlog built-in agent.
func qanMySQLSlowlogAgentConfig(service *models.Service, agent *models.Agent) *agentpb.SetStateRequest_BuiltinAgent {
	return &agentpb.SetStateRequest_BuiltinAgent{
		Type: agentpb.Type_QAN_MYSQL_SLOWLOG_AGENT,
		Dsn:  agent.DSN(service, time.Second, ""),
	}
}
