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

package models

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// EnrollmentTokenPrefix marks a token as authorizing node enrollment. AuthServer uses it to
// tell an enrollment token from an agent token without a database lookup.
const EnrollmentTokenPrefix = "pmmet_"

// DefaultEnrollmentTokenTTL bounds a token that was minted without an explicit expiry.
// A token that never expires is a standing credential to enrol nodes, which is the shape of
// thing agent tokens were just changed to stop being. Long-lived tokens remain possible, but
// only by asking for one.
const DefaultEnrollmentTokenTTL = 30 * time.Minute

// ErrInvalidEnrollmentToken is returned when a token is malformed or matches nothing issued.
var ErrInvalidEnrollmentToken = errors.New("invalid enrollment token")

// IsEnrollmentToken reports whether a credential is an enrollment token.
func IsEnrollmentToken(token string) bool {
	return strings.HasPrefix(token, EnrollmentTokenPrefix)
}

// CreateEnrollmentTokenParams contains parameters for minting an enrollment token.
type CreateEnrollmentTokenParams struct {
	Description string
	// ExpiresAt is when the token stops working. Nil applies DefaultEnrollmentTokenTTL.
	ExpiresAt *time.Time
	// MaxUses is how many nodes it may enroll. Zero means unlimited.
	MaxUses int
}

// Validate checks the parameters.
func (p *CreateEnrollmentTokenParams) Validate() error {
	if p.MaxUses < 0 {
		return status.Error(codes.InvalidArgument, "Maximum uses cannot be negative.")
	}
	if p.ExpiresAt != nil && !p.ExpiresAt.After(Now()) {
		return status.Error(codes.InvalidArgument, "Expiry must be in the future.")
	}

	return nil
}

// CreateEnrollmentToken mints a token and returns it. The plaintext is returned only here;
// afterwards only its hash exists, so a lost token cannot be recovered and must be replaced.
func CreateEnrollmentToken(q *reform.Querier, params *CreateEnrollmentTokenParams) (*EnrollmentToken, string, error) {
	err := params.Validate()
	if err != nil {
		return nil, "", err
	}

	expiresAt := params.ExpiresAt
	if expiresAt == nil {
		bounded := Now().Add(DefaultEnrollmentTokenTTL)
		expiresAt = &bounded
	}

	buf := make([]byte, agentTokenBytes)
	_, err = rand.Read(buf)
	if err != nil {
		return nil, "", err
	}
	token := EnrollmentTokenPrefix + base64.RawURLEncoding.EncodeToString(buf)

	row := &EnrollmentToken{
		TokenHash:   HashAgentToken(token),
		Description: params.Description,
		ExpiresAt:   expiresAt,
		MaxUses:     params.MaxUses,
	}
	err = q.Insert(row)
	if err != nil {
		return nil, "", err
	}

	return row, token, nil
}

// FindEnrollmentToken returns the token row for a token value, whether or not it is usable.
func FindEnrollmentToken(q *reform.Querier, token string) (*EnrollmentToken, error) {
	if !IsEnrollmentToken(token) {
		return nil, ErrInvalidEnrollmentToken
	}

	var row EnrollmentToken
	err := q.FindOneTo(&row, "token_hash", HashAgentToken(token))
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrInvalidEnrollmentToken
		}
		return nil, err
	}

	return &row, nil
}

// UseEnrollmentToken checks that a token may still enroll a node and records the use.
// Call it inside the same transaction as the registration it authorizes, so a failed
// registration does not consume a use.
func UseEnrollmentToken(q *reform.Querier, token string) error {
	row, err := FindEnrollmentToken(q, token)
	if err != nil {
		return err
	}

	switch {
	case row.Expired():
		return status.Error(codes.PermissionDenied, "Enrollment token has expired.")
	case row.Exhausted():
		return status.Error(codes.PermissionDenied, "Enrollment token has no uses left.")
	}

	row.UsedCount++

	return q.Update(row)
}

// FindEnrollmentTokens returns every enrollment token, newest first.
func FindEnrollmentTokens(q *reform.Querier) ([]*EnrollmentToken, error) {
	structs, err := q.SelectAllFrom(EnrollmentTokenTable, "ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}

	tokens := make([]*EnrollmentToken, len(structs))
	for i, str := range structs {
		tokens[i] = str.(*EnrollmentToken) //nolint:forcetypeassert
	}

	return tokens, nil
}

// RemoveEnrollmentToken revokes a token by its hash, which is what listing exposes.
func RemoveEnrollmentToken(q *reform.Querier, tokenHash string) error {
	err := q.Delete(&EnrollmentToken{TokenHash: tokenHash})
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return status.Error(codes.NotFound, "Enrollment token not found.")
		}
		return err
	}

	return nil
}
