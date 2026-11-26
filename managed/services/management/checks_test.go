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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestSendTestAdvisorNotification(t *testing.T) {
	t.Run("no recipients", func(t *testing.T) {
		var checksService mockChecksService
		s := NewChecksAPIService(&checksService)

		resp, err := s.SendTestAdvisorNotification(t.Context(), &advisorsv1.SendTestAdvisorNotificationRequest{})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "At least one email address is required."), err)
		assert.Nil(t, resp)
	})

	t.Run("invalid recipient", func(t *testing.T) {
		var checksService mockChecksService
		s := NewChecksAPIService(&checksService)

		resp, err := s.SendTestAdvisorNotification(t.Context(), &advisorsv1.SendTestAdvisorNotificationRequest{
			EmailAddresses: []string{"not-an-address"},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument,
			"advisor_notification_email_addresses: invalid address 'not-an-address'."), err)
		assert.Nil(t, resp)
	})

	t.Run("no severities", func(t *testing.T) {
		var checksService mockChecksService
		s := NewChecksAPIService(&checksService)

		resp, err := s.SendTestAdvisorNotification(t.Context(), &advisorsv1.SendTestAdvisorNotificationRequest{
			EmailAddresses: []string{"dba@example.com"},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "At least one severity is required."), err)
		assert.Nil(t, resp)
	})

	t.Run("unsupported severity", func(t *testing.T) {
		var checksService mockChecksService
		s := NewChecksAPIService(&checksService)

		resp, err := s.SendTestAdvisorNotification(t.Context(), &advisorsv1.SendTestAdvisorNotificationRequest{
			EmailAddresses: []string{"dba@example.com"},
			Severities:     []managementv1.Severity{managementv1.Severity_SEVERITY_DEBUG},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument,
			"advisor_notification_severities: unsupported severity level: debug."), err)
		assert.Nil(t, resp)
	})

	t.Run("SMTP not configured", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("SendTestNotification", []string{"dba@example.com"}, []common.Severity{common.Critical}).
			Return(fmt.Errorf("%w: GF_SMTP_ENABLED is not set", services.ErrSMTPNotConfigured))

		s := NewChecksAPIService(&checksService)

		resp, err := s.SendTestAdvisorNotification(t.Context(), &advisorsv1.SendTestAdvisorNotificationRequest{
			EmailAddresses: []string{"dba@example.com"},
			Severities:     []managementv1.Severity{managementv1.Severity_SEVERITY_CRITICAL},
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
			"SMTP is not configured in PMM Server: GF_SMTP_ENABLED is not set."), err)
		assert.Nil(t, resp)
	})

	t.Run("sent", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("SendTestNotification",
			[]string{"dba@example.com", "oncall@example.com"},
			[]common.Severity{common.Critical, common.Error, common.Warning},
		).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.SendTestAdvisorNotification(t.Context(), &advisorsv1.SendTestAdvisorNotificationRequest{
			EmailAddresses: []string{"dba@example.com", "oncall@example.com"},
			Severities: []managementv1.Severity{
				managementv1.Severity_SEVERITY_CRITICAL,
				managementv1.Severity_SEVERITY_ERROR,
				managementv1.Severity_SEVERITY_WARNING,
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		checksService.AssertExpectations(t)
	})
}

func TestEmailInsights(t *testing.T) {
	recipients := []string{"dba@example.com", "oncall@example.com"}

	t.Run("invalid recipient", func(t *testing.T) {
		var checksService mockChecksService
		s := NewChecksAPIService(&checksService)

		resp, err := s.EmailInsights(t.Context(), &advisorsv1.EmailInsightsRequest{
			EmailAddresses: []string{"not-an-address"},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument,
			"advisor_notification_email_addresses: invalid address 'not-an-address'."), err)
		assert.Nil(t, resp)
	})

	t.Run("emails the filtered insights", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EmailInsights", mock.Anything, models.InsightFilters{
			ServiceName: "mysql",
			RunID:       "run-1",
			Status:      new(models.CheckResultFailed),
			Severity:    new(models.Severity(common.Warning)),
		}, recipients).Return(3, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.EmailInsights(t.Context(), &advisorsv1.EmailInsightsRequest{
			EmailAddresses: recipients,
			ServiceName:    "mysql",
			RunId:          "run-1",
			Status:         new(advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_FAILED),
			Severity:       new(managementv1.Severity_SEVERITY_WARNING),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(3), resp.InsightsCount)
		checksService.AssertExpectations(t)
	})

	for _, tc := range []struct {
		name string
		err  error
		want *status.Status
	}{
		{
			name: "no insights match",
			err:  services.ErrNoInsightsToEmail,
			want: status.New(codes.InvalidArgument, "no insights match the filters."),
		},
		{
			name: "too many insights match",
			err:  fmt.Errorf("%w: 201 match, narrow them to 200 or fewer", services.ErrTooManyInsightsToEmail),
			want: status.New(codes.InvalidArgument,
				"too many insights match the filters: 201 match, narrow them to 200 or fewer."),
		},
		{
			name: "SMTP not configured",
			err:  fmt.Errorf("%w: GF_SMTP_ENABLED is not set", services.ErrSMTPNotConfigured),
			want: status.New(codes.FailedPrecondition, "SMTP is not configured in PMM Server: GF_SMTP_ENABLED is not set."),
		},
		{
			name: "send failure",
			err:  errors.New("connection refused"),
			want: status.New(codes.Internal, "Failed to email the insights: connection refused."),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var checksService mockChecksService
			checksService.On("EmailInsights", mock.Anything, models.InsightFilters{}, recipients).Return(0, tc.err)

			s := NewChecksAPIService(&checksService)

			resp, err := s.EmailInsights(t.Context(), &advisorsv1.EmailInsightsRequest{EmailAddresses: recipients})
			tests.AssertGRPCError(t, tc.want, err)
			assert.Nil(t, resp)
		})
	}

	t.Run("requires a recipient", func(t *testing.T) {
		err := (&advisorsv1.EmailInsightsRequest{}).Validate()
		require.ErrorContains(t, err, "EmailAddresses: value must contain at least 1 item(s)")
	})
}

