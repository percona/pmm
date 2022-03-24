// pmm-agent
// Copyright 2019 Percona LLC
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
	"go/build"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
//
// goroutine 1 [running]:
// main.main()
//         /home/user/go/src/github.com/percona/pmm-agent/main.go:34 +0x22a
// exit status 2
// 3. Do we really need to test the output of a command? It is making local tests to always fail.
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
		"github.com/percona/pmm-agent/agents/mongodb",
		"github.com/percona/pmm-agent/agents/mysql/perfschema",
		"github.com/percona/pmm-agent/agents/mysql/slowlog",
		"github.com/percona/pmm-agent/agents/noop",
		"github.com/percona/pmm-agent/agents/postgres/pgstatmonitor",
		"github.com/percona/pmm-agent/agents/postgres/pgstatstatements",
		"github.com/percona/pmm-agent/agents/process",
		"github.com/percona/pmm-agent/agents/cache",
	} {
		c := constraint{
			denyPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		}

		// TODO move parser to pgstatstatements/parser unless pgstatmonitor will actually use it
		if strings.HasSuffix(a, "/pgstatstatements") {
			c.allowPrefixes = []string{
				"github.com/percona/pmm-agent/agents/postgres/parser",
			}
		}

		// allows agents to use cache
		for _, cachedAgent := range agentsUsingCache {
			if strings.HasSuffix(a, cachedAgent) {
				c.allowPrefixes = append(c.allowPrefixes, "github.com/percona/pmm-agent/agents/cache")
			}
		}

		constraints[a] = c
	}

	// slowlog parser should be fully independent
	constraints["github.com/percona/pmm-agent/agents/mysql/slowlog/parser"] = constraint{
		denyPrefixes: []string{
			"github.com/percona/pmm-agent",
		},
	}

	// those packages should be independent from each other
	packs := []string{
		// TODO https://jira.percona.com/browse/PMM-7206
		// "github.com/percona/pmm-agent/actions",

		"github.com/percona/pmm-agent/agentlocal",
		"github.com/percona/pmm-agent/agents/supervisor",
		"github.com/percona/pmm-agent/client",
		"github.com/percona/pmm-agent/connectionchecker",
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
		"github.com/percona/pmm-agent/commands",
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

		for _, d := range c.denyPrefixes {
			for i := range allImports[path] {
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

		p = strings.TrimPrefix(p, "github.com/percona/pmm-agent")
		if p == "" {
			p = "/"
		}
		for _, i := range imports {
			if strings.Contains(i, "/utils/") {
				continue
			}
			if strings.HasPrefix(i, "github.com/percona/pmm-agent") {
				i = strings.TrimPrefix(i, "github.com/percona/pmm-agent")
				fmt.Fprintf(f, "\t%q -> %q;\n", p, i)
			}
		}
	}

	fmt.Fprintf(f, "}\n")
}
