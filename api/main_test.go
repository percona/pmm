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
		"github.com/percona/pmm/api/inventory": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocal",
				"github.com/percona/pmm/api/agent",
			},
		},
		"github.com/percona/pmm/api/management": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocal",
				"github.com/percona/pmm/api/agent",
			},
		},
		"github.com/percona/pmm/api/server": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocal",
				"github.com/percona/pmm/api/agent",
			},
		},

		// public qan-api APIs should not import private APIs
		"github.com/percona/pmm/api/qan/v1": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocal",
				"github.com/percona/pmm/api/agent",
			},
		},

		// pmm-agent<->pmm-managed and pmm-managed<->qan-api APIs should be independent from each other
		"github.com/percona/pmm/api/agent": {
			blacklistPrefixes: []string{
				"github.com/percona/pmm/api/agentlocal",
				"github.com/percona/pmm/api/qan/v1",
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
