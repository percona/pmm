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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestStartAdvisorChecks(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", []string(nil)).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to start advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("Advisors disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", []string(nil)).Return(services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartAdvisorChecks(t.Context(), &advisorsv1.StartAdvisorChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})
}

func TestGetFailedChecks(t *testing.T) {
	t.Parallel()

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)
		serviceID := "test_svc"

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: serviceID,
		})
		require.EqualError(t, err, fmt.Sprintf("failed to get check results for service '%s': random error", serviceID))
		assert.Nil(t, resp)
	})

	t.Run("Advisors disabled error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(nil, services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})

	t.Run("get failed checks for requested service", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Emergency,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc", ServiceID: "test_svc"},
				CheckName: "test_check",
			},
		}
		response := &advisorsv1.GetFailedChecksResponse{
			Results: []*advisorsv1.CheckResult{
				{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreUrl: "https://www.example.com",
					Severity:    managementv1.Severity(common.Emergency),
					Labels:      map[string]string{"label_key": "label_value"},
					ServiceName: "svc",
					ServiceId:   "test_svc",
					CheckName:   "test_check",
				},
			},
			TotalPages: 1,
			TotalItems: 1,
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
		})
		require.NoError(t, err)
		assert.Equal(t, response, resp)
	})

	t.Run("get failed checks with pagination", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Critical,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc", ServiceID: "test_svc"},
				CheckName: "test_check1",
			},
			{
				Result: check.Result{
					Summary:     "Check summary 2",
					Description: "Check Description 2",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc", ServiceID: "test_svc"},
				CheckName: "test_check2",
			},
			{
				Result: check.Result{
					Summary:     "Check summary 3",
					Description: "Check Description 3",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Notice,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc", ServiceID: "test_svc"},
				CheckName: "test_check3",
			},
		}
		response := &advisorsv1.GetFailedChecksResponse{
			Results: []*advisorsv1.CheckResult{
				{
					Summary:     "Check summary 2",
					Description: "Check Description 2",
					ReadMoreUrl: "https://www.example.com",
					Severity:    managementv1.Severity(common.Warning),
					Labels:      map[string]string{"label_key": "label_value"},
					ServiceName: "svc",
					ServiceId:   "test_svc",
					CheckName:   "test_check2",
				},
			},
			TotalPages: 3,
			TotalItems: 3,
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageSize:  new(int32(1)),
			PageIndex: new(int32(1)),
		})
		require.NoError(t, err)
		assert.Equal(t, response, resp)
	})

	t.Run("get failed checks pagination calculations", func(t *testing.T) {
		t.Parallel()

		// Setup 5 results: with page size 2, we expect 3 pages
		checkResult := make([]services.CheckResult, 5)
		for i := range checkResult {
			checkResult[i].CheckName = fmt.Sprintf("check_%d", i)
			checkResult[i].Target = services.Target{ServiceName: "svc", ServiceID: "test_svc"}
		}

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageSize:  new(int32(2)),
			PageIndex: new(int32(2)), // Requesting the 3rd page (index 2)
		})
		require.NoError(t, err)
		assert.Equal(t, int32(5), resp.TotalItems)
		assert.Equal(t, int32(3), resp.TotalPages)
		assert.Len(t, resp.Results, 1) // Only one item left for the last page
		assert.Equal(t, "check_4", resp.Results[0].CheckName)
	})

	t.Run("get failed checks with zero items", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(nil, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageSize:  new(int32(10)),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.TotalItems)
		// When pageSize is provided, totalPages is calculated from results (0/10 = 0)
		assert.Equal(t, int32(0), resp.TotalPages)
		assert.Empty(t, resp.Results)
	})

	t.Run("get failed checks with nil pagination", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{CheckName: "check1", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
			{CheckName: "check2", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageIndex: nil,
			PageSize:  nil,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.TotalItems)
		assert.Equal(t, int32(1), resp.TotalPages)
		assert.Len(t, resp.Results, 2)
	})

	t.Run("get failed checks with zero page size", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{CheckName: "check1", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageSize:  new(int32(0)),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.TotalItems)
		assert.Len(t, resp.Results, 1)
	})

	t.Run("get failed checks with page index out of bounds", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{CheckName: "check1", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageIndex: new(int32(100)),
			PageSize:  new(int32(10)),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.TotalItems)
		assert.Empty(t, resp.Results)
	})

	t.Run("get failed checks with negative page index", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{CheckName: "check1", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		// Negative page index/size should be clamped to 0. PageSize 0 means return "all".
		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageIndex: new(int32(-1)),
			PageSize:  new(int32(10)),
		})
		require.ErrorContains(t, err, "page index must be non-negative")
		require.Nil(t, resp)
	})

	t.Run("get failed checks with negative page size", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{CheckName: "check1", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		// Negative page index/size should be clamped to 0. PageSize 0 means return "all".
		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageIndex: new(int32(1)),
			PageSize:  new(int32(-10)),
		})
		require.ErrorContains(t, err, "page size must be non-negative")
		require.Nil(t, resp)
	})

	t.Run("get failed checks with pagination overflow", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{CheckName: "check1", Target: services.Target{ServiceName: "svc", ServiceID: "test_svc"}},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		// Extremely large values should not cause a panic and should be handled by wider integer math.
		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageIndex: new(int32(math.MaxInt32)),
			PageSize:  new(int32(math.MaxInt32)),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.TotalItems)
		assert.Empty(t, resp.Results)
	})

	t.Run("get failed checks with severity overflow", func(t *testing.T) {
		t.Parallel()

		// Simulate a check result with a severity value that exceeds int32 range
		// to test the clamping logic in the service.
		checkResult := []services.CheckResult{
			{
				Result: check.Result{
					Summary:  "Overflow severity check",
					Severity: common.Severity(math.MaxInt32 + 1),
				},
				Target:    services.Target{ServiceName: "svc", ServiceID: "test_svc"},
				CheckName: "overflow_check",
			},
		}

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, "test_svc").Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(t.Context(), &advisorsv1.GetFailedChecksRequest{
			ServiceId: "test_svc",
		})
		require.Nil(t, resp)
		require.ErrorContains(t, err, "check result severity 2147483648 is out of range for int32")
	})
}