func TestStartAdvisorChecks(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything, []string(nil), []string(nil)).Return("", errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to start advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("Advisors disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything, []string(nil), []string(nil)).Return("", services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})

	t.Run("run in progress", func(t *testing.T) {
		now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
		inProgress := &services.AdvisorRunInProgressError{
			Run: &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByScheduler, StartedAt: now.Add(-3 * time.Minute)},
			Now: now,
		}
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything, []string{"check_a"}, []string(nil)).Return("", inProgress)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{Names: []string{"check_a"}})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
			"Advisor checks are already running (started 3 minutes ago by the scheduler). Try again when the run finishes."), err)
		assert.Nil(t, resp)
	})

	t.Run("passes the scope through", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything, []string{"check_a"}, []string{"svc-1"}).Return("run-1", nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{
			Names:      []string{"check_a"},
			ServiceIds: []string{"svc-1"},
		})
		require.NoError(t, err)
		assert.Equal(t, "run-1", resp.RunId)
		checksService.AssertExpectations(t)
	})
}

func TestTestAdvisorCheck(t *testing.T) {
	t.Parallel()

	apiCheck := &advisorsv1.AdvisorCheck{
		Name:        "custom_mysql_version",
		Summary:     "Check summary",
		Description: "Check description",
		Category:    "configuration",
		Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL,
		Queries:     []*advisorsv1.AdvisorCheckQuery{{Type: "MYSQL_SHOW", Query: "version"}},
		Script:      "def check_context(docs, context):\n    return []",
	}

	t.Run("check is required", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService

		s := NewChecksAPIService(&checksService)

		resp, err := s.TestAdvisorCheck(t.Context(), &advisorsv1.TestAdvisorCheckRequest{ServiceId: "test_svc"})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Check is required."), err)
		assert.Nil(t, resp)
		checksService.AssertNotCalled(t, "TestAdvisorCheck")
	})

	t.Run("Advisors disabled error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("TestAdvisorCheck", mock.Anything, mock.Anything, "test_svc").Return(nil, "", services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.TestAdvisorCheck(t.Context(), &advisorsv1.TestAdvisorCheckRequest{Check: apiCheck, ServiceId: "test_svc"})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})

	t.Run("execution errors keep their message", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("TestAdvisorCheck", mock.Anything, mock.Anything, "test_svc").
			Return(nil, "", status.Errorf(codes.FailedPrecondition,
				"failed to execute check 'custom_mysql_version' on service 'svc': random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.TestAdvisorCheck(t.Context(), &advisorsv1.TestAdvisorCheckRequest{Check: apiCheck, ServiceId: "test_svc"})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
			"failed to execute check 'custom_mysql_version' on service 'svc': random error"), err)
		assert.Nil(t, resp)
	})

	t.Run("passes the converted check and returns converted results", func(t *testing.T) {
		t.Parallel()

		expectedCheck := check.Check{
			Name:        "custom_mysql_version",
			Summary:     "Check summary",
			Description: "Check description",
			Category:    "configuration",
			Technology:  check.MySQL,
			Queries:     []check.Query{{Type: check.MySQLShow, Query: "version"}},
			Script:      "def check_context(docs, context):\n    return []",
		}
		checkResult := []services.CheckResult{
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc", ServiceID: "test_svc"},
				CheckName: "custom_mysql_version",
			},
		}
		var checksService mockChecksService
		checksService.On("TestAdvisorCheck", mock.Anything, expectedCheck, "test_svc").Return(checkResult, "print output", nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.TestAdvisorCheck(t.Context(), &advisorsv1.TestAdvisorCheckRequest{Check: apiCheck, ServiceId: "test_svc"})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.TestAdvisorCheckResponse{
			Results: []*advisorsv1.TestAdvisorCheckResult{
				{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreUrl: "https://www.example.com",
					Severity:    managementv1.Severity_SEVERITY_WARNING,
					Labels:      map[string]string{"label_key": "label_value"},
					ServiceName: "svc",
					ServiceId:   "test_svc",
					CheckName:   "custom_mysql_version",
				},
			},
			ScriptOutput: "print output",
		}, resp)
	})
}

