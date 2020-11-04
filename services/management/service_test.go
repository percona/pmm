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

package management

import (
	"context"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
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

func TestServiceService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, s *ServiceService, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		r := new(mockAgentsRegistry)
		r.Test(t)

		vmdb := new(mockPrometheusService)
		vmdb.Test(t)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)

			require.NoError(t, sqlDB.Close())
			r.AssertExpectations(t)
		}
		s = NewServiceService(db, r, vmdb)

		return
	}

	t.Run("Remove", func(t *testing.T) {
		t.Run("No params", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			response, err := s.RemoveService(ctx, &managementpb.RemoveServiceRequest{})
			assert.Nil(t, response)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `params not found`), err)
		})

		t.Run("Both params", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			response, err := s.RemoveService(ctx, &managementpb.RemoveServiceRequest{ServiceId: "some-id", ServiceName: "some-service-name"})
			assert.Nil(t, response)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `service_id or service_name expected; not both`), err)
		})

		t.Run("Not found", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			response, err := s.RemoveService(ctx, &managementpb.RemoveServiceRequest{ServiceName: "some-service-name"})
			assert.Nil(t, response)
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with name "some-service-name" not found.`), err)
		})

		t.Run("Wrong service type", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			response, err := s.RemoveService(ctx, &managementpb.RemoveServiceRequest{ServiceId: service.ServiceID, ServiceType: inventorypb.ServiceType_POSTGRESQL_SERVICE})
			assert.Nil(t, response)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `wrong service type`), err)
		})

		t.Run("Basic", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			service, err := models.AddNewService(s.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName: "test-mysql",
				NodeID:      models.PMMServerNodeID,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16(3306),
			})
			require.NoError(t, err)

			pmmAgent, err := models.CreatePMMAgent(s.db.Querier, models.PMMServerNodeID, nil)
			require.NoError(t, err)

			mysqldExporter, err := models.CreateAgent(s.db.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				ServiceID:  service.ServiceID,
				Password:   "password",
				Username:   "username",
				// TODO TLS
			})
			require.NoError(t, err)

			s.registry.(*mockAgentsRegistry).On("SendSetStateRequest", ctx, pmmAgent.AgentID)
			response, err := s.RemoveService(ctx, &managementpb.RemoveServiceRequest{ServiceName: service.ServiceName, ServiceType: inventorypb.ServiceType_MYSQL_SERVICE})
			assert.NotNil(t, response)
			assert.NoError(t, err)

			agent, err := models.FindAgentByID(s.db.Querier, mysqldExporter.AgentID)
			assert.Nil(t, agent)
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Agent with ID "/agent_id/00000000-0000-4000-8000-000000000007" not found.`), err)

			service, err = models.FindServiceByID(s.db.Querier, service.ServiceID)
			assert.Nil(t, service)
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "/service_id/00000000-0000-4000-8000-000000000005" not found.`), err)
		})
	})
}
