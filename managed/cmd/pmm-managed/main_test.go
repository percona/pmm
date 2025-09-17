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

	fmt.Fprintf(f, "digraph packages {\n") //nolint:errcheck
	duplicate := make(map[string]struct{})
	for _, line := range lines {
		if _, ok := duplicate[line]; !ok {
			duplicate[line] = struct{}{}
			fmt.Fprint(f, line) //nolint:errcheck
		}
	}
	fmt.Fprintf(f, "}\n")
}

func formatPkgName(t *testing.T, name string) string {
	t.Helper()

	name = strings.TrimPrefix(name, "github.com/percona/pmm/managed")
	if name == "" {
		name = "/"
	}

	return name
}
