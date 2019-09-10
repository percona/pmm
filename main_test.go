// pmm-admin
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
	cmd := exec.Command("pmm-admin", "-h") //nolint:gosec
	b, err := cmd.CombinedOutput()
	require.NoError(t, err)

	out := string(b)
	assert.False(t, strings.Contains(out, "httptest.serve"), `pmm-admin should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "test.run"), `pmm-admin should not import package "testing"`)
}

func TestImports(t *testing.T) {
	type constraint struct {
		blacklist []string
	}

	for path, c := range map[string]constraint{
		// "github.com/percona/pmm-admin/commands": {
		// 	blacklist: []string{
		// 		"gopkg.in/alecthomas/kingpin.v2",
		// 	},
		// },
	} {
		p, err := build.Import(path, ".", build.IgnoreVendor)
		require.NoError(t, err)

		allImports := map[string]struct{}{}
		for _, i := range p.Imports {
			allImports[i] = struct{}{}
		}
		for _, i := range p.TestImports {
			allImports[i] = struct{}{}
		}
		for _, i := range p.XTestImports {
			allImports[i] = struct{}{}
		}

		for _, i := range c.blacklist {
			if _, ok := allImports[i]; ok {
				t.Errorf("Package %q should not import %q.", path, i)
			}
		}
	}
}
