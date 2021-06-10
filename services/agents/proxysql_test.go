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
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

func TestProxySQLExporterConfig(t *testing.T) {
	proxysql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(3306),
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.ProxySQLExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
	}
	actual := proxysqlExporterConfig(proxysql, exporter, redactSecrets)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_PROXYSQL_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"-collect.mysql_connection_list",
			"-collect.mysql_connection_pool",
			"-collect.mysql_status",
			"-collect.stats_command_counter",
			"-collect.stats_memory_metrics",
			"-web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:3306)/?timeout=1s",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual := proxysqlExporterConfig(proxysql, exporter, exposeSecrets)
		assert.Equal(t, "DATA_SOURCE_NAME=username@tcp(1.2.3.4:3306)/?timeout=1s", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual := proxysqlExporterConfig(proxysql, exporter, exposeSecrets)
		assert.Equal(t, "DATA_SOURCE_NAME=tcp(1.2.3.4:3306)/?timeout=1s", actual.Env[0])
	})

	t.Run("DisabledCollector", func(t *testing.T) {
		exporter.DisabledCollectors = []string{"mysql_connection_list", "stats_memory_metrics"}
		actual := proxysqlExporterConfig(proxysql, exporter, exposeSecrets)
		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_PROXYSQL_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"-collect.mysql_connection_pool",
				"-collect.mysql_status",
				"-collect.stats_command_counter",
				"-web.listen-address=:{{ .listen_port }}",
			},
		}
		require.Equal(t, expected.Args, actual.Args)
	})
}
