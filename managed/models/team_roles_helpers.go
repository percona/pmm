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
	"fmt"

	"gopkg.in/reform.v1"
)

// ListTeams lists all teams and their roles. Returns map with team ID as key and role ID as values.
func ListTeams(q *reform.Querier) (map[int][]uint32, error) {
	rows, err := q.SelectAllFrom(TeamRolesView, "")
	if err != nil {
		return nil, err
	}

	roles := make(map[int][]uint32)
	for _, row := range rows {
		teamRoles, ok := row.(*TeamRoles)
		if !ok {
			return nil, fmt.Errorf("invalid data in team_roles table")
		}

		_, ok = roles[teamRoles.TeamID]
		if !ok {
			roles[teamRoles.TeamID] = []uint32{}
		}

		roles[teamRoles.TeamID] = append(roles[teamRoles.TeamID], teamRoles.RoleID)
	}

	return roles, nil
}
