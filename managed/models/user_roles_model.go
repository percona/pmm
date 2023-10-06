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
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// UserRoles represents mapping of users to roles.
//
//reform:user_roles
type UserRoles struct {
	UserID int    `reform:"user_id"`
	RoleID uint32 `reform:"role_id"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//
//nolint:unparam
func (t *UserRoles) BeforeInsert() error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//
//nolint:unparam
func (t *UserRoles) BeforeUpdate() error {
	t.UpdatedAt = Now()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Template)(nil)
	_ reform.BeforeUpdater  = (*Template)(nil)
)
