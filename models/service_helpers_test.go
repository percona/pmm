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

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestServiceHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SkipFixtures)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (q *reform.Querier, teardown func(t *testing.T)) {
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q = tx.Querier

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
			},
			&models.Service{
				ServiceID:   "S2",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service with Agents",
				NodeID:      "N1",
			},
			&models.Service{
				ServiceID:   "S3",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Third service",
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

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		return
	}

	t.Run("FindServices", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		services, err := models.FindServices(q, models.ServiceFilters{})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(services))

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N1"})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(services))
		assert.Equal(t, services, []*models.Service{{
			ServiceID:   "S1",
			ServiceType: models.MongoDBServiceType,
			ServiceName: "Service without Agents",
			NodeID:      "N1",
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S2",
			ServiceType: models.MySQLServiceType,
			ServiceName: "Service with Agents",
			NodeID:      "N1",
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
			CreatedAt:   now,
			UpdatedAt:   now,
		}})
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
}

func pointerToServiceType(serviceType models.ServiceType) *models.ServiceType {
	return &serviceType
}
