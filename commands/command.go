// pmm-admin
// Copyright (C) 2018 Percona LLC
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

package commands

import (
	"github.com/percona/pmm/api/inventory/json/client"
)

// FIXME Expand this interface to cover our use cases:
// * Normal results output
// * Live progress output
// * JSON output (do we need it?)

// Command is a common interface for all commands.
type Command interface {
	Run()
}

// CommonParams contains common parameters for all commands.
type CommonParams struct {
	Client *client.PMMServerInventory
}
