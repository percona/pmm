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
	"fmt"
	"reflect"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	commonv1 "github.com/percona/pmm/api/common"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/management/common"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func setup(t *testing.T) (*ServicesService, *AgentsService, *NodesService, func(t *testing.T), context.Context, *mockPrometheusService) {
	t.Helper()

	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	r := &mockAgentsRegistry{}
	r.Test(t)

	vmdb := &mockPrometheusService{}
	vmdb.Test(t)

	state := &mockAgentsStateUpdater{}
	state.Test(t)

	cc := &mockConnectionChecker{}
	cc.Test(t)

	vc := &mockVersionCache{}
	vc.Test(t)

	as := &mockAgentService{}
	as.Test(t)

	sib := &mockServiceInfoBroker{}
	sib.Test(t)

	mgmtServices := &common.MgmtServices{
		BackupService:  nil, // FIXME: &backup.mockBackupService{} is not public
		RestoreService: nil, // FIXME: &backup.mockRestoreService{} does not exist
	}

	teardown := func(t *testing.T) {
		t.Helper()
		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())

		r.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		state.AssertExpectations(t)
		cc.AssertExpectations(t)
		sib.AssertExpectations(t)
	}

	return NewServicesService(db, r, state, vmdb, vc, mgmtServices),
		NewAgentsService(db, r, state, vmdb, cc, sib, as),
		NewNodesService(db, r, state, vmdb),
		teardown,
		logger.Set(context.Background(), t.Name()),
		vmdb
}

