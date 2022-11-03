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

package models

import (
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

var (
	// ErrRoleNotFound is returned when a role is not found.
	ErrRoleNotFound = fmt.Errorf("RoleNotFound")
	// ErrRoleIsAssigned is returned when a role is assigned to a user and cannot be removed.
	ErrRoleIsAssigned = fmt.Errorf("RoleIsAssigned")
)

// CreateRole creates a new role.
func CreateRole(q *reform.Querier, role *Role) error {
	if err := q.Insert(role); err != nil {
		return err
	}

	return nil
}

// AssignRole assigns a role to a user.
func AssignRole(tx *reform.TX, userID, roleID int) error {
	q := tx.Querier

	var role Role

	if err := q.SelectOneTo(&role, "WHERE id = $1 FOR UPDATE", roleID); err != nil {
		if ok := errors.As(err, &reform.ErrNoRows); ok {
			return ErrRoleNotFound
		}

		return err
	}

	user, err := GetOrCreateUser(q, userID)
	if err != nil {
		return err
	}

	user.RoleID = role.ID
	err = tx.UpdateColumns(user, "role_id")

	return err
}

// DeleteRole deletes a role.
func DeleteRole(tx *reform.TX, roleID int) error {
	q := tx.Querier

	var role Role
	err := q.SelectOneTo(&role, "WHERE id = $1 FOR UPDATE", roleID)
	if err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return ErrRoleNotFound
		}
		return err
	}

	s, err := q.FindOneFrom(UserDetailsTable, "role_id", roleID)
	if err != nil && !errors.As(err, &reform.ErrNoRows) {
		return err
	}

	if s != nil {
		return ErrRoleIsAssigned
	}

	if err := q.Delete(&role); err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return ErrRoleNotFound
		}

		return err
	}

	return nil
}
