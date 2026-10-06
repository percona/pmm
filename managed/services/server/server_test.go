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
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
	"github.com/percona/pmm/managed/utils/env"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

// newTestServer builds a Server with every dependency mocked, for tests that drive the real
// GetSettings/ChangeSettings paths. It returns the state updater alongside the Server so a caller
// can assert on the pmm-agent that gets signalled: the only one this package ever signals is PMM
// Server's own, and passing the changed agent's own ID instead was the bug fixed here, so the
// expectation is pinned to models.PMMServerAgentID and any other argument panics the test.
func newTestServer(t *testing.T, db *reform.DB) (*Server, *mockAgentsStateUpdater) {
	t.Helper()

	return newTestServerWithHA(t, db, false)
}

// newTestServerWithHA is newTestServer with HA mode set to haEnabled.
func newTestServerWithHA(t *testing.T, db *reform.DB, haEnabled bool) (*Server, *mockAgentsStateUpdater) {
	t.Helper()

	var supervisord mockSupervisordService
	supervisord.Test(t)
	supervisord.On("UpdateConfiguration", mock.Anything).Return(nil)

	var vmdb mockPrometheusService
	vmdb.Test(t)
	vmdb.On("RequestConfigurationUpdate").Return(nil)

	var vmalert mockPrometheusService
	vmalert.Test(t)
	vmalert.On("RequestConfigurationUpdate").Return(nil)

	state := &mockAgentsStateUpdater{}
	state.Test(t)
	state.On("UpdateAgentsState", mock.Anything).Return(nil)
	state.On("RequestStateUpdate", mock.Anything, models.PMMServerAgentID).Return(nil)

	var templatesService mockTemplatesService
	templatesService.Test(t)
	templatesService.On("CollectTemplates", context.TODO()).Return(nil)

	var checksService mockChecksService
	checksService.Test(t)
	checksService.On("UpdateAdvisorsList", context.TODO()).Return(nil)

	var externalRules mockVmAlertExternalRules
	externalRules.Test(t)
	externalRules.On("ReadRules").Return("", nil)

	var telemetry mockTelemetryService
	telemetry.Test(t)
	telemetry.On("GetSummaries").Return(nil)

	var nomad mockNomadService
	nomad.Test(t)
	nomad.On("UpdateConfiguration", mock.Anything).Return(nil)

	var ha mockHaService
	ha.Test(t)
	ha.On("IsLeader").Return(true)
	ha.On("Params").Return(&models.HAParams{Enabled: haEnabled})

	s, err := NewServer(&Params{
		DB:                   db,
		VMDB:                 &vmdb,
		VMAlert:              &vmalert,
		ChecksService:        &checksService,
		TemplatesService:     &templatesService,
		AgentsStateUpdater:   state,
		Supervisord:          &supervisord,
		VMAlertExternalRules: &externalRules,
		TelemetryService:     &telemetry,
		Nomad:                &nomad,
		HAService:            &ha,
	})
	require.NoError(t, err)

	return s, state
}

