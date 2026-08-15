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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/auth"
)

func toAPIEnrollmentToken(token *models.EnrollmentToken) *managementv1.EnrollmentToken {
	res := &managementv1.EnrollmentToken{
		TokenHash:   token.TokenHash,
		Description: token.Description,
		MaxUses:     int32(token.MaxUses),   //nolint:gosec
		UsedCount:   int32(token.UsedCount), //nolint:gosec
		CreatedAt:   timestamppb.New(token.CreatedAt),
	}
	if token.ExpiresAt != nil {
		res.ExpiresAt = timestamppb.New(*token.ExpiresAt)
	}

	return res
}

// CreateEnrollmentToken mints a token that authorizes enrolling Nodes.
//
// The token is returned once, here, and only its hash is kept. Managing these tokens is an
// administrative act and stays behind the admin rule, so that handing an ops team the
// ability to enroll nodes does not mean handing them the ability to mint that ability.
func (s *ManagementService) CreateEnrollmentToken(ctx context.Context, req *managementv1.CreateEnrollmentTokenRequest) (*managementv1.CreateEnrollmentTokenResponse, error) { //nolint:lll
	err := s.requireUnscopedCaller(ctx)
	if err != nil {
		return nil, err
	}

	params := &models.CreateEnrollmentTokenParams{
		Description: req.Description,
		MaxUses:     int(req.MaxUses),
	}
	if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.AsTime()
		params.ExpiresAt = &expiresAt
	}

	var row *models.EnrollmentToken
	var token string
	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		row, token, err = models.CreateEnrollmentToken(tx.Querier, params)
		return err
	})
	if errTX != nil {
		return nil, errTX
	}

	s.l.WithField("description", row.Description).Info("Created an enrollment token.")

	return &managementv1.CreateEnrollmentTokenResponse{
		EnrollmentToken: toAPIEnrollmentToken(row),
		Token:           token,
	}, nil
}

// ListEnrollmentTokens lists enrollment tokens. Token values are not stored, so they are
// never returned; a token is identified for revocation by its hash.
func (s *ManagementService) ListEnrollmentTokens(ctx context.Context, _ *managementv1.ListEnrollmentTokensRequest) (*managementv1.ListEnrollmentTokensResponse, error) { //nolint:lll
	err := s.requireUnscopedCaller(ctx)
	if err != nil {
		return nil, err
	}

	tokens, err := models.FindEnrollmentTokens(s.db.Querier)
	if err != nil {
		return nil, err
	}

	res := &managementv1.ListEnrollmentTokensResponse{
		EnrollmentTokens: make([]*managementv1.EnrollmentToken, len(tokens)),
	}
	for i, token := range tokens {
		res.EnrollmentTokens[i] = toAPIEnrollmentToken(token)
	}

	return res, nil
}

// DeleteEnrollmentToken revokes an enrollment token. Nodes it already enrolled keep working:
// they hold their own agent tokens, which are unrelated to the one that enrolled them.
func (s *ManagementService) DeleteEnrollmentToken(ctx context.Context, req *managementv1.DeleteEnrollmentTokenRequest) (*managementv1.DeleteEnrollmentTokenResponse, error) { //nolint:lll
	err := s.requireUnscopedCaller(ctx)
	if err != nil {
		return nil, err
	}

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return models.RemoveEnrollmentToken(tx.Querier, req.TokenHash)
	})
	if errTX != nil {
		return nil, errTX
	}

	s.l.Info("Revoked an enrollment token.")

	return &managementv1.DeleteEnrollmentTokenResponse{}, nil
}

// requireUnscopedCaller refuses callers that are confined to a node. The rule table already
// gates these operations at admin, but a node's own token reaches some management paths
// through the binding, and minting enrollment tokens must never be one of them.
func (s *ManagementService) requireUnscopedCaller(ctx context.Context) error {
	if _, scoped := auth.NodeScope(ctx); scoped {
		return status.Error(codes.PermissionDenied, "This token may not manage enrollment tokens.")
	}

	return nil
}
