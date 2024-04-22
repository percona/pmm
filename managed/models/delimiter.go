// Copyright (C) 2024 Percona LLC
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

import "strings"

// DelimiterPair contains a pair of safe template delimiters.
type DelimiterPair struct {
	Left  string
	Right string
}

var pairs = []DelimiterPair{
	{Left: "{{", Right: "}}"},
	{Left: "[[", Right: "]]"},
	{Left: "((", Right: "))"},
	{Left: "<<", Right: ">>"},
	{Left: "<%", Right: "%>"},
}

// TemplateDelimsPair returns a pair of safe template delimiters that are not present in any given string.
func TemplateDelimsPair(str ...string) DelimiterPair {
	for _, p := range pairs {
		var found bool
		for _, s := range str {
			if strings.Contains(s, p.Left) {
				found = true
				break
			}
			if strings.Contains(s, p.Right) {
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
