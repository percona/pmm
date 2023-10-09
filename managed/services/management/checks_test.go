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
	"fmt"
	"testing"

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestStartSecurityChecks(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", []string(nil)).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartSecurityChecks(context.Background(), &managementpb.StartSecurityChecksRequest{})
		assert.EqualError(t, err, "failed to start security checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("STT disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", []string(nil)).Return(services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartSecurityChecks(context.Background(), &managementpb.StartSecurityChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})
}

func TestGetSecurityCheckResults(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetSecurityCheckResults", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetSecurityCheckResults(context.Background(), nil)
		assert.EqualError(t, err, "failed to get security check results: random error")
		assert.Nil(t, resp)
	})

	t.Run("STT disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetSecurityCheckResults", mock.Anything).Return(nil, services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetSecurityCheckResults(context.Background(), nil)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Advisor checks are disabled."), err)
		assert.Nil(t, resp)
	})

	t.Run("STT enabled", func(t *testing.T) {
		checkResult := []services.CheckResult{
			{
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreURL: "https://www.example.com",
					Severity:    1,
					Labels:      map[string]string{"label_key": "label_value"},
				},
				Target: services.Target{ServiceName: "svc"},
			},
		}
		response := &managementpb.GetSecurityCheckResultsResponse{ //nolint:staticcheck
			Results: []*managementpb.SecurityCheckResult{
				{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreUrl: "https://www.example.com",
					Severity:    1,
					Labels:      map[string]string{"label_key": "label_value"},
					ServiceName: "svc",
				},
			},
		}
		var checksService mockChecksService
		checksService.On("GetSecurityCheckResults", mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetSecurityCheckResults(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, resp, response)
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

		resp, err := s.GetFailedChecks(context.Background(), &managementpb.GetFailedChecksRequest{
			ServiceId: serviceID,
		})
		assert.EqualError(t, err, fmt.Sprintf("failed to get check results for service '%s': random error", serviceID))
		assert.Nil(t, resp)
	})

	t.Run("STT disabled error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(nil, services.ErrAdvisorsDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(context.Background(), &managementpb.GetFailedChecksRequest{
			ServiceId: "test_svc",
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Advisor checks are disabled."), err)
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
		response := &managementpb.GetFailedChecksResponse{
			Results: []*managementpb.CheckResult{
				{
					Summary:     "Check summary",
					Description: "Check Description",
					ReadMoreUrl: "https://www.example.com",
					Severity:    managementpb.Severity(common.Emergency),
					Labels:      map[string]string{"label_key": "label_value"},
					ServiceName: "svc",
					ServiceId:   "test_svc",
					CheckName:   "test_check",
				},
			},
			PageTotals: &managementpb.PageTotals{
				TotalPages: 1,
				TotalItems: 1,
			},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(context.Background(), &managementpb.GetFailedChecksRequest{
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
		response := &managementpb.GetFailedChecksResponse{
			Results: []*managementpb.CheckResult{
				{
					Summary:     "Check summary 2",
					Description: "Check Description 2",
					ReadMoreUrl: "https://www.example.com",
					Severity:    managementpb.Severity(common.Warning),
					Labels:      map[string]string{"label_key": "label_value"},
					ServiceName: "svc",
					ServiceId:   "test_svc",
					CheckName:   "test_check2",
				},
			},
			PageTotals: &managementpb.PageTotals{
				TotalPages: 3,
				TotalItems: 3,
			},
		}
		var checksService mockChecksService
		checksService.On("GetChecksResults", mock.Anything, mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetFailedChecks(context.Background(), &managementpb.GetFailedChecksRequest{
			ServiceId: "test_svc",
			PageParams: &managementpb.PageParams{
				PageSize: 1,
				Index:    1,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, response, resp)
	})
}

func TestListFailedServices(t *testing.T) {
	t.Parallel()

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("GetSecurityCheckResults", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListFailedServices(context.Background(), &managementpb.ListFailedServicesRequest{})
		assert.EqualError(t, err, "failed to get check results: random error")
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
		response := &managementpb.ListFailedServicesResponse{
			Result: []*managementpb.CheckResultSummary{
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
		checksService.On("GetSecurityCheckResults", mock.Anything).Return(checkResult, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListFailedServices(context.Background(), &managementpb.ListFailedServicesRequest{})
		require.NoError(t, err)
		assert.ElementsMatch(t, resp.Result, response.Result)
	})
}

func TestListSecurityChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string{"two"}, nil)
		checksService.On("GetChecks", mock.Anything).
			Return(map[string]check.Check{
				"one":   {Name: "one", Interval: check.Standard},
				"two":   {Name: "two", Interval: check.Frequent},
				"three": {Name: "three", Interval: check.Rare},
				"four":  {Name: "four", Interval: ""},
			}, nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListSecurityChecks(context.Background(), nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.ElementsMatch(t, resp.Checks,
			[]*managementpb.SecurityCheck{
				{Name: "one", Disabled: false, Interval: managementpb.SecurityCheckInterval_STANDARD},
				{Name: "two", Disabled: true, Interval: managementpb.SecurityCheckInterval_FREQUENT},
				{Name: "three", Disabled: false, Interval: managementpb.SecurityCheckInterval_RARE},
				{Name: "four", Disabled: false, Interval: managementpb.SecurityCheckInterval_STANDARD},
			},
		)
	})

	t.Run("get disabled checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return(nil, errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListSecurityChecks(context.Background(), nil)
		assert.EqualError(t, err, "failed to get disabled checks list: random error")
		assert.Nil(t, resp)
	})
}

func TestUpdateSecurityChecks(t *testing.T) {
	t.Run("enable security checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecks", mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeSecurityChecks(context.Background(), &managementpb.ChangeSecurityChecksRequest{})
		assert.EqualError(t, err, "failed to enable disabled security checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("disable security checks error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("EnableChecks", mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeSecurityChecks(context.Background(), &managementpb.ChangeSecurityChecksRequest{})
		assert.EqualError(t, err, "failed to disable security checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("change interval error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("ChangeInterval", mock.Anything).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeSecurityChecks(context.Background(), &managementpb.ChangeSecurityChecksRequest{
			Params: []*managementpb.ChangeSecurityCheckParams{{
				Name:     "check-name",
				Interval: managementpb.SecurityCheckInterval_STANDARD,
			}},
		})
		assert.EqualError(t, err, "failed to change security check interval: random error")
		assert.Nil(t, resp)
	})

	t.Run("ChangeInterval success", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("ChangeInterval", mock.Anything).Return(nil)
		checksService.On("EnableChecks", mock.Anything).Return(nil)
		checksService.On("DisableChecks", mock.Anything).Return(nil)

		s := NewChecksAPIService(&checksService)

		resp, err := s.ChangeSecurityChecks(context.Background(), &managementpb.ChangeSecurityChecksRequest{
			Params: []*managementpb.ChangeSecurityCheckParams{{
				Name:     "check-name",
				Interval: managementpb.SecurityCheckInterval_STANDARD,
			}},
		})
		require.NoError(t, err)
		assert.Equal(t, &managementpb.ChangeSecurityChecksResponse{}, resp)
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
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.Comment, createComment(tc.Checks))
		})
	}
}
