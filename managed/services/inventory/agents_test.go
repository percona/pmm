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

package inventory

import (
	"reflect"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/common"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestAgents(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		// FIXME split this test into several smaller

		ss, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "/agent_id/00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.vmdb.(*mockPrometheusService).On("RequestConfigurationUpdate").Return()

		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent)

		actualNodeExporter, err := as.AddNodeExporter(ctx, &inventorypb.AddNodeExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
		})
		require.NoError(t, err)
		expectedNodeExporter := &inventorypb.NodeExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000005",
			Status:     inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter)

		actualNodeExporter, err = as.ChangeNodeExporter(ctx, &inventorypb.ChangeNodeExporterRequest{
			AgentId: "/agent_id/00000000-0000-4000-8000-000000000006",
			Common: &inventorypb.ChangeCommonAgentParams{
				Disable: true,
				MetricsResolutions: &common.MetricsResolutions{
					Hr: durationpb.New(10 * time.Second),
				},
			},
		})
		require.NoError(t, err)
		expectedNodeExporter = &inventorypb.NodeExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000005",
			Disabled:   true,
			Status:     inventorypb.AgentStatus_UNKNOWN,
			MetricsResolutions: &common.MetricsResolutions{
				Hr: durationpb.New(10 * time.Second),
			},
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter)

		actualAgent, err := as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedNodeExporter, actualAgent)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		s, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		actualAgent, _, err = as.AddMySQLdExporter(ctx, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedMySQLdExporter := &inventorypb.MySQLdExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-000000000008",
			PmmAgentId: "/agent_id/00000000-0000-4000-8000-000000000005",
			ServiceId:  s.ServiceId,
			Username:   "username",
			Status:     inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedMySQLdExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000008")
		require.NoError(t, err)
		assert.Equal(t, expectedMySQLdExporter, actualAgent)

		ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
		})
		require.NoError(t, err)

		actualAgent, err = as.AddMongoDBExporter(ctx, &inventorypb.AddMongoDBExporterRequest{
			PmmAgentId:       pmmAgent.AgentId,
			ServiceId:        ms.ServiceId,
			Username:         "username",
			StatsCollections: nil,
			CollectionsLimit: 0, // no limit
		})
		require.NoError(t, err)
		expectedMongoDBExporter := &inventorypb.MongoDBExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-00000000000a",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ms.ServiceId,
			Username:   "username",
			Status:     inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedMongoDBExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-00000000000a")
		require.NoError(t, err)
		assert.Equal(t, expectedMongoDBExporter, actualAgent)

		actualAgent, err = as.AddQANMySQLSlowlogAgent(ctx, &inventorypb.AddQANMySQLSlowlogAgentRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedQANMySQLSlowlogAgent := &inventorypb.QANMySQLSlowlogAgent{
			AgentId:    "/agent_id/00000000-0000-4000-8000-00000000000b",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  s.ServiceId,
			Username:   "username",
			Status:     inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-00000000000b")
		require.NoError(t, err)
		assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgent)

		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(5432),
		})
		require.NoError(t, err)

		actualAgent, err = as.AddPostgresExporter(ctx, &inventorypb.AddPostgresExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ps.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
		expectedPostgresExporter := &inventorypb.PostgresExporter{
			AgentId:    "/agent_id/00000000-0000-4000-8000-00000000000d",
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  ps.ServiceId,
			Username:   "username",
			Status:     inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedPostgresExporter, actualAgent)

		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-00000000000d")
		require.NoError(t, err)
		assert.Equal(t, expectedPostgresExporter, actualAgent)

		actualAgent, err = as.AddExternalExporter(ctx, &inventorypb.AddExternalExporterRequest{
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    ps.ServiceId,
			Username:     "username",
			ListenPort:   9222,
		})
		require.NoError(t, err)
		expectedExternalExporter := &inventorypb.ExternalExporter{
			AgentId:      "/agent_id/00000000-0000-4000-8000-00000000000e",
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    ps.ServiceId,
			Username:     "username",
			Scheme:       "http",
			MetricsPath:  "/metrics",
			ListenPort:   9222,
		}
		assert.Equal(t, expectedExternalExporter, actualAgent)

		t.Run("ListAllAgents", func(t *testing.T) {
			actualAgents, err = as.List(ctx, models.AgentFilters{})
			require.NoError(t, err)
			for i, a := range actualAgents {
				t.Logf("%d: %T %s", i, a, a)
			}
			require.Len(t, actualAgents, 11)

			// TODO: fix protobuf equality https://jira.percona.com/browse/PMM-6743
			assert.Equal(t, pmmAgent.AgentId, actualAgents[3].(*inventorypb.PMMAgent).AgentId)
			assert.Equal(t, expectedNodeExporter.AgentId, actualAgents[4].(*inventorypb.NodeExporter).AgentId)
			assert.Equal(t, expectedMySQLdExporter.AgentId, actualAgents[5].(*inventorypb.MySQLdExporter).AgentId)
			assert.Equal(t, expectedMongoDBExporter.AgentId, actualAgents[6].(*inventorypb.MongoDBExporter).AgentId)
			assert.Equal(t, expectedQANMySQLSlowlogAgent.AgentId, actualAgents[7].(*inventorypb.QANMySQLSlowlogAgent).AgentId)
			assert.Equal(t, expectedPostgresExporter.AgentId, actualAgents[8].(*inventorypb.PostgresExporter).AgentId)
			assert.Equal(t, expectedExternalExporter.AgentId, actualAgents[9].(*inventorypb.ExternalExporter).AgentId)
		})

		t.Run("FilterByServiceID", func(t *testing.T) {
			actualAgents, err = as.List(ctx, models.AgentFilters{ServiceID: s.ServiceId})
			require.NoError(t, err)
			require.Len(t, actualAgents, 2)
			assert.Equal(t, expectedMySQLdExporter, actualAgents[0])
			assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[1])
		})

		t.Run("FilterByPMMAgent", func(t *testing.T) {
			actualAgents, err = as.List(ctx, models.AgentFilters{PMMAgentID: pmmAgent.AgentId})
			require.NoError(t, err)
			require.Len(t, actualAgents, 5)
			assert.Equal(t, expectedNodeExporter, actualAgents[0])
			assert.Equal(t, expectedMySQLdExporter, actualAgents[1])
			assert.Equal(t, expectedMongoDBExporter, actualAgents[2])
			assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[3])
			assert.Equal(t, expectedPostgresExporter, actualAgents[4])
		})

		t.Run("FilterByNode", func(t *testing.T) {
			actualAgents, err = as.List(ctx, models.AgentFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualAgents, 2)
			assert.Equal(t, expectedNodeExporter, actualAgents[1])
		})

		t.Run("FilterByAgentType", func(t *testing.T) {
			agentType := models.ExternalExporterType
			actualAgents, err = as.List(ctx, models.AgentFilters{AgentType: &agentType})
			require.NoError(t, err)
			require.Len(t, actualAgents, 1)
			assert.Equal(t, expectedExternalExporter, actualAgents[0])
		})

		t.Run("FilterByMultipleFields", func(t *testing.T) {
			actualAgents, err = as.List(ctx, models.AgentFilters{PMMAgentID: pmmAgent.AgentId, NodeID: models.PMMServerNodeID})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `expected at most one param: pmm_agent_id, node_id or service_id`), err)
			assert.Nil(t, actualAgents)
		})

		as.r.(*mockAgentsRegistry).On("Kick", ctx, "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		err = as.Remove(ctx, "/agent_id/00000000-0000-4000-8000-000000000005", true)
		require.NoError(t, err)
		actualAgent, err = as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualAgent)

		actualAgents, err = as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 5) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN, External exporter
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualNode, err := as.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Agent ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddPMMAgent", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(false)
		actualAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    false,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent)

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000006").Return(true)
		actualAgent, err = as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent = &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000006",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent)
	})

	t.Run("AddPmmAgentNotFound", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		_, err := as.AddNodeExporter(ctx, &inventorypb.AddNodeExporterRequest{
			PmmAgentId: "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})

	t.Run("AddRDSExporter", func(t *testing.T) {
		_, as, ns, teardown, ctx, _ := setup(t)
		defer teardown(t)

		node, err := ns.AddRemoteRDSNode(ctx, &inventorypb.AddRemoteRDSNodeRequest{
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		})
		require.NoError(t, err)
		expectedNode := &inventorypb.RemoteRDSNode{
			NodeId:       "/node_id/00000000-0000-4000-8000-000000000005",
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		}
		assert.Equal(t, expectedNode, node)

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		agent, err := as.AddRDSExporter(ctx, &inventorypb.AddRDSExporterRequest{
			PmmAgentId:   "pmm-server",
			NodeId:       node.NodeId,
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			AwsSecretKey: "EXAMPLE_SECRET_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
		})
		require.NoError(t, err)
		expectedAgent := &inventorypb.RDSExporter{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId:   "pmm-server",
			NodeId:       "/node_id/00000000-0000-4000-8000-000000000005",
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
			Status:       inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedAgent, agent)
	})

	t.Run("AddExternalExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.vmdb.(*mockPrometheusService).On("RequestConfigurationUpdate").Return()

		service, err := ss.AddExternalService(ctx, &models.AddDBMSServiceParams{
			ServiceName:   "External service",
			NodeID:        models.PMMServerNodeID,
			ExternalGroup: "external",
		})
		require.NoError(t, err)
		require.NotNil(t, service)

		agent, err := as.AddExternalExporter(ctx, &inventorypb.AddExternalExporterRequest{
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    service.ServiceId,
			Username:     "username",
			ListenPort:   12345,
		})
		require.NoError(t, err)
		expectedExternalExporter := &inventorypb.ExternalExporter{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000006",
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    service.ServiceId,
			Username:     "username",
			Scheme:       "http",
			MetricsPath:  "/metrics",
			ListenPort:   12345,
		}
		assert.Equal(t, expectedExternalExporter, agent)

		actualAgent, err := as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalExporter, actualAgent)
	})

	t.Run("AddServiceNotFound", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		_, _, err = as.AddMySQLdExporter(ctx, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId: pmmAgent.AgentId,
			ServiceId:  "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		err := as.Remove(ctx, "no-such-id", false)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "no-such-id" not found.`), err)
	})
	t.Run("PushMetricsMongodbExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "/agent_id/00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent)
		ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
		})
		require.NoError(t, err)
		actualAgent, err := as.AddMongoDBExporter(ctx, &inventorypb.AddMongoDBExporterRequest{
			PmmAgentId:  pmmAgent.AgentId,
			ServiceId:   ms.ServiceId,
			Username:    "username",
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedMongoDBExporter := &inventorypb.MongoDBExporter{
			AgentId:            "/agent_id/00000000-0000-4000-8000-000000000007",
			PmmAgentId:         pmmAgent.AgentId,
			ServiceId:          ms.ServiceId,
			Username:           "username",
			PushMetricsEnabled: true,
			Status:             inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedMongoDBExporter, actualAgent)
	})
	t.Run("PushMetricsNodeExporter", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "/agent_id/00000000-0000-4000-8000-000000000005")

		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent)

		actualNodeExporter, err := as.AddNodeExporter(ctx, &inventorypb.AddNodeExporterRequest{
			PmmAgentId:  pmmAgent.AgentId,
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedNodeExporter := &inventorypb.NodeExporter{
			AgentId:            "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId:         "/agent_id/00000000-0000-4000-8000-000000000005",
			PushMetricsEnabled: true,
			Status:             inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter)
	})
	t.Run("PushMetricsPostgresSQLExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "/agent_id/00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent)
		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(5432),
		})
		require.NoError(t, err)

		actualAgent, err := as.AddPostgresExporter(ctx, &inventorypb.AddPostgresExporterRequest{
			PmmAgentId:  pmmAgent.AgentId,
			ServiceId:   ps.ServiceId,
			Username:    "username",
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedPostgresExporter := &inventorypb.PostgresExporter{
			AgentId:            "/agent_id/00000000-0000-4000-8000-000000000007",
			PmmAgentId:         pmmAgent.AgentId,
			ServiceId:          ps.ServiceId,
			Username:           "username",
			PushMetricsEnabled: true,
			Status:             inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedPostgresExporter, actualAgent)
	})
	t.Run("PushMetricsMySQLExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "/agent_id/00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventorypb.AddPMMAgentRequest{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventorypb.PMMAgent{
			AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		s, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		actualAgent, _, err := as.AddMySQLdExporter(ctx, &inventorypb.AddMySQLdExporterRequest{
			PmmAgentId:  pmmAgent.AgentId,
			ServiceId:   s.ServiceId,
			Username:    "username",
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedMySQLdExporter := &inventorypb.MySQLdExporter{
			AgentId:            "/agent_id/00000000-0000-4000-8000-000000000007",
			PmmAgentId:         "/agent_id/00000000-0000-4000-8000-000000000005",
			ServiceId:          s.ServiceId,
			Username:           "username",
			PushMetricsEnabled: true,
			Status:             inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedMySQLdExporter, actualAgent)
	})
	t.Run("PushMetricsRdsExporter", func(t *testing.T) {
		_, as, ns, teardown, ctx, _ := setup(t)
		defer teardown(t)

		node, err := ns.AddRemoteRDSNode(ctx, &inventorypb.AddRemoteRDSNodeRequest{
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		})
		require.NoError(t, err)
		expectedNode := &inventorypb.RemoteRDSNode{
			NodeId:       "/node_id/00000000-0000-4000-8000-000000000005",
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		}
		assert.Equal(t, expectedNode, node)

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		agent, err := as.AddRDSExporter(ctx, &inventorypb.AddRDSExporterRequest{
			PmmAgentId:   "pmm-server",
			NodeId:       node.NodeId,
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			AwsSecretKey: "EXAMPLE_SECRET_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
			PushMetrics:  true,
		})
		require.NoError(t, err)
		expectedAgent := &inventorypb.RDSExporter{
			AgentId:            "/agent_id/00000000-0000-4000-8000-000000000006",
			PmmAgentId:         "pmm-server",
			NodeId:             "/node_id/00000000-0000-4000-8000-000000000005",
			AwsAccessKey:       "EXAMPLE_ACCESS_KEY",
			CustomLabels:       map[string]string{"baz": "qux"},
			PushMetricsEnabled: true,
			Status:             inventorypb.AgentStatus_UNKNOWN,
		}
		assert.Equal(t, expectedAgent, agent)
	})
	t.Run("PushMetricsExternalExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		defer teardown(t)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		service, err := ss.AddExternalService(ctx, &models.AddDBMSServiceParams{
			ServiceName:   "External service",
			NodeID:        models.PMMServerNodeID,
			ExternalGroup: "external",
		})
		require.NoError(t, err)
		require.NotNil(t, service)

		agent, err := as.AddExternalExporter(ctx, &inventorypb.AddExternalExporterRequest{
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    service.ServiceId,
			Username:     "username",
			ListenPort:   12345,
			PushMetrics:  true,
		})
		require.NoError(t, err)
		expectedExternalExporter := &inventorypb.ExternalExporter{
			AgentId:            "/agent_id/00000000-0000-4000-8000-000000000006",
			RunsOnNodeId:       models.PMMServerNodeID,
			ServiceId:          service.ServiceId,
			Username:           "username",
			Scheme:             "http",
			MetricsPath:        "/metrics",
			ListenPort:         12345,
			PushMetricsEnabled: true,
		}
		assert.Equal(t, expectedExternalExporter, agent)

		actualAgent, err := as.Get(ctx, "/agent_id/00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalExporter, actualAgent)
	})
}
