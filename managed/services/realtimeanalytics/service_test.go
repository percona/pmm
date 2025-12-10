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

package realtimeanalytics

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

type mockAgentsRegistry struct {
	connectedAgents map[string]bool
}

func (m *mockAgentsRegistry) IsConnected(pmmAgentID string) bool {
	return m.connectedAgents[pmmAgentID]
}

func TestGetRunningRealtimeAgents(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Create test data
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	service, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "test-mongodb",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(27017),
		Cluster:     "test-cluster",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	// Create a MongoDB Realtime Agent with EnabledAt timestamp
	now := time.Now()
	agent, err := models.CreateAgent(db.Querier, models.MongoDBRealtimeAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		RTAOptions: models.RTAOptions{EnabledAt: &now},
	})
	require.NoError(t, err)

	// Create service with mock registry
	registry := &mockAgentsRegistry{
		connectedAgents: map[string]bool{
			pmmAgent.AgentID: true,
		},
	}
	svc := NewService(db, registry)

	t.Run("list running agents", func(t *testing.T) {
		resp, err := svc.ListRunningRealtimeAgents(context.Background(), &rtav1.ListRunningRealtimeAgentsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Agents, 1)

		assert.Equal(t, agent.AgentID, resp.Agents[0].AgentId)
		assert.Equal(t, service.ServiceID, resp.Agents[0].ServiceId)
		assert.Equal(t, service.ServiceName, resp.Agents[0].ServiceName)
		assert.Equal(t, "test-cluster", resp.Agents[0].Cluster)
		assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN, resp.Agents[0].Status) // Default status from database
		// Verify StartedAt uses EnabledAt from RTAOptions
		assert.WithinDuration(t, now, resp.Agents[0].StartedAt.AsTime(), time.Second)
	})

	t.Run("filter by cluster", func(t *testing.T) {
		resp, err := svc.ListRunningRealtimeAgents(context.Background(), &rtav1.ListRunningRealtimeAgentsRequest{Cluster: "test-cluster"})
		require.NoError(t, err)
		require.Len(t, resp.Agents, 1)

		resp, err = svc.ListRunningRealtimeAgents(context.Background(), &rtav1.ListRunningRealtimeAgentsRequest{Cluster: "other-cluster"})
		require.NoError(t, err)
		require.Len(t, resp.Agents, 0)
	})

	t.Run("show disconnected agents with unknown status", func(t *testing.T) {
		registry.connectedAgents[pmmAgent.AgentID] = false
		resp, err := svc.ListRunningRealtimeAgents(context.Background(), &rtav1.ListRunningRealtimeAgentsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Agents, 1)
		assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN, resp.Agents[0].Status)
	})
}
