package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestImports(t *testing.T) {
	type constraint struct {
		blacklistPrefixes []string
	}

	config := &packages.Config{
		Mode:  packages.NeedImports,
		Tests: true,
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
		pkgs, err := packages.Load(config, path)
		require.NoError(t, err)

		for _, p := range pkgs {
			for _, b := range c.blacklistPrefixes {
				for i := range p.Imports {
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
}
