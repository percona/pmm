// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestServices(t *testing.T) {
	var ctx context.Context

	setup := func(t *testing.T) (ss *ServicesService, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		r := new(mockAgentsRegistry)
		r.Test(t)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)

			require.NoError(t, sqlDB.Close())
			r.AssertExpectations(t)
		}
		ss = NewServicesService(db, r)

		return
	}

	t.Run("BasicMySQL", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualMySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)
		expectedService := &inventorypb.MySQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
			ServiceName: "test-mysql",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        3306,
		}
		assert.Equal(t, expectedService, actualMySQLService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedService, actualServices[1])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("BasicMySQLWithSocket", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 1) // PMM Server PostgreSQL

		actualMySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket",
			NodeID:      models.PMMServerNodeID,
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		require.NoError(t, err)
		expectedService := &inventorypb.MySQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
			ServiceName: "test-mysql-socket",
			NodeId:      models.PMMServerNodeID,
			Socket:      "/var/run/mysqld/mysqld.sock",
		}
		assert.Equal(t, expectedService, actualMySQLService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedService, actualServices[1])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("MySQLSocketAddressConflict", func(t *testing.T) {
		ss, teardown := setup(t)
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
		ss, teardown := setup(t)
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
		ss, teardown := setup(t)
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

		expectedMongoDBService := &inventorypb.MongoDBService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
			ServiceName: "test-mongo",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        27017,
		}
		assert.Equal(t, expectedMongoDBService, actualMongoDBService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedMongoDBService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedMongoDBService, actualServices[1])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			ss, teardown := setup(t)
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
			expectedPostgreSQLService := &inventorypb.PostgreSQLService{
				ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
				ServiceName: "test-postgres",
				NodeId:      models.PMMServerNodeID,
				Address:     "127.0.0.1",
				Port:        5432,
			}
			assert.Equal(t, expectedPostgreSQLService, actualPostgreSQLService)

			actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgreSQLService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedPostgreSQLService, actualServices[1])

			err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("WithSocket", func(t *testing.T) {
			ss, teardown := setup(t)
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
			expectedPostgreSQLService := &inventorypb.PostgreSQLService{
				ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
				ServiceName: "test-postgres",
				NodeId:      models.PMMServerNodeID,
				Socket:      "/var/run/postgresql",
			}
			assert.Equal(t, expectedPostgreSQLService, actualPostgreSQLService)

			actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedPostgreSQLService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedPostgreSQLService, actualServices[1])

			err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("WithSocketAddressConflict", func(t *testing.T) {
			ss, teardown := setup(t)
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
			ss, teardown := setup(t)
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

	t.Run("BasicProxySQL", func(t *testing.T) {
		ss, teardown := setup(t)
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
		expectedProxySQLService := &inventorypb.ProxySQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
			ServiceName: "test-proxysql",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        6033,
		}
		assert.Equal(t, expectedProxySQLService, actualProxySQLService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedProxySQLService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedProxySQLService, actualServices[1])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("BasicProxySQLWithSocket", func(t *testing.T) {
		ss, teardown := setup(t)
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
		expectedService := &inventorypb.ProxySQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
			ServiceName: "test-proxysql-socket",
			NodeId:      models.PMMServerNodeID,
			Socket:      "/tmp/proxysql.sock",
		}
		assert.Equal(t, expectedService, actualProxySQLService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedService, actualServices[1])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("ProxySQLSocketAddressConflict", func(t *testing.T) {
		ss, teardown := setup(t)
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
		ss, teardown := setup(t)
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

	t.Run("BasicExternalService", func(t *testing.T) {
		ss, teardown := setup(t)
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
		expectedExternalService := &inventorypb.ExternalService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
			ServiceName: "test-external-service",
			NodeId:      models.PMMServerNodeID,
			Group:       "external",
		}
		assert.Equal(t, expectedExternalService, actualExternalService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		require.NoError(t, err)
		assert.Equal(t, expectedExternalService, actualService)

		actualServices, err = ss.List(ctx, models.ServiceFilters{NodeID: models.PMMServerNodeID})
		require.NoError(t, err)
		require.Len(t, actualServices, 2)
		assert.Equal(t, expectedExternalService, actualServices[1])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		assert.Nil(t, actualService)
	})

	t.Run("GetEmptyID", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualNode, err := ss.Get(ctx, "")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Service ID.`), err)
		assert.Nil(t, actualNode)
	})

	t.Run("AddNameNotUnique", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

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
		ss, teardown := setup(t)
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
		ss, teardown := setup(t)
		defer teardown(t)

		err := ss.Remove(ctx, "no-such-id", false)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})

	t.Run("MongoDB", func(t *testing.T) {
		t.Run("WithSocket", func(t *testing.T) {
			ss, teardown := setup(t)
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
			expectedService := &inventorypb.MongoDBService{
				ServiceId:   "/service_id/00000000-0000-4000-8000-000000000005",
				ServiceName: "test-mongodb-socket",
				NodeId:      models.PMMServerNodeID,
				Socket:      "/tmp/mongodb-27017.sock",
			}
			assert.Equal(t, expectedService, actualMongoDBService)

			actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
			require.NoError(t, err)
			assert.Equal(t, expectedService, actualService)

			actualServices, err = ss.List(ctx, models.ServiceFilters{})
			require.NoError(t, err)
			require.Len(t, actualServices, 2)
			assert.Equal(t, expectedService, actualServices[1])

			err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000005", false)
			require.NoError(t, err)
			actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000005")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
			assert.Nil(t, actualService)
		})

		t.Run("SocketAddressConflict", func(t *testing.T) {
			ss, teardown := setup(t)
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
			ss, teardown := setup(t)
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
}
