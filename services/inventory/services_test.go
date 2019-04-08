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
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestServices(t *testing.T) {
	ctx := logger.Set(context.Background(), t.Name())

	setup := func(t *testing.T) (ss *ServicesService, teardown func(t *testing.T)) {
		uuid.SetRand(new(tests.IDReader))

		sqlDB := tests.OpenTestDB(t)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		r := new(mockRegistry)
		r.Test(t)
		teardown = func(t *testing.T) {
			require.NoError(t, sqlDB.Close())
			r.AssertExpectations(t)
		}

		ss = NewServicesService(db, r)
		return
	}

	t.Run("Basic", func(t *testing.T) {
		ss, teardown := setup(t)
		defer teardown(t)

		actualServices, err := ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 0)

		actualMySQLService, err := ss.AddMySQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)
		expectedService := &inventorypb.MySQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000001",
			ServiceName: "test-mysql",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        3306,
		}
		assert.Equal(t, expectedService, actualMySQLService)

		actualService, err := ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		assert.Equal(t, expectedService, actualService)

		actualServices, err = ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedService, actualServices[0])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000001")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000001")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000001" not found.`), err)
		assert.Nil(t, actualService)

		actualService, err = ss.AddMongoDB(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-mongo",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
		})
		require.NoError(t, err)
		expectedMdbService := &inventorypb.MongoDBService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000002",
			ServiceName: "test-mongo",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        27017,
		}
		assert.Equal(t, expectedMdbService, actualService)

		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		assert.Equal(t, expectedMdbService, actualService)

		actualServices, err = ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedMdbService, actualServices[0])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000002")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000002")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000002" not found.`), err)
		assert.Nil(t, actualService)

		actualService, err = ss.AddPostgreSQL(ctx, &models.AddDBMSServiceParams{
			ServiceName: "test-postgres",
			NodeID:      models.PMMServerNodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(5432),
		})
		require.NoError(t, err)
		expectedPostgreSQLService := &inventorypb.PostgreSQLService{
			ServiceId:   "/service_id/00000000-0000-4000-8000-000000000003",
			ServiceName: "test-postgres",
			NodeId:      models.PMMServerNodeID,
			Address:     "127.0.0.1",
			Port:        5432,
		}
		assert.Equal(t, expectedPostgreSQLService, actualService)

		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		assert.Equal(t, expectedPostgreSQLService, actualService)

		actualServices, err = ss.List(ctx)
		require.NoError(t, err)
		require.Len(t, actualServices, 1)
		assert.Equal(t, expectedPostgreSQLService, actualServices[0])

		err = ss.Remove(ctx, "/service_id/00000000-0000-4000-8000-000000000003")
		require.NoError(t, err)
		actualService, err = ss.Get(ctx, "/service_id/00000000-0000-4000-8000-000000000003")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000003" not found.`), err)
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

		err := ss.Remove(ctx, "no-such-id")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "no-such-id" not found.`), err)
	})
}
