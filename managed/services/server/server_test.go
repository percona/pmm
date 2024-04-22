// Copyright (C) 2024 Percona LLC
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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/serverpb"
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

		var malertmanager mockAlertmanagerService
		malertmanager.Test(t)
		malertmanager.On("RequestConfigurationUpdate").Return(nil)

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
			Alertmanager:         &malertmanager,
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
				"DISABLE_UPDATES=true",
				"DISABLE_TELEMETRY=1",
				"METRICS_RESOLUTION_HR=1s",
				"METRICS_RESOLUTION_MR=2s",
				"METRICS_RESOLUTION_LR=3s",
				"DATA_RETENTION=240h",
				"PMM_PUBLIC_ADDRESS=1.2.3.4:5678",
			})
			require.Empty(t, errs)
			assert.Equal(t, true, s.envSettings.DisableUpdates)
			assert.Equal(t, true, s.envSettings.DisableTelemetry)
			assert.Equal(t, time.Second, s.envSettings.MetricsResolutions.HR)
			assert.Equal(t, 2*time.Second, s.envSettings.MetricsResolutions.MR)
			assert.Equal(t, 3*time.Second, s.envSettings.MetricsResolutions.LR)
			assert.Equal(t, 10*24*time.Hour, s.envSettings.DataRetention)
			assert.Equal(t, "1.2.3.4:5678", s.envSettings.PMMPublicAddress)
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
			var errInvalidArgument *models.InvalidArgumentError
			assert.True(t, errors.As(errs[0], &errInvalidArgument))
			require.EqualError(t, errs[0], `invalid argument: hr: minimal resolution is 1s`)
			assert.Zero(t, s.envSettings.MetricsResolutions.HR)
		})

		t.Run("DataRetentionLessThenMin", func(t *testing.T) {
			s := newServer(t)
			errs := s.UpdateSettingsFromEnv([]string{
				"DATA_RETENTION=12h",
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
				"DATA_RETENTION=30h",
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

		s.envSettings.DisableUpdates = true
		expected = status.New(codes.FailedPrecondition, "Updates are disabled via DISABLE_UPDATES environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			EnableUpdates: true,
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			DisableUpdates: true,
		}))

		s.envSettings.DisableTelemetry = true
		expected = status.New(codes.FailedPrecondition, "Telemetry is disabled via DISABLE_TELEMETRY environment variable.")
		tests.AssertGRPCError(t, expected, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			EnableTelemetry: true,
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			DisableTelemetry: true,
		}))

		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			EnableStt: true,
		}))
		assert.NoError(t, s.validateChangeSettingsRequest(ctx, &serverpb.ChangeSettingsRequest{
			DisableStt: true,
		}))
	})

	t.Run("ChangeSettings", func(t *testing.T) {
		server := newServer(t)

		server.UpdateSettingsFromEnv([]string{
			"ENABLE_DBAAS=1",
			"ENABLE_ALERTING=1",
			"ENABLE_AZUREDISCOVER=1",
		})

		ctx := context.TODO()

		s, err := server.ChangeSettings(ctx, &serverpb.ChangeSettingsRequest{
			EnableTelemetry: true,
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		settings, err := server.GetSettings(ctx, &serverpb.GetSettingsRequest{})

		require.NoError(t, err)
		assert.True(t, settings.Settings.DbaasEnabled)
		assert.True(t, settings.Settings.AlertingEnabled)
		assert.True(t, settings.Settings.AzurediscoverEnabled)
	})

	t.Run("ChangeSettings IA", func(t *testing.T) {
		server := newServer(t)
		var rs mockRulesService
		server.rulesService = &rs
		server.UpdateSettingsFromEnv([]string{})

		ctx := context.TODO()
		rs.On("RemoveVMAlertRulesFiles").Return(nil)
		defer rs.AssertExpectations(t)
		s, err := server.ChangeSettings(ctx, &serverpb.ChangeSettingsRequest{
			DisableAlerting: true,
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		rs.On("WriteVMAlertRulesFiles")
		s, err = server.ChangeSettings(ctx, &serverpb.ChangeSettingsRequest{
			EnableAlerting: true,
		})
		require.NoError(t, err)
		require.NotNil(t, s)

		rs.On("RemoveVMAlertRulesFiles").Return(nil)
		s, err = server.ChangeSettings(ctx, &serverpb.ChangeSettingsRequest{
			DisableAlerting: true,
		})
		require.NoError(t, err)
		require.NotNil(t, s)
	})
}

func TestServer_TestEmailAlertingSettings(t *testing.T) { //nolint:tparallel
	t.Parallel()

	var server Server

	var e mockEmailer
	server.emailer = &e

	ctx := context.TODO()

	normalRequest := &serverpb.TestEmailAlertingSettingsRequest{
		EmailAlertingSettings: &serverpb.EmailAlertingSettings{
			From:       "me@example.com",
			Smarthost:  "example.com:465",
			Hello:      "example.com",
			Username:   "example-user",
			Password:   "some-password",
			Identity:   "example",
			Secret:     "example-secret",
			RequireTls: true,
		},
		EmailTo: "to@example.com",
	}
	eas := normalRequest.EmailAlertingSettings

	for _, tc := range []struct {
		testName string
		req      *serverpb.TestEmailAlertingSettingsRequest
		respErr  string
		mock     func()
	}{
		{
			testName: "normal",
			req:      normalRequest,
			respErr:  "",
			mock: func() {
				s := &models.EmailAlertingSettings{
					From:       eas.From,
					Smarthost:  eas.Smarthost,
					Hello:      eas.Hello,
					Username:   eas.Username,
					Password:   eas.Password,
					Identity:   eas.Identity,
					Secret:     eas.Secret,
					RequireTLS: eas.RequireTls,
				}
				e.On("Send", mock.Anything, s, normalRequest.EmailTo).Return(nil).Once()
			},
		},
		{
			testName: "failed to send: invalid argument",
			req:      normalRequest,
			respErr:  "rpc error: code = InvalidArgument desc = Cannot send email: invalid argument.",
			mock: func() {
				s := &models.EmailAlertingSettings{
					From:       eas.From,
					Smarthost:  eas.Smarthost,
					Hello:      eas.Hello,
					Username:   eas.Username,
					Password:   eas.Password,
					Identity:   eas.Identity,
					Secret:     eas.Secret,
					RequireTLS: eas.RequireTls,
				}
				e.On("Send", mock.Anything, s, normalRequest.EmailTo).
					Return(models.NewInvalidArgumentError("invalid argument")).Once()
			},
		},
		{
			testName: "invalid argument: from",
			respErr: "rpc error: code = InvalidArgument desc = " +
				"Invalid argument: invalid \"from\" email \"invalid-from\".",
			req: &serverpb.TestEmailAlertingSettingsRequest{
				EmailAlertingSettings: &serverpb.EmailAlertingSettings{
					From:       "invalid-from",
					Smarthost:  eas.Smarthost,
					Hello:      eas.Hello,
					Username:   eas.Username,
					Password:   eas.Password,
					Identity:   eas.Identity,
					Secret:     eas.Secret,
					RequireTls: eas.RequireTls,
				},
				EmailTo: normalRequest.EmailTo,
			},
		},
		{
			testName: "invalid argument: smarthost",
			respErr: "rpc error: code = InvalidArgument desc = " +
				"Invalid argument: invalid server address, expected format host:port.",
			req: &serverpb.TestEmailAlertingSettingsRequest{
				EmailAlertingSettings: &serverpb.EmailAlertingSettings{
					From:       eas.From,
					Smarthost:  "invalid-smart-host",
					Hello:      eas.Hello,
					Username:   eas.Username,
					Password:   eas.Password,
					Identity:   eas.Identity,
					Secret:     eas.Secret,
					RequireTls: eas.RequireTls,
				},
				EmailTo: normalRequest.EmailTo,
			},
		},
		{
			testName: "invalid argument: hello",
			respErr: "rpc error: code = InvalidArgument desc = " +
				"Invalid argument: invalid hello field, expected valid host.",
			req: &serverpb.TestEmailAlertingSettingsRequest{
				EmailAlertingSettings: &serverpb.EmailAlertingSettings{
					From:       eas.From,
					Smarthost:  eas.Smarthost,
					Hello:      "@invalid hello",
					Username:   eas.Username,
					Password:   eas.Password,
					Identity:   eas.Identity,
					Secret:     eas.Secret,
					RequireTls: eas.RequireTls,
				},
				EmailTo: normalRequest.EmailTo,
			},
		},
		{
			testName: "invalid argument: emailTo",
			respErr:  "rpc error: code = InvalidArgument desc = invalid \"emailTo\" email \"invalid email\"",
			req: &serverpb.TestEmailAlertingSettingsRequest{
				EmailAlertingSettings: &serverpb.EmailAlertingSettings{
					From:       eas.From,
					Smarthost:  eas.Smarthost,
					Hello:      eas.Hello,
					Username:   eas.Username,
					Password:   eas.Password,
					Identity:   eas.Identity,
					Secret:     eas.Secret,
					RequireTls: eas.RequireTls,
				},
				EmailTo: "invalid email",
			},
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.mock != nil {
				tc.mock()
			}
			resp, err := server.TestEmailAlertingSettings(ctx, tc.req)
			if tc.respErr != "" {
				assert.Nil(t, resp)
				assert.EqualError(t, err, tc.respErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}

	mock.AssertExpectationsForObjects(t, &e)
}
