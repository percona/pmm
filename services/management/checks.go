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

	"github.com/percona/pmm/api/managementpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services"
)

// ChecksAPIService represents security checks service API.
type ChecksAPIService struct {
	checksService checksService
}

// NewChecksAPIService creates new Checks API Service.
func NewChecksAPIService(checksService checksService) *ChecksAPIService {
	return &ChecksAPIService{checksService: checksService}
}

// StartSecurityChecks starts STT checks execution.
func (s *ChecksAPIService) StartSecurityChecks(ctx context.Context) (*managementpb.StartSecurityChecksResponse, error) {
	err := s.checksService.StartChecks(ctx)
	if err != nil {
		if err == services.ErrSTTDisabled {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, status.Error(codes.Internal, "Failed to start security checks.")
	}

	return &managementpb.StartSecurityChecksResponse{}, nil
}

// GetSecurityCheckResults returns the results of the STT checks that were run.
func (s *ChecksAPIService) GetSecurityCheckResults() (*managementpb.GetSecurityCheckResultsResponse, error) {
	results, err := s.checksService.GetSecurityCheckResults()
	if err != nil {
		if err == services.ErrSTTDisabled {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, status.Error(codes.Internal, "Failed to get security check results.")
	}

	checkResults := make([]*managementpb.SecurityCheckResult, 0, len(results))
	for _, result := range results {
		checkResults = append(checkResults, &managementpb.SecurityCheckResult{
			Summary:     result.Summary,
			Description: result.Description,
			Severity:    managementpb.Severity(result.Severity),
			Labels:      result.Labels,
		})
	}

	return &managementpb.GetSecurityCheckResultsResponse{Results: checkResults}, nil
}
