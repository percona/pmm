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
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestListSessions(t *testing.T) {
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

	// Create a MongoDB Realtime Agent
	rtaAgent, err := models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		Disabled:   false,
		RTAOptions: models.RTAOptions{CollectInterval: pointer.To(2 * time.Second)},
	})
	require.NoError(t, err)

	// Create service with mock registry and store
	stateUpdater := newMockAgentsStateUpdater(t)
	store := NewStore()

	t.Run("list running sessions", func(t *testing.T) {
		registry := newMockAgentsRegistry(t)
		registry.On("IsConnected", pmmAgent.AgentID).Return(true)
		svc := NewService(db, registry, stateUpdater, store)

		rtaAgent.Status = inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_RUNNING)]
		err = db.Querier.Update(rtaAgent)

		resp, err := svc.ListSessions(context.Background(), &rtav1.ListSessionsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Sessions, 1)

		assert.Equal(t, service.ServiceID, resp.Sessions[0].ServiceId)
		assert.Equal(t, service.ServiceName, resp.Sessions[0].ServiceName)
		assert.Equal(t, "test-cluster", resp.Sessions[0].ClusterName)
		assert.Equal(t, rtav1.SessionStatus_SESSION_STATUS_RUNNING, resp.Sessions[0].Status)
		assert.NotNil(t, resp.Sessions[0].StartTime)
	})

	t.Run("filter sessions by cluster", func(t *testing.T) {
		registry := newMockAgentsRegistry(t)
		registry.On("IsConnected", pmmAgent.AgentID).Return(true)
		svc := NewService(db, registry, stateUpdater, store)

		resp, err := svc.ListSessions(context.Background(), &rtav1.ListSessionsRequest{ClusterName: "test-cluster"})
		require.NoError(t, err)
		require.Len(t, resp.Sessions, 1)

		resp, err = svc.ListSessions(context.Background(), &rtav1.ListSessionsRequest{ClusterName: "absent-cluster"})
		require.NoError(t, err)
		require.Empty(t, resp.Sessions)
	})

	t.Run("show disconnected agents with unknown status", func(t *testing.T) {
		registry := newMockAgentsRegistry(t)
		registry.On("IsConnected", pmmAgent.AgentID).Return(false)
		svc := NewService(db, registry, stateUpdater, store)

		resp, err := svc.ListSessions(context.Background(), &rtav1.ListSessionsRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Sessions, 1)
		assert.Equal(t, rtav1.SessionStatus_SESSION_STATUS_UNSPECIFIED, resp.Sessions[0].Status)
	})
}

