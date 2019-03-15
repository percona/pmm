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
		blacklist []string
	}

	for path, c := range map[string]constraint{
		// models should not import services
		"github.com/percona/pmm-managed/models": {
			blacklist: []string{
				"github.com/percona/pmm-managed/services/agents",
				"github.com/percona/pmm-managed/services/inventory",
				"github.com/percona/pmm-managed/services/prometheus",
				"github.com/percona/pmm-managed/services/qan",
			},
		},

		// services should be independent: agent, inventory, prometheus, qan
		"github.com/percona/pmm-managed/services/agents": {
			blacklist: []string{
				"github.com/percona/pmm-managed/services/inventory",
				"github.com/percona/pmm-managed/services/prometheus",
				"github.com/percona/pmm-managed/services/qan",
			},
		},
		"github.com/percona/pmm-managed/services/inventory": {
			blacklist: []string{
				"github.com/percona/pmm-managed/services/agents",
				"github.com/percona/pmm-managed/services/prometheus",
				"github.com/percona/pmm-managed/services/qan",
			},
		},
		"github.com/percona/pmm-managed/services/prometheus": {
			blacklist: []string{
				"github.com/percona/pmm-managed/services/agents",
				"github.com/percona/pmm-managed/services/inventory",
				"github.com/percona/pmm-managed/services/qan",
			},
		},
		"github.com/percona/pmm-managed/services/qan": {
			blacklist: []string{
				"github.com/percona/pmm-managed/services/agents",
				"github.com/percona/pmm-managed/services/inventory",
				"github.com/percona/pmm-managed/services/prometheus",
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
