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

package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/version"
)

// TestWithLogLevel covers the pmm-agent version gate, the fatal downgrade and that args
// passed in are preserved.
func TestWithLogLevel(t *testing.T) {
	t.Parallel()

	supported := version.MustParse("2.28.0")

	for name, tc := range map[string]struct {
		args                 []string
		level                *string
		pmmAgentVersion      *version.Parsed
		supportLogLevelFatal bool
		expected             []string
	}{
		"debug appended to existing args": {
			args:            []string{"--web.listen-address=:42000"},
			level:           new("debug"),
			pmmAgentVersion: supported,
			expected:        []string{"--web.listen-address=:42000", "--log.level=debug"},
		},
		"fatal supported": {
			level:                new("fatal"),
			pmmAgentVersion:      supported,
			supportLogLevelFatal: true,
			expected:             []string{"--log.level=fatal"},
		},
		"fatal falls back to error": {
			// Exporters which dropped the fatal level would refuse to start otherwise.
			level:           new("fatal"),
			pmmAgentVersion: supported,
			expected:        []string{"--log.level=error"},
		},
		"no level": {
			level:           nil,
			pmmAgentVersion: supported,
			expected:        nil,
		},
		"empty level": {
			level:           new(""),
			pmmAgentVersion: supported,
			expected:        nil,
		},
		"pmm-agent too old": {
			// The flag only exists from PMM 2.28 onwards.
			level:           new("debug"),
			pmmAgentVersion: version.MustParse("2.27.0"),
			expected:        nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := withLogLevel(tc.args, tc.level, tc.pmmAgentVersion, tc.supportLogLevelFatal)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
