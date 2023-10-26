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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/version"
)

func requireNoDuplicateFlags(t *testing.T, flags []string) {
	t.Helper()
	s := make(map[string]struct{})
	for _, f := range flags {
		name := strings.Split(f, "=")[0]
		if strings.HasPrefix(name, "--no-") { // kingpin's --no-<name> disables --<name>
			name = "--" + strings.TrimPrefix(name, "--no-")
		}
		if _, present := s[name]; present {
			assert.Failf(t, "flag (or --no- form) is already present", "%q", name)
		}
		s[name] = struct{}{}
	}
}

func TestPathsBaseForDifferentVersions(t *testing.T) {
	left := "{{"
	right := "}}"
	assert.Equal(t, "/usr/local/percona/pmm3", pathsBase(version.MustParse("2.22.01"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0-3-g7aa417c"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0-beta4"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0-rc1"), left, right))
}
