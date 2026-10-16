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
	"fmt"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/common"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	mgmtcommon "github.com/percona/pmm/managed/services/management/common"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

type updateServiceMocks struct {
	state *mockAgentsStateUpdater
	cc    *mockConnectionChecker
	sib   *mockServiceInfoBroker
	vmdb  *mockPrometheusService
	vc    *mockVersionCache
	stm   *mockScheduledTasksRemover
}

func setupUpdateService(t *testing.T) (context.Context, *ManagementService, *updateServiceMocks) {
	t.Helper()

	ctx := logger.Set(t.Context(), t.Name())
	uuid.SetRand(&tests.IDReader{})
	t.Cleanup(func() { uuid.SetRand(nil) })

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	m := &updateServiceMocks{
		state: newMockAgentsStateUpdater(t),
		cc:    newMockConnectionChecker(t),
		sib:   newMockServiceInfoBroker(t),
		vmdb:  newMockPrometheusService(t),
		vc:    newMockVersionCache(t),
		stm:   newMockScheduledTasksRemover(t),
	}
	ar := newMockAgentsRegistry(t)
	ar.On("IsConnected", mock.Anything).Return(false).Maybe()

	s := NewManagementService(db, ar, m.state, m.cc, m.sib, m.vmdb, m.vc, nil, nil, m.stm, nil, false)

	return ctx, s, m
}

// expectApplied sets the expectations of an update that is applied rather than rejected or dry-run.
func (m *updateServiceMocks) expectApplied() {
	m.state.On("RequestStateUpdate", mock.Anything, mock.Anything)
	m.vmdb.On("RequestConfigurationUpdate")
}

// expectConnectionCheck sets the expectations of an update that checks the connection to the Service.
func (m *updateServiceMocks) expectConnectionCheck() {
	m.cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	m.sib.On("GetInfoFromService", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	m.vc.On("RequestSoftwareVersionsUpdate").Maybe()
}

func addTestMongoDB(ctx context.Context, t *testing.T, s *ManagementService, m *updateServiceMocks) string {
	t.Helper()

	m.state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()
	m.vc.On("RequestSoftwareVersionsUpdate").Once()

	resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
		Service: &managementv1.AddServiceRequest_Mongodb{
			Mongodb: &managementv1.AddMongoDBServiceParams{
				NodeId:              models.PMMServerNodeID,
				PmmAgentId:          models.PMMServerAgentID,
				ServiceName:         "mongodb-srv-1",
				Address:             "127.0.0.1",
				Port:                27017,
				Environment:         "prod",
				CustomLabels:        map[string]string{"team": "db"},
				Username:            "admin",
				Password:            "admin_pass",
				QanMongodbProfiler:  true,
				RtaMongodbAgent:     true,
				EnableAllCollectors: true,
				CollectionsLimit:    -1,
				MaxQueryLength:      2048,
				LogLevel:            inventoryv1.LogLevel_LOG_LEVEL_WARN,
				ConnectionTimeout:   durationpb.New(5 * time.Second),
				SkipConnectionCheck: true,
			},
		},
	})
	require.NoError(t, err)

	return resp.GetMongodb().Service.ServiceId
}

func addTestMySQL(ctx context.Context, t *testing.T, s *ManagementService, m *updateServiceMocks, params *managementv1.AddMySQLServiceParams) string {
	t.Helper()

	m.state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()
	m.vc.On("RequestSoftwareVersionsUpdate").Once()

	params.NodeId = models.PMMServerNodeID
	params.PmmAgentId = models.PMMServerAgentID
	params.ServiceName = "mysql-srv-1"
	params.Username = "root"
	params.Password = "root_pass"
	params.SkipConnectionCheck = true

	resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
		Service: &managementv1.AddServiceRequest_Mysql{Mysql: params},
	})
	require.NoError(t, err)

	return resp.GetMysql().Service.ServiceId
}

