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
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
)

var (
	// ErrRoleNotFound is returned when a role is not found.
	ErrRoleNotFound = fmt.Errorf("RoleNotFound")
	// ErrRoleIsAssigned is returned when a role is assigned to a user and cannot be removed.
	ErrRoleIsAssigned = fmt.Errorf("RoleIsAssigned")
	// ErrRoleIsDefaultRole is returned when trying to delete a default role.
	ErrRoleIsDefaultRole = fmt.Errorf("RoleIsDefaultRole")
)

// CreateRole creates a new role.
func CreateRole(q *reform.Querier, role *Role) error {
	if err := q.Insert(role); err != nil {
		return err
	}

	return nil
}

// AssignRoles assigns a set of roles to a user. This replaces all existing roles.
func AssignRoles(tx *reform.TX, userID int, roleIDs []int) error {
	q := tx.Querier

	if _, err := q.DeleteFrom(UserRolesView, " WHERE user_id = $1", userID); err != nil {
		return err
	}

	s := make([]reform.Struct, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		var role Role
		if err := FindAndLockRole(tx, roleID, &role); err != nil {
			return err
		}

		var userRole UserRoles
		userRole.UserID = userID
		userRole.RoleID = uint32(roleID)
		s = append(s, &userRole)
	}

	return q.InsertMulti(s...)
}

// AssignDefaultRole assigns a default role to a user.
func AssignDefaultRole(tx *reform.TX, userID int) error {
	settings, err := GetSettings(tx)
	if err != nil {
		return err
	}

	if settings.DefaultRoleID <= 0 {
		logrus.Panicf("Default role ID is %d", settings.DefaultRoleID)
	}

	return AssignRoles(tx, userID, []int{settings.DefaultRoleID})
}

// DeleteRole deletes a role, if possible.
func DeleteRole(tx *reform.TX, roleID int) error {
	q := tx.Querier

	var role Role
	if err := FindAndLockRole(tx, roleID, &role); err != nil {
		return err
	}

	settings, err := GetSettings(tx)
	if err != nil {
		return err
	}

	if settings.DefaultRoleID == roleID {
		return ErrRoleIsDefaultRole
	}

	s, err := q.FindOneFrom(UserRolesView, "role_id", roleID)
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

// GetUserRoles retrieves all roles assigned to a user.
func GetUserRoles(q *reform.Querier, userID int) ([]Role, error) {
	query := fmt.Sprintf(`
		SELECT 
			%s
		FROM 
			%[2]s
			INNER JOIN %[3]s ON (%[2]s.role_id = %[3]s.id)
		WHERE
			user_roles.user_id = $1`,
		strings.Join(q.QualifiedColumns(RoleTable), ","),
		UserRolesView.Name(),
		RoleTable.Name())

	rows, err := q.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	roles := []Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(role.Pointers()...); err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}
