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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestServiceHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Node",
			},
			&models.Node{
				NodeID:   "N2",
				NodeType: models.GenericNodeType,
				NodeName: "Node 2",
			},

			&models.Service{
				ServiceID:   "S1",
				ServiceType: models.MongoDBServiceType,
				ServiceName: "Service without Agents",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(27017),
			},
			&models.Service{
				ServiceID:   "S2",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service with Agents",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(3306),
			},
			&models.Service{
				ServiceID:   "S21",
				ServiceType: models.ValkeyServiceType,
				ServiceName: "Standalone Valkey Service",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(6379),
			},
			&models.Service{
				ServiceID:   "S3",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Third service",
				NodeID:      "N2",
				Socket:      pointer.ToStringOrNil("/var/run/mysqld/mysqld.sock"),
			},
			&models.Service{
				ServiceID:     "S4",
				ServiceType:   models.ExternalServiceType,
				ExternalGroup: "external",
				ServiceName:   "Fourth service",
				NodeID:        "N2",
			},
			&models.Service{
				ServiceID:   "S5",
				ServiceType: models.ProxySQLServiceType,
				ServiceName: "Fifth service",
				NodeID:      "N1",
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(6032),
			},
			&models.Service{
				ServiceID:   "S6",
				ServiceType: models.ProxySQLServiceType,
				ServiceName: "Sixth service",
				NodeID:      "N2",
				Socket:      pointer.ToStringOrNil("/tmp/proxysql_admin.sock"),
			},
			&models.Service{
				ServiceID:     "S7",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Seventh service",
				NodeID:        "N2",
				Address:       pointer.ToString("127.0.0.1"),
				Port:          pointer.ToUint16OrNil(6379),
				ExternalGroup: "redis",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			&models.Service{
				ServiceID:   "S8",
				ServiceType: models.HAProxyServiceType,
				ServiceName: "Eighth service",
				NodeID:      "N2",
			},

			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
			},
			&models.Agent{
				AgentID:    "A2",
				AgentType:  models.MySQLdExporterType,
				PMMAgentID: pointer.ToString("A1"),
				ServiceID:  pointer.ToString("S2"),
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return q, teardown
	}

	t.Run("FindServices", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		services, err := models.FindServices(q, models.ServiceFilters{})
		assert.NoError(t, err)
		assert.Equal(t, 9, len(services))

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N1"})
		assert.NoError(t, err)
		assert.Equal(t, 4, len(services))
		assert.Equal(t, services, []*models.Service{{
			ServiceID:   "S1",
			ServiceType: models.MongoDBServiceType,
			ServiceName: "Service without Agents",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(27017),
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S2",
			ServiceType: models.MySQLServiceType,
			ServiceName: "Service with Agents",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(3306),
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S21",
			ServiceType: models.ValkeyServiceType,
			ServiceName: "Standalone Valkey Service",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(6379),
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S5",
			ServiceType: models.ProxySQLServiceType,
			ServiceName: "Fifth service",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(6032),
			CreatedAt:   now,
			UpdatedAt:   now,
		}})

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N1", ServiceType: pointerToServiceType(models.MySQLServiceType)})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(services))
		assert.Equal(t, services, []*models.Service{{
			ServiceID:   "S2",
			ServiceType: models.MySQLServiceType,
			ServiceName: "Service with Agents",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(3306),
			CreatedAt:   now,
			UpdatedAt:   now,
		}})

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N2", ServiceType: pointerToServiceType(models.ExternalServiceType)})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(services))
		assert.Equal(t, services, []*models.Service{
			{
				ServiceID:     "S4",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Fourth service",
				ExternalGroup: "external",
				NodeID:        "N2",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			{
				ServiceID:     "S7",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Seventh service",
				NodeID:        "N2",
				Address:       pointer.ToString("127.0.0.1"),
				Port:          pointer.ToUint16OrNil(6379),
				ExternalGroup: "redis",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		})

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N2", ServiceType: pointerToServiceType(models.ProxySQLServiceType)})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(services))
		assert.Equal(t, services, []*models.Service{{
			ServiceID:   "S6",
			ServiceType: models.ProxySQLServiceType,
			ServiceName: "Sixth service",
			Socket:      pointer.ToStringOrNil("/tmp/proxysql_admin.sock"),
			NodeID:      "N2",
			CreatedAt:   now,
			UpdatedAt:   now,
		}})

		services, err = models.FindServices(q, models.ServiceFilters{ExternalGroup: "redis"})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(services))
		assert.Equal(t, services, []*models.Service{{
			ServiceID:     "S7",
			ServiceType:   models.ExternalServiceType,
			ServiceName:   "Seventh service",
			NodeID:        "N2",
			Address:       pointer.ToString("127.0.0.1"),
			Port:          pointer.ToUint16OrNil(6379),
			ExternalGroup: "redis",
			CreatedAt:     now,
			UpdatedAt:     now,
		}})

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N2", ServiceType: pointerToServiceType(models.HAProxyServiceType)})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(services))
		assert.Equal(t, services, []*models.Service{{
			ServiceID:   "S8",
			ServiceType: models.HAProxyServiceType,
			ServiceName: "Eighth service",
			NodeID:      "N2",
			CreatedAt:   now,
			UpdatedAt:   now,
		}})
	})

	t.Run("FindActiveServiceTypes", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		types, err := models.FindActiveServiceTypes(q)
		assert.NoError(t, err)
		assert.Equal(t, len(types), 6)
	})

	t.Run("RemoveService", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		err := models.RemoveService(q, "", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Service ID.`), err)

		err = models.RemoveService(q, "S0", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "S0" not found.`), err)

		_, err = models.FindServiceByID(q, "S1")
		require.NoError(t, err)
		err = models.RemoveService(q, "S1", models.RemoveRestrict)
		assert.NoError(t, err)
		_, err = models.FindServiceByID(q, "S1")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "S1" not found.`), err)

		err = models.RemoveService(q, "S2", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Service with ID "S2" has agents.`), err)

		_, err = models.FindServiceByID(q, "S2")
		require.NoError(t, err)
		err = models.RemoveService(q, "S2", models.RemoveCascade)
		assert.NoError(t, err)
		_, err = models.FindServiceByID(q, "S2")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "S2" not found.`), err)
	})

	t.Run("MySQL Conflict socket and address", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket-address",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
			Socket:      pointer.ToString("/var/run/mysqld/mysqld.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("MySQL empty connection", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket-address",
			NodeID:      "N1",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("PostgreSQL conflict socket and address", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-postgresql-socket-address",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(5432),
			Socket:      pointer.ToString("/var/run/postgresql"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("PostgreSQL empty connection", func(t *testing.T) {
		q, teardown := setup(t)

		defer teardown(t)
		_, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-postgresql-socket-address",
			NodeID:      "N1",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("MongoDB conflict socket and address", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mongodb-socket-address",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(27017),
			Socket:      pointer.ToString("/tmp/mongodb-27017.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("MongoDB empty connection", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mongodb-socket-address",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("ProxySQL empty connection", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.ProxySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-socket-address",
			NodeID:      "N1",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("ProxySQL conflict socket and address", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.ProxySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-socket-address",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(6032),
			Socket:      pointer.ToString("/tmp/proxysql_admin.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("MongoDB find services in the same cluster", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)
		s1, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongors1",
			NodeID:      "N1",
			Cluster:     "cluster0",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(27017),
		})
		require.NoError(t, err)

		s2, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongors2",
			NodeID:      "N1",
			Cluster:     "cluster0",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(27017),
		})
		require.NoError(t, err)
		_, err = models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongors3",
			NodeID:      "N1",
			Cluster:     "cluster1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(27017),
		})
		require.NoError(t, err)

		services, err := models.FindServices(q, models.ServiceFilters{
			ServiceType: pointerToServiceType(models.MongoDBServiceType),
			Cluster:     "cluster0",
		})
		assert.NoError(t, err)
		assert.NotNil(t, services)
		assert.ElementsMatch(t, []*models.Service{s1, s2}, services)
	})

	t.Run("Change standard labels", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)
		s, err := models.AddNewService(q, models.ExternalServiceType, &models.AddDBMSServiceParams{
			ServiceName:   "mongors1",
			NodeID:        "N1",
			Cluster:       "cluster0",
			ExternalGroup: "ext",
			Address:       pointer.ToString("127.0.0.1"),
			Port:          pointer.ToUint16OrNil(27017),
		})
		require.NoError(t, err)

		err = models.ChangeStandardLabels(q, s.ServiceID, models.ServiceStandardLabelsParams{
			Cluster:        pointer.ToString("cluster"),
			Environment:    pointer.ToString("env"),
			ReplicationSet: pointer.ToString("rs"),
			ExternalGroup:  pointer.ToString("external"),
		})
		require.NoError(t, err)

		ns, err := models.FindServiceByID(q, s.ServiceID)
		require.NoError(t, err)

		assert.Equal(t, ns.Cluster, "cluster")
		assert.Equal(t, ns.Environment, "env")
		assert.Equal(t, ns.ReplicationSet, "rs")
		assert.Equal(t, ns.ExternalGroup, "external")
	})

	t.Run("Software versions record created when adding a service", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		emptyVersionsCreatedByServiceType := map[models.ServiceType]bool{
			models.MySQLServiceType:      true,
			models.MongoDBServiceType:    true,
			models.PostgreSQLServiceType: false,
		}

		s1, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mysql",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(3306),
		})
		require.NoError(t, err)

		s2, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "mongo",
			NodeID:      "N1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(27017),
		})
		require.NoError(t, err)

		s3, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "postgres",
			NodeID:      "N1",
			Cluster:     "cluster1",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(5432),
		})
		require.NoError(t, err)

		for _, service := range []*models.Service{s1, s2, s3} {
			swVersions, err := models.FindServiceSoftwareVersionsByServiceID(q, service.ServiceID)

			if emptyVersionsCreatedByServiceType[service.ServiceType] {
				require.NoError(t, err)
				assert.NotNil(t, swVersions)
				return
			}

			assert.ErrorIs(t, err, models.ErrNotFound)
			assert.Nil(t, swVersions)
		}
	})
}

func pointerToServiceType(serviceType models.ServiceType) *models.ServiceType {
	return &serviceType
}
