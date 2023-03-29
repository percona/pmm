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
	// ErrRoleIsDefaultRole is returned when trying to delete a default role.
	ErrRoleIsDefaultRole = fmt.Errorf("RoleIsDefaultRole")
)

type RoleBeforeDeleter interface {
	BeforeDeleteRole(tx *reform.TX, roleID, newRoleID int) error
}

// CreateRole creates a new role.
func CreateRole(q *reform.Querier, role *Role) error {
	if err := q.Insert(role); err != nil {
		return err
	}

	return nil
}

// DeleteRole deletes a role, if possible.
func DeleteRole(tx *reform.TX, beforeDelete RoleBeforeDeleter, roleID, replacementRoleID int) error {
	q := tx.Querier

	var role Role
	if err := FindAndLockRole(tx, roleID, &role); err != nil {
		return err
	}

	// Check if it's the default role.
	settings, err := GetSettings(tx)
	if err != nil {
		return err
	}

	if settings.DefaultRoleID == roleID {
		return ErrRoleIsDefaultRole
	}

	if err := beforeDelete.BeforeDeleteRole(tx, roleID, replacementRoleID); err != nil {
		return err
	}

	if err := q.Delete(&role); err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return ErrRoleNotFound
		}

		return err
	}

	return nil
}

// FindAndLockRole retrieves a role by ID and locks it for update.
func FindAndLockRole(tx *reform.TX, roleID int, role *Role) error {
	err := tx.Querier.SelectOneTo(role, "WHERE id = $1 FOR UPDATE", roleID)
	if err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return ErrRoleNotFound
		}
		return err
	}

	return nil
}

// ChangeDefaultRole changes default role in the settings.
func ChangeDefaultRole(tx *reform.TX, roleID int) error {
	var role Role
	if err := FindAndLockRole(tx, roleID, &role); err != nil {
		return err
	}

	var p ChangeSettingsParams
	p.DefaultRoleID = roleID

	_, err := UpdateSettings(tx, &p)

	return err
}
