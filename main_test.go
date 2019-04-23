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
	assert.False(t, strings.Contains(out, "httptest.serve"), `pmm-agent should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "test.run"), `pmm-agent should not import package "testing"`)
}

func TestImports(t *testing.T) {
	type constraint struct {
		blacklistPrefixes []string
	}

	for path, c := range map[string]constraint{
		// agents code should be independent
		"github.com/percona/pmm-agent/agents/process": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/builtin",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/builtin/mysql/perfschema": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/builtin",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/builtin/mysql/slowlog": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/builtin",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/builtin/noop": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/builtin",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},

		// agentlocal server, supervisor, and client should be independent
		"github.com/percona/pmm-agent/agentlocal": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agents/supervisor",
				"github.com/percona/pmm-agent/client",
			},
		},
		"github.com/percona/pmm-agent/agents/supervisor": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/client",
			},
		},
		"github.com/percona/pmm-agent/client": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/supervisor",
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

		for _, b := range c.blacklistPrefixes {
			for i := range allImports {
				// whitelist own subpackages
				if strings.HasPrefix(i, path) {
					continue
				}

				// check blacklist
				if strings.HasPrefix(i, b) {
					t.Errorf("Package %q should not import package %q (blacklisted by %q).", path, i, b)
				}
			}
		}
	}
}
