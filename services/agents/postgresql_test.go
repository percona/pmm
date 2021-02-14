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

func TestPostgresExporterConfig(t *testing.T) {
	postgresql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(5432),
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.PostgresExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
	}
	actual := postgresExporterConfig(postgresql, exporter, redactSecrets)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.custom_query.hr",
			"--collect.custom_query.hr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/high-resolution",
			"--collect.custom_query.lr",
			"--collect.custom_query.lr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/low-resolution",
			"--collect.custom_query.mr",
			"--collect.custom_query.mr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/medium-resolution",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	requireNoDuplicateFlags(t, actual.Args)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual := postgresExporterConfig(postgresql, exporter, exposeSecrets)
		assert.Equal(t, "DATA_SOURCE_NAME=postgres://username@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual := postgresExporterConfig(postgresql, exporter, exposeSecrets)
		assert.Equal(t, "DATA_SOURCE_NAME=postgres://1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
	})

	t.Run("Socket", func(t *testing.T) {
		postgresql.Address = nil
		postgresql.Port = nil
		postgresql.Socket = pointer.ToString("/var/run/postgres")
		actual := postgresExporterConfig(postgresql, exporter, exposeSecrets)
		assert.Equal(t, "DATA_SOURCE_NAME=postgres:///postgres?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=disable", actual.Env[0])
	})

	t.Run("DisabledCollectors", func(t *testing.T) {
		postgresql.Address = nil
		postgresql.Port = nil
		postgresql.Socket = pointer.ToString("/var/run/postgres")
		exporter.DisabledCollectors = []string{"custom_query.hr", "custom_query.hr.directory"}
		actual := postgresExporterConfig(postgresql, exporter, exposeSecrets)
		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_POSTGRES_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collect.custom_query.lr",
				"--collect.custom_query.lr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/low-resolution",
				"--collect.custom_query.mr",
				"--collect.custom_query.mr.directory=/usr/local/percona/pmm2/collectors/custom-queries/postgresql/medium-resolution",
				"--web.listen-address=:{{ .listen_port }}",
			},
		}
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected.Args, actual.Args)
	})
}