func TestServer(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	newServerWithHA := func(t *testing.T, haEnabled bool) *Server {
		t.Helper()

		s, _ := newTestServerWithHA(t, reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), haEnabled)

		return s
	}

	newServer := func(t *testing.T) *Server {
		t.Helper()

		return newServerWithHA(t, false)
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

	t.Run("DataRetentionIsEnvOnlyInHA", func(t *testing.T) {
		retention := func(d time.Duration) *serverv1.ChangeSettingsRequest {
			return &serverv1.ChangeSettingsRequest{DataRetention: durationpb.New(d)}
		}

		t.Run("a changed value is refused and not written", func(t *testing.T) {
			s := newServerWithHA(t, true)

			stored, err := models.GetSettings(s.db)
			require.NoError(t, err)

			_, err = s.ChangeSettings(context.TODO(), retention(stored.DataRetention+24*time.Hour))
			tests.AssertGRPCErrorRE(t, codes.FailedPrecondition, "Data retention cannot be changed at runtime", err)

			after, err := models.GetSettings(s.db)
			require.NoError(t, err)
			assert.Equal(t, stored.DataRetention, after.DataRetention)
		})

		// The UI submits the whole settings form, so refusing an unchanged retention would
		// block every other setting on the page.
		t.Run("the value already in force is not a change", func(t *testing.T) {
			s := newServerWithHA(t, true)

			stored, err := models.GetSettings(s.db)
			require.NoError(t, err)

			req := retention(stored.DataRetention)
			req.PmmPublicAddress = new("1.2.3.4:5678")
			_, err = s.ChangeSettings(context.TODO(), req)
			require.NoError(t, err)
		})

		// Malformed input is a client error whether or not HA is enabled, and must not be
		// reported as a refusal to change a valid value.
		t.Run("a malformed value is an invalid argument", func(t *testing.T) {
			s := newServerWithHA(t, true)

			_, err := s.ChangeSettings(t.Context(), retention(36*time.Hour))
			tests.AssertGRPCErrorRE(t, codes.InvalidArgument, `Invalid argument: data_retention: should be a natural number of days\.`, err)

			_, err = s.ChangeSettings(t.Context(), retention(10*time.Second))
			tests.AssertGRPCErrorRE(t, codes.InvalidArgument, `Invalid argument: data_retention: minimal resolution is 24h\.`, err)
		})

		t.Run("leaving the value out is not a change", func(t *testing.T) {
			s := newServerWithHA(t, true)

			_, err := s.ChangeSettings(t.Context(), &serverv1.ChangeSettingsRequest{PmmPublicAddress: new("1.2.3.4:5678")})
			require.NoError(t, err)
		})

		// Another replica can commit a new retention between the transaction's two reads of the
		// row. That cannot be staged on one test database, so this pins the rule that avoids it
		// instead: the check keys off the request alone, and a request without a retention passes.
		t.Run("a request without a retention is never refused", func(t *testing.T) {
			s := newServerWithHA(t, true)

			stored, err := models.GetSettings(s.db)
			require.NoError(t, err)

			assert.NoError(t, s.refuseDataRetentionChangeInHA(0, stored))
		})

		t.Run("nothing is refused when HA is disabled", func(t *testing.T) {
			s := newServer(t)

			stored, err := models.GetSettings(s.db)
			require.NoError(t, err)

			_, err = s.ChangeSettings(context.TODO(), retention(stored.DataRetention+24*time.Hour))
			require.NoError(t, err)
		})
	})

	t.Run("DataRetentionIsReportedAtStartUp", func(t *testing.T) {
		// A boot-time setting has no other feedback channel, so the line itself is the
		// contract: assert the level, since that is what decides whether an operator sees it.
		run := func(t *testing.T, haEnabled bool, env []string) *logrustest.Hook {
			t.Helper()

			s := newServerWithHA(t, haEnabled)
			l, hook := logrustest.NewNullLogger()
			s.l = l.WithField("component", "server-test")
			require.Empty(t, s.UpdateSettingsFromEnv(context.TODO(), env))

			return hook
		}

		retentionEntry := func(t *testing.T, hook *logrustest.Hook) *logrus.Entry {
			t.Helper()

			for _, e := range hook.AllEntries() {
				if strings.HasPrefix(e.Message, "Data retention:") {
					return e
				}
			}
			t.Fatal("the effective data retention was never reported")

			return nil
		}

		t.Run("HA without the environment variable warns", func(t *testing.T) {
			e := retentionEntry(t, run(t, true, nil))
			assert.Equal(t, logrus.WarnLevel, e.Level, "an HA deployment with no retention supplied must be warned about")
			assert.Contains(t, e.Message, "dataRetentionDays", "the message must say where the value should come from")
		})

		t.Run("HA with the environment variable is informational", func(t *testing.T) {
			e := retentionEntry(t, run(t, true, []string{"PMM_DATA_RETENTION=240h"}))
			assert.Equal(t, logrus.InfoLevel, e.Level)
			assert.Equal(t, 10, e.Data["days"])
			assert.Contains(t, e.Message, "PMM_DATA_RETENTION")
		})

		t.Run("standalone is informational", func(t *testing.T) {
			e := retentionEntry(t, run(t, false, nil))
			assert.Equal(t, logrus.InfoLevel, e.Level)
			assert.Contains(t, e.Message, "changeable through the settings API")
		})

		retentionEntries := func(hook *logrustest.Hook) []*logrus.Entry {
			var res []*logrus.Entry
			for _, e := range hook.AllEntries() {
				if strings.HasPrefix(e.Message, "Data retention:") {
					res = append(res, e)
				}
			}

			return res
		}

		// setup() retries UpdateSettingsFromEnv until start-up succeeds, so the line must not
		// repeat on every retry, and must not be written by an attempt that failed to apply it.
		t.Run("reported once, after it is applied", func(t *testing.T) {
			s := newServerWithHA(t, true)
			l, hook := logrustest.NewNullLogger()
			s.l = l.WithField("component", "server-test")

			var sup mockSupervisordService
			sup.Test(t)
			sup.On("UpdateConfiguration", mock.Anything).Return(errors.New("supervisord is not ready")).Once()
			sup.On("UpdateConfiguration", mock.Anything).Return(nil)
			s.supervisord = &sup

			require.NotEmpty(t, s.UpdateSettingsFromEnv(t.Context(), nil))
			assert.Empty(t, retentionEntries(hook), "nothing was applied yet")

			require.Empty(t, s.UpdateSettingsFromEnv(t.Context(), nil))
			assert.Len(t, retentionEntries(hook), 1)

			require.Empty(t, s.UpdateSettingsFromEnv(t.Context(), nil))
			assert.Len(t, retentionEntries(hook), 1, "an unchanged value must not be reported again")
		})

		// qan-api2 takes its retention from the supervisord configuration, so the value is in force
		// once supervisord has rendered it, even if a later step of the re-render fails.
		t.Run("reported once supervisord applies it, even if the agents update fails", func(t *testing.T) {
			s := newServerWithHA(t, true)
			l, hook := logrustest.NewNullLogger()
			s.l = l.WithField("component", "server-test")

			mState := &mockAgentsStateUpdater{}
			mState.Test(t)
			mState.On("UpdateAgentsState", mock.Anything).Return(errors.New("agents are not ready"))
			s.agentsState = mState

			require.NotEmpty(t, s.UpdateSettingsFromEnv(t.Context(), nil))
			assert.Len(t, retentionEntries(hook), 1)
		})

		// Another replica can write its own environment to the shared row; this one applies it on
		// its next re-render, and the line must say so rather than keep the start-up value.
		t.Run("a value written by another replica is reported when applied", func(t *testing.T) {
			s := newServerWithHA(t, true)
			l, hook := logrustest.NewNullLogger()
			s.l = l.WithField("component", "server-test")
			require.Empty(t, s.UpdateSettingsFromEnv(t.Context(), []string{"PMM_DATA_RETENTION=240h"}))

			_, err := models.UpdateSettings(s.db, &models.ChangeSettingsParams{DataRetention: 20 * 24 * time.Hour})
			require.NoError(t, err)
			_, err = s.ChangeSettings(t.Context(), &serverv1.ChangeSettingsRequest{PmmPublicAddress: new("1.2.3.4:5678")})
			require.NoError(t, err)

			entries := retentionEntries(hook)
			require.Len(t, entries, 2)
			e := entries[1]
			assert.Equal(t, logrus.WarnLevel, e.Level, "a replica enforcing a value other than its own must be warned about")
			assert.Equal(t, 20, e.Data["days"])
			assert.Equal(t, 10, e.Data["env_days"])
			assert.Contains(t, e.Message, "another replica")
		})

		t.Run("a change through the settings API is reported", func(t *testing.T) {
			s := newServer(t)
			l, hook := logrustest.NewNullLogger()
			s.l = l.WithField("component", "server-test")
			require.Empty(t, s.UpdateSettingsFromEnv(t.Context(), nil))

			stored, err := models.GetSettings(s.db)
			require.NoError(t, err)
			_, err = s.ChangeSettings(t.Context(), &serverv1.ChangeSettingsRequest{DataRetention: durationpb.New(stored.DataRetention + 24*time.Hour)})
			require.NoError(t, err)

			entries := retentionEntries(hook)
			require.Len(t, entries, 2)
			assert.Equal(t, stored.DataRetentionDays()+1, entries[1].Data["days"])
		})
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

// TestInternalPgQANSettings covers the two call sites in this package that now key the internal QAN
// agent off the Service name through models.FindInternalPgQANAgent: GetSettings and
// handleInternalQANToggle. TestServer above opens its database with models.SkipFixtures, so
// pmm-server-postgresql never exists there and both paths only ever take the NotFound branch.
func TestInternalPgQANSettings(t *testing.T) {
	// The fixtures read PMM_ENABLE_INTERNAL_PG_QAN to decide the agent's initial state, so unset it
	// for a known starting point rather than inheriting the developer's or CI's environment.
	tests.UnsetEnv(t, env.EnableInternalPgQAN)

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	s, state := newTestServer(t, db)
	ctx := context.TODO()

	agent, err := models.FindInternalPgQANAgent(db.Querier)
	require.NoError(t, err)
	require.True(t, agent.Disabled, "the fixtures create the agent disabled when the variable is unset")

	t.Run("GetSettingsReportsTheAgentState", func(t *testing.T) {
		resp, err := s.GetSettings(ctx, &serverv1.GetSettingsRequest{})
		require.NoError(t, err)
		assert.False(t, resp.Settings.EnableInternalPgQan)
	})

	t.Run("ChangeSettingsTogglesTheAgent", func(t *testing.T) {
		resp, err := s.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
			EnableInternalPgQan: new(true),
		})
		require.NoError(t, err)
		assert.True(t, resp.Settings.EnableInternalPgQan)

		// The toggle has to reach the row the Service-keyed lookup finds, not just the response.
		stored, err := models.FindInternalPgQANAgent(db.Querier)
		require.NoError(t, err)
		assert.False(t, stored.Disabled)

		// RequestStateUpdate takes a pmm-agent ID, not the ID of the agent that changed.
		state.AssertCalled(t, "RequestStateUpdate", ctx, models.PMMServerAgentID)
	})
}

// TestInternalPgQANSettingsWithEnvPin pins the reason checkInternalPgQANEnvOverride stays in the
// inventory service layer instead of moving into models.ApplyAgentChange: ChangeSettings is the
// legitimate owner of this state, so it has to keep working while the variable is set to the
// opposite value. Moving that guard down makes this fail, which the comment on it asks for and
// nothing else in either suite checks.
func TestInternalPgQANSettingsWithEnvPin(t *testing.T) {
	// Pinned off, so enabling through the settings API contradicts the variable -- the case a
	// models-level guard would reject.
	t.Setenv(env.EnableInternalPgQAN, "false")

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	s, _ := newTestServer(t, db)
	ctx := context.TODO()

	agent, err := models.FindInternalPgQANAgent(db.Querier)
	require.NoError(t, err)
	require.True(t, agent.Disabled)

	resp, err := s.ChangeSettings(ctx, &serverv1.ChangeSettingsRequest{
		EnableInternalPgQan: new(true),
	})
	require.NoError(t, err)
	assert.True(t, resp.Settings.EnableInternalPgQan)

	stored, err := models.FindInternalPgQANAgent(db.Querier)
	require.NoError(t, err)
	assert.False(t, stored.Disabled)
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

func TestConvertReadOnlySettings(t *testing.T) {
	s := &Server{}

	t.Run("reports PMM Extensions as enabled when the process was started with it", func(t *testing.T) {
		t.Setenv(env.EnableExtensions, "1")

		assert.True(t, s.convertReadOnlySettings(&models.Settings{}).ExtensionsEnabled)
	})

	t.Run("reports PMM Extensions as disabled when the variable is absent", func(t *testing.T) {
		t.Setenv(env.EnableExtensions, "")
		os.Unsetenv(env.EnableExtensions)

		assert.False(t, s.convertReadOnlySettings(&models.Settings{}).ExtensionsEnabled)
	})
}
