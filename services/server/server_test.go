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

package server

import (
	"context"
	"testing"
	"time"

	"github.com/percona/pmm/api/serverpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestServer(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	newServer := func(t *testing.T) *Server {
		r := new(mockSupervisordService)
		r.Test(t)
		r.On("UpdateConfiguration", mock.Anything).Return(nil)

		mvmdb := new(mockPrometheusService)
		mvmdb.Test(t)
		mvmdb.On("RequestConfigurationUpdate").Return(nil)

		mvmalert := new(mockPrometheusService)
		mvmalert.Test(t)
		mvmalert.On("RequestConfigurationUpdate").Return(nil)

		par := new(mockPrometheusAlertingRules)
		par.Test(t)
		par.On("ReadRules").Return("", nil)

		ts := new(mockTelemetryService)
		ts.Test(t)

		ps := new(mockPlatformService)
		ps.Test(t)

		s, err := NewServer(&Params{
			DB:                      reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)),
			VMDB:                    mvmdb,
			VMAlert:                 mvmalert,
			Supervisord:             r,
			PrometheusAlertingRules: par,
			TelemetryService:        ts,
			PlatformService:         ps,
		})
		require.NoError(t, err)
		return s
	}

	t.Run("UpdateSettingsFromEnv", func(t *testing.T) {
		t.Run("Typical", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DISABLE_UPDATES=true",
				"DISABLE_TELEMETRY=1",
				"METRICS_RESOLUTION_HR=1s",
				"METRICS_RESOLUTION_MR=2s",
				"METRICS_RESOLUTION_LR=3s",
				"DATA_RETENTION=240h",
			})
			require.Empty(t, errs)
			assert.Equal(t, true, s.envSettings.DisableUpdates)
			assert.Equal(t, true, s.envSettings.DisableTelemetry)
			assert.Equal(t, time.Second, s.envSettings.MetricsResolutions.HR)
			assert.Equal(t, 2*time.Second, s.envSettings.MetricsResolutions.MR)
			assert.Equal(t, 3*time.Second, s.envSettings.MetricsResolutions.LR)
			assert.Equal(t, 10*24*time.Hour, s.envSettings.DataRetention)
		})

		t.Run("Untypical", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY=TrUe",
				"METRICS_RESOLUTION=3S",
				"DATA_RETENTION=360H",
			})
			require.Empty(t, errs)
			assert.Equal(t, true, s.envSettings.DisableTelemetry)
			assert.Equal(t, 3*time.Second, s.envSettings.MetricsResolutions.HR)
			assert.Equal(t, 15*24*time.Hour, s.envSettings.DataRetention)
		})

		t.Run("NoValue", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `failed to parse environment variable "DISABLE_TELEMETRY"`)
			assert.False(t, s.envSettings.DisableTelemetry)
		})

		t.Run("InvalidValue", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY=",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `invalid value "" for environment variable "DISABLE_TELEMETRY"`)
			assert.False(t, s.envSettings.DisableTelemetry)
		})

		t.Run("MetricsLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"METRICS_RESOLUTION=5ns",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `hr: minimal resolution is 1s`)
			assert.Zero(t, s.envSettings.MetricsResolutions.HR)
		})

		t.Run("DataRetentionLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DATA_RETENTION=12h",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `data_retention: minimal resolution is 24h`)
			assert.Zero(t, s.envSettings.DataRetention)
		})

		t.Run("Data retention is not a natural number of days", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DATA_RETENTION=30h",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `data_retention: should be a natural number of days`)
			assert.Zero(t, s.envSettings.DataRetention)
		})

		t.Run("Data retention without suffix", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DATA_RETENTION=30",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `environment variable "DATA_RETENTION=30" has invalid duration 30`)
			assert.Zero(t, s.envSettings.DataRetention)
		})
	})

	t.Run("ValidateChangeSettingsRequest", func(t *testing.T) {
		s := newServer(t)

		ctx := context.TODO()

		expected := status.New(codes.InvalidArgument, "Both alert_manager_rules and remove_alert_manager_rules are present.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			AlertManagerRules:       "something",
			RemoveAlertManagerRules: true,
		}))

		s.envSettings.DisableTelemetry = true

		expected = status.New(codes.FailedPrecondition, "Telemetry is disabled via DISABLE_TELEMETRY environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			EnableTelemetry: true,
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			DisableTelemetry: true,
		}))

		expected = status.New(codes.FailedPrecondition, "STT cannot be enabled because telemetry is disabled via DISABLE_TELEMETRY environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			EnableStt: true,
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			DisableStt: true,
		}))
	})

	t.Run("ChangeSettings", func(t *testing.T) {
		server := newServer(t)

		server.UpdateSettingsFromEnv([]string{
			"PERCONA_TEST_DBAAS=1",
		})

		ctx := context.TODO()

		s, err := server.ChangeSettings(ctx, &serverpb.ChangeSettingsRequest{
			EnableTelemetry: true,
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		settings, err := server.GetSettings(ctx, new(serverpb.GetSettingsRequest))
		require.NoError(t, err)
		assert.True(t, settings.Settings.DbaasEnabled)
	})
}