func TestListAdvisorCheckTestTargets(t *testing.T) {
	t.Parallel()

	var checksService mockChecksService
	checksService.On("ListTestTargets", mock.Anything, check.PostgreSQL).Return([]services.Target{
		{ServiceID: "svc-1", ServiceName: "pg-1"},
		{ServiceID: "svc-2", ServiceName: "pg-2"},
	}, nil)

	s := NewChecksAPIService(&checksService)

	resp, err := s.ListAdvisorCheckTestTargets(t.Context(), &advisorsv1.ListAdvisorCheckTestTargetsRequest{
		Technology: advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_POSTGRESQL,
	})
	require.NoError(t, err)
	assert.Equal(t, &advisorsv1.ListAdvisorCheckTestTargetsResponse{
		Targets: []*advisorsv1.AdvisorCheckTestTarget{
			{ServiceId: "svc-1", ServiceName: "pg-1"},
			{ServiceId: "svc-2", ServiceName: "pg-2"},
		},
	}, resp)
}

func TestListAdvisorServices(t *testing.T) {
	t.Parallel()

	t.Run("converts services", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("ListServices", mock.Anything).Return([]services.Target{
			{ServiceID: "svc-1", ServiceName: "mongo-1", ServiceType: models.MongoDBServiceType, NodeName: "node-1", Environment: "prod", Cluster: "rs0"},
			{ServiceID: "svc-2", ServiceName: "mysql-1", ServiceType: models.MySQLServiceType, NodeName: "node-2"},
			{ServiceID: "svc-3", ServiceName: "pg-1", ServiceType: models.PostgreSQLServiceType, NodeName: "node-3", Environment: "staging"},
		}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisorServices(t.Context(), &advisorsv1.ListAdvisorServicesRequest{})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.ListAdvisorServicesResponse{
			Services: []*advisorsv1.AdvisorTargetService{
				{
					ServiceId:   "svc-1",
					ServiceName: "mongo-1",
					Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MONGODB,
					NodeName:    "node-1",
					Environment: "prod",
					Cluster:     "rs0",
				},
				{
					ServiceId:   "svc-2",
					ServiceName: "mysql-1",
					Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL,
					NodeName:    "node-2",
				},
				{
					ServiceId:   "svc-3",
					ServiceName: "pg-1",
					Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_POSTGRESQL,
					NodeName:    "node-3",
					Environment: "staging",
				},
			},
		}, resp)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("ListServices", mock.Anything).Return(nil, errors.New("db down"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisorServices(t.Context(), &advisorsv1.ListAdvisorServicesRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestGetAdvisorNotificationSettings(t *testing.T) {
	t.Parallel()

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()

		settings := &models.Settings{}
		settings.AdvisorNotifications.Enabled = new(true)
		settings.AdvisorNotifications.EmailAddresses = []string{"dba@example.com", "oncall@example.com"}
		settings.AdvisorNotifications.Severities = []common.Severity{common.Critical, common.Error, common.Warning}

		var checksService mockChecksService
		checksService.On("GetSettings", mock.Anything).Return(settings, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorNotificationSettings(t.Context(), &advisorsv1.GetAdvisorNotificationSettingsRequest{})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.GetAdvisorNotificationSettingsResponse{
			Enabled:        true,
			EmailAddresses: []string{"dba@example.com", "oncall@example.com"},
			Severities: []managementv1.Severity{
				managementv1.Severity_SEVERITY_CRITICAL,
				managementv1.Severity_SEVERITY_ERROR,
				managementv1.Severity_SEVERITY_WARNING,
			},
		}, resp)
	})

	t.Run("never configured", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetSettings", mock.Anything).Return(&models.Settings{}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorNotificationSettings(t.Context(), &advisorsv1.GetAdvisorNotificationSettingsRequest{})
		require.NoError(t, err)
		assert.Equal(t, models.AdvisorNotificationsEnabledDefault, resp.Enabled)
		assert.Empty(t, resp.EmailAddresses)
		assert.Empty(t, resp.Severities)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetSettings", mock.Anything).Return(nil, errors.New("db down"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorNotificationSettings(t.Context(), &advisorsv1.GetAdvisorNotificationSettingsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestListRuns(t *testing.T) {
	t.Parallel()

	t.Run("converts runs and pagination totals", func(t *testing.T) {
		t.Parallel()

		startedAt := time.Date(2026, time.August, 4, 19, 57, 28, 0, time.UTC)
		finishedAt := startedAt.Add(2*time.Minute + 7*time.Second)
		finished := &models.AdvisorRun{
			ID:                   "run-1",
			TriggeredBy:          models.CheckTriggeredByUser,
			Status:               models.AdvisorRunStatusCompleted,
			CheckNames:           []string{"check_a"},
			ServiceIDs:           []string{"svc-1"},
			StartedAt:            startedAt,
			FinishedAt:           &finishedAt,
			PlannedChecksCount:   109,
			PlannedServicesCount: 4,
			ChecksCount:          107,
			ServicesCount:        3,
			FindingsCount:        28,
			ErrorsCount:          1,
		}
		require.NoError(t, finished.SetSeverityCounts(map[models.Severity]int{
			models.Severity(common.Error):   4,
			models.Severity(common.Warning): 22,
		}))
		// a run still in flight has no completion and no totals yet; it waited for the one above
		scheduledFor := startedAt.Add(time.Minute)
		running := &models.AdvisorRun{
			ID:           "run-2",
			TriggeredBy:  models.CheckTriggeredByScheduler,
			Status:       models.AdvisorRunStatusRunning,
			ScheduleID:   new("schedule-1"),
			ScheduleName: "All checks daily",
			ScheduledFor: &scheduledFor,
			StartedAt:    startedAt.Add(5 * time.Minute),
		}

		var checksService mockChecksService
		checksService.On("GetRuns", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]*models.AdvisorRun{finished, running}, 3, nil)
		checksService.On("GetRunsWaitedFor", mock.Anything, []*models.AdvisorRun{finished, running}).
			Return(map[string]*models.AdvisorRun{"run-2": finished}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListRuns(t.Context(), &advisorsv1.ListRunsRequest{
			PageSize:  new(int32(2)),
			PageIndex: new(int32(0)),
		})
		require.NoError(t, err)

		expected := &advisorsv1.ListRunsResponse{
			Results: []*advisorsv1.AdvisorRun{
				{
					Id:                   "run-1",
					TriggeredBy:          advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_USER,
					Status:               advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_COMPLETED,
					CheckNames:           []string{"check_a"},
					ServiceIds:           []string{"svc-1"},
					StartedAt:            timestamppb.New(startedAt),
					FinishedAt:           timestamppb.New(finishedAt),
					PlannedChecksCount:   109,
					PlannedServicesCount: 4,
					ChecksCount:          107,
					ServicesCount:        3,
					FindingsCount:        28,
					ErrorsCount:          1,
					SeverityCounts: []*advisorsv1.SeverityCount{
						{Severity: managementv1.Severity_SEVERITY_ERROR, Count: 4},
						{Severity: managementv1.Severity_SEVERITY_WARNING, Count: 22},
					},
				},
				{
					Id:           "run-2",
					TriggeredBy:  advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER,
					Status:       advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_RUNNING,
					ScheduleId:   "schedule-1",
					ScheduleName: "All checks daily",
					ScheduledFor: timestamppb.New(scheduledFor),
					WaitedForRun: &advisorsv1.AdvisorRunReference{
						RunId:       "run-1",
						TriggeredBy: advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_USER,
						StartedAt:   timestamppb.New(startedAt),
					},
					StartedAt:      timestamppb.New(startedAt.Add(5 * time.Minute)),
					SeverityCounts: []*advisorsv1.SeverityCount{},
				},
			},
			TotalItems: 3,
			TotalPages: 2,
		}
		assert.Equal(t, expected, resp)
		checksService.AssertExpectations(t)
	})

	t.Run("converts filters", func(t *testing.T) {
		t.Parallel()

		from := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
		triggeredBy := models.CheckTriggeredByScheduler

		var checksService mockChecksService
		checksService.On("GetRuns", mock.Anything, models.AdvisorRunFilters{
			RunID:       "run-1",
			ScheduleID:  "schedule-1",
			TriggeredBy: &triggeredBy,
			From:        &from,
			To:          &to,
		}, 0, 25).Return([]*models.AdvisorRun{}, 0, nil)
		checksService.On("GetRunsWaitedFor", mock.Anything, []*models.AdvisorRun{}).
			Return(map[string]*models.AdvisorRun{}, nil)

		s := NewChecksAPIService(&checksService)

		_, err := s.ListRuns(t.Context(), &advisorsv1.ListRunsRequest{
			PageSize:    new(int32(25)),
			TriggeredBy: new(advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER),
			From:        timestamppb.New(from),
			To:          timestamppb.New(to),
			RunId:       "run-1",
			ScheduleId:  "schedule-1",
		})
		require.NoError(t, err)
		checksService.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetRuns", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, 0, errors.New("boom"))

		s := NewChecksAPIService(&checksService)

		_, err := s.ListRuns(t.Context(), &advisorsv1.ListRunsRequest{})
		require.Error(t, err)
	})
}

func TestListInsightsFilterValues(t *testing.T) {
	t.Parallel()

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetInsightsFilterValues", mock.Anything).
			Return(nil, nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListInsightsFilterValues(t.Context(), &advisorsv1.ListInsightsFilterValuesRequest{})
		require.EqualError(t, err, "failed to get insights filter values: random error")
		assert.Nil(t, resp)
	})

	t.Run("returns distinct values", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetInsightsFilterValues", mock.Anything).
			Return([]string{"mysql-1", "pg-1"}, []string{"node-a"}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListInsightsFilterValues(t.Context(), &advisorsv1.ListInsightsFilterValuesRequest{})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.ListInsightsFilterValuesResponse{
			ServiceNames: []string{"mysql-1", "pg-1"},
			NodeNames:    []string{"node-a"},
		}, resp)
	})
}

func TestListInsights(t *testing.T) {
	t.Parallel()

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetInsights", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, 0, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListInsights(t.Context(), &advisorsv1.ListInsightsRequest{})
		require.EqualError(t, err, "failed to get insights: random error")
		assert.Nil(t, resp)
	})

	t.Run("returns converted history with pagination totals", func(t *testing.T) {
		t.Parallel()

		checkedAt := time.Date(2026, time.June, 27, 10, 0, 0, 0, time.UTC)
		record := &models.Insight{
			ID:          "id1",
			CheckName:   "test_check",
			Category:    "configuration",
			ServiceID:   "test_svc",
			ServiceName: "svc",
			ServiceType: models.MySQLServiceType,
			NodeID:      "node1",
			NodeName:    "node",
			Status:      models.CheckResultFailed,
			Summary:     "Check summary",
			Description: "Check Description",
			ReadMoreURL: "https://www.example.com",
			Severity:    new(models.Severity(common.Critical)),
			CheckedAt:   &checkedAt,
			Region:      "us-east-1",
			AZ:          "us-east-1f",
		}
		require.NoError(t, record.SetLabels(map[string]string{"label_key": "label_value"}))
		// a pending check has no outcome yet
		pending := &models.Insight{
			ID:          "id2",
			CheckName:   "test_check",
			ServiceID:   "test_svc",
			ServiceType: models.MySQLServiceType,
			Status:      models.CheckResultPending,
		}

		var checksService mockChecksService
		checksService.On("GetInsights", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]*models.Insight{record, pending}, 3, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListInsights(t.Context(), &advisorsv1.ListInsightsRequest{
			ServiceId: "test_svc",
			PageSize:  new(int32(2)),
			PageIndex: new(int32(0)),
		})
		require.NoError(t, err)

		expected := &advisorsv1.ListInsightsResponse{
			Results: []*advisorsv1.Insight{
				{
					Id:          "id1",
					CheckName:   "test_check",
					Category:    "configuration",
					ServiceId:   "test_svc",
					ServiceName: "svc",
					ServiceType: string(models.MySQLServiceType),
					NodeId:      "node1",
					NodeName:    "node",
					Status:      advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_FAILED,
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreUrl: "https://www.example.com",
					Severity:    managementv1.Severity_SEVERITY_CRITICAL,
					Labels:      map[string]string{"label_key": "label_value"},
					CheckedAt:   timestamppb.New(checkedAt),
					Region:      "us-east-1",
					Az:          "us-east-1f",
				},
				{
					Id:          "id2",
					CheckName:   "test_check",
					ServiceId:   "test_svc",
					ServiceType: string(models.MySQLServiceType),
					Status:      advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_PENDING,
					Severity:    managementv1.Severity_SEVERITY_UNSPECIFIED,
				},
			},
			TotalItems: 3,
			TotalPages: 2,
		}
		assert.Equal(t, expected, resp)
	})
}

func TestListAdvisorChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string{"two"}, nil)
		checksService.On("GetDisabledServicesForChecks", mock.Anything).
			Return(map[string][]string{"three": {"svc-1", "svc-2"}}, nil)
		checksService.On("GetChecks").
			Return(map[string]check.Check{
				"one":   {Name: "one"},
				"two":   {Name: "two"},
				"three": {Name: "three"},
				"four":  {Name: "four"},
			}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisorChecks(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(
			t,
			[]*advisorsv1.AdvisorCheck{
				{Name: "four", Enabled: true},
				{Name: "one", Enabled: true},
				{Name: "three", Enabled: true, DisabledServiceIds: []string{"svc-1", "svc-2"}},
				{Name: "two", Enabled: false},
			},
			resp.Checks,
		)
	})

	t.Run("get disabled checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisorChecks(t.Context(), nil)
		require.EqualError(t, err, "failed to get disabled checks list: random error")
		assert.Nil(t, resp)
	})
}

func TestGetAdvisorCheck(t *testing.T) {
	t.Parallel()

	storedCheck := check.Check{
		Name:        "custom_pg_settings",
		Summary:     "Check summary",
		Description: "Check description",
		Category:    "configuration",
		Technology:  check.PostgreSQL,
		UserDefined: true,
		Queries: []check.Query{{
			Type:       check.PostgreSQLSelect,
			Query:      "name FROM pg_settings",
			Parameters: map[check.Parameter]string{check.AllDBs: "true"},
		}},
		Script: "def check_context(docs, context):\n    return []",
	}

	t.Run("returns the check with its enabled state and disabled services", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_pg_settings": storedCheck}, nil)
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string{"custom_pg_settings"}, nil)
		checksService.On("GetDisabledServicesForChecks", mock.Anything).
			Return(map[string][]string{"custom_pg_settings": {"svc-1", "svc-2"}}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorCheck(t.Context(), &advisorsv1.GetAdvisorCheckRequest{Name: "custom_pg_settings"})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.GetAdvisorCheckResponse{
			Check: &advisorsv1.AdvisorCheck{
				Name:        "custom_pg_settings",
				Enabled:     false,
				Summary:     "Check summary",
				Description: "Check description",
				Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_POSTGRESQL,
				Category:    "configuration",
				UserDefined: true,
				Queries: []*advisorsv1.AdvisorCheckQuery{{
					Type:       "POSTGRESQL_SELECT",
					Query:      "name FROM pg_settings",
					Parameters: map[string]string{"all_dbs": "true"},
				}},
				Script:             "def check_context(docs, context):\n    return []",
				DisabledServiceIds: []string{"svc-1", "svc-2"},
			},
		}, resp)
	})

	t.Run("unknown check", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_pg_settings": storedCheck}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorCheck(t.Context(), &advisorsv1.GetAdvisorCheckRequest{Name: "custom_missing"})
		tests.AssertGRPCError(t, status.New(codes.NotFound, "Advisor check custom_missing not found"), err)
		assert.Nil(t, resp)
	})

	t.Run("get checks error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecks").Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorCheck(t.Context(), &advisorsv1.GetAdvisorCheckRequest{Name: "custom_pg_settings"})
		require.EqualError(t, err, "failed to get available checks list: random error")
		assert.Nil(t, resp)
	})

	t.Run("get disabled checks error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_pg_settings": storedCheck}, nil)
		checksService.On("GetDisabledChecks", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorCheck(t.Context(), &advisorsv1.GetAdvisorCheckRequest{Name: "custom_pg_settings"})
		require.EqualError(t, err, "failed to get disabled checks list: random error")
		assert.Nil(t, resp)
	})

	t.Run("get disabled services error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_pg_settings": storedCheck}, nil)
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string(nil), nil)
		checksService.On("GetDisabledServicesForChecks", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetAdvisorCheck(t.Context(), &advisorsv1.GetAdvisorCheckRequest{Name: "custom_pg_settings"})
		require.EqualError(t, err, "failed to get disabled services list: random error")
		assert.Nil(t, resp)
	})
}

