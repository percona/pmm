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

package grafana

import (
	"github.com/percona/pmm/managed/models"
	"gopkg.in/reform.v1"
)

//go:generate ../../../bin/mockery -name=awsInstanceChecker -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=defaultRoleAssigner -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=userRolesGetter -case=snake -inpkg -testonly

// checker is a subset of methods of server.AWSInstanceChecker used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type awsInstanceChecker interface {
	MustCheck() bool
}

type defaultRoleAssigner interface {
	AssignDefaultRole(tx *reform.TX, userID int) error
}

type userRolesGetter interface {
	GetUserRoles(q *reform.Querier, userID int, teamIDs []int) ([]models.Role, error)
}