// serviceAgents returns the Agents of the Service by type.
func serviceAgents(t *testing.T, q *reform.Querier, serviceID string) map[models.AgentType]*models.Agent {
	t.Helper()

	agents, err := models.FindAgents(q, models.AgentFilters{ServiceID: serviceID})
	require.NoError(t, err)

	res := make(map[models.AgentType]*models.Agent, len(agents))
	for _, agent := range agents {
		res[agent.AgentType] = agent
	}

	return res
}

func TestUpdateService(t *testing.T) {
	t.Run("ChangesOnlyPassedFields", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)
		before := serviceAgents(t, s.db.Querier, serviceID)

		m.expectApplied()
		resp, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: "mongodb-srv-1",
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					CollectionsLimit:  new(int32(0)),
					DisableCollectors: &common.StringArray{Values: []string{"collstats"}},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, serviceID, resp.After.ServiceId)

		after := serviceAgents(t, s.db.Querier, serviceID)
		exporter := after[models.MongoDBExporterType]
		assert.Equal(t, before[models.MongoDBExporterType].AgentID, exporter.AgentID)
		assert.Equal(t, int32(0), exporter.MongoDBOptions.CollectionsLimit)
		assert.Equal(t, []string{"collstats"}, []string(exporter.ExporterOptions.DisabledCollectors))
		assert.True(t, exporter.MongoDBOptions.EnableAllCollectors)
		assert.Equal(t, "admin_pass", pointer.GetString(exporter.Password))
		assert.Equal(t, new(5*time.Second), exporter.ExporterOptions.ConnectionTimeout)
		assert.Equal(t, before[models.QANMongoDBProfilerAgentType].AgentID, after[models.QANMongoDBProfilerAgentType].AgentID)
		assert.Equal(t, before[models.RTAMongoDBAgentType].AgentID, after[models.RTAMongoDBAgentType].AgentID)

		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Equal(t, "prod", service.Environment)
		labels, err := service.GetCustomLabels()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"team": "db"}, labels)
	})

	t.Run("TurnsOffAndClears", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					EnableAllCollectors: new(false),
					DisableCollectors:   &common.StringArray{},
					CustomLabels:        &common.StringMap{},
					Environment:         new(""),
					ConnectionTimeout:   durationpb.New(0),
				},
			},
		})
		require.NoError(t, err)

		exporter := serviceAgents(t, s.db.Querier, serviceID)[models.MongoDBExporterType]
		assert.False(t, exporter.MongoDBOptions.EnableAllCollectors)
		assert.Empty(t, exporter.ExporterOptions.DisabledCollectors)
		assert.Nil(t, exporter.ExporterOptions.ConnectionTimeout)

		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Empty(t, service.Environment)
		labels, err := service.GetCustomLabels()
		require.NoError(t, err)
		assert.Empty(t, labels)
	})

	t.Run("CredentialsReachEveryAgent", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		m.expectConnectionCheck()
		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					Password:               new("new_pass"),
					AuthenticationDatabase: new("admin"),
					LogLevel:               new(inventoryv1.LogLevel_LOG_LEVEL_DEBUG),
				},
			},
		})
		require.NoError(t, err)

		for agentType, agent := range serviceAgents(t, s.db.Querier, serviceID) {
			assert.Equal(t, "new_pass", pointer.GetString(agent.Password), agentType)
			assert.Equal(t, "admin", agent.MongoDBOptions.AuthenticationDatabase, agentType)
			assert.Equal(t, "debug", pointer.GetString(agent.LogLevel), agentType)
		}

		// LOG_LEVEL_UNSPECIFIED is Auto, as when the Service is added: no level is passed to the Agents.
		m.expectApplied()
		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{LogLevel: new(inventoryv1.LogLevel_LOG_LEVEL_UNSPECIFIED)},
			},
		})
		require.NoError(t, err)

		for agentType, agent := range serviceAgents(t, s.db.Querier, serviceID) {
			assert.Empty(t, pointer.GetString(agent.LogLevel), agentType)
		}
	})

	t.Run("FailedConnectionCheckChangesNothing", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		m.cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(status.Error(codes.FailedPrecondition, "Connection check failed.")).Once()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					Password:    new("wrong_pass"),
					Environment: new("qa"),
				},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Connection check failed."), err)

		exporter := serviceAgents(t, s.db.Querier, serviceID)[models.MongoDBExporterType]
		assert.Equal(t, "admin_pass", pointer.GetString(exporter.Password))
		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Equal(t, "prod", service.Environment)
	})

	t.Run("SkipConnectionCheck", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		m.expectApplied()
		m.vc.On("RequestSoftwareVersionsUpdate").Once()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					Password:            new("new_pass"),
					SkipConnectionCheck: true,
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("SwitchesQuerySource", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		m.expectConnectionCheck()
		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					QanMongodbProfiler: new(false),
					QanMongodbMongolog: new(true),
				},
			},
		})
		require.NoError(t, err)

		agents := serviceAgents(t, s.db.Querier, serviceID)
		assert.NotContains(t, agents, models.QANMongoDBProfilerAgentType)
		mongolog := agents[models.QANMongoDBMongologAgentType]
		require.NotNil(t, mongolog)
		assert.Equal(t, "admin", pointer.GetString(mongolog.Username))
		assert.Equal(t, "admin_pass", pointer.GetString(mongolog.Password))
		assert.Equal(t, int32(2048), mongolog.QANOptions.MaxQueryLength)
		assert.Equal(t, "warn", pointer.GetString(mongolog.LogLevel))
		assert.Contains(t, agents, models.RTAMongoDBAgentType)
	})

	t.Run("DryRunChangesNothing", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		// The cluster changes, so the dry run checks the lock; it removes no scheduled tasks.
		m.stm.On("CheckClusterUnlocked", ctx, s.db, &models.ChangeStandardLabelsParams{
			ServiceID: serviceID,
			Cluster:   new("rs-cluster"),
		}).Return(nil).Once()
		resp, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			DryRun:    true,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					Cluster:          new("rs-cluster"),
					CollectionsLimit: new(int32(500)),
				},
			},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Before.Cluster)
		assert.Equal(t, "rs-cluster", resp.After.Cluster)

		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Empty(t, service.Cluster)
		exporter := serviceAgents(t, s.db.Querier, serviceID)[models.MongoDBExporterType]
		assert.Equal(t, int32(-1), exporter.MongoDBOptions.CollectionsLimit)
	})

	t.Run("ClusterChangeRemovesScheduledTasks", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		m.stm.On("RemoveScheduledTasks", ctx, s.db, &models.ChangeStandardLabelsParams{
			ServiceID: serviceID,
			Cluster:   new("rs-cluster"),
		}).Return(nil).Once()
		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{Cluster: new("rs-cluster")},
			},
		})
		require.NoError(t, err)

		// The unchanged cluster leaves the scheduled tasks alone.
		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{Cluster: new("rs-cluster")},
			},
		})
		require.NoError(t, err)
	})

	t.Run("ClusterLockedRejectsChange", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		errLocked := fmt.Errorf("there is an unfinished backup job: %w", mgmtcommon.ErrClusterLocked)
		update := &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{Cluster: new("rs-cluster")},
			},
		}

		m.stm.On("RemoveScheduledTasks", ctx, s.db, mock.Anything).Return(errLocked).Once()
		_, err := s.UpdateService(ctx, update)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, errLocked.Error()), err)

		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Empty(t, service.Cluster)

		// A dry run checks the lock too, and removes nothing either way.
		update.DryRun = true
		m.stm.On("CheckClusterUnlocked", ctx, s.db, mock.Anything).Return(errLocked).Once()
		_, err = s.UpdateService(ctx, update)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, errLocked.Error()), err)

		m.stm.On("CheckClusterUnlocked", ctx, s.db, mock.Anything).Return(nil).Once()
		res, err := s.UpdateService(ctx, update)
		require.NoError(t, err)
		assert.Equal(t, "rs-cluster", res.After.Cluster)
		m.stm.AssertNumberOfCalls(t, "RemoveScheduledTasks", 1)
	})

	// A rejected update must not remove the scheduled backups of the old and new clusters.
	t.Run("RejectedClusterChangeKeepsScheduledTasks", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{Cluster: new("rs-cluster")},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Service mongodb-srv-1 is a mongodb Service, not mysql."), err)

		m.cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(status.Error(codes.FailedPrecondition, "Connection check failed.")).Once()
		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{Cluster: new("rs-cluster"), Password: new("wrong_pass")},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Connection check failed."), err)

		m.stm.AssertNotCalled(t, "RemoveScheduledTasks", mock.Anything, mock.Anything, mock.Anything)
	})

	// Settings owns the query source of PMM's internal PostgreSQL, pinned by PMM_ENABLE_INTERNAL_PG_QAN.
	t.Run("InternalPostgreSQLQuerySourceIsFixed", func(t *testing.T) {
		ctx, s, _ := setupUpdateService(t)

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: models.PMMServerPostgreSQLServiceName,
			Service: &managementv1.UpdateServiceRequest_Postgresql{
				Postgresql: &managementv1.UpdatePostgreSQLServiceParams{
					QanPostgresqlPgstatementsAgent:  new(false),
					QanPostgresqlPgstatmonitorAgent: new(true),
					SkipConnectionCheck:             true,
				},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
			"The query source of PMM's internal PostgreSQL server cannot be changed; QAN for it is turned on and off in Settings."), err)

		agent, err := models.FindInternalPgQANAgent(s.db.Querier)
		require.NoError(t, err)
		assert.Equal(t, models.QANPostgreSQLPgStatementsAgentType, agent.AgentType)
	})

	t.Run("WrongServiceType", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{Environment: new("qa")},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Service mongodb-srv-1 is a mongodb Service, not mysql."), err)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx, s, _ := setupUpdateService(t)

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: "no-such-service",
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "no-such-service" not found.`), err)
	})

	t.Run("NoServiceParams", func(t *testing.T) {
		ctx, s, _ := setupUpdateService(t)

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{ServiceId: "mongodb-srv-1"})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Service parameters are expected."), err)
	})

	t.Run("PushOnServerAgentRejected", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMongoDB(ctx, t, s, m)

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mongodb{
				Mongodb: &managementv1.UpdateMongoDBServiceParams{
					MetricsMode: new(managementv1.MetricsMode_METRICS_MODE_PUSH),
				},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "push metrics mode is not allowed for exporters running on pmm-server"), err)
	})

	// A QAN Agent with none to take its settings from has comments parsing off, as pmm-admin add leaves it.
	t.Run("NewQANAgentHasCommentsParsingOff", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMySQL(ctx, t, s, m, &managementv1.AddMySQLServiceParams{Address: "127.0.0.1", Port: 3306})

		turnOn := func(params *managementv1.UpdateMySQLServiceParams) *models.Agent {
			m.expectConnectionCheck()
			m.expectApplied()
			_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
				ServiceId: serviceID,
				Service:   &managementv1.UpdateServiceRequest_Mysql{Mysql: params},
			})
			require.NoError(t, err)

			agent := serviceAgents(t, s.db.Querier, serviceID)[models.QANMySQLPerfSchemaAgentType]
			require.NotNil(t, agent)
			return agent
		}

		agent := turnOn(&managementv1.UpdateMySQLServiceParams{QanMysqlPerfschema: new(true)})
		assert.True(t, agent.QANOptions.CommentsParsingDisabled)

		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{QanMysqlPerfschema: new(false)},
			},
		})
		require.NoError(t, err)

		agent = turnOn(&managementv1.UpdateMySQLServiceParams{QanMysqlPerfschema: new(true), DisableCommentsParsing: new(false)})
		assert.False(t, agent.QANOptions.CommentsParsingDisabled)
	})

	// A Service on a remote RDS or Azure Node gets only the QAN Agents that adding it creates.
	t.Run("CloudServiceQANAgents", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMySQL(ctx, t, s, m, &managementv1.AddMySQLServiceParams{Address: "127.0.0.1", Port: 3306})

		rds, err := models.CreateNode(s.db.Querier, models.RemoteRDSNodeType, &models.CreateNodeParams{
			NodeName:   "rds-node",
			Address:    "rds.abc.us-east-1.rds.amazonaws.com",
			InstanceID: "rds",
			Region:     new("us-east-1"),
		})
		require.NoError(t, err)
		_, err = s.db.Exec("UPDATE services SET node_id = $1 WHERE service_id = $2", rds.NodeID, serviceID)
		require.NoError(t, err)

		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{QanMysqlSlowlog: new(true)},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument,
			"qan-mysql-slowlog-agent is not supported for a Service on a remote_rds Node."), err)

		m.expectConnectionCheck()
		m.expectApplied()
		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{QanMysqlPerfschema: new(true)},
			},
		})
		require.NoError(t, err)
		assert.Contains(t, serviceAgents(t, s.db.Querier, serviceID), models.QANMySQLPerfSchemaAgentType)
	})

	t.Run("MySQLLimitsAreStoredAsAddStoresThem", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMySQL(ctx, t, s, m, &managementv1.AddMySQLServiceParams{
			Address:            "127.0.0.1",
			Port:               3306,
			QanMysqlPerfschema: true,
			MaxQueryLength:     1024,
		})

		m.expectConnectionCheck()
		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{
					TablestatsGroupTableLimit: new(int32(0)),
					QanMysqlPerfschema:        new(false),
					QanMysqlSlowlog:           new(true),
				},
			},
		})
		require.NoError(t, err)

		agents := serviceAgents(t, s.db.Querier, serviceID)
		assert.Equal(t, int32(defaultTablestatsGroupTableLimit), agents[models.MySQLdExporterType].MySQLOptions.TableCountTablestatsGroupLimit)
		assert.NotContains(t, agents, models.QANMySQLPerfSchemaAgentType)
		slowlog := agents[models.QANMySQLSlowlogAgentType]
		require.NotNil(t, slowlog)
		assert.Equal(t, int64(defaultMaxSlowlogFileSize), slowlog.QANOptions.MaxQueryLogSize)
		assert.Equal(t, int32(1024), slowlog.QANOptions.MaxQueryLength)

		m.expectApplied()
		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{
					TablestatsGroupTableLimit: new(int32(-5)),
					MaxSlowlogFileSize:        new(int64(-1)),
				},
			},
		})
		require.NoError(t, err)

		agents = serviceAgents(t, s.db.Querier, serviceID)
		assert.Equal(t, int32(-1), agents[models.MySQLdExporterType].MySQLOptions.TableCountTablestatsGroupLimit)
		assert.Equal(t, int64(0), agents[models.QANMySQLSlowlogAgentType].QANOptions.MaxQueryLogSize)
	})

	t.Run("SocketReplacesAddress", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMySQL(ctx, t, s, m, &managementv1.AddMySQLServiceParams{Address: "127.0.0.1", Port: 3306})

		m.expectConnectionCheck()
		m.expectApplied()
		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{Socket: new("/var/run/mysqld/mysqld.sock")},
			},
		})
		require.NoError(t, err)

		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Equal(t, "/var/run/mysqld/mysqld.sock", pointer.GetString(service.Socket))
		assert.Nil(t, service.Address)
		assert.Nil(t, service.Port)

		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{Address: new("10.0.0.1")},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Port is expected to be passed along with the host address."), err)
	})

	t.Run("RejectsUnsupportedExtraDSNParams", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)
		serviceID := addTestMySQL(ctx, t, s, m, &managementv1.AddMySQLServiceParams{Address: "127.0.0.1", Port: 3306})

		_, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{
					ExtraDsnParams:      &common.StringMap{Values: map[string]string{"tls": "false"}},
					SkipConnectionCheck: true,
				},
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Unsupported DSN parameter: tls"), err)
	})

	t.Run("PGStatMonitorFallsBackWithoutExtension", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)

		m.state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()
		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_Postgresql{
				Postgresql: &managementv1.AddPostgreSQLServiceParams{
					NodeId:              models.PMMServerNodeID,
					PmmAgentId:          models.PMMServerAgentID,
					ServiceName:         "postgresql-srv-1",
					Address:             "127.0.0.1",
					Port:                5432,
					Username:            "postgres",
					SkipConnectionCheck: true,
				},
			},
		})
		require.NoError(t, err)
		serviceID := resp.GetPostgresql().Service.ServiceId

		m.cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		m.sib.On("GetInfoFromService", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once().
			Run(func(args mock.Arguments) {
				args.Get(3).(*models.Agent).PostgreSQLOptions.PGSMVersion = new("")
			})
		m.expectApplied()
		updated, err := s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_Postgresql{
				Postgresql: &managementv1.UpdatePostgreSQLServiceParams{
					QanPostgresqlPgstatmonitorAgent: new(true),
				},
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, updated.Warning)

		agents := serviceAgents(t, s.db.Querier, serviceID)
		assert.Contains(t, agents, models.QANPostgreSQLPgStatementsAgentType)
		assert.NotContains(t, agents, models.QANPostgreSQLPgStatMonitorAgentType)
	})

	t.Run("ExternalListenPort", func(t *testing.T) {
		ctx, s, m := setupUpdateService(t)

		// The exporter is scraped by the Server, so no pmm-agent gets a new state.
		m.vmdb.On("RequestConfigurationUpdate").Twice()
		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_External{
				External: &managementv1.AddExternalServiceParams{
					RunsOnNodeId:        models.PMMServerNodeID,
					NodeId:              models.PMMServerNodeID,
					ServiceName:         "external-srv-1",
					ListenPort:          9104,
					CustomLabels:        map[string]string{"team": "db"},
					SkipConnectionCheck: true,
				},
			},
		})
		require.NoError(t, err)
		serviceID := resp.GetExternal().Service.ServiceId

		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_External{
				External: &managementv1.UpdateExternalServiceParams{
					ListenPort:          new(uint32(9105)),
					CustomLabels:        &common.StringMap{Values: map[string]string{"team": "ops"}},
					Group:               new("exporters"),
					SkipConnectionCheck: true,
				},
			},
		})
		require.NoError(t, err)

		service, err := models.FindServiceByID(s.db.Querier, serviceID)
		require.NoError(t, err)
		assert.Equal(t, uint16(9105), pointer.GetUint16(service.Port))
		assert.Equal(t, "exporters", service.ExternalGroup)

		exporter := serviceAgents(t, s.db.Querier, serviceID)[models.ExternalExporterType]
		assert.Equal(t, uint16(9105), pointer.GetUint16(exporter.ListenPort))
		labels, err := exporter.GetCustomLabels()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"team": "ops"}, labels)

		// An empty scheme or metrics path resets it to the default, as adding the Service does.
		_, err = s.UpdateService(ctx, &managementv1.UpdateServiceRequest{
			ServiceId: serviceID,
			Service: &managementv1.UpdateServiceRequest_External{
				External: &managementv1.UpdateExternalServiceParams{
					Scheme:              new(""),
					MetricsPath:         new(""),
					SkipConnectionCheck: true,
				},
			},
		})
		require.NoError(t, err)

		exporter = serviceAgents(t, s.db.Querier, serviceID)[models.ExternalExporterType]
		assert.Equal(t, "http", exporter.ExporterOptions.MetricsScheme)
		assert.Equal(t, "/metrics", exporter.ExporterOptions.MetricsPath)
	})
}

// A log level outside the enum would reach the exporters as "--log.level=99"; the request validation refuses it.
func TestUpdateServiceRequestLogLevelValidation(t *testing.T) {
	t.Parallel()

	request := func(level inventoryv1.LogLevel) *managementv1.UpdateServiceRequest {
		return &managementv1.UpdateServiceRequest{
			ServiceId: "service-id",
			Service: &managementv1.UpdateServiceRequest_Mysql{
				Mysql: &managementv1.UpdateMySQLServiceParams{LogLevel: &level},
			},
		}
	}

	require.NoError(t, request(inventoryv1.LogLevel_LOG_LEVEL_UNSPECIFIED).Validate())
	require.NoError(t, request(inventoryv1.LogLevel_LOG_LEVEL_DEBUG).Validate())
	require.ErrorContains(t, request(99).Validate(), "value must be one of the defined enum values")
}