func TestCreateAdvisorCheck(t *testing.T) {
	t.Parallel()

	apiCheck := &advisorsv1.AdvisorCheck{
		Name:        "custom_mysql_version",
		Summary:     "Check summary",
		Description: "Check description",
		Category:    "configuration",
		Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL,
		Queries:     []*advisorsv1.AdvisorCheckQuery{{Type: "MYSQL_SHOW", Query: "version"}},
		Script:      "def check_context(docs, context):\n    return []",
	}
	expectedCheck := check.Check{
		Name:        "custom_mysql_version",
		Summary:     "Check summary",
		Description: "Check description",
		Category:    "configuration",
		Technology:  check.MySQL,
		Queries:     []check.Query{{Type: check.MySQLShow, Query: "version"}},
		Script:      "def check_context(docs, context):\n    return []",
	}

	t.Run("check is required", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService

		s := NewChecksAPIService(&checksService)

		resp, err := s.CreateAdvisorCheck(t.Context(), &advisorsv1.CreateAdvisorCheckRequest{})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Check is required."), err)
		assert.Nil(t, resp)
		checksService.AssertNotCalled(t, "CreateAdvisorCheck")
	})

	t.Run("service errors keep their gRPC code", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("CreateAdvisorCheck", mock.Anything, expectedCheck).
			Return(status.Errorf(codes.AlreadyExists, "advisor check custom_mysql_version already exists"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.CreateAdvisorCheck(t.Context(), &advisorsv1.CreateAdvisorCheckRequest{Check: apiCheck})
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, "advisor check custom_mysql_version already exists"), err)
		assert.Nil(t, resp)
		checksService.AssertNotCalled(t, "GetChecks")
	})

	t.Run("reload error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("CreateAdvisorCheck", mock.Anything, expectedCheck).Return(nil)
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_mysql_version": expectedCheck}, nil)
		checksService.On("GetDisabledChecks", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.CreateAdvisorCheck(t.Context(), &advisorsv1.CreateAdvisorCheckRequest{Check: apiCheck})
		require.EqualError(t, err, "failed to get disabled checks list: random error")
		assert.Nil(t, resp)
	})

	t.Run("creates the check and returns it as stored", func(t *testing.T) {
		t.Parallel()

		storedCheck := expectedCheck
		storedCheck.UserDefined = true

		var checksService mockChecksService
		checksService.On("CreateAdvisorCheck", mock.Anything, expectedCheck).Return(nil)
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_mysql_version": storedCheck}, nil)
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string(nil), nil)
		checksService.On("GetDisabledServicesForChecks", mock.Anything).Return(map[string][]string(nil), nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.CreateAdvisorCheck(t.Context(), &advisorsv1.CreateAdvisorCheckRequest{Check: apiCheck})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.CreateAdvisorCheckResponse{
			Check: &advisorsv1.AdvisorCheck{
				Name:        "custom_mysql_version",
				Enabled:     true,
				Summary:     "Check summary",
				Description: "Check description",
				Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL,
				Category:    "configuration",
				UserDefined: true,
				Queries:     []*advisorsv1.AdvisorCheckQuery{{Type: "MYSQL_SHOW", Query: "version"}},
				Script:      "def check_context(docs, context):\n    return []",
			},
		}, resp)
		checksService.AssertExpectations(t)
	})
}

