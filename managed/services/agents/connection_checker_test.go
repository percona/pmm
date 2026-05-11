// Copyright (C) 2026 Percona LLC
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
	"time"

	"github.com/AlekSi/pointer"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
)

func TestConnectionRequestUsesExporterConnectionTimeout(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close() //nolint:errcheck

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
		WithArgs("pmm-agent-id").
		WillReturnError(reform.ErrNoRows)

	connectionTimeout := 7 * time.Second
	service := &models.Service{
		ServiceType:  models.MySQLServiceType,
		Address:      pointer.ToString("127.0.0.1"),
		Port:         pointer.ToUint16(3306),
		DatabaseName: "mysql",
	}
	agent := &models.Agent{
		AgentType:  models.MySQLdExporterType,
		PMMAgentID: pointer.ToString("pmm-agent-id"),
		Username:   pointer.ToString("pmm-agent"),
		Password:   pointer.ToString("password"),
		ExporterOptions: models.ExporterOptions{
			ConnectionTimeout: pointer.ToDuration(connectionTimeout),
		},
	}

	request, err := connectionRequest(db.Querier, service, agent)
	require.NoError(t, err)

	assert.Contains(t, request.Dsn, "timeout=7s")
	assert.Equal(t, 8*time.Second, request.Timeout.AsDuration())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionRequestDialTimeoutRoundsPostgreSQLTimeoutUp(t *testing.T) {
	t.Parallel()

	connectionTimeout := 1500 * time.Millisecond
	service := &models.Service{
		ServiceType: models.PostgreSQLServiceType,
	}
	agent := &models.Agent{
		AgentType: models.PostgresExporterType,
		ExporterOptions: models.ExporterOptions{
			ConnectionTimeout: pointer.ToDuration(connectionTimeout),
		},
	}

	assert.Equal(t, 2*time.Second, connectionRequestDialTimeout(service, agent))
}
