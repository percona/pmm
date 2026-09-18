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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeDisabledCollectors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			// nil must survive: ChangeExporterOptions reads it as "no change".
			name:     "nil is preserved",
			input:    nil,
			expected: nil,
		}, {
			// An empty non-nil slice means "clear the list" and must not become nil.
			name:     "empty stays non-nil",
			input:    []string{},
			expected: []string{},
		}, {
			name:     "surrounding whitespace is trimmed",
			input:    []string{"cpu", " meminfo", "diskstats "},
			expected: []string{"cpu", "meminfo", "diskstats"},
		}, {
			// Two --no-collector.cpu arguments stop node_exporter from starting.
			name:     "repeats are dropped, order preserved",
			input:    []string{"cpu", "diskstats", "cpu"},
			expected: []string{"cpu", "diskstats"},
		}, {
			name:     "repeats that differ only by whitespace are dropped",
			input:    []string{"cpu", " cpu "},
			expected: []string{"cpu"},
		}, {
			name:     "blank entries are dropped",
			input:    []string{"", "  ", "cpu"},
			expected: []string{"cpu"},
		}, {
			name:     "all entries blank",
			input:    []string{"", "   "},
			expected: []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := normalizeDisabledCollectors(tc.input)
			assert.Equal(t, tc.expected, actual)
			if tc.input != nil {
				assert.NotNil(t, actual, "a non-nil input must stay non-nil, it means clear")
			}
		})
	}
}