func TestStartSession(t *testing.T) {
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

	registry := newMockAgentsRegistry(t)
	registry.On("IsConnected", pmmAgent.AgentID).Return(true)

	stateUpdater := newMockAgentsStateUpdater(t)
	stateUpdater.On("RequestStateUpdate", context.Background(), pmmAgent.AgentID).Return()

	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	t.Run("start session for single service", func(t *testing.T) {
		resp, err := svc.StartSession(context.Background(), &rtav1.StartSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Session.StartTime)

		// Verify RTA agent was created
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.False(t, agents[0].Disabled)
		// assert.NotNil(t, agents[0].RTAOptions.EnabledAt)
	})

	t.Run("idempotent start session", func(t *testing.T) {
		// Enable twice
		resp1, err := svc.StartSession(context.Background(), &rtav1.StartSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp1)
		assert.NotNil(t, resp1.Session.StartTime)

		resp2, err := svc.StartSession(context.Background(), &rtav1.StartSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp2)
		assert.NotNil(t, resp2.Session.StartTime)
		assert.Equal(t, resp1.Session.StartTime, resp2.Session.StartTime)

		// Should still have only one agent
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
	})

	t.Run("start session for existing disabled agent", func(t *testing.T) {
		// Create second service in same cluster
		service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongodb-2",
			NodeID:      node.NodeID,
			Address:     pointer.ToString("127.0.0.2"),
			Port:        pointer.ToUint16(27017),
			Cluster:     "cluster-1",
		})
		require.NoError(t, err)

		// Create a MongoDB Realtime Agent
		_, err = models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
			PMMAgentID: pmmAgent.AgentID,
			ServiceID:  service2.ServiceID,
			Username:   "test-user",
			Password:   "test-pass",
			Disabled:   true,
			RTAOptions: models.RTAOptions{CollectInterval: pointer.To(2 * time.Second)},
		})
		require.NoError(t, err)

		resp, err := svc.StartSession(context.Background(), &rtav1.StartSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Session.StartTime)

		// Verify RTA agent was created
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service2.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.False(t, agents[0].Disabled)
	})

	t.Run("error on non-existent service", func(t *testing.T) {
		_, err := svc.StartSession(context.Background(), &rtav1.StartSessionRequest{
			ServiceId: "absent-service",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestStopSession(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Create test data
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	// Create MongoDB services
	service1, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-1",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(27017),
		Cluster:     "cluster-1",
	})
	require.NoError(t, err)

	service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-2",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(27017),
		Cluster:     "cluster-2",
	})
	require.NoError(t, err)

	// Create a MongoDB Realtime Agents
	_, err = models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service1.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		Disabled:   false,
		RTAOptions: models.RTAOptions{CollectInterval: pointer.To(2 * time.Second)},
	})
	require.NoError(t, err)

	_, err = models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service2.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		Disabled:   false,
		RTAOptions: models.RTAOptions{CollectInterval: pointer.To(2 * time.Second)},
	})
	require.NoError(t, err)

	registry := newMockAgentsRegistry(t)
	stateUpdater := newMockAgentsStateUpdater(t)
	stateUpdater.On("RequestStateUpdate", context.Background(), pmmAgent.AgentID).Return()
	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	t.Run("stop session for single service", func(t *testing.T) {
		resp, err := svc.StopSession(context.Background(), &rtav1.StopSessionRequest{
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
	})

	t.Run("idempotent stop session", func(t *testing.T) {
		// Enable twice
		resp, err := svc.StopSession(context.Background(), &rtav1.StopSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		resp, err = svc.StopSession(context.Background(), &rtav1.StopSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Should still have only one agent
		agentType := models.RTAMongoDBAgentType
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service2.ServiceID,
			AgentType: &agentType,
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		require.True(t, agents[0].Disabled)
	})

	t.Run("error on non-existent service", func(t *testing.T) {
		_, err = svc.StopSession(context.Background(), &rtav1.StopSessionRequest{
			ServiceId: "absent-service",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("stop non-existent session is a no-op", func(t *testing.T) {
		// Create a new service without RTA agent
		service3, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongodb-3",
			NodeID:      node.NodeID,
			Address:     pointer.ToString("127.0.0.3"),
			Port:        pointer.ToUint16(27017),
			Cluster:     "cluster-3",
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
		resp, err := svc.StopSession(context.Background(), &rtav1.StopSessionRequest{
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

func TestSearchQueries(t *testing.T) {
	t.Parallel()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Create test data
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
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

	service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-2",
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(27017),
		Cluster:     "cluster-2",
	})
	require.NoError(t, err)

	registry := newMockAgentsRegistry(t)
	stateUpdater := newMockAgentsStateUpdater(t)
	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	// Populate store with static query data for service1
	store.Set(service1.ServiceID, []*rtav1.QueryData{
		{
			ServiceId:         service1.ServiceID,
			ServiceName:       service1.ServiceName,
			QueryId:           "static-query-1",
			QueryText:         `{ find: "mycollection", filter: { status: "active" } }`,
			State:             "RUNNING",
			ExecutionDuration: durationpb.New(15),
			RowsExamined:      200,
			RowsSent:          100,
			CollectTime:       timestamppb.Now(),
			RawQueryJson:      `{ find: "mycollection", filter: { status: "active" } }`,
			Payload: &rtav1.QueryData_MongoDbPayload{
				MongoDbPayload: &rtav1.QueryMongoDBData{
					Opid:           "1",
					Client:         "127.0.0.1:5060",
					WaitingForLock: false,
					IndexUtilized:  "COLLSCAN",
				},
			},
		},
		{
			ServiceId:         service1.ServiceID,
			ServiceName:       service1.ServiceName,
			QueryId:           "static-query-2",
			QueryText:         `{ find: "mycollection", filter: { status: "active" } }`,
			State:             "PROCESSING",
			ExecutionDuration: durationpb.New(25),
			RowsExamined:      200,
			RowsSent:          100,
			CollectTime:       timestamppb.Now(),
			RawQueryJson:      `{ find: "mycollection", filter: { status: "active" } }`,
			Payload: &rtav1.QueryData_MongoDbPayload{
				MongoDbPayload: &rtav1.QueryMongoDBData{
					Opid:           "2",
					Client:         "127.0.0.1:5061",
					WaitingForLock: true,
					IndexUtilized:  "IXSCAN",
				},
			},
		},
	})

	// Populate store with static query data for service2
	store.Set(service2.ServiceID, []*rtav1.QueryData{
		{
			ServiceId:         service2.ServiceID,
			ServiceName:       service2.ServiceName,
			QueryId:           "static-query-3",
			QueryText:         `{ find: "mycollection", filter: { status: "active" } }`,
			State:             "FINISHED",
			ExecutionDuration: durationpb.New(35),
			RowsExamined:      200,
			RowsSent:          100,
			CollectTime:       timestamppb.Now(),
			RawQueryJson:      `{ find: "mycollection", filter: { status: "active" } }`,
			Payload: &rtav1.QueryData_MongoDbPayload{
				MongoDbPayload: &rtav1.QueryMongoDBData{
					Opid:           "1",
					Client:         "127.0.0.1:5062",
					WaitingForLock: true,
					IndexUtilized:  "COLLSCAN",
				},
			},
		},
	})

	t.Run("search all queries for service1", func(t *testing.T) {
		t.Parallel()

		resp, err := svc.SearchQueries(context.Background(), &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service1.ServiceID},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, resp.Queries, 2)
		queryIDs := []string{resp.Queries[0].QueryId, resp.Queries[1].QueryId}
		assert.Contains(t, queryIDs, "static-query-1")
		assert.Contains(t, queryIDs, "static-query-2")
	})

	t.Run("search all queries for service2", func(t *testing.T) {
		t.Parallel()

		resp, err := svc.SearchQueries(context.Background(), &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service2.ServiceID},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, resp.Queries, 1)
		assert.Equal(t, "static-query-3", resp.Queries[0].QueryId)
	})

	t.Run("search all queries for both services", func(t *testing.T) {
		t.Parallel()

		resp, err := svc.SearchQueries(context.Background(), &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service1.ServiceID, service2.ServiceID},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, resp.Queries, 3)
		queryIDs := []string{resp.Queries[0].QueryId, resp.Queries[1].QueryId, resp.Queries[2].QueryId}
		assert.Contains(t, queryIDs, "static-query-1")
		assert.Contains(t, queryIDs, "static-query-2")
		assert.Contains(t, queryIDs, "static-query-3")
	})

	t.Run("search all queries for absent service", func(t *testing.T) {
		t.Parallel()

		_, err := svc.SearchQueries(context.Background(), &rtav1.SearchQueriesRequest{
			ServiceIds: []string{"absent-serivice"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("one of the services is absent", func(t *testing.T) {
		t.Parallel()

		_, err := svc.SearchQueries(context.Background(), &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service1.ServiceID, "absent-serivice"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
