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
	"context"
	"reflect"
	"testing"
	"time"

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
	"github.com/percona/pmm/managed/utils/env"
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
				mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
				mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
				mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)
			as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
				mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
				mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
				mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)
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
					Enable: new(false),
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
				Status:     inventoryv1.AgentStatus_AGENT_STATUS_DONE,
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
				Address:     new("127.0.0.1"),
				Port:        new(uint16(3306)),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
				PmmAgentId:        pmmAgentID,
				ServiceId:         ms.ServiceId,
				Username:          "username",
				ConnectionTimeout: durationpb.New(11 * time.Second),
			})
			require.NoError(t, err)
			expectedMySQLdExporter = &inventoryv1.MySQLdExporter{
				AgentId:           "00000000-0000-4000-8000-000000000008",
				PmmAgentId:        "00000000-0000-4000-8000-000000000005",
				ServiceId:         ms.ServiceId,
				Username:          "username",
				Status:            inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
				ConnectionTimeout: durationpb.New(11 * time.Second),
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
				Address:     new("127.0.0.1"),
				Port:        new(uint16(27017)),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddMongoDBExporter(ctx, &inventoryv1.AddMongoDBExporterParams{
				PmmAgentId:                     pmmAgentID,
				ServiceId:                      ms.ServiceId,
				Username:                       "username",
				StatsCollections:               nil,
				CollectionsLimit:               0, // no limit
				EnableDiagnosticDataHistograms: true,
			})
			require.NoError(t, err)
			expectedMongoDBExporter = &inventoryv1.MongoDBExporter{
				AgentId:                        "00000000-0000-4000-8000-00000000000a",
				PmmAgentId:                     pmmAgentID,
				ServiceId:                      ms.ServiceId,
				Username:                       "username",
				Status:                         inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
				EnableDiagnosticDataHistograms: true,
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
				Address:     new("127.0.0.1"),
				Port:        new(uint16(5432)),
			})
			require.NoError(t, err)

			actualAgent, err := as.AddPostgresExporter(ctx, &inventoryv1.AddPostgresExporterParams{
				PmmAgentId:        pmmAgentID,
				ServiceId:         ps.ServiceId,
				Username:          "username",
				ConnectionTimeout: durationpb.New(13 * time.Second),
			})
			require.NoError(t, err)
			expectedPostgresExporter = &inventoryv1.PostgresExporter{
				AgentId:           "00000000-0000-4000-8000-00000000000d",
				PmmAgentId:        pmmAgentID,
				ServiceId:         ps.ServiceId,
				Username:          "username",
				Status:            inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
				ConnectionTimeout: durationpb.New(13 * time.Second),
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
				Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			}
			assert.Equal(t, expectedExternalExporter, actualAgent.GetExternalExporter())
		})

		t.Run("AddValkeyExporter", func(t *testing.T) {
			var err error
			valkey, err = ss.AddValkey(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-valkey",
				NodeID:      models.PMMServerNodeID,
				Address:     new("127.0.0.1"),
				Port:        new(uint16(6379)),
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
			actualAgents, err := as.List(ctx, models.AgentFilters{AgentType: new(models.ExternalExporterType)})
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

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		changedAgent, err := as.ChangeRDSExporter(ctx, "00000000-0000-4000-8000-000000000006", &inventoryv1.ChangeRDSExporterParams{})
		require.NoError(t, err)
		expectedAgent = &inventoryv1.RDSExporter{
			AgentId:      "00000000-0000-4000-8000-000000000006",
			PmmAgentId:   "pmm-server",
			NodeId:       "00000000-0000-4000-8000-000000000005",
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
			Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedAgent, changedAgent.GetRdsExporter())

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		changedAgent, err = as.ChangeRDSExporter(ctx, "00000000-0000-4000-8000-000000000006", &inventoryv1.ChangeRDSExporterParams{})
		require.NoError(t, err)
		expectedAgent = &inventoryv1.RDSExporter{
			AgentId:      "00000000-0000-4000-8000-000000000006",
			PmmAgentId:   "pmm-server",
			NodeId:       "00000000-0000-4000-8000-000000000005",
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			CustomLabels: map[string]string{"baz": "qux"},
			Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedAgent, changedAgent.GetRdsExporter())
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
			Status:       inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
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
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)

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
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
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
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)

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
			Address:     new("127.0.0.1"),
			Port:        new(uint16(5432)),
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
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)

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
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
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
			Status:             inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
		}
		assert.Equal(t, expectedExternalExporter, agent.GetExternalExporter())

		actualAgent, err := as.Get(ctx, "00000000-0000-4000-8000-000000000006")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalExporter, actualAgent)
	})

	t.Run("AddRTAMongoDBAgent", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name())).Return(nil)

		// Add PMM Agent
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

		// Add MongoDB Service
		ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo-rta",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
		})
		require.NoError(t, err)

		// Add RTA MongoDB Agent
		actualAgent, err := as.AddRTAMongoDBAgent(ctx, &inventoryv1.AddRTAMongoDBAgentParams{
			PmmAgentId: pmmAgent.GetPmmAgent().AgentId,
			ServiceId:  ms.ServiceId,
			Username:   "username",
			RtaOptions: &inventoryv1.RTAOptions{
				CollectInterval: durationpb.New(3 * time.Second),
			},
		})
		require.NoError(t, err)

		expectedRTAMongoDBAgent := &inventoryv1.RTAMongoDBAgent{
			AgentId:    "00000000-0000-4000-8000-000000000007",
			PmmAgentId: pmmAgent.GetPmmAgent().AgentId,
			ServiceId:  ms.ServiceId,
			Username:   "username",
			Status:     inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN,
			RtaOptions: &inventoryv1.RTAOptions{
				CollectInterval: durationpb.New(3 * time.Second),
			},
		}
		assert.Equal(t, expectedRTAMongoDBAgent, actualAgent.GetRtaMongodbAgent())
	})
}