func TestServices(t *testing.T) {
	t.Run("BasicMySQL", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		actualMySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)
		expectedService := &inventoryv1.MySQLService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-mysql",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        3306,
		}
		assert.Equal(t, expectedService, actualMySQLService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("RDSServiceRemoving", func(t *testing.T) {
		ss, as, ns, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server")
		as.vmdb.(*mockPrometheusService).On("RequestConfigurationUpdate")
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		node, err := ns.AddRemoteRDSNode(ctx, &inventoryv1.AddRemoteRDSNodeParams{NodeName: "test1", Region: "test-region", Address: "test"})
		require.NoError(t, err)

		rdsAgent, err := as.AddRDSExporter(ctx, &inventoryv1.AddRDSExporterParams{
			PmmAgentId:   "pmm-server",
			NodeId:       node.NodeId,
			AwsAccessKey: "EXAMPLE_ACCESS_KEY",
			AwsSecretKey: "EXAMPLE_SECRET_KEY",
			PushMetrics:  true,
		})
		require.NoError(t, err)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		mySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket",
			NodeID:      node.NodeId,
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		require.NoError(t, err)

		mySQLAgent, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
			PmmAgentId: "pmm-server",
			ServiceId:  mySQLService.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)

		err = ss.Remove(ctx, mySQLService.ServiceId, true)
		require.NoError(t, err)

		_, err = ss.Get(ctx, mySQLService.ServiceId)
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Service with ID "%s" not found.`, mySQLService.ServiceId)), err)

		_, err = as.Get(ctx, rdsAgent.GetRdsExporter().AgentId)
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Agent with ID %s not found.`, rdsAgent.GetRdsExporter().AgentId)), err)

		_, err = as.Get(ctx, mySQLAgent.GetMysqldExporter().AgentId)
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Agent with ID %s not found.`, mySQLAgent.GetMysqldExporter().AgentId)), err)

		_, err = ns.Get(ctx, &inventoryv1.GetNodeRequest{NodeId: node.NodeId})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Node with ID "%s" not found.`, node.NodeId)), err)
	})

	t.Run("AzureServiceRemoving", func(t *testing.T) {
		ss, as, ns, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		as.vmdb.(*mockPrometheusService).On("RequestConfigurationUpdate")
		as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server").Times(0)
		as.cc.(*mockConnectionChecker).On("CheckConnectionToService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)
		as.sib.(*mockServiceInfoBroker).On("GetInfoFromService", ctx,
			mock.AnythingOfType(reflect.TypeOf(&reform.TX{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Service{}).Name()),
			mock.AnythingOfType(reflect.TypeOf(&models.Agent{}).Name())).Return(nil)

		node, err := ns.AddRemoteAzureDatabaseNode(ctx, &inventoryv1.AddRemoteAzureNodeParams{NodeName: "test1", Region: "test-region", Address: "test"})
		require.NoError(t, err)

		azureAgent, err := as.AddAzureDatabaseExporter(ctx, &inventoryv1.AddAzureDatabaseExporterParams{
			PmmAgentId:    "pmm-server",
			NodeId:        node.NodeId,
			PushMetrics:   true,
			AzureClientId: "test",
		})
		require.NoError(t, err)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		mySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket",
			NodeID:      node.NodeId,
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		require.NoError(t, err)

		mySQLAgent, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
			PmmAgentId: "pmm-server",
			ServiceId:  mySQLService.ServiceId,
			Username:   "username",
		})
		require.NoError(t, err)

		err = ss.Remove(ctx, mySQLService.ServiceId, true)
		require.NoError(t, err)

		_, err = ss.Get(ctx, mySQLService.ServiceId)
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Service with ID "%s" not found.`, mySQLService.ServiceId)), err)

		_, err = as.Get(ctx, azureAgent.GetAzureDatabaseExporter().AgentId)
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Agent with ID %s not found.`, azureAgent.GetAzureDatabaseExporter().AgentId)), err)

		_, err = as.Get(ctx, mySQLAgent.GetMysqldExporter().AgentId)
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Agent with ID %s not found.`, mySQLAgent.GetMysqldExporter().AgentId)), err)

		_, err = ns.Get(ctx, &inventoryv1.GetNodeRequest{NodeId: node.NodeId})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf(`Node with ID "%s" not found.`, node.NodeId)), err)
	})

	t.Run("BasicMySQLWithSocket", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		actualMySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket",
			NodeID:      models.PMMServerNodeID,
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		require.NoError(t, err)
		expectedService := &inventoryv1.MySQLService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-mysql-socket",
			NodeId:      models.PMMServerNodeID,
			Socket:      "/var/run/mysqld/mysqld.sock",
		}
		assert.Equal(t, expectedService, actualMySQLService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("MySQLSocketAddressConflict", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		_, err = ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket-conflict",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("MySQLSocketAndPort", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		_, err = ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-invalid-port",
			NodeID:      models.PMMServerNodeID,
			Port:        pointer.ToUint16(3306),
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and port cannot be specified together.`), err)
	})

	t.Run("BasicMongoDB", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualMongoDBService, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
		})
		assert.NoError(t, err)

		expectedMongoDBService := &inventoryv1.MongoDBService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-mongo",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        27017,
		}
		assert.Equal(t, expectedMongoDBService, actualMongoDBService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedMongoDBService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedMongoDBService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			actualPostgreSQLService, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(5432),
			})
			require.NoError(t, err)
			expectedPostgreSQLService := &inventoryv1.PostgreSQLService{
				ServiceId:    "00000000-0000-4000-8000-000000000005",
				ServiceName:  "test-postgres",
				DatabaseName: "postgres",
				NodeId:       models.PMMServerNodeID,
				Address:      "127.0.0.1",
				Port:         5432,
			}
			assert.Equal(t, expectedPostgreSQLService, actualPostgreSQLService)

			actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgreSQLService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedPostgreSQLService, actualServices[1])

			err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("WithSocket", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			actualPostgreSQLService, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/var/run/postgresql"),
			})
			require.NoError(t, err)
			expectedPostgreSQLService := &inventoryv1.PostgreSQLService{
				ServiceId:    "00000000-0000-4000-8000-000000000005",
				ServiceName:  "test-postgres",
				DatabaseName: "postgres",
				NodeId:       models.PMMServerNodeID,
				Socket:       "/var/run/postgresql",
			}
			assert.Equal(t, expectedPostgreSQLService, actualPostgreSQLService)

			actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgreSQLService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedPostgreSQLService, actualServices[1])

			err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("WithSocketAddressConflict", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			_, err = ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/var/run/postgresql"),
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(5432),
			})

			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Socket and address cannot be specified together."), err)
		})

		t.Run("WithSocketAndPort", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			_, err = ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/var/run/postgresql"),
				Port:        pointer.ToUint16(5432),
			})

			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Socket and port cannot be specified together."), err)
		})
	})

	/*t.Run("Valkey", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			actualPostgreSQLService, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(5432),
			})
			require.NoError(t, err)
			expectedPostgreSQLService := &inventoryv1.PostgreSQLService{
				ServiceId:    "00000000-0000-4000-8000-000000000005",
				ServiceName:  "test-postgres",
				DatabaseName: "postgres",
				NodeId:       models.PMMServerNodeID,
				Address:      "127.0.0.1",
				Port:         5432,
			}
			assert.Equal(t, expectedPostgreSQLService, actualPostgreSQLService)

			actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgreSQLService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedPostgreSQLService, actualServices[1])

			err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("WithSocket", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			actualPostgreSQLService, err := ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/var/run/postgresql"),
			})
			require.NoError(t, err)
			expectedPostgreSQLService := &inventoryv1.PostgreSQLService{
				ServiceId:    "00000000-0000-4000-8000-000000000005",
				ServiceName:  "test-postgres",
				DatabaseName: "postgres",
				NodeId:       models.PMMServerNodeID,
				Socket:       "/var/run/postgresql",
			}
			assert.Equal(t, expectedPostgreSQLService, actualPostgreSQLService)

			actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgreSQLService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedPostgreSQLService, actualServices[1])

			err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("WithSocketAddressConflict", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			_, err = ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/var/run/postgresql"),
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(5432),
			})

			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Socket and address cannot be specified together."), err)
		})

		t.Run("WithSocketAndPort", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			_, err = ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-postgres",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/var/run/postgresql"),
				Port:        pointer.ToUint16(5432),
			})

			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Socket and port cannot be specified together."), err)
		})
	})*/

	t.Run("BasicProxySQL", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualProxySQLService, err := ss.AddProxySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(6033),
		})
		require.NoError(t, err)
		expectedProxySQLService := &inventoryv1.ProxySQLService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-proxysql",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        6033,
		}
		assert.Equal(t, expectedProxySQLService, actualProxySQLService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedProxySQLService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedProxySQLService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("BasicProxySQLWithSocket", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualProxySQLService, err := ss.AddProxySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-socket",
			NodeID:      models.PMMServerNodeID,
			Socket:      pointer.ToString("/tmp/proxysql.sock"),
		})
		require.NoError(t, err)
		expectedService := &inventoryv1.ProxySQLService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-proxysql-socket",
			NodeId:      models.PMMServerNodeID,
			Socket:      "/tmp/proxysql.sock",
		}
		assert.Equal(t, expectedService, actualProxySQLService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("ProxySQLSocketAddressConflict", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		_, err = ss.AddProxySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-socket-conflict",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Socket:      pointer.ToString("/tmp/proxysql.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("ProxySQLSocketAndPort", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		_, err = ss.AddProxySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-invalid-port",
			NodeID:      models.PMMServerNodeID,
			Port:        pointer.ToUint16(6033),
			Socket:      pointer.ToString("/tmp/proxysql.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and port cannot be specified together.`), err)
	})

	t.Run("BasicHAProxyService", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualHAProxyService, err := ss.AddHAProxyService(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-haproxy-service",
			NodeID:      models.PMMServerNodeID,
		})
		require.NoError(t, err)
		expectedHAProxyService := &inventoryv1.HAProxyService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-haproxy-service",
			NodeId:      models.PMMServerNodeID,
		}
		assert.Equal(t, expectedHAProxyService, actualHAProxyService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedHAProxyService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedHAProxyService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("BasicExternalService", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualExternalService, err := ss.AddExternalService(ctx, &models.AddDBMSServiceParams{
			ServiceName:   "test-external-service",
			NodeID:        models.PMMServerNodeID,
			ExternalGroup: "external",
		})
		require.NoError(t, err)
		expectedExternalService := &inventoryv1.ExternalService{
			ServiceId:   "00000000-0000-4000-8000-000000000005",
			ServiceName: "test-external-service",
			NodeId:      models.PMMServerNodeID,
			Group:       "external",
		}
		assert.Equal(t, expectedExternalService, actualExternalService)

		actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedExternalService, actualServices[1])

		err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		actualNode, err := ss.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Service ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		ss.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		_, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Service with name "test-mysql" already exists.`), err)
	})

	t.Run("AddNodeNotFound", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		_, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      "no-such-id",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "no-such-id" not found.`), err)
	})

	t.Run("RemoveNotFound", func(t *testing.T) {
		ss, _, _, teardown, ctx, _ := setup(t)
		defer teardown(t)

		err := ss.Remove(ctx, "no-such-id", false)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("MongoDB", func(t *testing.T) {
		t.Run("WithSocket", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			actualMongoDBService, err := ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-mongodb-socket",
				NodeID:      models.PMMServerNodeID,
				Socket:      pointer.ToString("/tmp/mongodb-27017.sock"),
			})
			require.NoError(t, err)
			expectedService := &inventoryv1.MongoDBService{
				ServiceId:   "00000000-0000-4000-8000-000000000005",
				ServiceName: "test-mongodb-socket",
				NodeId:      models.PMMServerNodeID,
				Socket:      "/tmp/mongodb-27017.sock",
			}
			assert.Equal(t, expectedService, actualMongoDBService)

			actualService, err := ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedService, actualServices[1])

			err = ss.Remove(ctx, "00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("SocketAddressConflict", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			_, err = ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-mongodb-socket-conflict",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Socket:      pointer.ToString("/tmp/mongodb-27017.sock"),
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
		})

		t.Run("SocketAndPort", func(t *testing.T) {
			ss, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			actualServices, err := ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 1) // PMM Server PostgreSQL

			_, err = ss.AddProxySQL(ctx, &models.AddDBMSServiceParams{
				ServiceName: "test-mongodb-invalid-port",
				NodeID:      models.PMMServerNodeID,
				Port:        pointer.ToUint16(27017),
				Socket:      pointer.ToString("/tmp/mongodb-27017.sock"),
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and port cannot be specified together.`), err)
		})
	})

	t.Run("AddCustomLabels", func(t *testing.T) {
		t.Run("No Service ID", func(t *testing.T) {
			t.Skip("TODO: fix")
			s, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			response, err := s.ChangeService(ctx, &models.ChangeStandardLabelsParams{}, nil)
			assert.Nil(t, response)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Empty Service ID."), err)
		})

		t.Run("Add a label", func(t *testing.T) {
			t.Skip("FIXME: fix")
			s, _, _, teardown, ctx, vmdb := setup(t)
			defer teardown(t)

			vmdb.Mock.On("RequestConfigurationUpdate").Once().Return()

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			response, err := s.ChangeService(
				ctx,
				&models.ChangeStandardLabelsParams{
					ServiceID: service.ServiceID,
				},
				&commonv1.StringMap{
					Values: map[string]string{
						"newKey":  "newValue",
						"newKey2": "newValue2",
					},
				},
			)
			assert.NotNil(t, response)
			assert.NoError(t, err)

			service, err = models.FindServiceByID(s.db.Querier, service.ServiceID)
			assert.NoError(t, err)
			assert.NotNil(t, service)

			labels, err := service.GetCustomLabels()
			assert.NoError(t, err)
			assert.Equal(t, len(labels), 2)
			assert.Equal(t, labels["newKey"], "newValue")
			assert.Equal(t, labels["newKey2"], "newValue2")
		})

		t.Run("Replace a label", func(t *testing.T) {
			t.Skip("FIXME: fix")
			s, _, _, teardown, ctx, vmdb := setup(t)
			defer teardown(t)

			vmdb.Mock.On("RequestConfigurationUpdate").Once().Return()

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
				CustomLabels: map[string]string{
					"newKey":  "newValue",
					"newKey2": "newValue2",
				},
			})
			require.NoError(t, err)

			_, err = s.ChangeService(
				ctx,
				&models.ChangeStandardLabelsParams{
					ServiceID: service.ServiceID,
				},
				&commonv1.StringMap{
					Values: map[string]string{
						"newKey2": "newValue-replaced",
					},
				},
			)

			assert.NoError(t, err)

			service, err = models.FindServiceByID(s.db.Querier, service.ServiceID)
			assert.NoError(t, err)
			assert.NotNil(t, service)

			labels, err := service.GetCustomLabels()
			assert.NoError(t, err)
			assert.Equal(t, len(labels), 2)
			assert.Equal(t, labels["newKey"], "newValue")
			assert.Equal(t, labels["newKey2"], "newValue-replaced")
		})
	})

	t.Run("RemoveCustomLabels", func(t *testing.T) {
		t.Run("No Service ID", func(t *testing.T) {
			s, _, _, teardown, ctx, _ := setup(t)
			defer teardown(t)

			service, err := s.ChangeService(ctx, &models.ChangeStandardLabelsParams{}, nil)
			assert.Nil(t, service)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Empty Service ID."), err)
		})

		t.Run("Remove a label", func(t *testing.T) {
			t.Skip("FIXME: fix")
			s, _, _, teardown, ctx, vmdb := setup(t)
			defer teardown(t)

			vmdb.Mock.On("RequestConfigurationUpdate").Once().Return()

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
				CustomLabels: map[string]string{
					"newKey":  "newValue",
					"newKey2": "newValue2",
					"newKey3": "newValue3",
				},
			})
			require.NoError(t, err)

			_, err = s.ChangeService(
				ctx,
				&models.ChangeStandardLabelsParams{
					ServiceID: service.ServiceID,
				},
				nil)
			assert.NoError(t, err)

			service, err = models.FindServiceByID(s.db.Querier, service.ServiceID)
			assert.NoError(t, err)
			assert.NotNil(t, service)

			labels, err := service.GetCustomLabels()
			assert.NoError(t, err)
			assert.Equal(t, len(labels), 1)
			assert.Equal(t, labels["newKey3"], "newValue3")
		})
	})
}
