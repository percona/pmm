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

type EntityType string

const (
	EntityTypeUser EntityType = "user"
)

//go:generate ../../bin/reform

// EntityRoles represents mapping of entities to roles.
//
//reform:entity_roles
type EntityRoles struct {
	EntityID   int        `reform:"entity_id"`
	EntityType EntityType `reform:"entity_type"`
	RoleID     uint32     `reform:"role_id"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//
//nolint:unparam
func (t *EntityRoles) BeforeInsert() error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//
//nolint:unparam
func (t *EntityRoles) BeforeUpdate() error {
	t.UpdatedAt = Now()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Template)(nil)
	_ reform.BeforeUpdater  = (*Template)(nil)
)
