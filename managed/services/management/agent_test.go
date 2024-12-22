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

package management

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	agentv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

var now time.Time

func setup(t *testing.T) (context.Context, *ManagementService, func(t *testing.T)) {
	t.Helper()

	now = models.Now()
	origNowF := models.Now
	models.Now = func() time.Time {
		return now
	}

	ctx := logger.Set(context.Background(), t.Name())
	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, database.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	vmdb := &mockPrometheusService{}
	vmdb.Test(t)

	state := &mockAgentsStateUpdater{}
	state.Test(t)

	ar := &mockAgentsRegistry{}
	ar.Test(t)

	cc := &mockConnectionChecker{}
	cc.Test(t)

	sib := &mockServiceInfoBroker{}
	sib.Test(t)

	vc := &mockVersionCache{}
	vc.Test(t)

	grafanaClient := &mockGrafanaClient{}
	grafanaClient.Test(t)

	vmClient := &mockVictoriaMetricsClient{}
	vmClient.Test(t)

	teardown := func(t *testing.T) {
		t.Helper()
		models.Now = origNowF
		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())

		ar.AssertExpectations(t)
		state.AssertExpectations(t)
		cc.AssertExpectations(t)
		sib.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		vc.AssertExpectations(t)
		grafanaClient.AssertExpectations(t)
		vmClient.AssertExpectations(t)
	}

	s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)

	return ctx, s, teardown
}

