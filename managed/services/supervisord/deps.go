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

package supervisord

import "github.com/percona/pmm/managed/models"

//go:generate ../../../bin/mockery -name=alertFlagsProvider -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=vmBaseFileProvider -case=snake -inpkg -testonly

// alertFlagsProvider is a subset of methods of victoria.Metrics service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type alertFlagsProvider interface {
	ListAlertFlags() []string
}

// vmBaseFileProvider is a subset of methods of victoria.Metrics service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type vmBaseFileProvider interface {
	GetBaseFile() (models.File, error)
}
