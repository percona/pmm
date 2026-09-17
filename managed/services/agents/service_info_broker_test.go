// Copyright (C) 2023 Percona LLC
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

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
)

func TestServiceInfoRequestValkeyForwardsTLSSettings(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
		WithArgs("pmm-agent-id").
		WillReturnError(reform.ErrNoRows)

	service := &models.Service{
		ServiceType: models.ValkeyServiceType,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(6379)),
	}
	agent := &models.Agent{
		AgentType:     models.ValkeyExporterType,
		PMMAgentID:    new("pmm-agent-id"),
		Username:      new("pmm-agent"),
		Password:      new("password"),
		TLS:           true,
		TLSSkipVerify: true,
		ValkeyOptions: models.ValkeyOptions{SSLCa: "ca-pem"},
	}

	request, err := serviceInfoRequest(db.Querier, service, agent)
	require.NoError(t, err)

	assert.True(t, request.Tls)
	assert.True(t, request.TlsSkipVerify)
	assert.Contains(t, request.Dsn, "rediss://")
	assert.Equal(t, map[string]string{"tlsCa": "ca-pem"}, request.TextFiles.Files)
	require.NoError(t, mock.ExpectationsWereMet())
}
