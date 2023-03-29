// Copyright (C) 2022 Percona LLC
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

// TeamRoles represents mapping of teams to roles.
//
//reform:team_roles
type TeamRoles struct {
	TeamID int    `reform:"team_id"`
	RoleID uint32 `reform:"role_id"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// SetEntityID sets team ID
func (t *TeamRoles) SetEntityID(teamID int) {
	t.TeamID = teamID
}

// SetRoleID sets role ID
func (t *TeamRoles) SetRoleID(roleID uint32) {
	t.RoleID = roleID
}

// BeforeInsert implements reform.BeforeInserter interface.
//
//nolint:unparam
func (t *TeamRoles) BeforeInsert() error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//
//nolint:unparam
func (t *TeamRoles) BeforeUpdate() error {
	t.UpdatedAt = Now()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Template)(nil)
	_ reform.BeforeUpdater  = (*Template)(nil)
)
