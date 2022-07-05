// // pmm-managed
// // Copyright (C) 2019 Percona LLC
// //
// // This program is free software: you can redistribute it and/or modify
// // it under the terms of the GNU Affero General Public License as published by
// // the Free Software Foundation, either version 3 of the License, or
// // (at your option) any later version.
// //
// // This program is distributed in the hope that it will be useful,
// // but WITHOUT ANY WARRANTY; without even the implied warranty of
// // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// // GNU Affero General Public License for more details.
// //
// // You should have received a copy of the GNU Affero General Public License
// // along with this program. If not, see <https://www.gnu.org/licenses/>.

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

		allImports := make(map[string]struct{}, len(p.Imports)+len(p.TestImports)+len(p.XTestImports))
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
