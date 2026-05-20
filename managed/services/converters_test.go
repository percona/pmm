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

package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestToAPIAgent(t *testing.T) {
	t.Parallel()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	service, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "test-mongodb",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
		Cluster:     "test-cluster",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	mysqlService, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "test-mysql",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(3306)),
		Cluster:     "test-cluster",
	})
	require.NoError(t, err)

	type args struct {
		q     *reform.Querier
		agent *models.Agent
	}

	tests := []struct {
		name    string
		args    args
		want    inventoryv1.Agent
		wantErr error
	}{
		{
			name: "valid RTA MongoDB agent",
			args: args{
				q: db.Querier,
				agent: &models.Agent{
					AgentID:       "agent-123",
					PMMAgentID:    &pmmAgent.AgentID,
					ServiceID:     &service.ServiceID,
					AgentType:     models.RTAMongoDBAgentType,
					Disabled:      false,
					Username:      new("test-user"),
					Password:      new("test-pass"),
					TLS:           true,
					TLSSkipVerify: true,
					RTAOptions:    models.RTAOptions{CollectInterval: new(2 * time.Second)},
					Status:        inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_RUNNING)],
				},
			},
			want: &inventoryv1.RTAMongoDBAgent{
				AgentId:       "agent-123",
				PmmAgentId:    pmmAgent.AgentID,
				ServiceId:     service.ServiceID,
				Disabled:      false,
				Username:      "test-user",
				Tls:           true,
				TlsSkipVerify: true,
				Status:        inventoryv1.AgentStatus_AGENT_STATUS_RUNNING,
				RtaOptions:    &inventoryv1.RTAOptions{CollectInterval: durationpb.New(2 * time.Second)},
			},
			wantErr: nil,
		},
		{
			name: "mysqld exporter with ExporterOptions timeout",
			args: args{
				q: db.Querier,
				agent: &models.Agent{
					AgentID:    "mysqld-agent-1",
					PMMAgentID: &pmmAgent.AgentID,
					ServiceID:  &mysqlService.ServiceID,
					AgentType:  models.MySQLdExporterType,
					Disabled:   false,
					Username:   new("exporter-user"),
					Status:     inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN)],
					MySQLOptions: models.MySQLOptions{
						TableCountTablestatsGroupLimit: 1000,
					},
					ExporterOptions: models.ExporterOptions{
						ConnectionTimeout: new(9 * time.Second),
					},
				},
			},
			want: &inventoryv1.MySQLdExporter{
				AgentId:                   "mysqld-agent-1",
				PmmAgentId:                pmmAgent.AgentID,
				ServiceId:                 mysqlService.ServiceID,
				Username:                  "exporter-user",
				Disabled:                  false,
				Status:                    inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
				TablestatsGroupTableLimit: 1000,
				ConnectionTimeout:         durationpb.New(9 * time.Second),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ToAPIAgent(tt.args.q, tt.args.agent)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.Equalf(t, tt.want, got, "ToAPIAgent(%v, %v)", tt.args.q, tt.args.agent)
			}
		})
	}
}
