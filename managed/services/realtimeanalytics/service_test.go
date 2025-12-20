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

func TestListRunningRealtimeAgents(t *testing.T) {
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
	agent, err := models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		RTAOptions: models.RTAOptions{EnabledAt: &now},
	})
	require.NoError(t, err)

	// Create service with mock registry and store
	registry := &mockAgentsRegistry{
		connectedAgents: map[string]bool{
			pmmAgent.AgentID: true,
		},
	}
	store := NewStore()
	svc := NewService(db, registry, store)

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

	t.Run("show disconnected agents with unknown status", func(t *testing.T) {
		registry.connectedAgents[pmmAgent.AgentID] = false
		resp, err := svc.ListRunningRealtimeAgents(context.Background(), &rtav1.ListRunningRealtimeAgentsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Agents, 1)
		assert.Equal(t, inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN, resp.Agents[0].Status)
	})
}

func TestChangeRealtimeAnalytics(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Create test data
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	// Create MongoDB service with QAN agent (needed for credentials)
	service1, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-1",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(27017),
		Cluster:     "cluster-1",
	})
	require.NoError(t, err)

	// Create QAN agent to provide credentials
	_, err = models.CreateAgent(db.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service1.ServiceID,
		Username:   "qan-user",
		Password:   "qan-pass",
	})
	require.NoError(t, err)

	// Create second service in same cluster
	service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-2",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.2"),
		Port:        pointer.ToUint16(27017),
		Cluster:     "cluster-1",
	})
	require.NoError(t, err)

	_, err = models.CreateAgent(db.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service2.ServiceID,
		Username:   "qan-user",
		Password:   "qan-pass",
	})
	require.NoError(t, err)

	registry := &mockAgentsRegistry{
		connectedAgents: map[string]bool{
			pmmAgent.AgentID: true,
		},
	}
	store := NewStore()
	svc := NewService(db, registry, store)

	t.Run("enable RTA for single service", func(t *testing.T) {
		resp, err := svc.ChangeRealtimeAnalytics(context.Background(), &rtav1.ChangeRealtimeAnalyticsRequest{
			Enable:    true,
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify RTA agent was created
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.False(t, agents[0].Disabled)
		assert.NotNil(t, agents[0].RTAOptions.EnabledAt)
	})

	t.Run("disable RTA for single service", func(t *testing.T) {
		resp, err := svc.ChangeRealtimeAnalytics(context.Background(), &rtav1.ChangeRealtimeAnalyticsRequest{
			Enable:    false,
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify RTA agent was disabled
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.True(t, agents[0].Disabled)
		assert.Nil(t, agents[0].RTAOptions.EnabledAt)
	})

	t.Run("error on non-existent service", func(t *testing.T) {
		_, err := svc.ChangeRealtimeAnalytics(context.Background(), &rtav1.ChangeRealtimeAnalyticsRequest{
			Enable:    true,
			ServiceId: "non-existent",
		})
		require.Error(t, err)
		// CreateMongoDBRealtimeAgent validates the service exists, so we get NotFound
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("idempotent enable", func(t *testing.T) {
		// Enable twice
		resp, err := svc.ChangeRealtimeAnalytics(context.Background(), &rtav1.ChangeRealtimeAnalyticsRequest{
			Enable:    true,
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		resp, err = svc.ChangeRealtimeAnalytics(context.Background(), &rtav1.ChangeRealtimeAnalyticsRequest{
			Enable:    true,
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Should still have only one agent
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
	})

	t.Run("disable non-existent agent is a no-op", func(t *testing.T) {
		// Create a new service without RTA agent
		service3, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongodb-3",
			NodeID:      node.NodeID,
			Address:     pointer.ToString("127.0.0.3"),
			Port:        pointer.ToUint16(27017),
			Cluster:     "cluster-2",
		})
		require.NoError(t, err)

		_, err = models.CreateAgent(db.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
			PMMAgentID: pmmAgent.AgentID,
			ServiceID:  service3.ServiceID,
			Username:   "qan-user",
			Password:   "qan-pass",
		})
		require.NoError(t, err)

		// Call disable on service that has no RTA agent yet
		resp, err := svc.ChangeRealtimeAnalytics(context.Background(), &rtav1.ChangeRealtimeAnalyticsRequest{
			Enable:    false,
			ServiceId: service3.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify no agent was created (disable non-existent is a no-op)
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service3.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Empty(t, agents, "No agent should be created when disabling non-existent agent")
	})
}
