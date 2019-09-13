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
	require.NoError(t, err, "%s", b)

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
		"github.com/percona/pmm-agent/agents/mongodb": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/mysql/perfschema": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/mysql/slowlog": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/postgres/pgstatstatements": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/mongodb",
				"github.com/percona/pmm-agent/agents/mysql",
				"github.com/percona/pmm-agent/agents/noop",
				"github.com/percona/pmm-agent/agents/process",
				"github.com/percona/pmm-agent/agents/supervisor",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/noop": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/",
				"github.com/percona/pmm-agent/client",
				"github.com/percona/pmm-agent/config",
			},
		},
		"github.com/percona/pmm-agent/agents/process": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentpb",
				"github.com/percona/pmm-agent/agentlocal",
				"github.com/percona/pmm-agent/agents/",
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

		// slowlog parser should be fully independent
		"github.com/percona/pmm-agent/agents/mysql/slowlog/parser": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm-agent",
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
