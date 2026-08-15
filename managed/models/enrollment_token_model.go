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

import "time"

//go:generate go tool reform

// EnrollmentToken authorizes enrolling a Node and nothing else.
//
// Registering a node requires Grafana Org Admin, which means handing an ops team full
// administrative access to PMM just so they can add hosts to monitoring. An enrollment token
// is the narrower grant: it may create a node and obtain that node's agent token, and it can
// do nothing else at all.
//
// Only TokenHash is stored. The token itself is returned once, when it is minted.
//
//reform:enrollment_tokens
type EnrollmentToken struct {
	TokenHash   string     `reform:"token_hash,pk"`
	Description string     `reform:"description"`
	ExpiresAt   *time.Time `reform:"expires_at"`
	MaxUses     int        `reform:"max_uses"`
	UsedCount   int        `reform:"used_count"`
	CreatedAt   time.Time  `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (t *EnrollmentToken) BeforeInsert() error { //nolint:unparam
	t.CreatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (t *EnrollmentToken) AfterFind() error { //nolint:unparam
	t.CreatedAt = t.CreatedAt.UTC()
	if t.ExpiresAt != nil {
		expiresAt := t.ExpiresAt.UTC()
		t.ExpiresAt = &expiresAt
	}
	return nil
}

// Expired reports whether the token is past its expiry. A token with no expiry never is.
func (t *EnrollmentToken) Expired() bool {
	return t.ExpiresAt != nil && !Now().Before(*t.ExpiresAt)
}

// Exhausted reports whether the token has been used as many times as it is allowed.
// MaxUses of zero means unlimited.
func (t *EnrollmentToken) Exhausted() bool {
	return t.MaxUses > 0 && t.UsedCount >= t.MaxUses
}

// Usable reports whether the token may still enroll a node.
func (t *EnrollmentToken) Usable() bool {
	return !t.Expired() && !t.Exhausted()
}