func TestChangeQANPostgreSQLPgStatementsAgentWithEnvVar(t *testing.T) {
	// The QAN agent of PMM's internal PostgreSQL, created by the test fixtures. It runs under
	// PMM Server's own pmm-agent and is disabled, because PMM_ENABLE_INTERNAL_PG_QAN is not set
	// while the fixtures are created.
	const internalPgQANAgentID = "00000000-0000-4000-8000-000000000004"

	t.Run("FailWhenEnvVarSet", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		// Set the environment variable
		t.Setenv(env.EnableInternalPgQAN, "true")

		// Try to disable the internal PostgreSQL QAN agent while the environment variable enables it
		_, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			Enable: new(false),
		})

		// Expect a FailedPrecondition error
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "QAN for PMM's internal PostgreSQL server is set to true via an environment variable."), err)
	})

	t.Run("KeepAgentIntactWhenRejected", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		t.Setenv(env.EnableInternalPgQAN, "false")

		_, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			Enable:   new(true),
			LogLevel: inventoryv1.LogLevel_LOG_LEVEL_DEBUG.Enum(),
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "QAN for PMM's internal PostgreSQL server is set to false via an environment variable."), err)

		// A rejected request must not leave any of the requested changes behind.
		agent, err := models.FindAgentByID(as.db.Querier, internalPgQANAgentID)
		require.NoError(t, err)
		assert.True(t, agent.Disabled)
		assert.Nil(t, agent.LogLevel)
	})

	t.Run("FailWhenRequestedThroughParamsOfAnotherAgentType", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		t.Setenv(env.EnableInternalPgQAN, "true")

		// The inventory API picks the Change*Agent method from the request payload and not from
		// the type of the agent being changed, so any of them can be pointed at the internal QAN
		// agent. The guard must hold no matter which one was called.
		_, err := as.ChangeQANPostgreSQLPgStatMonitorAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentParams{
			Enable:   new(false),
			LogLevel: inventoryv1.LogLevel_LOG_LEVEL_DEBUG.Enum(),
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "QAN for PMM's internal PostgreSQL server is set to true via an environment variable."), err)

		agent, err := models.FindAgentByID(as.db.Querier, internalPgQANAgentID)
		require.NoError(t, err)
		assert.True(t, agent.Disabled)
		assert.Nil(t, agent.LogLevel)
	})

	t.Run("SucceedForParametersUnrelatedToEnvVar", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		t.Setenv(env.EnableInternalPgQAN, "true")
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, models.PMMServerAgentID)

		// The environment variable pins only the enabled state, everything else stays changeable.
		agent, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			LogLevel:       inventoryv1.LogLevel_LOG_LEVEL_DEBUG.Enum(),
			MaxQueryLength: new(int32(2048)),
		})

		require.NoError(t, err)
		assert.Equal(t, inventoryv1.LogLevel_LOG_LEVEL_DEBUG, agent.GetQanPostgresqlPgstatementsAgent().LogLevel)
		assert.Equal(t, int32(2048), agent.GetQanPostgresqlPgstatementsAgent().MaxQueryLength)
		assert.True(t, agent.GetQanPostgresqlPgstatementsAgent().Disabled)
	})

	t.Run("SucceedWhenRequestedStateMatchesEnvVar", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		t.Setenv(env.EnableInternalPgQAN, "true")
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, models.PMMServerAgentID)

		agent, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			Enable: new(true),
		})

		require.NoError(t, err)
		assert.False(t, agent.GetQanPostgresqlPgstatementsAgent().Disabled)
	})

	t.Run("SucceedForAgentOutsideOfPMMServer", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		// One state update for adding the agent, another one for changing it.
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005").Times(2)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(5432)),
		})
		require.NoError(t, err)

		added, err := as.AddQANPostgreSQLPgStatementsAgent(ctx, &inventoryv1.AddQANPostgreSQLPgStatementsAgentParams{
			PmmAgentId:          pmmAgent.GetPmmAgent().AgentId,
			ServiceId:           ps.ServiceId,
			Username:            "username",
			SkipConnectionCheck: true,
		})
		require.NoError(t, err)

		t.Setenv(env.EnableInternalPgQAN, "true")

		// The environment variable only covers PMM's internal PostgreSQL, agents monitoring
		// other PostgreSQL services are not affected by it.
		agent, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, added.GetQanPostgresqlPgstatementsAgent().AgentId,
			&inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
				Enable: new(false),
			})

		require.NoError(t, err)
		assert.True(t, agent.GetQanPostgresqlPgstatementsAgent().Disabled)
	})

	t.Run("SucceedForOtherAgentTypesOfPMMServer", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		// The postgres_exporter of PMM's internal PostgreSQL, running under the same pmm-agent
		// as the QAN agent that the environment variable pins.
		pgExporters, err := models.FindAgents(as.db.Querier, models.AgentFilters{
			PMMAgentID: models.PMMServerAgentID,
			AgentType:  new(models.PostgresExporterType),
		})
		require.NoError(t, err)
		require.Len(t, pgExporters, 1)

		t.Setenv(env.EnableInternalPgQAN, "true")
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, models.PMMServerAgentID)

		// The environment variable covers QAN only, the other agents of PMM Server stay changeable.
		agent, err := as.ChangePostgresExporter(ctx, pgExporters[0].AgentID, &inventoryv1.ChangePostgresExporterParams{
			Enable: new(false),
		})

		require.NoError(t, err)
		assert.True(t, agent.GetPostgresExporter().Disabled)
	})

	t.Run("SucceedWhenEnvVarValueIsNotABool", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		// A value that is not a boolean is reported by the environment variable parser during
		// startup and pins nothing, exactly like an unset variable.
		t.Setenv(env.EnableInternalPgQAN, "not-a-bool")
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, models.PMMServerAgentID)

		agent, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			Enable: new(true),
		})

		require.NoError(t, err)
		assert.False(t, agent.GetQanPostgresqlPgstatementsAgent().Disabled)
	})

	t.Run("SucceedWhenEnvVarNotSet", func(t *testing.T) {
		_, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		// Ensure the environment variable is not set
		// (It shouldn't be set by default, but we explicitly unset it to be safe)
		t.Setenv(env.EnableInternalPgQAN, "")

		// Mock the state update request
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")

		// Try to change the internal PostgreSQL QAN agent
		agent, err := as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			Enable: new(false),
		})

		// Should succeed
		require.NoError(t, err)
		assert.True(t, agent.GetQanPostgresqlPgstatementsAgent().Disabled)

		// Change it back to enabled
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")
		agent, err = as.ChangeQANPostgreSQLPgStatementsAgent(ctx, internalPgQANAgentID, &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams{
			Enable: new(true),
		})

		// Should succeed
		require.NoError(t, err)
		assert.False(t, agent.GetQanPostgresqlPgstatementsAgent().Disabled)
	})
}

