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

func TestWithLogLevel(t *testing.T) {
	t.Parallel()

	supported := version.MustParse("2.28.0")

	for name, tc := range map[string]struct {
		level                *string
		pmmAgentVersion      *version.Parsed
		supportLogLevelFatal bool
		expected             []string
	}{
		"debug": {
			level:           new("debug"),
			pmmAgentVersion: supported,
			expected:        []string{"--log.level=debug"},
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

			actual := withLogLevel(nil, tc.level, tc.pmmAgentVersion, tc.supportLogLevelFatal)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestWithValkeyLogLevel covers PMM-15201: valkey_exporter parses its command line with the
// standard library flag package, which rejects --log.level and exits with code 2, leaving the
// agent in the DONE state. The per-level matrix lives in TestValkeyExporterConfig, so only the
// flag spelling and the fatal fallback are checked here.
func TestWithValkeyLogLevel(t *testing.T) {
	t.Parallel()

	supported := version.MustParse("2.28.0")

	for name, tc := range map[string]struct {
		level    *string
		expected []string
	}{
		"dashed flag": {new("info"), []string{"--log-level=info"}},
		// valkey_exporter silently falls back to info on an unknown level, so a stored
		// "fatal" must be translated rather than passed through.
		"fatal falls back to error": {new("fatal"), []string{"--log-level=error"}},
		"no level":                  {nil, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := withValkeyLogLevel(nil, tc.level, supported)
			assert.Equal(t, tc.expected, actual)
			for _, arg := range actual {
				assert.NotContains(t, arg, "--log.level", "valkey_exporter rejects the dotted flag")
			}
		})
	}
}

// TestWithLogLevelAppends makes sure existing args are preserved.
func TestWithLogLevelAppends(t *testing.T) {
	t.Parallel()

	args := withLogLevel([]string{"--web.listen-address=:42000"}, new("info"), version.MustParse("2.28.0"), false)
	assert.Equal(t, []string{"--web.listen-address=:42000", "--log.level=info"}, args)
}
