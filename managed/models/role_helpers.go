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

// ErrRoleNotFound is returned when a role is not found.
var ErrRoleNotFound = fmt.Errorf("RoleNotFound")

// CreateRole creates a new role.
func CreateRole(q *reform.Querier, role *Role) error {
	if err := q.Insert(role); err != nil {
		return err
	}

	return nil
}

// AssignRole assigns a role to a user.
func AssignRole(tx *reform.TX, userID, roleID int) error {
	var role Role
	if err := tx.Querier.FindByPrimaryKeyTo(&role, roleID); err != nil {
		if ok := errors.As(err, &reform.ErrNoRows); ok {
			return ErrRoleNotFound
		}

		return err
	}

	user, err := GetOrCreateUser(tx.Querier, int(userID))
	if err != nil {
		return err
	}

	user.RoleID = role.ID
	err = tx.UpdateColumns(user, "role_id")

	return err
}
