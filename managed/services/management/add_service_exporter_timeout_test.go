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
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

// Verifies management AddService paths persist ExporterOptions.Timeout (dial / scrape timeout API).
func TestAddServiceExporterTimeout(t *testing.T) {
	uuid.SetRand(&tests.IDReader{})
	defer uuid.SetRand(nil)

	ctx := logger.Set(context.Background(), t.Name())
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() { sqlDB.Close() }) //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	cc := &mockConnectionChecker{}
	cc.Test(t)
	sib := &mockServiceInfoBroker{}
	sib.Test(t)
	state := &mockAgentsStateUpdater{}
	state.Test(t)
	ar := &mockAgentsRegistry{}
	ar.Test(t)
	vmdb := &mockPrometheusService{}
	vmdb.Test(t)
	vc := &mockVersionCache{}
	vc.Test(t)
	grafanaClient := &mockGrafanaClient{}
	grafanaClient.Test(t)
	vmClient := &mockVictoriaMetricsClient{}
	vmClient.Test(t)

	t.Cleanup(func() {
		cc.AssertExpectations(t)
		sib.AssertExpectations(t)
		state.AssertExpectations(t)
		ar.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		vc.AssertExpectations(t)
		grafanaClient.AssertExpectations(t)
		vmClient.AssertExpectations(t)
	})

	s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)
	want := durationpb.New(17 * time.Second)

	t.Run("MySQL", func(t *testing.T) {
		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()
		vc.On("RequestSoftwareVersionsUpdate").Once()

		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_Mysql{
				Mysql: &managementv1.AddMySQLServiceParams{
					NodeId:              models.PMMServerNodeID,
					ServiceName:         "mgmt-test-mysql-timeout",
					Address:             "127.0.0.1",
					Port:                3306,
					PmmAgentId:          models.PMMServerAgentID,
					Username:            "root",
					SkipConnectionCheck: true,
					MetricsMode:         managementv1.MetricsMode_METRICS_MODE_PULL,
					ConnectionTimeout:   want,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetMysql())
		assert.Equal(t, want, resp.GetMysql().GetMysqldExporter().GetConnectionTimeout())
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()

		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_Postgresql{
				Postgresql: &managementv1.AddPostgreSQLServiceParams{
					NodeId:              models.PMMServerNodeID,
					ServiceName:         "mgmt-test-pg-timeout",
					Address:             "127.0.0.1",
					Port:                5432,
					Database:            "postgres",
					PmmAgentId:          models.PMMServerAgentID,
					Username:            "postgres",
					SkipConnectionCheck: true,
					MetricsMode:         managementv1.MetricsMode_METRICS_MODE_PULL,
					ConnectionTimeout:   want,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetPostgresql())
		assert.Equal(t, want, resp.GetPostgresql().GetPostgresExporter().GetConnectionTimeout())
	})

	t.Run("ProxySQL", func(t *testing.T) {
		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()

		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_Proxysql{
				Proxysql: &managementv1.AddProxySQLServiceParams{
					NodeId:              models.PMMServerNodeID,
					ServiceName:         "mgmt-test-px-timeout",
					Address:             "127.0.0.1",
					Port:                6033,
					PmmAgentId:          models.PMMServerAgentID,
					Username:            "admin",
					SkipConnectionCheck: true,
					MetricsMode:         managementv1.MetricsMode_METRICS_MODE_PULL,
					ConnectionTimeout:   want,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetProxysql())
		assert.Equal(t, want, resp.GetProxysql().GetProxysqlExporter().GetConnectionTimeout())
	})

	t.Run("Valkey", func(t *testing.T) {
		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()

		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_Valkey{
				Valkey: &managementv1.AddValkeyServiceParams{
					NodeId:              models.PMMServerNodeID,
					ServiceName:         "mgmt-test-valkey-timeout",
					Address:             "127.0.0.1",
					Port:                6379,
					PmmAgentId:          models.PMMServerAgentID,
					SkipConnectionCheck: true,
					MetricsMode:         managementv1.MetricsMode_METRICS_MODE_PULL,
					ConnectionTimeout:   want,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetValkey())
		assert.Equal(t, want, resp.GetValkey().GetValkeyExporter().GetConnectionTimeout())
	})

	t.Run("External", func(t *testing.T) {
		vmdb.On("RequestConfigurationUpdate").Once()

		resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
			Service: &managementv1.AddServiceRequest_External{
				External: &managementv1.AddExternalServiceParams{
					NodeId:              models.PMMServerNodeID,
					RunsOnNodeId:        models.PMMServerNodeID,
					ServiceName:         "mgmt-test-external-timeout",
					Address:             "127.0.0.1",
					ListenPort:          42000,
					Scheme:              "http",
					MetricsPath:         "/metrics",
					SkipConnectionCheck: true,
					MetricsMode:         managementv1.MetricsMode_METRICS_MODE_PULL,
					ConnectionTimeout:   want,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetExternal())
		assert.Equal(t, want, resp.GetExternal().GetExternalExporter().GetConnectionTimeout())
	})

	t.Run("Azure Database", func(t *testing.T) {
		_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
			EnableAzurediscover: pointer.ToBool(true),
		})
		require.NoError(t, err)

		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()

		_, err = s.AddAzureDatabase(ctx, &managementv1.AddAzureDatabaseRequest{
			Region:                "westeurope",
			InstanceId:            "mgmt-test-azure-timeout-instance",
			NodeName:              "mgmt-test-azure-timeout-node",
			ServiceName:           "mgmt-test-azure-timeout",
			NodeModel:             "general-purpose",
			Address:               "127.0.0.1",
			Port:                  3306,
			Username:              "root",
			Password:              "secret",
			AzureClientId:         "client-id",
			AzureClientSecret:     "client-secret",
			AzureTenantId:         "tenant-id",
			AzureSubscriptionId:   "subscription-id",
			AzureResourceGroup:    "resource-group",
			AzureDatabaseExporter: true,
			SkipConnectionCheck:   true,
			Type:                  managementv1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_MYSQL,
			ConnectionTimeout:     want,
		})
		require.NoError(t, err)

		service, err := models.FindServiceByName(db.Querier, "mgmt-test-azure-timeout")
		require.NoError(t, err)

		agents, err := models.FindAgents(db.Querier, models.AgentFilters{ServiceID: service.ServiceID})
		require.NoError(t, err)
		require.Len(t, agents, 2)

		got := map[models.AgentType]time.Duration{}
		for _, agent := range agents {
			got[agent.AgentType] = pointer.GetDuration(agent.ExporterOptions.ConnectionTimeout)
		}

		assert.Equal(t, want.AsDuration(), got[models.AzureDatabaseExporterType])
		assert.Equal(t, want.AsDuration(), got[models.MySQLdExporterType])
	})
}
