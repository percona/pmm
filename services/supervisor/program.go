// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package supervisor

import (
	servicelib "github.com/percona/kardianos-service"
)

// we always run external programs, so we don't need a real implementation
type program struct{}

func (p *program) Start(s servicelib.Service) error { return nil }
func (p *program) Stop(s servicelib.Service) error  { return nil }

// check interface
var _ servicelib.Interface = new(program)