func TestAgentService(t *testing.T) {
	t.Run("Should return a validation error when no params passed", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		response, err := s.ListAgents(ctx, &agentv1.ListAgentsRequest{})
		assert.Nil(t, response)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Either service_id or node_id is expected."), err)
	})

	t.Run("Should return a validation error when both params passed", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		response, err := s.ListAgents(ctx, &agentv1.ListAgentsRequest{ServiceId: "foo-id", NodeId: "bar-id"})
		assert.Nil(t, response)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Either service_id or node_id is expected, not both."), err)
	})

	t.Run("ListAgents", func(t *testing.T) {
		const (
			pgExporterID      = "00000000-0000-4000-8000-000000000003"
			pgStatStatementID = "00000000-0000-4000-8000-000000000004"
		)

		t.Run("should output a list of agents provisioned by default", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			services, err := models.FindServices(s.db.Querier, models.ServiceFilters{
				NodeID: models.PMMServerNodeID,
			})

			require.NoError(t, err)
			assert.Len(t, services, 1)
			service := services[0]

			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once() // PMM Server Agent
			s.r.(*mockAgentsRegistry).On("IsConnected", pgExporterID).Return(false).Once()           // PMM Server PostgreSQL exporter
			s.r.(*mockAgentsRegistry).On("IsConnected", pgStatStatementID).Return(false).Once()      // PMM Server PG Stat Statements agent
			response, err := s.ListAgents(ctx, &agentv1.ListAgentsRequest{
				ServiceId: service.ServiceID,
			})
			require.NoError(t, err)

			expected := []*agentv1.UniversalAgent{
				{
					AgentId:                 pgExporterID,
					AgentType:               "postgres_exporter",
					PmmAgentId:              models.PMMServerAgentID,
					IsConnected:             false,
					CreatedAt:               timestamppb.New(now),
					UpdatedAt:               timestamppb.New(now),
					Username:                "postgres",
					ServiceId:               "00000000-0000-4000-8000-000000000002",
					Status:                  "AGENT_STATUS_UNKNOWN",
					Tls:                     true,
					CommentsParsingDisabled: true,
					AzureOptions: &agentv1.UniversalAgent_AzureOptions{
						ClientId:          "",
						IsClientSecretSet: false,
						ResourceGroup:     "",
						SubscriptionId:    "",
						TenantId:          "",
					},
					MongoDbOptions: &agentv1.UniversalAgent_MongoDBOptions{
						IsTlsCertificateKeySet:             false,
						IsTlsCertificateKeyFilePasswordSet: false,
						AuthenticationMechanism:            "",
						AuthenticationDatabase:             "",
						StatsCollections:                   nil,
						CollectionsLimit:                   0,
						EnableAllCollectors:                false,
					},
					MysqlOptions: &agentv1.UniversalAgent_MySQLOptions{
						IsTlsKeySet: false,
					},
					PostgresqlOptions: &agentv1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet:            false,
						AutoDiscoveryLimit:     0,
						MaxExporterConnections: 0,
					},
				},
				{
					AgentId:                 pgStatStatementID,
					AgentType:               "qan-postgresql-pgstatements-agent",
					PmmAgentId:              models.PMMServerAgentID,
					IsConnected:             false,
					CreatedAt:               timestamppb.New(now),
					UpdatedAt:               timestamppb.New(now),
					Username:                "postgres",
					ServiceId:               "00000000-0000-4000-8000-000000000002",
					Status:                  "AGENT_STATUS_UNKNOWN",
					Tls:                     true,
					CommentsParsingDisabled: true,
					AzureOptions: &agentv1.UniversalAgent_AzureOptions{
						ClientId:          "",
						IsClientSecretSet: false,
						ResourceGroup:     "",
						SubscriptionId:    "",
						TenantId:          "",
					},
					MongoDbOptions: &agentv1.UniversalAgent_MongoDBOptions{
						IsTlsCertificateKeySet:             false,
						IsTlsCertificateKeyFilePasswordSet: false,
						AuthenticationMechanism:            "",
						AuthenticationDatabase:             "",
						StatsCollections:                   nil,
						CollectionsLimit:                   0,
						EnableAllCollectors:                false,
					},
					MysqlOptions: &agentv1.UniversalAgent_MySQLOptions{
						IsTlsKeySet: false,
					},
					PostgresqlOptions: &agentv1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet:            false,
						AutoDiscoveryLimit:     0,
						MaxExporterConnections: 0,
					},
				},
				{
					AgentId:      models.PMMServerAgentID,
					AgentType:    "pmm-agent",
					RunsOnNodeId: models.PMMServerAgentID,
					IsConnected:  true,
					CreatedAt:    timestamppb.New(now),
					UpdatedAt:    timestamppb.New(now),
					AzureOptions: &agentv1.UniversalAgent_AzureOptions{
						ClientId:          "",
						IsClientSecretSet: false,
						ResourceGroup:     "",
						SubscriptionId:    "",
						TenantId:          "",
					},
					MongoDbOptions: &agentv1.UniversalAgent_MongoDBOptions{
						IsTlsCertificateKeySet:             false,
						IsTlsCertificateKeyFilePasswordSet: false,
						AuthenticationMechanism:            "",
						AuthenticationDatabase:             "",
						StatsCollections:                   nil,
						CollectionsLimit:                   0,
						EnableAllCollectors:                false,
					},
					MysqlOptions: &agentv1.UniversalAgent_MySQLOptions{
						IsTlsKeySet: false,
					},
					PostgresqlOptions: &agentv1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet:            false,
						AutoDiscoveryLimit:     0,
						MaxExporterConnections: 0,
					},
				},
			}

			assert.Equal(t, expected, response.Agents)
		})

		t.Run("should output a list of agents provisioned for RDS service", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			node, err := models.CreateNode(s.db.Querier, models.RemoteRDSNodeType, &models.CreateNodeParams{
				NodeName: "test",
				Address:  "test-address",
				Region:   pointer.ToString("test-region"),
			})
			require.NoError(t, err)

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      node.NodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			pmmAgent, err := models.CreatePMMAgent(s.db.Querier, models.PMMServerNodeID, nil)
			require.NoError(t, err)

			rdsExporter, err := models.CreateAgent(s.db.Querier, models.RDSExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				ServiceID:  service.ServiceID,
			})
			require.NoError(t, err)

			s.r.(*mockAgentsRegistry).On("IsConnected", rdsExporter.AgentID).Return(false).Once()

			response, err := s.ListAgents(ctx, &agentv1.ListAgentsRequest{
				ServiceId: service.ServiceID,
			})
			require.NoError(t, err)

			expected := []*agentv1.UniversalAgent{
				{
					AgentId:     rdsExporter.AgentID,
					AgentType:   "rds_exporter",
					PmmAgentId:  "00000000-0000-4000-8000-000000000007",
					IsConnected: false,
					CreatedAt:   timestamppb.New(now),
					UpdatedAt:   timestamppb.New(now),
					ServiceId:   "00000000-0000-4000-8000-000000000006",
					Status:      "AGENT_STATUS_UNKNOWN",
					AzureOptions: &agentv1.UniversalAgent_AzureOptions{
						ClientId:          "",
						IsClientSecretSet: false,
						ResourceGroup:     "",
						SubscriptionId:    "",
						TenantId:          "",
					},
					MongoDbOptions: &agentv1.UniversalAgent_MongoDBOptions{
						IsTlsCertificateKeySet:             false,
						IsTlsCertificateKeyFilePasswordSet: false,
						AuthenticationMechanism:            "",
						AuthenticationDatabase:             "",
						StatsCollections:                   nil,
						CollectionsLimit:                   0,
						EnableAllCollectors:                false,
					},
					MysqlOptions: &agentv1.UniversalAgent_MySQLOptions{
						IsTlsKeySet: false,
					},
					PostgresqlOptions: &agentv1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet:            false,
						AutoDiscoveryLimit:     0,
						MaxExporterConnections: 0,
					},
				},
			}
			assert.Equal(t, expected, response.Agents)
		})

		t.Run("should output a list of agents provisioned for Azure service", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			node, err := models.CreateNode(s.db.Querier, models.RemoteAzureDatabaseNodeType, &models.CreateNodeParams{
				NodeName: "test",
				Address:  "test-address",
				Region:   pointer.ToString("test-region"),
			})
			require.NoError(t, err)

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      node.NodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			pmmAgent, err := models.CreatePMMAgent(s.db.Querier, models.PMMServerNodeID, nil)
			require.NoError(t, err)

			azureExporter, err := models.CreateAgent(s.db.Querier, models.AzureDatabaseExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				ServiceID:  service.ServiceID,
			})
			require.NoError(t, err)

			s.r.(*mockAgentsRegistry).On("IsConnected", azureExporter.AgentID).Return(false).Once()

			response, err := s.ListAgents(ctx, &agentv1.ListAgentsRequest{
				ServiceId: service.ServiceID,
			})
			require.NoError(t, err)

			expected := []*agentv1.UniversalAgent{
				{
					AgentId:     azureExporter.AgentID,
					AgentType:   "azure_database_exporter",
					PmmAgentId:  "00000000-0000-4000-8000-000000000007",
					IsConnected: false,
					CreatedAt:   timestamppb.New(now),
					UpdatedAt:   timestamppb.New(now),
					ServiceId:   "00000000-0000-4000-8000-000000000006",
					Status:      "AGENT_STATUS_UNKNOWN",
					AzureOptions: &agentv1.UniversalAgent_AzureOptions{
						ClientId:          "",
						IsClientSecretSet: false,
						ResourceGroup:     "",
						SubscriptionId:    "",
						TenantId:          "",
					},
					MongoDbOptions: &agentv1.UniversalAgent_MongoDBOptions{
						IsTlsCertificateKeySet:             false,
						IsTlsCertificateKeyFilePasswordSet: false,
						AuthenticationMechanism:            "",
						AuthenticationDatabase:             "",
						StatsCollections:                   nil,
						CollectionsLimit:                   0,
						EnableAllCollectors:                false,
					},
					MysqlOptions: &agentv1.UniversalAgent_MySQLOptions{
						IsTlsKeySet: false,
					},
					PostgresqlOptions: &agentv1.UniversalAgent_PostgreSQLOptions{
						IsSslKeySet:            false,
						AutoDiscoveryLimit:     0,
						MaxExporterConnections: 0,
					},
				},
			}
			assert.Equal(t, expected, response.Agents)
		})
	})
}