func TestUpdateAdvisorCheck(t *testing.T) {
	t.Parallel()

	definition := &advisorsv1.AdvisorCheckDefinition{
		Summary:     "Check summary",
		Description: "Check description",
		Category:    "configuration",
		Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL,
		Queries:     []*advisorsv1.AdvisorCheckQuery{{Type: "MYSQL_SHOW", Query: "version"}},
		Script:      "def check_context(docs, context):\n    return []",
	}

	t.Run("check is required", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService

		s := NewChecksAPIService(&checksService)

		resp, err := s.UpdateAdvisorCheck(t.Context(), &advisorsv1.UpdateAdvisorCheckRequest{Name: "custom_one"})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Check is required."), err)
		assert.Nil(t, resp)
		checksService.AssertNotCalled(t, "UpdateAdvisorCheck")
	})

	t.Run("updates the check named in the path", func(t *testing.T) {
		t.Parallel()

		expectedCheck := check.Check{
			Name:        "custom_one",
			Summary:     "Check summary",
			Description: "Check description",
			Category:    "configuration",
			Technology:  check.MySQL,
			Queries:     []check.Query{{Type: check.MySQLShow, Query: "version"}},
			Script:      "def check_context(docs, context):\n    return []",
		}

		var checksService mockChecksService
		checksService.On("UpdateAdvisorCheck", mock.Anything, expectedCheck).Return(nil)
		checksService.On("GetChecks").Return(map[string]check.Check{"custom_one": expectedCheck}, nil)
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string(nil), nil)
		checksService.On("GetDisabledServicesForChecks", mock.Anything).Return(map[string][]string(nil), nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.UpdateAdvisorCheck(t.Context(), &advisorsv1.UpdateAdvisorCheckRequest{
			Name:  "custom_one",
			Check: definition,
		})
		require.NoError(t, err)
		assert.Equal(t, "custom_one", resp.Check.Name)
		checksService.AssertExpectations(t)
	})
}

