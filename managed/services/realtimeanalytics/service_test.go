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
	"fmt"
	"net"
	"testing"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/interceptors"
	"github.com/percona/pmm/managed/utils/testdb"
)

func getServiceQueries(serviceID, serviceName string, count int) []*rtav1.QueryData {
	data := make([]*rtav1.QueryData, count)
	for i := range count {
		data[i] = &rtav1.QueryData{
			ServiceId:              serviceID,
			ServiceName:            serviceName,
			QueryId:                fmt.Sprintf("static-query-%d", i),
			QueryText:              `{ find: "mycollection", filter: { status: "active" } }`,
			QueryExecutionDuration: durationpb.New(time.Duration(15 * i)),
			QueryCollectTime:       timestamppb.Now(),
			QueryRawJson:           `{ find: "mycollection", filter: { status: "active" } }`,
			ClientAddress:          "127.0.0.1:5060",
			Payload: &rtav1.QueryData_MongoDbPayload{
				MongoDbPayload: &rtav1.QueryMongoDBData{
					DbInstanceAddress:  "c4486b1ebd30:27017",
					DatabaseName:       "mydb",
					ClientAppName:      "myapp",
					Collection:         "mycollection",
					Operation:          "find",
					OperationStartTime: timestamppb.Now(),
					Username:           "test-user",
					PlanSummary:        "COLLSCAN",
				},
			},
		}
	}

	return data
}

