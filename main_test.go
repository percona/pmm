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
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"sort"
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

	allImports := make(map[string]map[string]struct{})
	for path, c := range map[string]constraint{
		// models should not import services or APIs.
		"github.com/percona/pmm-managed/models": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
				"github.com/percona/pmm/api",
			},
		},

		// services should be independent
		"github.com/percona/pmm-managed/services/agents": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},
		"github.com/percona/pmm-managed/services/grafana": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},
		"github.com/percona/pmm-managed/services/logs": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},
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
		"github.com/percona/pmm-managed/services/telemetry": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		},

		// those services should be independent too, but import converters
		"github.com/percona/pmm-managed/services/inventory": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services/",
			},
		},
		"github.com/percona/pmm-managed/services/management": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services/",
			},
		},
	} {
		p, err := build.Import(path, ".", build.IgnoreVendor)
		require.NoError(t, err)

		if allImports[path] == nil {
			allImports[path] = make(map[string]struct{})
		}
		for _, i := range p.Imports {
			allImports[path][i] = struct{}{}
		}
		for _, i := range p.TestImports {
			allImports[path][i] = struct{}{}
		}
		for _, i := range p.XTestImports {
			allImports[path][i] = struct{}{}
		}

		for _, b := range c.blacklistPrefixes {
			for i := range allImports[path] {
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

	f, err := os.Create("packages.dot")
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	fmt.Fprintf(f, "digraph packages {\n")

	packages := make([]string, 0, len(allImports))
	for p := range allImports {
		packages = append(packages, p)
	}
	sort.Strings(packages)

	for _, p := range packages {
		imports := make([]string, 0, len(allImports[p]))
		for p := range allImports[p] {
			imports = append(imports, p)
		}
		sort.Strings(imports)

		p = strings.TrimPrefix(p, "github.com/percona/pmm-managed")
		for _, i := range imports {
			if strings.Contains(i, "/utils/") || strings.Contains(i, "/internal/") {
				continue
			}
			if strings.HasPrefix(i, "github.com/percona/pmm-managed") {
				i = strings.TrimPrefix(i, "github.com/percona/pmm-managed")
				fmt.Fprintf(f, "\t%q -> %q;\n", p, i)
			}
		}
	}

	fmt.Fprintf(f, "}\n")
}