func TestDeleteAdvisorCheck(t *testing.T) {
	t.Parallel()

	t.Run("service errors keep their gRPC code", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("DeleteAdvisorCheck", mock.Anything, "mysql_version").
			Return(status.Errorf(codes.FailedPrecondition, "Advisor check mysql_version is shipped by Percona and cannot be modified"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.DeleteAdvisorCheck(t.Context(), &advisorsv1.DeleteAdvisorCheckRequest{Name: "mysql_version"})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition,
			"Advisor check mysql_version is shipped by Percona and cannot be modified"), err)
		assert.Nil(t, resp)
	})

	t.Run("deletes the check named in the request", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("DeleteAdvisorCheck", mock.Anything, "custom_one").Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.DeleteAdvisorCheck(t.Context(), &advisorsv1.DeleteAdvisorCheckRequest{Name: "custom_one"})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.DeleteAdvisorCheckResponse{}, resp)
		checksService.AssertExpectations(t)
	})
}

func TestUpdateAdvisorChecks(t *testing.T) {
	t.Run("enable advisor checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecks", mock.Anything, mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to enable disabled advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("disable advisor checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecks", mock.Anything, mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything, mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to disable advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("disable for services", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("DisableChecksForServices", mock.Anything, "check-name", []string{"svc-1", "svc-2"}).Return(nil)
		checksService.On("EnableChecks", mock.Anything, mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything, mock.Anything).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{
			Params: []*advisorsv1.ChangeAdvisorCheckParams{{
				Name:       "check-name",
				Enable:     new(false),
				ServiceIds: []string{"svc-1", "svc-2"},
			}},
		})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.ChangeAdvisorChecksResponse{}, resp)
		checksService.AssertCalled(t, "DisableChecksForServices", mock.Anything, "check-name", []string{"svc-1", "svc-2"})
		// a per-service change must not touch the global enable/disable lists
		checksService.AssertNotCalled(t, "DisableChecks", mock.Anything, []string{"check-name"})
	})

	t.Run("enable for services", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecksForServices", mock.Anything, "check-name", []string{"svc-1"}).Return(nil)
		checksService.On("EnableChecks", mock.Anything, mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything, mock.Anything).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{
			Params: []*advisorsv1.ChangeAdvisorCheckParams{{
				Name:       "check-name",
				Enable:     new(true),
				ServiceIds: []string{"svc-1"},
			}},
		})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.ChangeAdvisorChecksResponse{}, resp)
		checksService.AssertCalled(t, "EnableChecksForServices", mock.Anything, "check-name", []string{"svc-1"})
	})

	t.Run("services without enable flag rejected", func(t *testing.T) {
		var checksService mockChecksService

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{
			Params: []*advisorsv1.ChangeAdvisorCheckParams{{
				Name:       "check-name",
				ServiceIds: []string{"svc-1"},
			}},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, resp)
	})
}

