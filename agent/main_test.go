// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

/*
Commenting out these tests because not always we have a proper executable in the path.
These tests should be moved to QA testing framework.

--- FAIL: TestPackages (0.00s)
    main_test.go:35:
        	Error Trace:	main_test.go:35
        	Error:      	Received unexpected error:
        	            	exec: "pmm-agent": executable file not found in $PATH
        	Test:       	TestPackages
--- FAIL: TestVersionJson (0.00s)
    main_test.go:67:
        	Error Trace:	main_test.go:67
        	Error:      	Received unexpected error:
        	            	exec: "pmm-agent": executable file not found in $PATH
        	Test:       	TestVersionJson
--- FAIL: TestVersionPlain (0.00s)
    main_test.go:46:
        	Error Trace:	main_test.go:46
        	Error:      	Received unexpected error:
        	            	exec: "pmm-agent": executable file not found in $PATH
        	Test:       	TestVersionPlain

func TestPackages(t *testing.T) {
	cmd := exec.Command("pmm-agent", "-h") //nolint:gosec
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.False(t, strings.Contains(out, "httptest.serve"), `pmm-agent should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "test.run"), `pmm-agent should not import package "testing"`)
}

func TestVersionPlain(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("pmm-agent", "--version")
	b, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", b)

	out := string(b)
	assert.True(t, strings.Contains(out, `Version:`), `'pmm-agent --version --json' produces incorrect output format`)
}

// TODO: Review/Rewrite this test.
// 1. Only works with a built agent installed in the path
// 2. Just building the agent does not guarantees that there is a version set.
// go run main.go --version
// panic: pmm-agent version is not set during build.

goroutine 1 [running]:
main.main()
        /home/user/go/src/github.com/percona/pmm-agent/main.go:34 +0x22a
exit status 2
3. Do we really need to test the output of a command? It is making local tests to always fail.

	func TestVersionJson(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("pmm-agent", "--version", "--json")
		b, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", b)

		var jsonStruct interface{}
		err = json.Unmarshal(b, &jsonStruct)
		require.NoError(t, err, `'pmm-agent --version --json' produces incorrect output format`)
	}
*/

func TestImports(t *testing.T) {
	type constraint struct {
		denyPrefixes  []string
		allowPrefixes []string
	}

	constraints := make(map[string]constraint)

	agentsUsingCache := []string{"/perfschema", "/pgstatstatements"}

	// agents code should be independent
	for _, a := range []string{
		"github.com/percona/pmm/agent/agents/mongodb/mongolog",
		"github.com/percona/pmm/agent/agents/mongodb/profiler",
		"github.com/percona/pmm/agent/agents/mysql/perfschema",
		"github.com/percona/pmm/agent/agents/mysql/slowlog",
		"github.com/percona/pmm/agent/agents/noop",
		"github.com/percona/pmm/agent/agents/postgres/pgstatmonitor",
		"github.com/percona/pmm/agent/agents/postgres/pgstatstatements",
		"github.com/percona/pmm/agent/agents/process",
		"github.com/percona/pmm/agent/agents/cache",
	} {
		c := constraint{
			denyPrefixes: []string{
				"github.com/percona/pmm/agent/agentlocal",
				"github.com/percona/pmm/agent/agents/",
				"github.com/percona/pmm/agent/client",
				"github.com/percona/pmm/agent/config",
			},
		}

		// TODO move parser to pgstatstatements/parser unless pgstatmonitor will actually use it
		if strings.HasSuffix(a, "/pgstatstatements") {
			c.allowPrefixes = []string{
				"github.com/percona/pmm/agent/agents/postgres/parser",
			}
		}

		// allows agents to use cache
		for _, cachedAgent := range agentsUsingCache {
			if strings.HasSuffix(a, cachedAgent) {
				c.allowPrefixes = append(c.allowPrefixes, "github.com/percona/pmm/agent/agents/cache")
			}
		}

		constraints[a] = c
	}

	// those packages should be independent from each other
	packs := []string{
		// TODO https://jira.percona.com/browse/PMM-7206
		// "github.com/percona/pmm/agent/actions",

		"github.com/percona/pmm/agent/agentlocal",
		"github.com/percona/pmm/agent/agents/supervisor",
		"github.com/percona/pmm/agent/client",
		"github.com/percona/pmm/agent/connectionchecker",
	}
	for _, p := range packs {
		var c constraint
		for _, d := range packs {
			if p == d {
				continue
			}
			c.denyPrefixes = append(c.denyPrefixes, d)
		}

		constraints[p] = c
	}

	// just to add them to packages.dot
	for _, service := range []string{
		"github.com/percona/pmm/agent/commands",
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
		allPkgs = append(allPkgs, pkgs...)

		for _, p := range pkgs {
			for _, d := range c.denyPrefixes {
				for i := range p.Imports {
					// allow own subpackages
					if strings.HasPrefix(i, path) {
						continue
					}

					// check allowlist
					var allow bool
					for _, a := range c.allowPrefixes {
						if strings.HasPrefix(i, a) {
							allow = true
							break
						}
					}
					if allow {
						continue
					}

					// check denylist
					if strings.HasPrefix(i, d) {
						t.Errorf("Package %q should not import package %q (denied by %q).", path, i, d)
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
			if strings.HasPrefix(i.PkgPath, "github.com/percona/pmm/agent") {
				iName := formatPkgName(t, i.PkgPath)
				if pName == iName {
					continue
				}
				lines = append(lines, fmt.Sprintf("\t%q -> %q;\n", pName, iName))
			}
		}
	}
	sort.Strings(lines)

	fmt.Fprintf(f, "digraph packages {\n")
	duplicate := make(map[string]struct{})
	for _, line := range lines {
		if _, ok := duplicate[line]; !ok {
			duplicate[line] = struct{}{}
			fmt.Fprint(f, line)
		}
	}
	fmt.Fprintf(f, "}\n")
}

func formatPkgName(t *testing.T, name string) string {
	t.Helper()

	name = strings.TrimPrefix(name, "github.com/percona/pmm/agent")
	if name == "" {
		name = "/"
	}

	return name
}
