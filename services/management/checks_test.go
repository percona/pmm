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

package management

import (
	"context"
	"testing"

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/api/managementpb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestStartSecurityChecks(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything, check.Interval(""), []string(nil)).Return(errors.New("random error"))

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartSecurityChecks(context.Background(), &managementpb.StartSecurityChecksRequest{})
		assert.EqualError(t, err, "failed to start security checks: random error")
		assert.Nil(t, resp)
	})

	t.Run("STT disabled error", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("StartChecks", mock.Anything, check.Interval(""), []string(nil)).Return(services.ErrSTTDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.StartSecurityChecks(context.Background(), &managementpb.StartSecurityChecksRequest{})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "STT is disabled."), err)
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
		checksService.On("GetSecurityCheckResults", mock.Anything).Return(nil, services.ErrSTTDisabled)

		s := NewChecksAPIService(&checksService)

		resp, err := s.GetSecurityCheckResults(context.Background(), nil)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "STT is disabled."), err)
		assert.Nil(t, resp)
	})

	t.Run("STT enabled", func(t *testing.T) {
		checkResult := []services.STTCheckResult{
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
		response := &managementpb.GetSecurityCheckResultsResponse{
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

func TestListSecurityChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		var checksService mockChecksService
		checksService.On("GetDisabledChecks", mock.Anything).Return([]string{"two"}, nil)
		checksService.On("GetAllChecks", mock.Anything).
			Return(map[string]check.Check{
				"one":   {Name: "one"},
				"two":   {Name: "two"},
				"three": {Name: "three"},
			})

		s := NewChecksAPIService(&checksService)

		resp, err := s.ListSecurityChecks(context.Background(), nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.ElementsMatch(t, resp.Checks,
			[]*managementpb.SecurityCheck{
				{Name: "one", Disabled: false},
				{Name: "two", Disabled: true},
				{Name: "three", Disabled: false},
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
