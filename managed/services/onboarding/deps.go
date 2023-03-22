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

package onboarding

import (
	"context"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
)

//go:generate ../../../bin/mockery -name=inventoryService -case=snake -inpkg -testonly

type inventoryService interface {
	List(ctx context.Context, filters models.ServiceFilters) ([]inventorypb.Service, error)
}
