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

package server

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestServer(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)

	newServer := func(t *testing.T) *Server {
		t.Helper()
		var r mockSupervisordService
		r.Test(t)
		r.On("UpdateConfiguration", mock.Anything, mock.Anything).Return(nil)

		var mvmdb mockPrometheusService
		mvmdb.Test(t)
		mvmdb.On("RequestConfigurationUpdate").Return(nil)
		mState := &mockAgentsStateUpdater{}
		mState.Test(t)
		mState.On("UpdateAgentsState", context.TODO()).Return(nil)

		var mvmalert mockPrometheusService
		mvmalert.Test(t)
		mvmalert.On("RequestConfigurationUpdate").Return(nil)

		var mtemplatesService mockTemplatesService
		mtemplatesService.Test(t)
		mtemplatesService.On("CollectTemplates", context.TODO()).Return(nil)

		var mchecksService mockChecksService
		mchecksService.Test(t)
		mchecksService.On("CollectAdvisors", context.TODO()).Return(nil)

		var par mockVmAlertExternalRules
		par.Test(t)
		par.On("ReadRules").Return("", nil)

		var ts mockTelemetryService
		ts.Test(t)
		ts.On("GetSummaries").Return(nil)

		s, err := NewServer(&Params{
			DB:                   reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)),
			VMDB:                 &mvmdb,
			VMAlert:              &mvmalert,
			ChecksService:        &mchecksService,
			TemplatesService:     &mtemplatesService,
			AgentsStateUpdater:   mState,
			Supervisord:          &r,
			VMAlertExternalRules: &par,
			TelemetryService:     &ts,
		})
		require.NoError(t, err)
		return s
	}

	t.Run("UpdateSettingsFromEnv", func(t *testing.T) {
		t.Run("Typical", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_ENABLE_UPDATES=true",
				"PMM_ENABLE_TELEMETRY=1",
				"PMM_METRICS_RESOLUTION_HR=1s",
				"PMM_METRICS_RESOLUTION_MR=2s",
				"PMM_METRICS_RESOLUTION_LR=3s",
				"PMM_DATA_RETENTION=240h",
				"PMM_PUBLIC_ADDRESS=1.2.3.4:5678",
			})
			require.Empty(t, errs)
			assert.True(t, *s.envSettings.EnableUpdates)
			assert.True(t, *s.envSettings.EnableTelemetry)
			assert.Equal(t, time.Second, s.envSettings.MetricsResolutions.HR)
			assert.Equal(t, 2*time.Second, s.envSettings.MetricsResolutions.MR)
			assert.Equal(t, 3*time.Second, s.envSettings.MetricsResolutions.LR)
			assert.Equal(t, 10*24*time.Hour, s.envSettings.DataRetention)
			assert.Equal(t, "1.2.3.4:5678", *s.envSettings.PMMPublicAddress)
		})

		t.Run("Untypical", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_ENABLE_TELEMETRY=TrUe",
				"PMM_METRICS_RESOLUTION=3S",
				"PMM_DATA_RETENTION=360H",
			})
			require.Empty(t, errs)
			assert.True(t, *s.envSettings.EnableTelemetry)
			assert.Equal(t, 3*time.Second, s.envSettings.MetricsResolutions.HR)
			assert.Equal(t, 15*24*time.Hour, s.envSettings.DataRetention)
		})

		t.Run("NoValue", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_ENABLE_TELEMETRY",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `failed to parse environment variable "PMM_ENABLE_TELEMETRY"`)
			assert.Nil(t, s.envSettings.EnableTelemetry)
		})

		t.Run("InvalidValue", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_ENABLE_TELEMETRY=",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `invalid value "" for environment variable "PMM_ENABLE_TELEMETRY"`)
			assert.Nil(t, s.envSettings.EnableTelemetry)
		})

		t.Run("MetricsLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_METRICS_RESOLUTION=5ns",
			})
			require.Len(t, errs, 1)
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(errs[0], &errInvalidArgument))
			require.EqualError(t, errs[0], `invalid argument: hr: minimal resolution is 1s`)
			assert.Zero(t, s.envSettings.MetricsResolutions.HR)
		})

		t.Run("DataRetentionLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_DATA_RETENTION=12h",
			})
			require.Len(t, errs, 1)
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(errs[0], &errInvalidArgument))
			require.EqualError(t, errs[0], `invalid argument: data_retention: minimal resolution is 24h`)
			assert.Zero(t, s.envSettings.DataRetention)
		})

		t.Run("Data retention is not a natural number of days", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_DATA_RETENTION=30h",
			})
			require.Len(t, errs, 1)
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(errs[0], &errInvalidArgument))
			require.EqualError(t, errs[0], `invalid argument: data_retention: should be a natural number of days`)
			assert.Zero(t, s.envSettings.DataRetention)
		})

		t.Run("Data retention without suffix", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"PMM_DATA_RETENTION=30",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `environment variable "PMM_DATA_RETENTION=30" has invalid duration 30`)
			assert.Zero(t, s.envSettings.DataRetention)
		})
	})

	t.Run("ValidateChangeSettingsRequest", func(t *testing.T) {
		s := newServer(t)

		ctx := context.TODO()

		s.envSettings.EnableUpdates = pointer.ToBool(true)
		expected := status.New(codes.FailedPrecondition, "Updates are configured via PMM_ENABLE_UPDATES environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableUpdates: pointer.ToBool(false),
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableUpdates: pointer.ToBool(true),
		}))

		s.envSettings.EnableTelemetry = pointer.ToBool(true)
		expected = status.New(codes.FailedPrecondition, "Telemetry is configured via PMM_ENABLE_TELEMETRY environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableTelemetry: pointer.ToBool(false),
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableTelemetry: pointer.ToBool(true),
		}))

		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisor: pointer.ToBool(false),
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisor: pointer.ToBool(true),
		}))
	})

	t.Run("ChangeSettings", func(t *testing.T) {
		server := newServer(t)

		server.UpdateSettingsFromEnv([]string{
			"ENABLE_ALERTING=1",
			"PMM_ENABLE_AZURE_DISCOVER=1",
		})

		ctx := context.TODO()

		s, err := server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableTelemetry: pointer.ToBool(true),
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		settings, err := server.GetSettings(ctx, &serverv1.GetSettingsRequest{})

		require.NoError(t, err)
		assert.True(t, settings.Settings.AlertingEnabled)
		assert.True(t, settings.Settings.AzurediscoverEnabled)
	})

	t.Run("ChangeSettings Alerting", func(t *testing.T) {
		server := newServer(t)
		server.UpdateSettingsFromEnv([]string{})

		ctx := context.TODO()
		s, err := server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableAlerting: pointer.ToBool(false),
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		s, err = server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableAlerting: pointer.ToBool(true),
		})
		require.NoError(t, err)
		require.NotNil(t, s)
	})
}
