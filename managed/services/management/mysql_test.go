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

	"github.com/google/uuid"
	"github.com/percona/pmm/api/managementpb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func setup(t *testing.T) (*MySQLService, func(t *testing.T), context.Context) {
	t.Helper()

	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	agentsRegistry := &mockAgentsRegistry{}
	agentsRegistry.Test(t)

	agentsStateUpdater := &mockAgentsStateUpdater{}
	agentsStateUpdater.Test(t)

	connectionChecker := &mockConnectionChecker{}
	connectionChecker.Test(t)

	defaultsFileParser := &mockDefaultsFileParser{}
	defaultsFileParser.Test(t)

	versionCache := &mockVersionCache{}
	versionCache.Test(t)

	teardown := func(t *testing.T) {
		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())

		agentsRegistry.AssertExpectations(t)
		agentsStateUpdater.AssertExpectations(t)
		connectionChecker.AssertExpectations(t)
		defaultsFileParser.AssertExpectations(t)
		versionCache.AssertExpectations(t)
	}

	return NewMySQLService(db, agentsStateUpdater, connectionChecker, versionCache, defaultsFileParser),
		teardown,
		logger.Set(context.Background(), t.Name())
}

func TestDefaultsFileParserRequest(t *testing.T) {
	t.Run("Add MySQLService defaults file test", func(t *testing.T) {
		service, teardown, ctx := setup(t)
		defer teardown(t)

		service.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server").Once()
		service.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		service.dfp.(*mockDefaultsFileParser).On("ParseDefaultsFile", ctx, mock.Anything, "/file/path", models.MySQLServiceType).Return(&models.ParseDefaultsFileResult{
			Username: "test",
			Password: "test",
			Host:     "192.168.2.1",
			Port:     6666,
		}, nil).Once()

		req := &managementpb.AddMySQLRequest{
			NodeId:              models.PMMServerNodeID,
			PmmAgentId:          models.PMMServerAgentID,
			ServiceName:         "test",
			DefaultsFile:        "/file/path",
			SkipConnectionCheck: true,
		}
		res, err := service.Add(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)

		require.Equal(t, res.Service.Address, "192.168.2.1")
		require.Equal(t, res.Service.Port, uint32(6666))
		require.Equal(t, res.MysqldExporter.Username, "test")
	})

	t.Run("Add MySQLService defaults file with overriden params", func(t *testing.T) {
		service, teardown, ctx := setup(t)
		defer teardown(t)

		service.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, "pmm-server").Once()
		service.vc.(*mockVersionCache).On("RequestSoftwareVersionsUpdate").Once()
		service.dfp.(*mockDefaultsFileParser).On("ParseDefaultsFile", ctx, mock.Anything, "/file/path", models.MySQLServiceType).Return(&models.ParseDefaultsFileResult{
			Username: "test",
			Socket:   "socks4://localhost",
		}, nil).Once()

		req := &managementpb.AddMySQLRequest{
			NodeId:              models.PMMServerNodeID,
			PmmAgentId:          models.PMMServerAgentID,
			ServiceName:         "test overriden",
			DefaultsFile:        "/file/path",
			Username:            "overriden",
			SkipConnectionCheck: true,
		}
		res, err := service.Add(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)

		require.Equal(t, res.Service.Socket, "socks4://localhost")
		require.Equal(t, res.MysqldExporter.Username, "overriden")
	})

	t.Run("Add MySQLService defaults file parse error", func(t *testing.T) {
		service, teardown, ctx := setup(t)
		defer teardown(t)

		service.dfp.(*mockDefaultsFileParser).On("ParseDefaultsFile", ctx, mock.Anything, "/file/path", models.MySQLServiceType).Return(nil, errors.New("dfp error")).Once()

		req := &managementpb.AddMySQLRequest{
			NodeId:              models.PMMServerNodeID,
			PmmAgentId:          models.PMMServerAgentID,
			ServiceName:         "not used",
			DefaultsFile:        "/file/path",
			Username:            "overriden",
			SkipConnectionCheck: true,
		}
		res, err := service.Add(ctx, req)
		require.Error(t, err, "dfp error")
		require.Nil(t, res)
	})
}
