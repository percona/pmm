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
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/collectors"
	"github.com/percona/pmm/version"
)

var (
	proxysqlExporterStatsCommandVersion = version.MustParse("2.18.99")
	proxysqlExporterRuntimeVersion      = version.MustParse("2.20.99")
)

// proxysqlExporterConfig returns desired configuration of proxysql_exporter process.
func proxysqlExporterConfig(service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) *agentpb.SetStateRequest_AgentProcess {
	tdp := exporter.TemplateDelimiters(service)

	args := []string{
		"-collect.mysql_connection_list",
		"-collect.mysql_connection_pool",
		"-collect.mysql_status",
		"-collect.stats_memory_metrics",
		"-web.listen-address=:" + tdp.Left + " .listen_port " + tdp.Right,
	}

	if !pmmAgentVersion.Less(proxysqlExporterStatsCommandVersion) {
		args = append(args, "-collect.stats_command_counter")
	}

	if !pmmAgentVersion.Less(proxysqlExporterRuntimeVersion) {
		args = append(args, "-collect.runtime_mysql_servers")
	}

	if pointer.GetString(exporter.MetricsPath) != "" {
		args = append(args, "-web.telemetry-path="+*exporter.MetricsPath)
	}

	args = collectors.FilterOutCollectors("-collect.", args, exporter.DisabledCollectors)

	args = withLogLevel(args, exporter.LogLevel, pmmAgentVersion)

	sort.Strings(args)

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_PROXYSQL_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		Env: []string{
			fmt.Sprintf("DATA_SOURCE_NAME=%s", exporter.DSN(service, time.Second, "", nil)),
			fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.GetAgentPassword()),
		},
	}
	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res
}
