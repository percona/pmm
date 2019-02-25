// pmm-managed
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

package agents

import (
	"strings"
)

type pair struct {
	left  string
	right string
}

var pairs = []pair{
	{left: "{{", right: "}}"},
	{left: "[[", right: "]]"},
	{left: "((", right: "))"},
	{left: "<<", right: ">>"},
	{left: "<%", right: "%>"},
}

// templateDelimsPair returns a pair of safe template delimeters that are not present in any given string.
func templateDelimsPair(str ...string) pair {
	for _, p := range pairs {
		var found bool
		for _, s := range str {
			if strings.Contains(s, p.left) {
				found = true
				break
			}
			if strings.Contains(s, p.right) {
				found = true
				break
			}
		}
		if !found {
			return p
		}
	}

	panic("failed to find a pair of safe template delimeters")
}
