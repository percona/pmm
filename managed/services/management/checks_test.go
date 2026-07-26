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

func TestStartAdvisorChecks(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", []string(nil)).Return("", errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to start advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("Advisors disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", []string(nil)).Return("", services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})
}

func TestTestAdvisorCheck(t *testing.T) {
	t.Parallel()

	apiCheck := &advisorsv1.AdvisorCheck{
		Name:        "custom_mysql_version",
		Summary:     "Check summary",
		Description: "Check description",
		Category:    "configuration",
		Subcategory: "version",
		Technology:  advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL,
		Interval:    advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD,
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
			Subcategory: "version",
			Technology:  check.MySQL,
			Interval:    check.Standard,
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
			Subcategory: "test_advisor",
			Category:    "configuration",
			Interval:    models.Standard,
			ServiceID:   "test_svc",
			ServiceName: "svc",
			ServiceType: models.MySQLServiceType,
			NodeID:      "node1",
			NodeName:    "node",
			Status:      models.CheckResultFailed,
			Summary:     "Check summary",
			Description: "Check Description",
			ReadMoreURL: "https://www.example.com",
			Severity:    models.Severity(common.Critical),
			CheckedAt:   checkedAt,
		}
		require.NoError(t, record.SetLabels(map[string]string{"label_key": "label_value"}))

		var checksService mockChecksService
		checksService.On("GetInsights", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]*models.Insight{record}, 3, nil)

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
					Subcategory: "test_advisor",
					Category:    "configuration",
					Interval:    advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD,
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
				},
			},
			TotalItems: 3,
			TotalPages: 2,
		}
		assert.Equal(t, expected, resp)
	})
}

func TestMarkInsightsRead(t *testing.T) {
	t.Parallel()

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("MarkInsightsRead", mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.MarkInsightsRead(t.Context(), &advisorsv1.MarkInsightsReadRequest{
			Ids:    []string{"id1"},
			IsRead: true,
		})
		require.EqualError(t, err, "failed to mark insights read: random error")
		assert.Nil(t, resp)
	})

	t.Run("passes ids and read state through", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("MarkInsightsRead", mock.Anything, []string{"id1", "id2"}, true).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.MarkInsightsRead(t.Context(), &advisorsv1.MarkInsightsReadRequest{
			Ids:    []string{"id1", "id2"},
			IsRead: true,
		})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.MarkInsightsReadResponse{}, resp)
		checksService.AssertExpectations(t)
	})

	t.Run("converts filters", func(t *testing.T) {
		t.Parallel()

		severity := models.Severity(common.Warning)
		status := models.CheckResultFailed
		var checksService mockChecksService
		checksService.On("MarkInsightsReadByFilters", mock.Anything, models.InsightFilters{
			ServiceName: "mysql-prod",
			NodeName:    "node-1",
			Category:    "security",
			BatchID:     "batch-1",
			Severity:    &severity,
			Status:      &status,
			IsRead:      new(false),
		}, true).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.MarkInsightsRead(t.Context(), &advisorsv1.MarkInsightsReadRequest{
			IsRead: true,
			Filters: &advisorsv1.InsightsFilters{
				ServiceName: "mysql-prod",
				NodeName:    "node-1",
				Category:    "security",
				BatchId:     "batch-1",
				Severity:    new(managementv1.Severity_SEVERITY_WARNING),
				Status:      new(advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_FAILED),
				IsRead:      new(false),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.MarkInsightsReadResponse{}, resp)
		checksService.AssertExpectations(t)
	})

	t.Run("empty filters match every record", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("MarkInsightsReadByFilters", mock.Anything, models.InsightFilters{}, true).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.MarkInsightsRead(t.Context(), &advisorsv1.MarkInsightsReadRequest{
			IsRead:  true,
			Filters: &advisorsv1.InsightsFilters{},
		})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.MarkInsightsReadResponse{}, resp)
		checksService.AssertExpectations(t)
	})

	t.Run("requires ids or filters", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		s := NewChecksAPIService(&checksService)

		resp, err := s.MarkInsightsRead(t.Context(), &advisorsv1.MarkInsightsReadRequest{
			IsRead: true,
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Either ids or filters must be provided."), err)
		assert.Nil(t, resp)
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
				"one":   {Name: "one", Interval: check.Standard},
				"two":   {Name: "two", Interval: check.Frequent},
				"three": {Name: "three", Interval: check.Rare},
				"four":  {Name: "four", Interval: ""},
			}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisorChecks(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.ElementsMatch(
			t, resp.Checks,
			[]*advisorsv1.AdvisorCheck{
				{Name: "one", Enabled: true, Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD},
				{Name: "two", Enabled: false, Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_FREQUENT},
				{Name: "three", Enabled: true, Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE, DisabledServiceIds: []string{"svc-1", "svc-2"}},
				{Name: "four", Enabled: true, Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD},
			},
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

	t.Run("change interval error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("ChangeInterval", mock.Anything, mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{
			Params: []*advisorsv1.ChangeAdvisorCheckParams{{
				Name:     "check-name",
				Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD,
			}},
		})
		require.EqualError(t, err, "failed to change advisor check interval: random error")
		assert.Nil(t, resp)
	})

	t.Run("ChangeInterval success", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("ChangeInterval", mock.Anything, mock.Anything).Return(nil)
		checksService.On("EnableChecks", mock.Anything, mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything, mock.Anything).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{
			Params: []*advisorsv1.ChangeAdvisorCheckParams{{
				Name:     "check-name",
				Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED,
			}},
		})
		require.NoError(t, err)
		assert.Equal(t, &advisorsv1.ChangeAdvisorChecksResponse{}, resp)
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

	t.Run("interval change with services rejected", func(t *testing.T) {
		var checksService mockChecksService

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{
			Params: []*advisorsv1.ChangeAdvisorCheckParams{{
				Name:       "check-name",
				Interval:   advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE,
				ServiceIds: []string{"svc-1"},
			}},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, resp)
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
