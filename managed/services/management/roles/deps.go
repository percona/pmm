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
	"github.com/percona/pmm/managed/models"
	"gopkg.in/reform.v1"
)

//go:generate ../../../../bin/mockery -name=EntityService -case=snake -inpkg -testonly

// EntityService allows for managing entity and its roles.
type EntityService interface {
	AssignRoles(tx *reform.TX, entityID int, roleIDs []int) error
	BeforeDeleteRole(tx *reform.TX, roleID, newRoleID int) error
	GetEntityRoles(q *reform.Querier, entityID int) ([]models.Role, error)
	RemoveEntityRoles(tx *reform.TX, entityID int) error
}
