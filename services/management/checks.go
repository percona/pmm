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

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/api/managementpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services"
)

// ChecksAPIService represents security checks service API.
type ChecksAPIService struct {
	checksService checksService
	l             *logrus.Entry

	managementpb.UnimplementedSecurityChecksServer
}

// NewChecksAPIService creates new Checks API Service.
func NewChecksAPIService(checksService checksService) *ChecksAPIService {
	return &ChecksAPIService{
		checksService: checksService,
		l:             logrus.WithField("component", "management/checks"),
	}
}

// GetSecurityCheckResults returns Security Thread Tool's latest checks results.
func (s *ChecksAPIService) GetSecurityCheckResults(ctx context.Context, req *managementpb.GetSecurityCheckResultsRequest) (*managementpb.GetSecurityCheckResultsResponse, error) {
	results, err := s.checksService.GetSecurityCheckResults()
	if err != nil {
		if err == services.ErrSTTDisabled {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrap(err, "failed to get security check results")
	}

	checkResults := make([]*managementpb.SecurityCheckResult, 0, len(results))
	for _, result := range results {
		checkResults = append(checkResults, &managementpb.SecurityCheckResult{
			Summary:     result.Result.Summary,
			Description: result.Result.Description,
			ReadMoreUrl: result.Result.ReadMoreURL,
			Severity:    managementpb.Severity(result.Result.Severity),
			Labels:      result.Result.Labels,
			ServiceName: result.Target.ServiceName,
		})
	}

	return &managementpb.GetSecurityCheckResultsResponse{Results: checkResults}, nil
}

// StartSecurityChecks executes Security Thread Tool checks and returns when all checks are executed.
func (s *ChecksAPIService) StartSecurityChecks(ctx context.Context, req *managementpb.StartSecurityChecksRequest) (*managementpb.StartSecurityChecksResponse, error) {
	// Start only specified checks from any group.
	err := s.checksService.StartChecks(req.Names)
	if err != nil {
		if errors.Is(err, services.ErrSTTDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrap(err, "failed to start security checks")
	}

	return &managementpb.StartSecurityChecksResponse{}, nil
}

// ListSecurityChecks returns a list of available Security Thread Tool checks and their statuses.
func (s *ChecksAPIService) ListSecurityChecks(ctx context.Context, req *managementpb.ListSecurityChecksRequest) (*managementpb.ListSecurityChecksResponse, error) {
	disChecks, err := s.checksService.GetDisabledChecks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get disabled checks list")
	}

	m := make(map[string]struct{}, len(disChecks))
	for _, c := range disChecks {
		m[c] = struct{}{}
	}

	checks, err := s.checksService.GetChecks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available checks list")
	}

	res := make([]*managementpb.SecurityCheck, 0, len(checks))
	for _, c := range checks {
		_, disabled := m[c.Name]
		res = append(res, &managementpb.SecurityCheck{
			Name:        c.Name,
			Disabled:    disabled,
			Summary:     c.Summary,
			Description: c.Description,
			Interval:    convertInterval(c.Interval),
		})
	}

	return &managementpb.ListSecurityChecksResponse{Checks: res}, nil
}

// ChangeSecurityChecks enables/disables Security Thread Tool checks by names or changes its execution interval.
func (s *ChecksAPIService) ChangeSecurityChecks(ctx context.Context, req *managementpb.ChangeSecurityChecksRequest) (*managementpb.ChangeSecurityChecksResponse, error) {
	var enableChecks, disableChecks []string
	changeIntervalParams := make(map[string]check.Interval)
	for _, check := range req.Params {
		if check.Enable && check.Disable {
			return nil, status.Errorf(codes.InvalidArgument, "Check %s has enable and disable parameters set to the true.", check.Name)
		}

		if check.Interval != managementpb.SecurityCheckInterval_SECURITY_CHECK_INTERVAL_INVALID {
			interval, err := convertAPIInterval(check.Interval)
			if err != nil {
				return nil, errors.Wrap(err, "failed to change security check interval")
			}
			changeIntervalParams[check.Name] = interval
		}

		if check.Enable {
			enableChecks = append(enableChecks, check.Name)
		}

		if check.Disable {
			disableChecks = append(disableChecks, check.Name)
		}
	}

	if len(changeIntervalParams) != 0 {
		err := s.checksService.ChangeInterval(changeIntervalParams)
		if err != nil {
			return nil, errors.Wrap(err, "failed to change security check interval")
		}
	}

	err := s.checksService.EnableChecks(enableChecks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enable disabled security checks")
	}

	err = s.checksService.DisableChecks(disableChecks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to disable security checks")
	}

	return &managementpb.ChangeSecurityChecksResponse{}, nil
}

// convertInterval converts check.Interval type to managementpb.SecurityCheckInterval.
func convertInterval(interval check.Interval) managementpb.SecurityCheckInterval {
	switch interval {
	case check.Standard:
		return managementpb.SecurityCheckInterval_STANDARD
	case check.Frequent:
		return managementpb.SecurityCheckInterval_FREQUENT
	case check.Rare:
		return managementpb.SecurityCheckInterval_RARE
	default:
		return managementpb.SecurityCheckInterval_SECURITY_CHECK_INTERVAL_INVALID
	}
}

// convertAPIInterval converts managementpb.SecurityCheckInterval type to check.Interval.
func convertAPIInterval(interval managementpb.SecurityCheckInterval) (check.Interval, error) {
	switch interval {
	case managementpb.SecurityCheckInterval_STANDARD:
		return check.Standard, nil
	case managementpb.SecurityCheckInterval_FREQUENT:
		return check.Frequent, nil
	case managementpb.SecurityCheckInterval_RARE:
		return check.Rare, nil
	case managementpb.SecurityCheckInterval_SECURITY_CHECK_INTERVAL_INVALID:
		return check.Interval(""), errors.New("invalid security check interval")
	default:
		return check.Interval(""), errors.New("unknown security check interval")
	}
}
