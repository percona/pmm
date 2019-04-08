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
	cmd := exec.Command("pmm-managed", "-h")
	b, err := cmd.CombinedOutput()
	require.EqualError(t, err, "exit status 2")

	out := string(b)
	assert.False(t, strings.Contains(out, "-httptest.serve"), `pmm-managed should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "-test.run"), `pmm-managed should not import package "testing"`)
}

func TestImports(t *testing.T) {
	type constraint struct {
		blacklistPrefixes []string
	}

	for path, c := range map[string]constraint{
		// models should not import services or APIs.
		"github.com/percona/pmm-managed/models": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
				"github.com/percona/pmm/api",
			},
		},

		// Services should be independent: agents, inventory, management, prometheus, qan.
		"github.com/percona/pmm-managed/services/agents": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},
		"github.com/percona/pmm-managed/services/inventory": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},
		// FIXME https://jira.percona.com/browse/PMM-3541
		// "github.com/percona/pmm-managed/services/management": {
		// 	blacklistPrefixes: []string{
		// 		"github.com/percona/pmm-managed/services",
		// 	},
		// },
		"github.com/percona/pmm-managed/services/prometheus": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},
		"github.com/percona/pmm-managed/services/qan": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
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
