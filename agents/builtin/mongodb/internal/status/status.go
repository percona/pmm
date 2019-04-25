// pmm-agent
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

package status

import (
	"fmt"

	"github.com/fatih/structs"
)

// Status converts stats into pct map status
type Status struct {
	stats interface{}
}

func New(stats interface{}) *Status {
	return &Status{
		stats: stats,
	}
}

// Map converts stats struct into a map
func (s *Status) Map() map[string]string {
	out := map[string]string{}
	for _, f := range structs.New(s.stats).Fields() {
		if f.IsZero() {
			continue
		}
		tag := f.Tag("name")
		if tag == "" {
			continue
		}
		v := fmt.Sprint(f.Value())
		if v == "" || v == `""` || v == "0" {
			continue
		}
		out[tag] = v
	}
	return out
}