func TestListAgentVersions(t *testing.T) {
	t.Run("Should suggest critical severity if major versions differ", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		pmmAgent := &models.Agent{
			AgentID:      uuid.New().String(),
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToString(models.PMMServerNodeID),
			Version:      pointer.ToString("2.0.0"),
		}

		err := s.db.Insert(pmmAgent)
		require.NoError(t, err)

		version.PMMVersion = "3.0.0"
		res, err := s.ListAgentVersions(ctx, &agentv1.ListAgentVersionsRequest{})
		require.NoError(t, err)
		require.Len(t, res.AgentVersions, 1)

		assert.Equal(t, agentv1.UpdateSeverity_UPDATE_SEVERITY_CRITICAL, res.AgentVersions[0].Severity)
	})

	t.Run("Should suggest an update if minor versions differ", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		pmmAgent := &models.Agent{
			AgentID:      uuid.New().String(),
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToString(models.PMMServerNodeID),
			Version:      pointer.ToString("3.0.0"),
		}

		err := s.db.Insert(pmmAgent)
		require.NoError(t, err)

		version.PMMVersion = "3.1.0"
		res, err := s.ListAgentVersions(ctx, &agentv1.ListAgentVersionsRequest{})
		require.NoError(t, err)
		require.Len(t, res.AgentVersions, 1)

		assert.Equal(t, agentv1.UpdateSeverity_UPDATE_SEVERITY_REQUIRED, res.AgentVersions[0].Severity)
	})

	t.Run("Should suggest an update if patch versions differ", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		pmmAgent := &models.Agent{
			AgentID:      uuid.New().String(),
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToString(models.PMMServerNodeID),
			Version:      pointer.ToString("3.0.0"),
		}

		err := s.db.Insert(pmmAgent)
		require.NoError(t, err)

		version.PMMVersion = "3.0.1"
		res, err := s.ListAgentVersions(ctx, &agentv1.ListAgentVersionsRequest{})
		require.NoError(t, err)
		require.Len(t, res.AgentVersions, 1)

		assert.Equal(t, agentv1.UpdateSeverity_UPDATE_SEVERITY_REQUIRED, res.AgentVersions[0].Severity)
	})

	t.Run("Should suggest no update if versions are the same", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		pmmAgent := &models.Agent{
			AgentID:      uuid.New().String(),
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToString(models.PMMServerNodeID),
			Version:      pointer.ToString("3.0.0"),
		}

		err := s.db.Insert(pmmAgent)
		require.NoError(t, err)

		version.PMMVersion = "3.0.0"
		res, err := s.ListAgentVersions(ctx, &agentv1.ListAgentVersionsRequest{})
		require.NoError(t, err)
		require.Len(t, res.AgentVersions, 1)

		assert.Equal(t, agentv1.UpdateSeverity_UPDATE_SEVERITY_UP_TO_DATE, res.AgentVersions[0].Severity)
	})

	t.Run("Should say unsupported if client version is newer", func(t *testing.T) {
		ctx, s, teardown := setup(t)
		t.Cleanup(func() { teardown(t) })

		pmmAgent := &models.Agent{
			AgentID:      uuid.New().String(),
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToString(models.PMMServerNodeID),
			Version:      pointer.ToString("3.0.0"),
		}

		err := s.db.Insert(pmmAgent)
		require.NoError(t, err)

		version.PMMVersion = "3.0.0-beta"
		res, err := s.ListAgentVersions(ctx, &agentv1.ListAgentVersionsRequest{})
		require.NoError(t, err)
		require.Len(t, res.AgentVersions, 1)

		assert.Equal(t, agentv1.UpdateSeverity_UPDATE_SEVERITY_UNSUPPORTED, res.AgentVersions[0].Severity)
	})
}
