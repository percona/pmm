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
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
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
		if err := findRole(tx, roleID, &role); err != nil {
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
func DeleteRole(tx *reform.TX, roleID, replacementRoleID int) error {
	q := tx.Querier

	var role Role
	if err := findRole(tx, roleID, &role); err != nil {
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

	if err := replaceRole(tx, roleID, replacementRoleID); err != nil {
		return err
	}

	if err := q.Delete(&role); err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return ErrRoleNotFound
		}

		return err
	}

	return nil
}

func replaceRole(tx *reform.TX, roleID, newRoleID int) error {
	q := tx.Querier

	// If no new role is assigned, we remove all role assignements.
	if newRoleID == 0 {
		_, err := q.DeleteFrom(UserRolesView, "WHERE role_id = $1", roleID)
		return err
	}

	if err := findRole(tx, newRoleID, &Role{}); err != nil {
		return err
	}

	// Check if the role is the last role for a user and apply special logic if it is.
	structs, err := q.FindAllFrom(UserRolesView, "role_id", roleID)
	if err != nil {
		return err
	}

	for _, s := range structs {
		ur, ok := s.(*UserRoles)
		if !ok {
			return fmt.Errorf("invalid data structure in user roles for role ID %d. Found %+v", roleID, s)
		}

		roleStructs, err := q.SelectAllFrom(UserRolesView, "WHERE user_id = $1 FOR UPDATE", ur.UserID)
		if err != nil {
			return err
		}

		// If there are more than 1 roles, we remove the role without a replacement.
		if len(roleStructs) > 1 {
			_, err := q.DeleteFrom(UserRolesView, "WHERE user_id = $1 AND role_id = $2", ur.UserID, roleID)
			if err != nil {
				return err
			}

			continue
		}

		// The removed role is the last one. We replace it with a new role.
		_, err = q.Exec(`
			UPDATE user_roles
			SET
				role_id = $1
			WHERE
				user_id = $2 AND
				role_id = $3 AND
				NOT EXISTS(
					SELECT 1
					FROM user_roles ur
					WHERE
						ur.user_id = $2 AND
						ur.role_id = $1
				)`,
			newRoleID,
			ur.UserID,
			roleID)
		if err != nil {
			return err
		}
	}

	return nil
}

// findRole retrieves a role by ID.
func findRole(tx *reform.TX, roleID int, role *Role) error {
	err := tx.Querier.SelectOneTo(role, "WHERE id = $1", roleID)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return ErrRoleNotFound
		}
		return err
	}

	return nil
}

// ChangeDefaultRole changes default role in the settings.
func ChangeDefaultRole(tx *reform.TX, roleID int) error {
	var role Role
	if err := findRole(tx, roleID, &role); err != nil {
		return err
	}

	var p ChangeSettingsParams
	p.DefaultRoleID = &roleID

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
