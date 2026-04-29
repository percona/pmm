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
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	managementv1 "github.com/percona/pmm/api/management/v1"
)

// Install-token lifetime is intentionally short: the token is only used for the
// initial pmm-admin config handshake, after which pmm-agent stores its own
// per-agent identity and no longer needs it. A 15-minute window covers normal
// install runs while limiting damage if the URL leaks. The Grafana install service
// account uses org Admin; the token TTL is capped at 15 minutes — treat it like a password.
const (
	defaultInstallTokenTTLSeconds = int64(15 * 60) // 15 minutes
	maxInstallTokenTTLSeconds     = int64(15 * 60) // 15 minutes; hard cap, no caller can exceed
	minInstallTokenTTLSeconds     = int64(60)      // 1 minute floor to leave room for slow installs
)

// IMPORTANT: keep this list in sync with the `Technology` union in
// ui/apps/pmm/src/pages/install-client/InstallClientPage.utils.ts — adding a tech
// to one without the other yields either a UI-unreachable code path (server-only)
// or an InvalidArgument at runtime (UI-only).
var installTokenTechnologies = map[string]struct{}{
	"mysql":      {},
	"postgresql": {},
	"mongodb":    {},
	"valkey":     {},
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
	switch {
	case ttl == 0:
		ttl = defaultInstallTokenTTLSeconds
	case ttl < minInstallTokenTTLSeconds:
		ttl = minInstallTokenTTLSeconds
	case ttl > maxInstallTokenTTLSeconds:
		ttl = maxInstallTokenTTLSeconds
	}

	saID, tok, exp, err := s.grafanaClient.CreateNodeInstallToken(ctx, tech, ttl)
	if err != nil {
		return nil, err
	}

	return &managementv1.CreateNodeInstallTokenResponse{
		Token:            tok,
		ExpiresAt:        timestamppb.New(exp),
		ServiceAccountId: saID,
	}, nil
}