func TestListFailedServices(t *testing.T) {
	t.Parallel()

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListFailedServices(t.Context(), &advisorsv1.ListFailedServicesRequest{})
		require.EqualError(t, err, "failed to get check results: random error")
		assert.Nil(t, resp)
	})

	t.Run("list services with failed checks", func(t *testing.T) {
		t.Parallel()

		checkResult := []services.CheckResult{
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Critical,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc1", ServiceID: "test_svc1"},
				CheckName: "test_check",
			},
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Error,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc1", ServiceID: "test_svc1"},
				CheckName: "test_check",
			},
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Emergency,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc1", ServiceID: "test_svc1"},
				CheckName: "test_check",
			},
			{
				Result: check.Result{
					Summary:     "Check summary 2",
					Description: "Check Description 2",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target:    services.Target{ServiceName: "svc2", ServiceID: "test_svc2"},
				CheckName: "test_check",
			},
		}
		response := &advisorsv1.ListFailedServicesResponse{
			Result: []*advisorsv1.CheckResultSummary{
				{
					ServiceName:    "svc1",
					ServiceId:      "test_svc1",
					EmergencyCount: 1,
					CriticalCount:  1,
					ErrorCount:     1,
				},
				{
					ServiceName:  "svc2",
					ServiceId:    "test_svc2",
					WarningCount: 1,
				},
			},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListFailedServices(t.Context(), &advisorsv1.ListFailedServicesRequest{})
		require.NoError(t, err)
		assert.ElementsMatch(t, resp.Result, response.Result)
	})
}

func TestListAdvisorChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string{"two"}, nil)
		checksService.On("GetChecks", mock.Anything).
			Return(map[string]check.Check{
				"one":   {Version: 1, Name: "one", Interval: check.Standard, Type: check.MySQLShow},
				"two":   {Version: 2, Name: "two", Interval: check.Frequent, Family: check.PostgreSQL},
				"three": {Version: 2, Name: "three", Interval: check.Rare, Family: check.MongoDB},
				// Version 1 check with a type that maps to MongoDB
				"four": {Version: 1, Name: "four", Interval: "", Type: check.MongoDBBuildInfo},
			}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisorChecks(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.ElementsMatch(
			t, resp.Checks,
			[]*advisorsv1.AdvisorCheck{
				{
					Name:     "one",
					Enabled:  true,
					Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD,
					Family:   advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MYSQL,
				},
				{
					Name:     "two",
					Enabled:  false,
					Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_FREQUENT,
					Family:   advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_POSTGRESQL,
				},
				{
					Name:     "three",
					Enabled:  true,
					Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE,
					Family:   advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MONGODB,
				},
				{
					Name:     "four",
					Enabled:  true,
					Interval: advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD,
					Family:   advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MONGODB,
				},
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

func TestListAdvisors(t *testing.T) {
	t.Parallel()

	t.Run("normal", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string{}, nil)
		checksService.On("GetAdvisors", mock.Anything).
			Return([]check.Advisor{
				{
					Name:        "Advisor 1",
					Description: "Description 1",
					Summary:     "Summary 1",
					Category:    "Category 1",
					Checks: []check.Check{
						{Version: 2, Name: "check1", Family: check.MySQL},
					},
				},
			}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListAdvisors(t.Context(), &advisorsv1.ListAdvisorsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Advisors, 1)

		adv := resp.Advisors[0]
		assert.Equal(t, "Advisor 1", adv.Name)
		assert.Equal(t, "Category 1", adv.Category)
		assert.Len(t, adv.Checks, 1)
		assert.Equal(t, "check1", adv.Checks[0].Name)
		assert.Equal(t, advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MYSQL, adv.Checks[0].Family)
		// Verify the comment generation logic is called
		assert.Equal(t, "Partial support (MySQL)", adv.Comment)
	})
}

func TestUpdateAdvisorChecks(t *testing.T) {
	t.Run("enable advisor checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecks", mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to enable disabled advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("disable advisor checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecks", mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeAdvisorChecks(t.Context(), &advisorsv1.ChangeAdvisorChecksRequest{})
		require.EqualError(t, err, "failed to disable advisor checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("change interval error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("ChangeInterval", mock.Anything).Return(errors.New("random error"))

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
		checksService.On("ChangeInterval", mock.Anything).Return(nil)
		checksService.On("EnableChecks", mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything).Return(nil)

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
}

func TestCreateComment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name    string
		Comment string
		Checks  []check.Check
	}{
		{
			Name:    "all technologies",
			Comment: "All technologies supported",
			Checks: []check.Check{
				{Version: 1, Name: "a", Type: check.MySQLShow},
				{Version: 1, Name: "b", Type: check.PostgreSQLSelect},
				{Version: 2, Name: "c", Family: check.MongoDB},
			},
		},
		{
			Name:    "partial support",
			Comment: "Partial support (MySQL, MongoDB)",
			Checks: []check.Check{
				{Version: 1, Name: "a", Type: check.MySQLShow},
				{Version: 2, Name: "b", Family: check.MongoDB},
			},
		},
		{
			Name:    "partial support",
			Comment: "Partial support (MySQL)",
			Checks: []check.Check{
				{Version: 1, Name: "a", Type: check.MySQLShow},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.Comment, createComment(tc.Checks))
		})
	}
}
