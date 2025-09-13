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
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestAgents(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		var (
			pmmAgentID                   string
			ms                           *inventoryv1.MySQLService
			ps                           *inventoryv1.PostgreSQLService
			valkey                       *inventoryv1.ValkeyService
			expectedNodeExporter         *inventoryv1.NodeExporter
			expectedMySQLdExporter       *inventoryv1.MySQLdExporter
			expectedMongoDBExporter      *inventoryv1.MongoDBExporter
			expectedQANMySQLSlowlogAgent *inventoryv1.QANMySQLSlowlogAgent
			expectedPostgresExporter     *inventoryv1.PostgresExporter
			expectedExternalExporter     *inventoryv1.ExternalExporter
			expectedValkeyExporter       *inventoryv1.ValkeyExporter
		)

		t.Run("AddPMMAgent", func(t *testing.T) {
			as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
			actualAgents, err := as.List(ctx, models.AgentFilters{})
			require.NoError(t, err)
			require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

			as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
			as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")
			as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
				mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
				mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
				mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
			as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
				mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
				mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
				mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
			as.vmdb.(*mockPrometheusService).On("RequestConfigurationUpdate").Return()

			pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
				RunsOnNodeId: models.PMMServerNodeID,
			})

			pmmAgentID = pmmAgent.GetPmmAgent().AgentId
			require.NoError(t, err)
			expectedPMMAgent := &inventoryv1.PMMAgent{
				AgentId:      "00000000-0000-4000-8000-000000000005",
				RunsOnNodeId: models.PMMServerNodeID,
				Connected:    true,
			}
			assert.Equal(t, expectedPMMAgent, pmmAgent.GetPmmAgent())
		})

		t.Run("AddNodeExporter", func(t *testing.T) {
			actualNodeExporter, err := as.AddNodeExporter(ctx, &inventoryv1.AddNodeExporterParams{
				PmmAgentId:   pmmAgentID,
				CustomLabels: map[string]string{"cluster": "test-cluster", "environment": "test-env"},
			})
			require.NoError(t, err)
			expectedNodeExporter := &inventoryv1.NodeExporter{
				AgentId:    "00000000-0000-4000-8000-000000000006",
				PmmAgentId: "00000000-0000-4000-8000-000000000005",
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
				CustomLabels: map[string]string{
					"cluster":     "test-cluster",
					"environment": "test-env",
				},
			}
			assert.Equal(t, expectedNodeExporter, actualNodeExporter.GetNodeExporter())
		})

		t.Run("ChangeNodeExporterAndRemoveCustomLabels", func(t *testing.T) {
			actualNodeExporter, err := as.ChangeNodeExporter(
				ctx,
				"00000000-0000-4000-8000-000000000006",
				&inventoryv1.ChangeNodeExporterParams{
					Enable: pointer.ToBool(false),
					// passing an empty map to remove custom labels
					CustomLabels: &common.StringMap{},
					MetricsResolutions: &common.MetricsResolutions{
						Hr: durationpb.New(10 * time.Second),
					},
				},
			)
			require.NoError(t, err)
			expectedNodeExporter = &inventoryv1.NodeExporter{
				AgentId:    "00000000-0000-4000-8000-000000000006",
				PmmAgentId: "00000000-0000-4000-8000-000000000005",
				Disabled:   true,
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
				MetricsResolutions: &common.MetricsResolutions{
					Hr: durationpb.New(10 * time.Second),
				},
			}
			assert.Equal(t, expectedNodeExporter, actualNodeExporter.GetNodeExporter())

			actualAgent, err := as.Get(ctx, "00000000-0000-4000-8000-000000000006")
			require.NoError(t, err)
			assert.Equal(t, expectedNodeExporter, actualAgent.(*inventoryv1.NodeExporter))
		})

		t.Run("AddMySQLExporter", func(t *testing.T) {
			var err error
			ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
			ms, err = ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
				PmmAgentId: pmmAgentID,
				ServiceId:  ms.ServiceId,
				Username:   "username",
			})
			require.NoError(t, err)
			expectedMySQLdExporter = &inventoryv1.MySQLdExporter{
				AgentId:    "00000000-0000-4000-8000-000000000008",
				PmmAgentId: "00000000-0000-4000-8000-000000000005",
				ServiceId:  ms.ServiceId,
				Username:   "username",
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			}
			assert.Equal(t, expectedMySQLdExporter, actualAgent.GetMysqldExporter())

			exporter, err := as.Get(ctx, "00000000-0000-4000-8000-000000000008")
			require.NoError(t, err)
			assert.Equal(t, expectedMySQLdExporter, exporter.(*inventoryv1.MySQLdExporter))
		})

		t.Run("AddMongoDBExporter", func(t *testing.T) {
			ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-mongo",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(27017),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddMongoDBExporter(ctx, &inventoryv1.AddMongoDBExporterParams{
				PmmAgentId:       pmmAgentID,
				ServiceId:        ms.ServiceId,
				Username:         "username",
				StatsCollections: nil,
				CollectionsLimit: 0, // no limit
			})
			require.NoError(t, err)
			expectedMongoDBExporter = &inventoryv1.MongoDBExporter{
				AgentId:    "00000000-0000-4000-8000-00000000000a",
				PmmAgentId: pmmAgentID,
				ServiceId:  ms.ServiceId,
				Username:   "username",
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			}
			assert.Equal(t, expectedMongoDBExporter, actualAgent.GetMongodbExporter())

			exporter, err := as.Get(ctx, "00000000-0000-4000-8000-00000000000a")
			require.NoError(t, err)
			assert.Equal(t, expectedMongoDBExporter, exporter.(*inventoryv1.MongoDBExporter))
		})

		t.Run("AddQANMySQLSlowlogAgent", func(t *testing.T) {
			actualAgent, err := as.AddQANMySQLSlowlogAgent(ctx, &inventoryv1.AddQANMySQLSlowlogAgentParams{
				PmmAgentId: pmmAgentID,
				ServiceId:  ms.ServiceId,
				Username:   "username",
			})
			require.NoError(t, err)
			expectedQANMySQLSlowlogAgent = &inventoryv1.QANMySQLSlowlogAgent{
				AgentId:    "00000000-0000-4000-8000-00000000000b",
				PmmAgentId: pmmAgentID,
				ServiceId:  ms.ServiceId,
				Username:   "username",
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			}
			assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgent.GetQanMysqlSlowlogAgent())

			exporter, err := as.Get(ctx, "00000000-0000-4000-8000-00000000000b")
			require.NoError(t, err)
			assert.Equal(t, expectedQANMySQLSlowlogAgent, exporter.(*inventoryv1.QANMySQLSlowlogAgent))
		})

		t.Run("AddPostgreSQLExporter", func(t *testing.T) {
			var err error
			ps, err = ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(5432),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddPostgresExporter(ctx, &inventoryv1.AddPostgresExporterParams{
				PmmAgentId: pmmAgentID,
				ServiceId:  ps.ServiceId,
				Username:   "username",
			})
			require.NoError(t, err)
			expectedPostgresExporter = &inventoryv1.PostgresExporter{
				AgentId:    "00000000-0000-4000-8000-00000000000d",
				PmmAgentId: pmmAgentID,
				ServiceId:  ps.ServiceId,
				Username:   "username",
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			}
			assert.Equal(t, expectedPostgresExporter, actualAgent.GetPostgresExporter())

			exporter, err := as.Get(ctx, "00000000-0000-4000-8000-00000000000d")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgresExporter, exporter.(*inventoryv1.PostgresExporter))
		})

		t.Run("AddExternalExporter", func(t *testing.T) {
			actualAgent, err := as.AddExternalExporter(ctx, &inventoryv1.AddExternalExporterParams{
				RunsOnNodeId: models.PMMServerNodeID,
				ServiceId:    ps.ServiceId,
				Username:     "username",
				ListenPort:   9222,
			})
			require.NoError(t, err)
			expectedExternalExporter = &inventoryv1.ExternalExporter{
				AgentId:      "00000000-0000-4000-8000-00000000000e",
				RunsOnNodeId: models.PMMServerNodeID,
				ServiceId:    ps.ServiceId,
				Username:     "username",
				Scheme:       "http",
				MetricsPath:  "/metrics",
				ListenPort:   9222,
			}
			assert.Equal(t, expectedExternalExporter, actualAgent.GetExternalExporter())
		})

		t.Run("AddValkeyExporter", func(t *testing.T) {
			var err error
			valkey, err = ss.AddValkey(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-valkey",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(6379),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddValkeyExporter(ctx, &inventoryv1.AddValkeyExporterParams{
				PmmAgentId: pmmAgentID,
				ServiceId:  valkey.ServiceId,
				Username:   "username",
				Password:   "password",
			})
			require.NoError(t, err)
			expectedValkeyExporter = &inventoryv1.ValkeyExporter{
				AgentId:    "00000000-0000-4000-8000-000000000010",
				PmmAgentId: pmmAgentID,
				ServiceId:  valkey.ServiceId,
				Username:   "username",
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			}
			assert.Equal(t, expectedValkeyExporter, actualAgent.GetValkeyExporter())

			exporter, err := as.Get(ctx, "00000000-0000-4000-8000-000000000010")
			require.NoError(t, err)
			assert.Equal(t, expectedValkeyExporter, exporter.(*inventoryv1.ValkeyExporter))
		})

		var actualAgents []inventoryv1.Agent
		t.Run("ListAllAgents", func(t *testing.T) {
			actualAgents, err := as.List(ctx, models.AgentFilters{})
			require.NoError(t, err)
			for i, a := range actualAgents {
				t.Logf("%d: %T %s", i, a, a)
			}
			require.Len(t, actualAgents, 12)

			// TODO: fix protobuf equality https://jira.percona.com/browse/PMM-6743
			assert.Equal(t, pmmAgentID, actualAgents[3].(*inventoryv1.PMMAgent).AgentId)
			assert.Equal(t, expectedNodeExporter.AgentId, actualAgents[4].(*inventoryv1.NodeExporter).AgentId)
			assert.Equal(t, expectedMySQLdExporter.AgentId, actualAgents[5].(*inventoryv1.MySQLdExporter).AgentId)
			assert.Equal(t, expectedMongoDBExporter.AgentId, actualAgents[6].(*inventoryv1.MongoDBExporter).AgentId)
			assert.Equal(t, expectedQANMySQLSlowlogAgent.AgentId, actualAgents[7].(*inventoryv1.QANMySQLSlowlogAgent).AgentId)
			assert.Equal(t, expectedPostgresExporter.AgentId, actualAgents[8].(*inventoryv1.PostgresExporter).AgentId)
			assert.Equal(t, expectedExternalExporter.AgentId, actualAgents[9].(*inventoryv1.ExternalExporter).AgentId)
		})

		t.Run("FilterByServiceID", func(t *testing.T) {
			actualAgents, err := as.List(ctx, models.AgentFilters{ServiceID: ms.ServiceId})
			require.NoError(t, err)
			require.Len(t, actualAgents, 2)
			assert.Equal(t, expectedMySQLdExporter, actualAgents[0])
			assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[1])
		})

		t.Run("FilterByPMMAgent", func(t *testing.T) {
			actualAgents, err := as.List(ctx, models.AgentFilters{PMMAgentID: pmmAgentID})
			require.NoError(t, err)
			require.Len(t, actualAgents, 6)
			assert.Equal(t, expectedNodeExporter, actualAgents[0])
			assert.Equal(t, expectedMySQLdExporter, actualAgents[1])
			assert.Equal(t, expectedMongoDBExporter, actualAgents[2])
			assert.Equal(t, expectedQANMySQLSlowlogAgent, actualAgents[3])
			assert.Equal(t, expectedPostgresExporter, actualAgents[4])
		})

		t.Run("FilterByNode", func(t *testing.T) {
			actualAgents, err := as.List(ctx, models.AgentFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualAgents, 2)
			assert.Equal(t, expectedNodeExporter, actualAgents[1])
		})

		t.Run("FilterByAgentType", func(t *testing.T) {
			agentType := models.ExternalExporterType
			actualAgents, err := as.List(ctx, models.AgentFilters{AgentType: &agentType})
			require.NoError(t, err)
			require.Len(t, actualAgents, 1)
			assert.Equal(t, expectedExternalExporter, actualAgents[0])
		})

		t.Run("FilterByMultipleFields", func(t *testing.T) {
			actualAgents, err := as.List(ctx, models.AgentFilters{PMMAgentID: pmmAgentID, NodeID: models.PMMServerNodeID})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `expected at most one param: pmm_agent_id, node_id or service_id`), err)
			assert.Nil(t, actualAgents)
		})

		t.Run("RemovePMMAgent", func(t *testing.T) {
			as.r.(*mockAgentsRegistry).On("Kick", ctx, "00000000-0000-4000-8000-000000000005").Return(true)
			err := as.Remove(ctx, "00000000-0000-4000-8000-000000000005", true)
			require.NoError(t, err)
			actualAgent, err := as.Get(ctx, "00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, "Agent with ID 00000000-0000-4000-8000-000000000005 not found."), err)
			assert.Nil(t, actualAgent)

			actualAgents, err = as.List(ctx, models.AgentFilters{})
			require.NoError(t, err)
			require.Len(t, actualAgents, 5) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN, External exporter
		})
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		actualNode, err := as.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Agent ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddPMMAgent", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(false)
		actualAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventoryv1.PMMAgent{
			AgentId:      "00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    false,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent.GetPmmAgent())

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000006").Return(true)
		actualAgent, err = as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent = &inventoryv1.PMMAgent{
			AgentId:      "00000000-0000-4000-8000-000000000006",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, actualAgent.GetPmmAgent())
	})

	t.Run("AddPmmAgentNotFound", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		_, err := as.AddNodeExporter(ctx, &inventoryv1.AddNodeExporterParams{
			PmmAgentId: "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID no-such-id not found.`), err)
	})

	t.Run("AddRDSExporter", func(t *testing.T) {
		_, as, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		node, err := ns.AddRemoteRDSNode(ctx, &inventoryv1.AddRemoteRDSNodeParams{
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		})
		require.NoError(t, err)
		expectedNode := &inventoryv1.RemoteRDSNode{
			NodeId:       "00000000-0000-4000-8000-000000000005",
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		}
		assert.Equal(t, expectedNode, node)

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		agent, err := as.AddRDSExporter(ctx, &inventoryv1.AddRDSExporterParams{
			PmmAgentId:   "pmm-server",
			NodeId:       node.NodeId,
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			AwsSecretKey: "EXAMPLE_SECRET_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
		})
		require.NoError(t, err)
		expectedAgent := &inventoryv1.RDSExporter{
			AgentId:      "00000000-0000-4000-8000-000000000006",
			PmmAgentId:   "pmm-server",
			NodeId:       "00000000-0000-4000-8000-000000000005",
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
			Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedAgent, agent.GetRdsExporter())
	})

	t.Run("AddExternalExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.vmdb.(*mockPrometheusService).On("RequestConfigurationUpdate").Return()

		service, err := ss.AddExternalService(ctx, &models.AddDBMSServiceParams{
			ServiceName:   "External service",
			NodeID:        models.PMMServerNodeID,
			ExternalGroup: "external",
		})
		require.NoError(t, err)
		require.NotNil(t, service)

		agent, err := as.AddExternalExporter(ctx, &inventoryv1.AddExternalExporterParams{
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    service.ServiceId,
			Username:     "username",
			ListenPort:   12345,
		})
		require.NoError(t, err)
		expectedExternalExporter := &inventoryv1.ExternalExporter{
			AgentId:      "00000000-0000-4000-8000-000000000006",
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    service.ServiceId,
			Username:     "username",
			Scheme:       "http",
			MetricsPath:  "/metrics",
			ListenPort:   12345,
		}
		assert.Equal(t, expectedExternalExporter, agent.GetExternalExporter())

		actualAgent, err := as.Get(ctx, "00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalExporter, actualAgent)
	})

	t.Run("AddServiceNotFound", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		_, err = as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
			PmmAgentId: pmmAgent.GetPmmAgent().AgentId,
			ServiceId:  "no-such-id",
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		err := as.Remove(ctx, "no-such-id", false)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID no-such-id not found.`), err)
	})

	t.Run("PushMetricsMongodbExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventoryv1.PMMAgent{
			AgentId:      "00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent.GetPmmAgent())
		ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
		})
		require.NoError(t, err)
		actualAgent, err := as.AddMongoDBExporter(ctx, &inventoryv1.AddMongoDBExporterParams{
			PmmAgentId:  pmmAgent.GetPmmAgent().AgentId,
			ServiceId:   ms.ServiceId,
			Username:    "username",
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedMongoDBExporter := &inventoryv1.MongoDBExporter{
			AgentId:            "00000000-0000-4000-8000-000000000007",
			PmmAgentId:         pmmAgent.GetPmmAgent().AgentId,
			ServiceId:          ms.ServiceId,
			Username:           "username",
			PushMetricsEnabled: true,
			Status:             inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedMongoDBExporter, actualAgent.GetMongodbExporter())
	})

	t.Run("PushMetricsNodeExporter", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventoryv1.PMMAgent{
			AgentId:      "00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent.GetPmmAgent())

		actualNodeExporter, err := as.AddNodeExporter(ctx, &inventoryv1.AddNodeExporterParams{
			PmmAgentId:  pmmAgent.GetPmmAgent().AgentId,
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedNodeExporter := &inventoryv1.NodeExporter{
			AgentId:            "00000000-0000-4000-8000-000000000006",
			PmmAgentId:         "00000000-0000-4000-8000-000000000005",
			PushMetricsEnabled: true,
			Status:             inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedNodeExporter, actualNodeExporter.GetNodeExporter())
	})

	t.Run("PushMetricsPostgresSQLExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventoryv1.PMMAgent{
			AgentId:      "00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent.GetPmmAgent())
		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(5432),
		})
		require.NoError(t, err)

		actualAgent, err := as.AddPostgresExporter(ctx, &inventoryv1.AddPostgresExporterParams{
			PmmAgentId:  pmmAgent.GetPmmAgent().AgentId,
			ServiceId:   ps.ServiceId,
			Username:    "username",
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedPostgresExporter := &inventoryv1.PostgresExporter{
			AgentId:            "00000000-0000-4000-8000-000000000007",
			PmmAgentId:         pmmAgent.GetPmmAgent().AgentId,
			ServiceId:          ps.ServiceId,
			Username:           "username",
			PushMetricsEnabled: true,
			Status:             inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedPostgresExporter, actualAgent.GetPostgresExporter())
	})

	t.Run("PushMetricsMySQLExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedPMMAgent := &inventoryv1.PMMAgent{
			AgentId:      "00000000-0000-4000-8000-000000000005",
			RunsOnNodeId: models.PMMServerNodeID,
			Connected:    true,
		}
		assert.Equal(t, expectedPMMAgent, pmmAgent.GetPmmAgent())

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		s, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		actualAgent, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
			PmmAgentId:  pmmAgent.GetPmmAgent().AgentId,
			ServiceId:   s.ServiceId,
			Username:    "username",
			PushMetrics: true,
		})
		require.NoError(t, err)
		expectedMySQLdExporter := &inventoryv1.MySQLdExporter{
			AgentId:            "00000000-0000-4000-8000-000000000007",
			PmmAgentId:         "00000000-0000-4000-8000-000000000005",
			ServiceId:          s.ServiceId,
			Username:           "username",
			PushMetricsEnabled: true,
			Status:             inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedMySQLdExporter, actualAgent.GetMysqldExporter())
	})

	t.Run("PushMetricsRdsExporter", func(t *testing.T) {
		_, as, ns, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		node, err := ns.AddRemoteRDSNode(ctx, &inventoryv1.AddRemoteRDSNodeParams{
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		})
		require.NoError(t, err)
		expectedNode := &inventoryv1.RemoteRDSNode{
			NodeId:       "00000000-0000-4000-8000-000000000005",
			NodeName:     "rds1",
			Address:      "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		}
		assert.Equal(t, expectedNode, node)

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		agent, err := as.AddRDSExporter(ctx, &inventoryv1.AddRDSExporterParams{
			PmmAgentId:   "pmm-server",
			NodeId:       node.NodeId,
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			AwsSecretKey: "EXAMPLE_SECRET_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
			PushMetrics:  true,
		})
		require.NoError(t, err)
		expectedAgent := &inventoryv1.RDSExporter{
			AgentId:            "00000000-0000-4000-8000-000000000006",
			PmmAgentId:         "pmm-server",
			NodeId:             "00000000-0000-4000-8000-000000000005",
			AwsAccessKey:       "EXAMPLE_ACCESS_KEY",
			CustomLabels:       map[string]string{"baz": "qux"},
			PushMetricsEnabled: true,
			Status:             inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedAgent, agent.GetRdsExporter())
	})

	t.Run("PushMetricsExternalExporter", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		service, err := ss.AddExternalService(ctx, &models.AddDBMSServiceParams{
			ServiceName:   "External service",
			NodeID:        models.PMMServerNodeID,
			ExternalGroup: "external",
		})
		require.NoError(t, err)
		require.NotNil(t, service)

		agent, err := as.AddExternalExporter(ctx, &inventoryv1.AddExternalExporterParams{
			RunsOnNodeId: models.PMMServerNodeID,
			ServiceId:    service.ServiceId,
			Username:     "username",
			ListenPort:   12345,
			PushMetrics:  true,
		})
		require.NoError(t, err)
		expectedExternalExporter := &inventoryv1.ExternalExporter{
			AgentId:            "00000000-0000-4000-8000-000000000006",
			RunsOnNodeId:       models.PMMServerNodeID,
			ServiceId:          service.ServiceId,
			Username:           "username",
			Scheme:             "http",
			MetricsPath:        "/metrics",
			ListenPort:         12345,
			PushMetricsEnabled: true,
		}
		assert.Equal(t, expectedExternalExporter, agent.GetExternalExporter())

		actualAgent, err := as.Get(ctx, "00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalExporter, actualAgent)
	})
}
