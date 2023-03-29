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
	"strings"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// EntityModel defines methods required on a model to be used with a generic struct.
type EntityModel interface {
	reform.Struct

	SetEntityID(int)
	SetRoleID(uint32)
}

// Generic is a generic entity structure used to access data about entity vs. role mapping.
// It expects the role ID to be stored in db in a column named "role_id".
type Generic struct {
	entityIDColumnName string
	newModel           func() EntityModel
	view               reform.View
}

// NewGeneric returns new generic entity struct.
func NewGeneric(entityIDColumnName string, newModel func() EntityModel, view reform.View) *Generic {
	return &Generic{
		entityIDColumnName: entityIDColumnName,
		newModel:           newModel,
		view:               view,
	}
}

// AssignRoles assigns roles to the entity.
func (g *Generic) AssignRoles(tx *reform.TX, entityID int, roleIDs []int) error {
	q := tx.Querier

	if _, err := q.DeleteFrom(g.view, " WHERE "+g.entityIDColumnName+" = $1", entityID); err != nil {
		return err
	}

	s := make([]reform.Struct, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		var role models.Role
		if err := models.FindAndLockRole(tx, roleID, &role); err != nil {
			return err
		}

		entityRole := g.newModel()
		entityRole.SetEntityID(entityID)
		entityRole.SetRoleID(uint32(roleID))

		s = append(s, entityRole)
	}

	return q.InsertMulti(s...)
}

// GetEntityRoles returns roles assigned to the entities.
func (g *Generic) GetEntityRoles(q *reform.Querier, entityIDs []int) ([]models.Role, error) {
	query := fmt.Sprintf(`
		SELECT 
			%s
		FROM 
			%[2]s
			INNER JOIN %[3]s ON (%[2]s.role_id = %[3]s.id)
		WHERE
			%[2]s.%[4]s IN (%[5]s)`,
		strings.Join(q.QualifiedColumns(models.RoleTable), ","),
		g.view.Name(),
		models.RoleTable.Name(),
		g.entityIDColumnName,
		strings.Join(q.Placeholders(1, len(entityIDs)), ","))

	params := make([]interface{}, 0, len(entityIDs))
	for _, e := range entityIDs {
		params = append(params, e)
	}

	rows, err := q.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	roles := []models.Role{}
	for rows.Next() {
		var role models.Role
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

// BeforeDeleteRole makes changes to the role assignments before the role is deleted.
func (g *Generic) BeforeDeleteRole(tx *reform.TX, roleID, _ int) error {
	_, err := tx.Querier.DeleteFrom(g.view, "WHERE role_id = $1", roleID)
	return err
}

// RemoveEntityRoles removes all roles for an entity.
func (g *Generic) RemoveEntityRoles(tx *reform.TX, entityID int) error {
	q := tx.Querier

	_, err := q.DeleteFrom(g.view, " WHERE "+g.entityIDColumnName+" = $1", entityID)
	return err
}
