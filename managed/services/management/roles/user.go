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

package roles

import (
	"fmt"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// User extends Generic struct and applies custom behavior for a user entity.
type User struct {
	Generic
}

// NewUser returns a new User.
func NewUser(entityIDColumnName string, newModel func() EntityModel, view reform.View) *User {
	return &User{
		Generic: Generic{
			entityIDColumnName: entityIDColumnName,
			newModel:           newModel,
			view:               view,
		},
	}
}

// BeforeDeleteRole makes changes to the role assignments before the role is deleted.
func (u *User) BeforeDeleteRole(tx *reform.TX, roleID, newRoleID int) error {
	q := tx.Querier

	// If no new role is assigned, we remove all role assignements.
	if newRoleID == 0 {
		_, err := q.DeleteFrom(u.view, "WHERE role_id = $1", roleID)
		return err
	}

	if err := models.FindAndLockRole(tx, newRoleID, &models.Role{}); err != nil {
		return err
	}

	// Check if the role is the last role for a user and apply special logic if it is.
	structs, err := q.FindAllFrom(u.view, "role_id", roleID)
	if err != nil {
		return err
	}

	for _, s := range structs {
		ur, ok := s.(*models.UserRoles)
		if !ok {
			return fmt.Errorf("invalid data structure in user roles for role ID %d. Found %+v", roleID, s)
		}

		roleStructs, err := q.SelectAllFrom(u.view, "WHERE user_id = $1 FOR UPDATE", ur.UserID)
		if err != nil {
			return err
		}

		// If there are more than 1 roles, we remove the role without a replacement.
		if len(roleStructs) > 1 {
			_, err := q.DeleteFrom(u.view, "WHERE user_id = $1 AND role_id = $2", ur.UserID, roleID)
			if err != nil {
				return err
			}

			continue
		}

		// The removed role is the last one. We replace it with a new role.
		query := fmt.Sprintf(`
			UPDATE %[1]s
			SET
				role_id = $1
			WHERE
				%[2]s = $2 AND
				role_id = $3 AND
				NOT EXISTS(
					SELECT 1
					FROM %[1]s ur
					WHERE
						ur.%[2]s = $2 AND
						ur.role_id = $1
				)`,
			u.view.Name(),
			u.entityIDColumnName,
		)
		_, err = q.Exec(query, newRoleID, ur.UserID, roleID)
		if err != nil {
			return err
		}
	}

	return nil
}
