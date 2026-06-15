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
	"time"

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
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mock.ExpectQuery(`SELECT .+ FROM "agents" WHERE .+ LIMIT 1`).
		WithArgs("pmm-agent-id").
		WillReturnError(reform.ErrNoRows)

	connectionTimeout := 7 * time.Second
	service := &models.Service{
		ServiceType:  models.MySQLServiceType,
		Address:      new("127.0.0.1"),
		Port:         new(uint16(3306)),
		DatabaseName: "mysql",
	}
	agent := &models.Agent{
		AgentType:  models.MySQLdExporterType,
		PMMAgentID: new("pmm-agent-id"),
		Username:   new("pmm-agent"),
		Password:   new("password"),
		ExporterOptions: models.ExporterOptions{
			ConnectionTimeout: new(connectionTimeout),
		},
	}

	request, err := connectionRequest(db.Querier, service, agent)
	require.NoError(t, err)

	assert.Contains(t, request.Dsn, "timeout=7s")
	assert.Equal(t, 8*time.Second, request.Timeout.AsDuration())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionRequestDialTimeoutRoundsWholeSecondTimeoutsUp(t *testing.T) {
	t.Parallel()

	connectionTimeout := 1500 * time.Millisecond
	for _, tc := range []struct {
		serviceType models.ServiceType
		agentType   models.AgentType
	}{
		{
			serviceType: models.MySQLServiceType,
			agentType:   models.MySQLdExporterType,
		},
		{
			serviceType: models.PostgreSQLServiceType,
			agentType:   models.PostgresExporterType,
		},
	} {
		t.Run(string(tc.serviceType), func(t *testing.T) {
			t.Parallel()

			agent := &models.Agent{
				AgentType: tc.agentType,
				ExporterOptions: models.ExporterOptions{
					ConnectionTimeout: new(connectionTimeout),
				},
			}

			timeout := connectionCheckDialTimeout(nil, agent)
			assert.Equal(t, 2*time.Second, timeout)
		})
	}
}

func TestConnectionRequestDialTimeoutPostgreSQLCloudDefaults(t *testing.T) {
	t.Parallel()

	t.Run("Azure", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{
			AgentType: models.PostgresExporterType,
			AzureOptions: models.AzureOptions{
				ClientID: "azure-client",
			},
		}

		timeout := connectionCheckDialTimeout(nil, agent)
		assert.Equal(t, 5*time.Second, timeout)
	})

	t.Run("RDS", func(t *testing.T) {
		t.Parallel()

		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = mock.ExpectClose()
			assert.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		nodeColumns := []string{
			"node_id", "node_type", "node_name", "machine_id", "distro", "node_model", "az", "custom_labels",
			"address", "instance_id", "created_at", "updated_at", "container_id", "container_name", "region", "is_pmm_server_node",
		}
		mock.ExpectQuery(`SELECT .+ FROM "nodes" WHERE .+ LIMIT 1`).
			WithArgs("node-id").
			WillReturnRows(sqlmock.NewRows(nodeColumns).AddRow(
				"node-id",
				string(models.RemoteRDSNodeType),
				"node-name",
				nil,
				"",
				"",
				"",
				nil,
				"1.2.3.4",
				"instance-id",
				time.Now(),
				time.Now(),
				nil,
				nil,
				nil,
				false,
			))

		service := &models.Service{
			ServiceType: models.PostgreSQLServiceType,
			NodeID:      "node-id",
		}
		agent := &models.Agent{
			AgentType: models.PostgresExporterType,
		}

		node, err := models.FindNodeByID(db.Querier, service.NodeID)
		require.NoError(t, err)
		timeout := connectionCheckDialTimeout(node, agent)
		assert.Equal(t, 5*time.Second, timeout)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestConnectionRequestTimeoutUsesConnectionTimeoutOverhead(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3*time.Second, requestTimeout(2*time.Second).AsDuration())
	assert.Equal(t, 11*time.Second, requestTimeout(10*time.Second).AsDuration())
}
