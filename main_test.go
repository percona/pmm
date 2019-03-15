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

package main

import (
	"go/build"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackages(t *testing.T) {
	cmd := exec.Command("pmm-agent", "-h") //nolint:gosec
	b, err := cmd.CombinedOutput()
	require.NoError(t, err)

	out := string(b)
	assert.False(t, strings.Contains(out, "-httptest.serve"), `pmm-agent should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "-test.run"), `pmm-agent should not import package "testing"`)
}

func TestImports(t *testing.T) {
	type constraint struct {
		blacklist []string
	}

	for path, c := range map[string]constraint{
		// agents code should not be concerned about pmm-agent<->pmm-managed protocol details
		"github.com/percona/pmm-agent/agents/process": {
			blacklist: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/server",
			},
		},
		"github.com/percona/pmm-agent/agents/builtin/mysql": {
			blacklist: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/server",
			},
		},
	} {
		p, err := build.Import(path, ".", build.IgnoreVendor)
		require.NoError(t, err)

		allImports := make(map[string]struct{})
		for _, i := range p.Imports {
			allImports[i] = struct{}{}
		}
		for _, i := range p.TestImports {
			allImports[i] = struct{}{}
		}
		for _, i := range p.XTestImports {
			allImports[i] = struct{}{}
		}

		for _, i := range c.blacklist {
			if _, ok := allImports[i]; ok {
				t.Errorf("Package %q should not import %q.", path, i)
			}
		}
	}
}