func TestChangeRTAMongoDBAgent(t *testing.T) {
	t.Run("update RTA options ", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true)
		actualAgents, err := as.List(ctx, models.AgentFilters{})
		require.NoError(t, err)
		require.Len(t, actualAgents, 4) // PMM Server's pmm-agent, node_exporter, postgres_exporter, PostgreSQL QAN

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005")

		// Add PMM Agent
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

		// Add MongoDB Service
		ms, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo-rta",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
		})
		require.NoError(t, err)

		// Add RTA MongoDB Agent with default RTA options
		rtaAgent, err := as.AddRTAMongoDBAgent(ctx, &inventoryv1.AddRTAMongoDBAgentParams{
			PmmAgentId:          pmmAgent.GetPmmAgent().AgentId,
			ServiceId:           ms.ServiceId,
			Username:            "username",
			SkipConnectionCheck: true,
		})
		require.NoError(t, err)
		assert.Equal(t, durationpb.New(2*time.Second), rtaAgent.GetRtaMongodbAgent().RtaOptions.CollectInterval)

		resp, err := as.ChangeRTAMongoDBAgent(ctx, rtaAgent.GetRtaMongodbAgent().AgentId, &inventoryv1.ChangeRTAMongoDBAgentParams{
			RtaOptions: &inventoryv1.RTAOptions{
				CollectInterval: durationpb.New(5 * time.Second),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, durationpb.New(5*time.Second), resp.GetRtaMongodbAgent().RtaOptions.CollectInterval)
	})
}

