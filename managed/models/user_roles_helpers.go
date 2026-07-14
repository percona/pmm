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
	"fmt"

	"gopkg.in/reform.v1"
)

// ListUsers lists all users and their roles. Returns map with user ID as key and role ID as values.
func ListUsers(q *reform.Querier) (map[int][]uint32, error) {
	rows, err := q.SelectAllFrom(UserRolesView, "")
	if err != nil {
		return nil, err
	}

	roles := make(map[int][]uint32)
	for _, row := range rows {
		userRole, ok := row.(*UserRoles)
		if !ok {
			return nil, fmt.Errorf("invalid data in user_roles table")
		}

		_, ok = roles[userRole.UserID]
		if !ok {
			roles[userRole.UserID] = []uint32{}
		}

		roles[userRole.UserID] = append(roles[userRole.UserID], userRole.RoleID)
	}

	return roles, nil
}
