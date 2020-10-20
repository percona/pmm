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

package grpc

import (
	"context"

	"github.com/percona/pmm/api/managementpb"

	"github.com/percona/pmm-managed/services/management"
)

type checksServer struct {
	svc *management.ChecksAPIService
}

// NewChecksServer creates Management Checks Server.
func NewChecksServer(s *management.ChecksAPIService) managementpb.SecurityChecksServer {
	return &checksServer{svc: s}
}

//  GetSecurityCheckResults returns the results of the STT checks that were run.
func (s *checksServer) GetSecurityCheckResults(ctx context.Context, request *managementpb.GetSecurityCheckResultsRequest) (*managementpb.GetSecurityCheckResultsResponse, error) {
	return s.svc.GetSecurityCheckResults()
}

// StartSecurityChecks starts STT checks execution.
func (s *checksServer) StartSecurityChecks(ctx context.Context, request *managementpb.StartSecurityChecksRequest) (*managementpb.StartSecurityChecksResponse, error) {
	return s.svc.StartSecurityChecks(ctx)
}

// ListSecurityChecks returns all available STT checks.
func (s *checksServer) ListSecurityChecks(ctx context.Context, req *managementpb.ListSecurityChecksRequest) (*managementpb.ListSecurityChecksResponse, error) {
	return s.svc.ListSecurityChecks()
}

// ChangeSecurityCheck allows to change STT checks state.
func (s *checksServer) ChangeSecurityChecks(ctx context.Context, req *managementpb.ChangeSecurityChecksRequest) (*managementpb.ChangeSecurityChecksResponse, error) {
	return s.svc.ChangeSecurityChecks(req)
}