func TestChangeAgentConnectionCheck(t *testing.T) {
	// Adds a pmm-agent, a PostgreSQL service and a postgres_exporter (without connection check)
	// and returns the exporter's agent ID. Expects stateUpdates calls to RequestStateUpdate:
	// one is made here (AddPostgresExporter), plus one per successful change.
	addPostgresExporter := func(t *testing.T, ss *ServicesService, as *AgentsService, ctx context.Context, stateUpdates int) string {
		t.Helper()

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005").Times(stateUpdates)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(5432)),
		})
		require.NoError(t, err)

		exporter, err := as.AddPostgresExporter(ctx, &inventoryv1.AddPostgresExporterParams{
			PmmAgentId:          pmmAgent.GetPmmAgent().AgentId,
			ServiceId:           ps.ServiceId,
			Username:            "username",
			SkipConnectionCheck: true,
		})
		require.NoError(t, err)

		return exporter.GetPostgresExporter().AgentId
	}

	// Adds a pmm-agent, a MySQL service and a mysqld_exporter (without connection check) and returns the exporter's agent ID.
	addMysqldExporter := func(t *testing.T, ss *ServicesService, as *AgentsService, ctx context.Context, stateUpdates int) string {
		t.Helper()

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005").Times(stateUpdates)

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		ms, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		exporter, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
			PmmAgentId:          pmmAgent.GetPmmAgent().AgentId,
			ServiceId:           ms.ServiceId,
			Username:            "username",
			SkipConnectionCheck: true,
		})
		require.NoError(t, err)

		return exporter.GetMysqldExporter().AgentId
	}

	connectionCheckCall := func(as *AgentsService, ctx context.Context) *mock.Call {
		return as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name()))
	}

	t.Run("CheckRunsOnCredentialChange", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		agentID := addPostgresExporter(t, ss, as, ctx, 2)

		connectionCheckCall(as, ctx).Return(nil).Once()

		resp, err := as.ChangePostgresExporter(ctx, agentID, &inventoryv1.ChangePostgresExporterParams{
			Username: new("new-username"),
		})
		require.NoError(t, err)
		assert.Equal(t, "new-username", resp.GetPostgresExporter().Username)
	})

	t.Run("RollbackOnFailedCheck", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		agentID := addPostgresExporter(t, ss, as, ctx, 1)

		checkErr := status.Error(codes.FailedPrecondition, "Connection check failed: FATAL: password authentication failed.")
		connectionCheckCall(as, ctx).Return(checkErr).Once()

		_, err := as.ChangePostgresExporter(ctx, agentID, &inventoryv1.ChangePostgresExporterParams{
			Username: new("wrong-username"),
			Password: new("wrong-password"),
		})
		tests.AssertGRPCError(t, status.Convert(checkErr), err)

		// The change must be rolled back.
		agent, err := as.Get(ctx, agentID)
		require.NoError(t, err)
		assert.Equal(t, "username", agent.(*inventoryv1.PostgresExporter).Username)
	})

	t.Run("NoCheckForUnrelatedChange", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		agentID := addPostgresExporter(t, ss, as, ctx, 2)

		// No CheckConnectionToService or GetInfoFromService expectations:
		// changing only labels must not trigger a connection check.
		resp, err := as.ChangePostgresExporter(ctx, agentID, &inventoryv1.ChangePostgresExporterParams{
			CustomLabels: &common.StringMap{Values: map[string]string{"environment": "test"}},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"environment": "test"}, resp.GetPostgresExporter().CustomLabels)
	})

	t.Run("SkipConnectionCheckHonored", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		agentID := addPostgresExporter(t, ss, as, ctx, 2)

		// No CheckConnectionToService or GetInfoFromService expectations:
		// the explicit skip flag must bypass the check.
		resp, err := as.ChangePostgresExporter(ctx, agentID, &inventoryv1.ChangePostgresExporterParams{
			Username:            new("new-username"),
			Password:            new("new-password"),
			SkipConnectionCheck: new(true),
		})
		require.NoError(t, err)
		assert.Equal(t, "new-username", resp.GetPostgresExporter().Username)
	})

	t.Run("MysqldOmittedSkipRunsCheck", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		agentID := addMysqldExporter(t, ss, as, ctx, 2)

		connectionCheckCall(as, ctx).Return(nil).Once()

		// SkipConnectionCheck left nil (omitted) -> GetSkipConnectionCheck() == false -> check runs.
		resp, err := as.ChangeMySQLdExporter(ctx, agentID, &inventoryv1.ChangeMySQLdExporterParams{
			Username: new("new-username"), // AffectsConnection() == true
		})
		require.NoError(t, err)
		assert.Equal(t, "new-username", resp.GetMysqldExporter().Username)
	})

	t.Run("MysqldExplicitFalseSkipRunsCheck", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		agentID := addMysqldExporter(t, ss, as, ctx, 2)

		connectionCheckCall(as, ctx).Return(nil).Once()

		// Explicit false must behave identically to omitted: the check still runs.
		resp, err := as.ChangeMySQLdExporter(ctx, agentID, &inventoryv1.ChangeMySQLdExporterParams{
			Username:            new("new-username"),
			SkipConnectionCheck: new(false),
		})
		require.NoError(t, err)
		assert.Equal(t, "new-username", resp.GetMysqldExporter().Username)
	})
}

