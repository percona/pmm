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
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	managementv1 "github.com/percona/pmm/api/management/v1"
)

const (
	defaultInstallTokenTTLSeconds = int64(86400)
	maxInstallTokenTTLSeconds     = int64(86400)
	minInstallTokenTTLSeconds     = int64(60)
)

var installTokenTechnologies = map[string]struct{}{
	"mysql":       {},
	"postgresql":  {},
	"mongodb":     {},
	"valkey":      {},
}

// CreateNodeInstallToken mints a short-lived Grafana token for PMM Client install; it does not create inventory rows.
func (s *ManagementService) CreateNodeInstallToken(
	ctx context.Context,
	req *managementv1.CreateNodeInstallTokenRequest,
) (*managementv1.CreateNodeInstallTokenResponse, error) {
	tech := strings.ToLower(strings.TrimSpace(req.GetTechnology()))
	if _, ok := installTokenTechnologies[tech]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported technology %q (expected mysql, postgresql, mongodb, or valkey)", req.GetTechnology())
	}

	ttl := int64(req.GetTtlSeconds())
	if ttl == 0 {
		ttl = defaultInstallTokenTTLSeconds
	}
	if ttl < minInstallTokenTTLSeconds {
		ttl = minInstallTokenTTLSeconds
	}
	if ttl > maxInstallTokenTTLSeconds {
		ttl = maxInstallTokenTTLSeconds
	}

	unique := fmt.Sprintf("%s-%d", tech, time.Now().UnixNano())
	saID, tok, exp, err := s.grafanaClient.CreateNodeInstallToken(ctx, unique, ttl)
	if err != nil {
		return nil, err
	}

	return &managementv1.CreateNodeInstallTokenResponse{
		Token:            tok,
		ExpiresAt:        timestamppb.New(exp),
		ServiceAccountId: saID,
	}, nil
}
