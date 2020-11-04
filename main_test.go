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
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.False(t, strings.Contains(out, "-httptest.serve"), `pmm-managed should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "-test.run"), `pmm-managed should not import package "testing"`)
}

func TestImports(t *testing.T) {
	type constraint struct {
		blacklistPrefixes []string
	}

	constraints := make(map[string]constraint)

	// models should not import services or APIs.
	constraints["github.com/percona/pmm-managed/models"] = constraint{
		blacklistPrefixes: []string{
			"github.com/percona/pmm-managed/services",
			"github.com/percona/pmm/api",
		},
	}

	// services should be independent
	for _, service := range []string{
		"github.com/percona/pmm-managed/services/agents",
		"github.com/percona/pmm-managed/services/alertmanager",
		"github.com/percona/pmm-managed/services/checks",
		"github.com/percona/pmm-managed/services/grafana",
		"github.com/percona/pmm-managed/services/qan",
		"github.com/percona/pmm-managed/services/server",
		"github.com/percona/pmm-managed/services/supervisord",
		"github.com/percona/pmm-managed/services/telemetry",

		// TODO add "github.com/percona/pmm-managed/services/victoriametrics" once we remove prometheus package
		"github.com/percona/pmm-managed/services/prometheus",
	} {
		constraints[service] = constraint{
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services",
			},
		}
	}

	for _, service := range []string{
		// those services should be independent too, but has some common code
		// as converters, errors, ...
		"github.com/percona/pmm-managed/services/inventory",
		"github.com/percona/pmm-managed/services/management",
		"github.com/percona/pmm-managed/services/server",
		"github.com/percona/pmm-managed/services/checks",
	} {
		constraints[service] = constraint{
			blacklistPrefixes: []string{
				"github.com/percona/pmm-managed/services/",
			},
		}
	}

	// just to add them to packages.dot
	for _, service := range []string{
		"github.com/percona/pmm-managed",
		"github.com/percona/pmm-managed/cmd/pmm-managed-init",
		"github.com/percona/pmm-managed/cmd/pmm-managed-starlark",
		"github.com/percona/pmm-managed/services/agents/grpc",
		"github.com/percona/pmm-managed/services/inventory/grpc",
		"github.com/percona/pmm-managed/services/management/grpc",

		// TODO remove from the once we add it above
		"github.com/percona/pmm-managed/services/victoriametrics",
	} {
		constraints[service] = constraint{}
	}

	allImports := make(map[string]map[string]struct{})
	for path, c := range constraints {
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
		if p == "" {
			p = "/"
		}
		for _, i := range imports {
			if strings.Contains(i, "/utils/") {
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
