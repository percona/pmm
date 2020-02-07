package main

import (
	"go/build"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImports(t *testing.T) {
	type constraint struct {
		blacklistPrefixes []string
	}

	for path, c := range map[string]constraint{
		// public pmm-managed APIs should not import private APIs
		"github.com/percona/pmm/api/inventorypb": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocalpb",
				"github.com/percona/pmm/api/agentpb",
			},
		},
		"github.com/percona/pmm/api/managementpb": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocalpb",
				"github.com/percona/pmm/api/agentpb",
			},
		},
		"github.com/percona/pmm/api/serverpb": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocalpb",
				"github.com/percona/pmm/api/agentpb",
			},
		},

		// public qan-api APIs should not import private APIs
		"github.com/percona/pmm/api/qanpb": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocalpb",
				"github.com/percona/pmm/api/agentpb",
			},
		},

		// pmm-agent<->pmm-managed and pmm-managed<->qan-api APIs should be independent from each other
		"github.com/percona/pmm/api/agentpb": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocalpb",
				"github.com/percona/pmm/api/qanpb",
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
