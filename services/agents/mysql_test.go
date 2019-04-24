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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm-managed/models"
)

func TestMySQLdExporterConfig(t *testing.T) {
	mysql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(3306),
	}
	exporter := &models.Agent{
		Username: pointer.ToString("username"),
		Password: pointer.ToString("s3cur3 p@$$w0r4."),
	}
	actual := mysqldExporterConfig(mysql, exporter)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               agentpb.Type_MYSQLD_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"-collect.auto_increment.columns",
			"-collect.binlog_size",
			"-collect.custom_query=false",
			"-collect.global_status",
			"-collect.global_variables",
			"-collect.info_schema.innodb_metrics",
			"-collect.info_schema.processlist",
			"-collect.info_schema.query_response_time",
			"-collect.info_schema.tables",
			"-collect.info_schema.tablestats",
			"-collect.info_schema.userstats",
			"-collect.perf_schema.eventswaits",
			"-collect.perf_schema.file_events",
			"-collect.perf_schema.indexiowaits",
			"-collect.perf_schema.tableiowaits",
			"-collect.perf_schema.tablelocks",
			"-collect.slave_status",
			"-web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:3306)/?clientFoundRows=true&parseTime=true&timeout=5s",
		},
	}
	assert.Equal(t, expected.Args, actual.Args)
	assert.Equal(t, expected.Env, actual.Env)
	assert.Equal(t, expected, actual)
}
