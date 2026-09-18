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
	"math"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/common"
	managementv1 "github.com/percona/pmm/api/management/v1"
	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestServer(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	newServer := func(t *testing.T) *Server {
		t.Helper()
		var r mockSupervisordService
		r.Test(t)
		r.On("UpdateConfiguration", mock.Anything).Return(nil)

		var mvmdb mockPrometheusService
		mvmdb.Test(t)
		mvmdb.On("RequestConfigurationUpdate").Return(nil)
		mState := &mockAgentsStateUpdater{}
		mState.Test(t)
		mState.On("UpdateAgentsState", t.Context()).Return(nil)
		mState.On("RequestStateUpdate", t.Context(), mock.Anything).Return(nil)

		var mvmalert mockPrometheusService
		mvmalert.Test(t)
		mvmalert.On("RequestConfigurationUpdate").Return(nil)

		var mtemplatesService mockTemplatesService
		mtemplatesService.Test(t)
		mtemplatesService.On("CollectTemplates", t.Context()).Return(nil)

		var mchecksService mockChecksService
		mchecksService.Test(t)
		mchecksService.On("UpdateAdvisorsList", t.Context()).Return(nil)

		var par mockVmAlertExternalRules
		par.Test(t)
		par.On("ReadRules").Return("", nil)

		var ts mockTelemetryService
		ts.Test(t)
		ts.On("GetSummaries").Return(nil)

		var nomad mockNomadService
		nomad.Test(t)
		nomad.On("UpdateConfiguration", mock.Anything).Return(nil)

		var ha mockHaService
		ha.Test(t)
		ha.On("IsLeader").Return(true)
		ha.On("Params").Return(&models.HAParams{Enabled: false})

		var mgrafana mockGrafanaClient
		mgrafana.Test(t)
		mgrafana.On("IsReady", mock.Anything).Return(nil)

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
			Nomad:                &nomad,
			HAService:            &ha,
			GrafanaClient:        &mgrafana,
		})
		require.NoError(t, err)
		return s
	}

	t.Run("UpdateSettingsFromEnv", func(t *testing.T) {
		t.Run("Typical", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
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
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
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
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
				"PMM_ENABLE_TELEMETRY",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `failed to parse environment variable "PMM_ENABLE_TELEMETRY"`)
			assert.Nil(t, s.envSettings.EnableTelemetry)
		})

		t.Run("InvalidValue", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
				"PMM_ENABLE_TELEMETRY=",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `invalid value "" for environment variable "PMM_ENABLE_TELEMETRY"`)
			assert.Nil(t, s.envSettings.EnableTelemetry)
		})

		t.Run("MetricsLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
				"PMM_METRICS_RESOLUTION=5ns",
			})
			require.Len(t, errs, 1)
			var errInvalidArgument *models.InvalidArgumentError
			require.ErrorAs(t, errs[0], &errInvalidArgument)
			require.EqualError(t, errs[0], `invalid argument: hr: minimal resolution is 1s`)
			assert.Zero(t, s.envSettings.MetricsResolutions.HR)
		})

		t.Run("DataRetentionLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
				"PMM_DATA_RETENTION=12h",
			})
			require.Len(t, errs, 1)
			var errInvalidArgument *models.InvalidArgumentError
			require.ErrorAs(t, errs[0], &errInvalidArgument)
			require.EqualError(t, errs[0], `invalid argument: data_retention: minimal resolution is 24h`)
			assert.Zero(t, s.envSettings.DataRetention)
		})

		t.Run("Data retention is not a natural number of days", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
				"PMM_DATA_RETENTION=30h",
			})
			require.Len(t, errs, 1)
			var errInvalidArgument *models.InvalidArgumentError
			require.ErrorAs(t, errs[0], &errInvalidArgument)
			require.EqualError(t, errs[0], `invalid argument: data_retention: should be a natural number of days`)
			assert.Zero(t, s.envSettings.DataRetention)
		})

		t.Run("Data retention without suffix", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv(t.Context(), []string{
				"PMM_DATA_RETENTION=30",
			})
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `environment variable "PMM_DATA_RETENTION=30" has invalid duration 30`)
			assert.Zero(t, s.envSettings.DataRetention)
		})
	})

	t.Run("ValidateChangeSettingsRequest", func(t *testing.T) {
		s := newServer(t)

		ctx := t.Context()

		s.envSettings.EnableUpdates = new(true)
		expected := status.New(codes.FailedPrecondition, "Updates are configured via PMM_ENABLE_UPDATES environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableUpdates: new(false),
		}))
		require.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableUpdates: new(true),
		}))

		s.envSettings.EnableTelemetry = new(true)
		expected = status.New(codes.FailedPrecondition, "Telemetry is configured via PMM_ENABLE_TELEMETRY environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableTelemetry: new(false),
		}))
		require.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableTelemetry: new(true),
		}))

		s.envSettings.EnableInternalPgQAN = new(true)
		expected = status.New(codes.FailedPrecondition, "QAN for internal PostgreSQL is already configured via an environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableInternalPgQan: new(false),
		}))
		require.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableInternalPgQan: new(true),
		}))

		require.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisor: new(false),
		}))
		require.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisor: new(true),
		}))

		s.envSettings.EnableAdvisorNotifications = new(true)
		expected = status.New(codes.FailedPrecondition, "Advisor notifications are configured via PMM_ENABLE_ADVISOR_NOTIFICATIONS environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisorNotifications: new(false),
		}))
		require.NoError(t, s.validateChangeSettingsRequest(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisorNotifications: new(true),
		}))
	})

	t.Run("ChangeSettings", func(t *testing.T) {
		server := newServer(t)

		server.UpdateSettingsFromEnv(t.Context(), []string{
			"ENABLE_ALERTING=1",
			"PMM_ENABLE_AZURE_DISCOVER=1",
		})

		ctx := t.Context()

		s, err := server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableTelemetry: new(true),
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
		server.UpdateSettingsFromEnv(t.Context(), []string{})

		ctx := t.Context()
		s, err := server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableAlerting: new(false),
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		s, err = server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableAlerting: new(true),
		})
		require.NoError(t, err)
		require.NotNil(t, s)
	})

	t.Run("ChangeSettings Advisor notifications", func(t *testing.T) {
		server := newServer(t)
		server.UpdateSettingsFromEnv(t.Context(), []string{})

		ctx := t.Context()
		s, err := server.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableAdvisorNotifications:           new(true),
			AdvisorNotificationSeverityThreshold: managementv1.Severity_SEVERITY_WARNING,
			AdvisorHistoryRetention:              durationpb.New(48 * time.Hour),
			// enabling the notifications requires at least one recipient
			AdvisorNotificationEmailAddresses: &common.StringArray{
				Values: []string{"dba@percona.com"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		settings, err := server.GetSettings(ctx, &serverv1.GetSettingsRequest{})
		require.NoError(t, err)
		assert.True(t, settings.Settings.AdvisorNotificationsEnabled)
		assert.Equal(t, managementv1.Severity_SEVERITY_WARNING, settings.Settings.AdvisorNotificationSeverityThreshold)
		assert.Equal(t, durationpb.New(48*time.Hour), settings.Settings.AdvisorHistoryRetention)
		assert.Equal(t, []string{"dba@percona.com"}, settings.Settings.AdvisorNotificationEmailAddresses)
	})
}

func TestConvertDefaultRoleID(t *testing.T) {
	tests := []struct {
		name   string
		roleID int
		want   uint32
	}{
		{
			name:   "positive",
			roleID: 1,
			want:   1,
		},
		{
			name:   "zero",
			roleID: 0,
			want:   0,
		},
		{
			name:   "negative",
			roleID: -1,
			want:   0,
		},
		{
			name:   "max uint32",
			roleID: math.MaxUint32,
			want:   math.MaxUint32,
		},
		{
			name:   "greater than max uint32",
			roleID: math.MaxUint32 + 1,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, convertDefaultRoleID(tt.roleID))
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	newServer := func(t *testing.T, initRunning bool) *Server {
		t.Helper()

		var sv mockSupervisordService
		sv.Test(t)
		sv.On("ProgramRunning", mock.Anything, pmmInitProgram).Return(initRunning)

		return &Server{
			supervisord: &sv,
			l:           logrus.WithField("component", "server-test"),
		}
	}

	t.Run("done once pmm-init is no longer running", func(t *testing.T) {
		res, err := newServer(t, false).UpdateStatus(t.Context(), &serverv1.UpdateStatusRequest{})
		require.NoError(t, err)
		assert.True(t, res.Done)
	})

	t.Run("not done while pmm-init is running", func(t *testing.T) {
		res, err := newServer(t, true).UpdateStatus(t.Context(), &serverv1.UpdateStatusRequest{})
		require.NoError(t, err)
		assert.False(t, res.Done)
	})

	t.Run("deprecated fields are ignored and left at their defaults", func(t *testing.T) {
		req := &serverv1.UpdateStatusRequest{}
		req.AuthToken = "issued-by-the-previous-instance" //nolint:staticcheck
		req.LogOffset = 1024                              //nolint:staticcheck

		res, err := newServer(t, false).UpdateStatus(t.Context(), req)
		require.NoError(t, err)
		assert.True(t, res.Done, "an unverifiable auth token must still be accepted")
		assert.Empty(t, res.LogLines, "the progress log is no longer served") //nolint:staticcheck
		assert.Zero(t, res.LogOffset)                                         //nolint:staticcheck
	})
}
