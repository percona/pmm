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

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOrphanedExporter(t *testing.T) {
	t.Parallel()

	const base = "/usr/local/percona/pmm/exporters"

	for _, tc := range []struct {
		name     string
		exe      string
		base     string
		ppid     int
		expected bool
	}{
		{"exporter reparented to init", base + "/vmagent", base, 1, true},
		{"exporter below the exporters directory", base + "/vm/vmagent", base, 1, true},
		{"exporters directory with a trailing separator", base + "/vmagent", base + "/", 1, true},

		// Somebody is still running this one - on a host with a second pmm-agent, that
		// somebody is the other pmm-agent.
		{"exporter whose pmm-agent is still alive", base + "/vmagent", base, 4242, false},

		// The reason this compares paths rather than string prefixes.
		{"sibling directory sharing the prefix", "/usr/local/percona/pmm/exporters-old/vmagent", base, 1, false},

		{"process outside the exporters directory", "/usr/bin/vmagent", base, 1, false},
		{"the exporters directory itself", base, base, 1, false},
		{"no executable", "", base, 1, false},
		{"no exporters directory configured", base + "/vmagent", "", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, isOrphanedExporter(tc.exe, tc.base, tc.ppid))
		})
	}
}