func TestConvertResultStatuses(t *testing.T) {
	t.Parallel()

	t.Run("round-trips every specified API status", func(t *testing.T) {
		t.Parallel()

		for value, name := range advisorsv1.AdvisorCheckResultStatus_name {
			apiStatus := advisorsv1.AdvisorCheckResultStatus(value)
			if apiStatus == advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_UNSPECIFIED {
				continue
			}

			assert.Equal(t, apiStatus, convertModelResultStatus(convertAPIResultStatus(apiStatus)), name)
		}
	})

	t.Run("unspecified and unknown values", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, convertAPIResultStatus(advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_UNSPECIFIED))
		assert.Empty(t, convertAPIResultStatus(advisorsv1.AdvisorCheckResultStatus(999)))
		assert.Equal(t, advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_UNSPECIFIED, convertModelResultStatus(""))
		assert.Equal(t, advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_UNSPECIFIED, convertModelResultStatus("unknown"))
	})
}

func TestConvertTriggeredBy(t *testing.T) {
	t.Parallel()

	t.Run("round-trips every specified API value", func(t *testing.T) {
		t.Parallel()

		for value, name := range advisorsv1.AdvisorCheckTriggeredBy_name {
			apiTriggeredBy := advisorsv1.AdvisorCheckTriggeredBy(value)
			if apiTriggeredBy == advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED {
				continue
			}

			assert.Equal(t, apiTriggeredBy, convertModelTriggeredBy(convertAPITriggeredBy(apiTriggeredBy)), name)
		}
	})

	t.Run("unspecified and unknown values", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, convertAPITriggeredBy(advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED))
		assert.Empty(t, convertAPITriggeredBy(advisorsv1.AdvisorCheckTriggeredBy(999)))
		assert.Equal(t, advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED, convertModelTriggeredBy(""))
		assert.Equal(t, advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED, convertModelTriggeredBy("unknown"))
	})
}

func TestConvertTechnologies(t *testing.T) {
	t.Parallel()

	t.Run("round-trips every specified API technology", func(t *testing.T) {
		t.Parallel()

		for value, name := range advisorsv1.AdvisorCheckTechnology_name {
			apiTechnology := advisorsv1.AdvisorCheckTechnology(value)
			if apiTechnology == advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_UNSPECIFIED {
				continue
			}

			assert.Equal(t, apiTechnology, convertTechnology(convertAPITechnology(apiTechnology)), name)
		}
	})

	t.Run("unspecified and unknown values", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, convertAPITechnology(advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_UNSPECIFIED))
		assert.Empty(t, convertAPITechnology(advisorsv1.AdvisorCheckTechnology(999)))
		assert.Equal(t, advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_UNSPECIFIED, convertTechnology(""))
		assert.Equal(t, advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_UNSPECIFIED, convertTechnology("CLICKHOUSE"))
	})
}

func TestConvertModelRunStatus(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		status   models.AdvisorRunStatus
		expected advisorsv1.AdvisorRunStatus
	}{
		{models.AdvisorRunStatusQueued, advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_QUEUED},
		{models.AdvisorRunStatusRunning, advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_RUNNING},
		{models.AdvisorRunStatusCompleted, advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_COMPLETED},
		{models.AdvisorRunStatusInterrupted, advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_INTERRUPTED},
		{models.AdvisorRunStatusAborted, advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_ABORTED},
		{"", advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_UNSPECIFIED},
		{"unknown", advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_UNSPECIFIED},
	} {
		assert.Equal(t, tc.expected, convertModelRunStatus(tc.status), string(tc.status))
	}
}

func TestConvertQueries(t *testing.T) {
	t.Parallel()

	t.Run("round-trips queries with and without parameters", func(t *testing.T) {
		t.Parallel()

		queries := []check.Query{
			{Type: check.MySQLShow, Query: "version"},
			{
				Type:       check.MetricsRange,
				Query:      "mysql_up",
				Parameters: map[check.Parameter]string{check.Lookback: "5m", check.Range: "1h", check.Step: "1m"},
			},
		}

		apiQueries := convertQueriesToAPI(queries)
		assert.Equal(t, []*advisorsv1.AdvisorCheckQuery{
			{Type: "MYSQL_SHOW", Query: "version"},
			{
				Type:       "METRICS_RANGE",
				Query:      "mysql_up",
				Parameters: map[string]string{"lookback": "5m", "range": "1h", "step": "1m"},
			},
		}, apiQueries)
		assert.Equal(t, queries, convertAPIQueries(apiQueries))
	})

	t.Run("nil and empty input", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, convertQueriesToAPI(nil))
		assert.Nil(t, convertQueriesToAPI([]check.Query{}))
		assert.Nil(t, convertAPIQueries(nil))
		assert.Nil(t, convertAPIQueries([]*advisorsv1.AdvisorCheckQuery{}))
	})

	t.Run("nil API queries are skipped", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []check.Query{{Type: check.MySQLShow, Query: "version"}},
			convertAPIQueries([]*advisorsv1.AdvisorCheckQuery{nil, {Type: "MYSQL_SHOW", Query: "version"}}))
	})
}
