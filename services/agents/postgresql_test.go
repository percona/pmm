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

func TestPostgresExporterConfig(t *testing.T) {
	postgresql := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(5432),
	}
	exporter := &models.Agent{
		Username: pointer.ToString("username"),
		Password: pointer.ToString("s3cur3 p@$$w0r4."),
	}
	actual := postgresExporterConfig(postgresql, exporter)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               agentpb.Type_POSTGRES_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"-web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"DATA_SOURCE_NAME=postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable",
		},
	}
	assert.Equal(t, expected.Args, actual.Args)
	assert.Equal(t, expected.Env, actual.Env)
	assert.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual := postgresExporterConfig(postgresql, exporter)
		assert.Equal(t, "DATA_SOURCE_NAME=postgres://username@1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual := postgresExporterConfig(postgresql, exporter)
		assert.Equal(t, "DATA_SOURCE_NAME=postgres://1.2.3.4:5432/postgres?connect_timeout=1&sslmode=disable", actual.Env[0])
	})
}
