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

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestPackages(t *testing.T) {
	cmd := exec.Command("pmm-managed", "-h")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.NotContains(t, out, "-httptest.serve", `pmm-managed should not import package "net/http/httptest"`)
	assert.NotContains(t, out, "-test.run", `pmm-managed should not import package "testing"`)
}

func TestImports(t *testing.T) {
	type constraint struct {
		blacklistPrefixes []string
		whitelistPrefixes []string
	}

	constraints := make(map[string]constraint)

	// models should not import services or APIs.
	constraints["github.com/percona/pmm/managed/models"] = constraint{
		blacklistPrefixes: []string{
			"github.com/percona/pmm/managed/services",
			"github.com/percona/pmm/api",
		},
	}

	// services should be independent
	for _, service := range []string{
		"github.com/percona/pmm/managed/services/agents",
		"github.com/percona/pmm/managed/services/checks",
		"github.com/percona/pmm/managed/services/qan",
		"github.com/percona/pmm/managed/services/server",
		"github.com/percona/pmm/managed/services/supervisord",
		"github.com/percona/pmm/managed/services/telemetry",
		"github.com/percona/pmm/managed/services/victoriametrics",
		"github.com/percona/pmm/managed/services/vmalert",
	} {
		constraints[service] = constraint{
			blacklistPrefixes: []string{
				"github.com/percona/pmm/managed/services",
			},
		}
	}

	for _, service := range []string{
		// those services should be independent too, but has some common code
		// as converters, errors, ...
		"github.com/percona/pmm/managed/services/grafana",
		"github.com/percona/pmm/managed/services/management",
		"github.com/percona/pmm/managed/services/server",
		"github.com/percona/pmm/managed/services/checks",
	} {
		constraints[service] = constraint{
			blacklistPrefixes: []string{
				"github.com/percona/pmm/managed/services/",
			},
		}
	}

	for _, service := range []string{
		// TODO come up with a new code structure that allows cross-service communication without the need to do tricks.
		"github.com/percona/pmm/managed/services/inventory",
	} {
		constraints[service] = constraint{
			blacklistPrefixes: []string{
				"github.com/percona/pmm/managed/services/",
			},
			whitelistPrefixes: []string{
				"github.com/percona/pmm/managed/services/management/common",
			},
		}
	}

	// validators should not import gRPC stack, including errors
	constraints["github.com/percona/pmm/managed/utils/validators"] = constraint{
		blacklistPrefixes: []string{
			"google.golang.org/grpc",
		},
	}

	// just to add them to packages.dot
	for _, service := range []string{
		"github.com/percona/pmm/managed",
		"github.com/percona/pmm/managed/cmd/pmm-managed-init",
		"github.com/percona/pmm/managed/cmd/pmm-managed-starlark",
		"github.com/percona/pmm/managed/services/agents/grpc",
		"github.com/percona/pmm/managed/services/inventory/grpc",
		"github.com/percona/pmm/managed/services/management/grpc",
	} {
		constraints[service] = constraint{}
	}

	config := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports,
		Tests: true,
	}

	var allPkgs []*packages.Package
	for path, c := range constraints {
		pkgs, err := packages.Load(config, path)
		require.NoError(t, err)

		for _, p := range pkgs {
			allPkgs = append(allPkgs, p)
			for _, b := range c.blacklistPrefixes {
				for i := range p.Imports {
					// whitelist own subpackages
					if strings.HasPrefix(i, path) {
						continue
					}

					// check allowlist
					var allow bool
					for _, a := range c.whitelistPrefixes {
						if strings.HasPrefix(i, a) {
							allow = true
							break
						}
					}
					if allow {
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

	f, err := os.Create("packages.dot")
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	var lines []string
	for _, p := range allPkgs {
		pName := formatPkgName(t, p.PkgPath)
		for _, i := range p.Imports {
			if strings.Contains(i.PkgPath, "/utils/") {
				continue
			}
			if strings.HasPrefix(i.PkgPath, "github.com/percona/pmm/managed") {
				iName := formatPkgName(t, i.PkgPath)
				if pName == iName {
					continue
				}
				lines = append(lines, fmt.Sprintf("\t%q -> %q;\n", pName, iName))
			}
		}
	}
	sort.Strings(lines)

	_, err = fmt.Fprintf(f, "digraph packages {\n")
	require.NoError(t, err)
	duplicate := make(map[string]struct{})
	for _, line := range lines {
		if _, ok := duplicate[line]; !ok {
			duplicate[line] = struct{}{}
			_, err = fmt.Fprint(f, line)
			require.NoError(t, err)
		}
	}
	_, err = fmt.Fprintf(f, "}\n")
	require.NoError(t, err)
}

func formatPkgName(t *testing.T, name string) string {
	t.Helper()

	name = strings.TrimPrefix(name, "github.com/percona/pmm/managed")
	if name == "" {
		name = "/"
	}

	return name
}

func TestDBPoolBudget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		maxConns  int
		instances int
		expected  int
	}{
		// PMM Server runs PostgreSQL with max_connections=2000.
		{name: "PMM Server", maxConns: 2000, instances: 0, expected: 1500},
		{name: "PMM Server, HA with 3 nodes", maxConns: 2000, instances: 3, expected: 500},
		// A stock managed PostgreSQL keeps a quarter of its connections.
		{name: "external PostgreSQL", maxConns: 100, instances: 1, expected: 75},
		{name: "external PostgreSQL, HA with 3 nodes", maxConns: 100, instances: 3, expected: 25},
		{name: "tiny PostgreSQL", maxConns: 4, instances: 1, expected: 3},
		{name: "never zero", maxConns: 1, instances: 8, expected: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, dbPoolBudget(tc.maxConns, tc.instances))
		})
	}
}

func TestFitDBPoolSizes(t *testing.T) {
	preferred := dbPoolSizes{internal: 20, api: 50}

	for _, tc := range []struct {
		name     string
		budget   int
		expected dbPoolSizes
	}{
		{name: "budget to spare", budget: 1500, expected: preferred},
		{name: "exactly enough", budget: 70, expected: preferred},
		{name: "too small, split", budget: 40, expected: dbPoolSizes{internal: 10, api: 30}},
		{name: "way too small", budget: 4, expected: dbPoolSizes{internal: 1, api: 3}},
		{name: "never zero", budget: 1, expected: dbPoolSizes{internal: 1, api: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sizes := fitDBPoolSizes(preferred, tc.budget)
			assert.Equal(t, tc.expected, sizes)
			if tc.budget >= 2 {
				assert.LessOrEqual(t, int(sizes.internal)+int(sizes.api), tc.budget)
			}
		})
	}
}

func TestPreferredDBPoolSizes(t *testing.T) {
	sizes := preferredDBPoolSizes()

	assert.GreaterOrEqual(t, sizes.internal, int32(internalDBMinOpenConns))
	assert.LessOrEqual(t, sizes.internal, int32(internalDBPoolCap))
	assert.GreaterOrEqual(t, sizes.api, int32(apiDBMinOpenConns))
	assert.LessOrEqual(t, sizes.api, int32(apiDBPoolCap))
}