func TestListServices(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Create test node and mongodbService
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	mongodbService, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "test-mongodb",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
		Cluster:     "test-cluster",
	})
	require.NoError(t, err)

	_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "test-mysql",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
	})
	require.NoError(t, err)

	_, err = models.AddNewService(db.Querier, models.ExternalServiceType, &models.AddDBMSServiceParams{
		ServiceName: "test-external",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	pmmAgent.Version = new("3.7.0")
	err = db.Update(pmmAgent)
	require.NoError(t, err)

	_, err = models.CreateAgent(db.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  mongodbService.ServiceID,
		Username:   "qan-user",
		Password:   "qan-pass",
	})
	require.NoError(t, err)

	registry := newMockAgentsRegistry(t)
	stateUpdater := newMockAgentsStateUpdater(t)
	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	t.Run("list all supported services", func(t *testing.T) {
		resp, err := svc.ListServices(t.Context(), &rtav1.ListServicesRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Mongodb, 1)
		assert.Equal(t, mongodbService.ServiceID, resp.Mongodb[0].ServiceId)
	})

	t.Run("filter by supported mongodbService type", func(t *testing.T) {
		resp, err := svc.ListServices(t.Context(), &rtav1.ListServicesRequest{
			ServiceType: inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Mongodb, 1)
		assert.Equal(t, mongodbService.ServiceID, resp.Mongodb[0].ServiceId)
	})

	t.Run("filter by unsupported mongodbService type", func(t *testing.T) {
		_, err := svc.ListServices(t.Context(), &rtav1.ListServicesRequest{
			ServiceType: inventoryv1.ServiceType_SERVICE_TYPE_EXTERNAL_SERVICE,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support Real-Time Analytics")
	})

	t.Run("skip mongodbService with unsupported pmm-agent version", func(t *testing.T) {
		// Create mongodbService with old agent version
		node2, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "test-node-2",
		})
		require.NoError(t, err)
		service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongodb-old",
			NodeID:      node2.NodeID,
			Address:     new("127.0.0.2"),
			Port:        new(uint16(27017)),
			Cluster:     "cluster-2",
		})
		require.NoError(t, err)
		pmmAgent2, err := models.CreatePMMAgent(db.Querier, node2.NodeID, nil)
		require.NoError(t, err)

		pmmAgent2.Version = new("3.6.0")
		err = db.Update(pmmAgent2)
		require.NoError(t, err)

		_, err = models.CreateAgent(db.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
			PMMAgentID: pmmAgent2.AgentID,
			ServiceID:  service2.ServiceID,
			Username:   "qan-user",
			Password:   "qan-pass",
		})
		require.NoError(t, err)

		resp, err := svc.ListServices(t.Context(), &rtav1.ListServicesRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		// Only the first mongodbService should be listed
		require.Len(t, resp.Mongodb, 1)
		assert.Equal(t, mongodbService.ServiceID, resp.Mongodb[0].ServiceId)
	})
}

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
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
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
		RTAOptions: models.RTAOptions{CollectInterval: new(2 * time.Second)},
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
		err = db.Update(rtaAgent)

		resp, err := svc.ListSessions(t.Context(), &rtav1.ListSessionsRequest{})
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

		resp, err := svc.ListSessions(t.Context(), &rtav1.ListSessionsRequest{ClusterName: "test-cluster"})
		require.NoError(t, err)
		require.Len(t, resp.Sessions, 1)

		resp, err = svc.ListSessions(t.Context(), &rtav1.ListSessionsRequest{ClusterName: "absent-cluster"})
		require.NoError(t, err)
		require.Empty(t, resp.Sessions)
	})

	t.Run("show disconnected agents with unknown status", func(t *testing.T) {
		registry := newMockAgentsRegistry(t)
		registry.On("IsConnected", pmmAgent.AgentID).Return(false)
		svc := NewService(db, registry, stateUpdater, store)

		resp, err := svc.ListSessions(t.Context(), &rtav1.ListSessionsRequest{})
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

	pmmAgent.Version = new("3.7.0")
	err = db.Update(pmmAgent)
	require.NoError(t, err)

	// Create MongoDB service with QAN agent (needed for credentials)
	service1, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-1",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
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
	stateUpdater.On("RequestStateUpdate", mock.Anything, pmmAgent.AgentID).Return()

	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	t.Run("start session for single service", func(t *testing.T) {
		resp, err := svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Session.StartTime)

		// Verify RTA agent was created
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: new(models.RTAMongoDBAgentType),
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.False(t, agents[0].Disabled)
	})

	t.Run("idempotent start session", func(t *testing.T) {
		// Enable twice
		resp1, err := svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp1)
		assert.NotNil(t, resp1.Session.StartTime)

		resp2, err := svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp2)
		assert.NotNil(t, resp2.Session.StartTime)
		assert.Equal(t, resp1.Session.StartTime, resp2.Session.StartTime)

		// Should still have only one agent
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: new(models.RTAMongoDBAgentType),
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
	})

	t.Run("start session for existing disabled agent", func(t *testing.T) {
		// Create second service in same cluster
		service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongodb-2",
			NodeID:      node.NodeID,
			Address:     new("127.0.0.2"),
			Port:        new(uint16(27017)),
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
			RTAOptions: models.RTAOptions{CollectInterval: new(2 * time.Second)},
		})
		require.NoError(t, err)

		resp, err := svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Session.StartTime)

		// Verify RTA agent was created
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service2.ServiceID,
			AgentType: new(models.RTAMongoDBAgentType),
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.False(t, agents[0].Disabled)
	})

	t.Run("error on non-existent service", func(t *testing.T) {
		_, err := svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: "absent-service",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("error on non-supported service type", func(t *testing.T) {
		service2, err := models.AddNewService(db.Querier, models.ExternalServiceType, &models.AddDBMSServiceParams{
			ServiceName: "external-1",
			NodeID:      node.NodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
		})
		require.NoError(t, err)
		_, err = svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
		assert.Equal(t, status.Convert(err).Message(), fmt.Sprintf("Service %s of type %s does not support Real-Time Analytics",
			service2.ServiceID, service2.ServiceType))
	})

	t.Run("no other agents available for RTA service", func(t *testing.T) {
		service3, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "external-psmdb-1",
			NodeID:      node.NodeID,
			Address:     new("127.0.0.2"),
			Port:        new(uint16(27017)),
		})
		require.NoError(t, err)
		_, err = svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: service3.ServiceID,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Convert(err).Code())
		assert.Equal(t, status.Convert(err).Message(), fmt.Sprintf("Service %s of type %s doesn't have agents to retrieve credentials and pmm-agent ID",
			service3.ServiceID, service3.ServiceType))
	})

	t.Run("pmm-agent doesn't support RTA", func(t *testing.T) {
		// Create test data
		nodeOld, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "test-node-2",
		})
		require.NoError(t, err)

		pmmAgentOld, err := models.CreatePMMAgent(db.Querier, nodeOld.NodeID, nil)
		require.NoError(t, err)

		pmmAgentOld.Version = new("3.6.0")
		err = db.Update(pmmAgentOld)
		require.NoError(t, err)

		// Create MongoDB service with QAN agent (needed for credentials)
		serviceOld, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongodb-old",
			NodeID:      nodeOld.NodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
			Cluster:     "cluster-2",
		})
		require.NoError(t, err)

		// Create QAN agent to provide credentials
		_, err = models.CreateAgent(db.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
			PMMAgentID: pmmAgentOld.AgentID,
			ServiceID:  serviceOld.ServiceID,
			Username:   "qan-user",
			Password:   "qan-pass",
		})
		require.NoError(t, err)

		_, err = svc.StartSession(t.Context(), &rtav1.StartSessionRequest{
			ServiceId: serviceOld.ServiceID,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Convert(err).Code())
		assert.Equal(t, status.Convert(err).Message(), fmt.Sprintf("Service %s has pmm-agent with version not supporting Real-Time Analytics.",
			serviceOld.ServiceID))
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
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
		Cluster:     "cluster-1",
	})
	require.NoError(t, err)

	service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-2",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
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
		RTAOptions: models.RTAOptions{CollectInterval: new(2 * time.Second)},
	})
	require.NoError(t, err)

	_, err = models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service2.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		Disabled:   false,
		RTAOptions: models.RTAOptions{CollectInterval: new(2 * time.Second)},
	})
	require.NoError(t, err)

	registry := newMockAgentsRegistry(t)
	stateUpdater := newMockAgentsStateUpdater(t)
	stateUpdater.On("RequestStateUpdate", mock.Anything, pmmAgent.AgentID).Return()

	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	t.Run("stop session for single service", func(t *testing.T) {
		resp, err := svc.StopSession(t.Context(), &rtav1.StopSessionRequest{
			ServiceId: service1.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify RTA agent was disabled
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service1.ServiceID,
			AgentType: new(models.RTAMongoDBAgentType),
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.True(t, agents[0].Disabled)
	})

	t.Run("idempotent stop session", func(t *testing.T) {
		// Enable twice
		resp, err := svc.StopSession(t.Context(), &rtav1.StopSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		resp, err = svc.StopSession(t.Context(), &rtav1.StopSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Should still have only one agent
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service2.ServiceID,
			AgentType: new(models.RTAMongoDBAgentType),
		})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		require.True(t, agents[0].Disabled)
	})

	t.Run("error on non-existent service", func(t *testing.T) {
		_, err = svc.StopSession(t.Context(), &rtav1.StopSessionRequest{
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
			Address:     new("127.0.0.3"),
			Port:        new(uint16(27017)),
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
		resp, err := svc.StopSession(t.Context(), &rtav1.StopSessionRequest{
			ServiceId: service3.ServiceID,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify no agent was created (disable non-existent is a no-op)
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{
			ServiceID: service3.ServiceID,
			AgentType: new(models.RTAMongoDBAgentType),
		})
		require.NoError(t, err)
		require.Empty(t, agents, "No agent should be created when disabling non-existent agent")
	})

	t.Run("error on non-supported service type", func(t *testing.T) {
		service2, err := models.AddNewService(db.Querier, models.ExternalServiceType, &models.AddDBMSServiceParams{
			ServiceName: "external-1",
			NodeID:      node.NodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
		})
		require.NoError(t, err)
		_, err = svc.StopSession(t.Context(), &rtav1.StopSessionRequest{
			ServiceId: service2.ServiceID,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
		assert.Equal(t, status.Convert(err).Message(), fmt.Sprintf("Service %s of type %s does not support Real-Time Analytics",
			service2.ServiceID, service2.ServiceType))
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
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
		Cluster:     "cluster-1",
	})
	require.NoError(t, err)

	service2, err := models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongodb-2",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(27017)),
		Cluster:     "cluster-2",
	})
	require.NoError(t, err)

	registry := newMockAgentsRegistry(t)
	stateUpdater := newMockAgentsStateUpdater(t)
	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)

	// Populate store with static query data for service1
	store.Set(service1.ServiceID, getServiceQueries(service1.ServiceID, service1.ServiceName, 2))

	// Populate store with static query data for service2
	store.Set(service2.ServiceID, getServiceQueries(service2.ServiceID, service2.ServiceName, 1))

	t.Run("search all queries for service1", func(t *testing.T) {
		t.Parallel()

		ctx := grpc.NewContextWithServerTransportStream(t.Context(), &grpc_gateway.ServerTransportStream{})
		resp, err := svc.SearchQueries(ctx, &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service1.ServiceID},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, resp.Queries, 2)

		assert.Equal(t, "static-query-1", resp.Queries[0].QueryId)
		assert.Equal(t, service1.ServiceID, resp.Queries[0].ServiceId)
		assert.Equal(t, service1.ServiceName, resp.Queries[0].ServiceName)

		assert.Equal(t, "static-query-0", resp.Queries[1].QueryId)
		assert.Equal(t, service1.ServiceID, resp.Queries[1].ServiceId)
		assert.Equal(t, service1.ServiceName, resp.Queries[1].ServiceName)
	})

	t.Run("search all queries for service2", func(t *testing.T) {
		t.Parallel()

		ctx := grpc.NewContextWithServerTransportStream(t.Context(), &grpc_gateway.ServerTransportStream{})
		resp, err := svc.SearchQueries(ctx, &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service2.ServiceID},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, resp.Queries, 1)
		assert.Equal(t, "static-query-0", resp.Queries[0].QueryId)
		assert.Equal(t, service2.ServiceID, resp.Queries[0].ServiceId)
		assert.Equal(t, service2.ServiceName, resp.Queries[0].ServiceName)
	})

	t.Run("search all queries for both services", func(t *testing.T) {
		t.Parallel()

		ctx := grpc.NewContextWithServerTransportStream(t.Context(), &grpc_gateway.ServerTransportStream{})
		resp, err := svc.SearchQueries(ctx, &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service1.ServiceID, service2.ServiceID},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, resp.Queries, 3)
		assert.Equal(t, "static-query-1", resp.Queries[0].QueryId)
		assert.Equal(t, service1.ServiceID, resp.Queries[0].ServiceId)
		assert.Equal(t, service1.ServiceName, resp.Queries[0].ServiceName)

		assert.Equal(t, "static-query-0", resp.Queries[1].QueryId)
		assert.Equal(t, service1.ServiceID, resp.Queries[1].ServiceId)
		assert.Equal(t, service1.ServiceName, resp.Queries[1].ServiceName)

		assert.Equal(t, "static-query-0", resp.Queries[2].QueryId)
		assert.Equal(t, service2.ServiceID, resp.Queries[2].ServiceId)
		assert.Equal(t, service2.ServiceName, resp.Queries[2].ServiceName)
	})

	t.Run("search all queries for absent service", func(t *testing.T) {
		t.Parallel()

		ctx := grpc.NewContextWithServerTransportStream(t.Context(), &grpc_gateway.ServerTransportStream{})
		_, err := svc.SearchQueries(ctx, &rtav1.SearchQueriesRequest{
			ServiceIds: []string{"absent-service"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("one of the services is absent", func(t *testing.T) {
		t.Parallel()

		ctx := grpc.NewContextWithServerTransportStream(t.Context(), &grpc_gateway.ServerTransportStream{})
		_, err := svc.SearchQueries(ctx, &rtav1.SearchQueriesRequest{
			ServiceIds: []string{service1.ServiceID, "absent-service"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func getTestClient(t *testing.T) rtav1.CollectorServiceClient {
	t.Helper()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := rtav1.NewCollectorServiceClient(conn)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	return client
}

func TestService_Collect(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Create test data
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

	// Create a MongoDB Realtime Agent
	_, err = models.CreateAgent(db.Querier, models.RTAMongoDBAgentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "test-user",
		Password:   "test-pass",
		Disabled:   false,
		RTAOptions: models.RTAOptions{CollectInterval: new(2 * time.Second)},
	})
	require.NoError(t, err)

	registry := newMockAgentsRegistry(t)
	stateUpdater := newMockAgentsStateUpdater(t)
	store := NewStore()
	svc := NewService(db, registry, stateUpdater, store)
	// // Create in-memory listener for testing
	const bufSize = 1024 * 1024

	lis = bufconn.Listen(bufSize)

	// Create and start server
	grpcMetrics := interceptors.NewServerMetricsWithExtension(&interceptors.GRPCMetricsExtension{})
	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			interceptors.Stream(grpcMetrics.StreamServerInterceptor()),
			interceptors.StreamServiceEnabledInterceptor(),
			grpc_validator.StreamServerInterceptor(),
		)),
	)
	rtav1.RegisterCollectorServiceServer(s, svc)

	serveError := make(chan error)
	go func() {
		serveError <- s.Serve(lis)
	}()
	t.Cleanup(func() {
		s.GracefulStop()
		require.NoError(t, <-serveError)
	})

	time.Sleep(1 * time.Second) // Give server time to start

	client := getTestClient(t)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	streamCtx := agentv1.AddAgentConnectMetadata(ctx, &agentv1.AgentConnectMetadata{
		ID:      pmmAgent.AgentID,
		Version: "1.0.0",
	})

	stream, err := client.Collect(streamCtx)
	require.NoError(t, err)

	err = stream.Send(&rtav1.CollectRequest{
		Queries: getServiceQueries("service-1", "mongodb-1", 3),
	})
	require.NoError(t, err)

	// Close and receive response
	_, err = stream.CloseAndRecv()
	require.NoError(t, err)

	storeqQs := store.Get("service-1")

	queryIDs := make([]string, len(storeqQs))
	for i, q := range storeqQs {
		queryIDs[i] = q.QueryId
	}

	assert.Contains(t, queryIDs, "static-query-0")
	assert.Contains(t, queryIDs, "static-query-1")
	assert.Contains(t, queryIDs, "static-query-2")

	for i := range storeqQs {
		assert.Equal(t, "service-1", storeqQs[i].ServiceId)
		assert.Equal(t, "mongodb-1", storeqQs[i].ServiceName)
	}
}