// TestAddAgentConnectionCheck locks in the Add-side connection-check behavior that
// executeAgentAdd centralizes: exporters run the check AND fetch service info, while
// QAN agents run only the check. The service-info broker is deliberately left
// unregistered on the QAN path so an unexpected GetInfoFromService call would fail.
func TestAddAgentConnectionCheck(t *testing.T) {
	connectionCheckCall := func(as *AgentsService, ctx context.Context) *mock.Call {
		return as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name()))
	}

	serviceInfoCall := func(as *AgentsService, ctx context.Context) *mock.Call {
		return as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeFor[*reform.TX]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Service]().Name()),
			mock.AnythingOfType(reflect.TypeFor[*models.Agent]().Name()))
	}

	t.Run("QANRunsCheckWithoutServiceInfo", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005").Once()

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		ms, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)

		// Only the connection check is expected. GetInfoFromService is intentionally not
		// registered, so a call to it would panic as an unexpected mock invocation.
		connectionCheckCall(as, ctx).Return(nil).Once()

		_, err = as.AddQANMySQLPerfSchemaAgent(ctx, &inventoryv1.AddQANMySQLPerfSchemaAgentParams{
			PmmAgentId: pmmAgent.GetPmmAgent().AgentId,
			ServiceId:  ms.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)

		as.sib.(*mockServiceInfoBroker).AssertNotCalled(t, "GetInfoFromService")
	})

	t.Run("ExporterRunsCheckWithServiceInfo", func(t *testing.T) {
		ss, as, _, teardown, ctx, _ := setup(t)
		t.Cleanup(func() { teardown(t) })

		as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-000000000005").Once()

		pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
			RunsOnNodeId: models.PMMServerNodeID,
		})
		require.NoError(t, err)

		ps, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(5432)),
		})
		require.NoError(t, err)

		// Exporters trigger both the connection check and the service-info fetch.
		connectionCheckCall(as, ctx).Return(nil).Once()
		serviceInfoCall(as, ctx).Return(nil).Once()

		_, err = as.AddPostgresExporter(ctx, &inventoryv1.AddPostgresExporterParams{
			PmmAgentId: pmmAgent.GetPmmAgent().AgentId,
			ServiceId:  ps.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)
	})
}
