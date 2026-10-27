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

func TestIsVMAgentExecutable(t *testing.T) {
	t.Parallel()

	const vmagent = "/usr/local/percona/pmm/exporters/vmagent"

	for _, tc := range []struct {
		name     string
		exe      string
		expected bool
	}{
		{"this pmm-agent's vmagent", vmagent, true},
		{"path not clean", "/usr/local/percona/pmm/exporters//vmagent", true},

		// An upgrade replaces the binary under a vmagent that keeps running from before it, the
		// case PMM-15773 was found in.
		{"binary replaced since it started", vmagent + " (deleted)", true},

		{"another vmagent binary", "/usr/bin/vmagent", false},
		{"another exporter", "/usr/local/percona/pmm/exporters/node_exporter", false},
		{"a sibling sharing the prefix", vmagent + "-old", false},
		{"no executable", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, isVMAgentExecutable(tc.exe, vmagent))
		})
	}

	t.Run("nothing configured", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isVMAgentExecutable(vmagent, ""))
	})
}

func TestTmpDataPath(t *testing.T) {
	t.Parallel()

	const queue = "/usr/local/percona/pmm/tmp/vmagent-temp-dir"
	args := func(rest ...string) []string {
		return append([]string{"/usr/local/percona/pmm/exporters/vmagent"}, rest...)
	}

	for _, tc := range []struct {
		name     string
		cmdline  []string
		expected string
	}{
		{"the flag pmm-agent passes", args("-envflag.enable=true", "-remoteWrite.tmpDataPath="+queue), queue},
		{"double-dash flag", args("--remoteWrite.tmpDataPath=" + queue), queue},

		{"queue only as a value of another flag", args("-promscrape.config=" + queue), ""},
		{"queue as a positional argument", args("remoteWrite.tmpDataPath=" + queue), ""},
		{"empty flag value", args("-remoteWrite.tmpDataPath="), ""},
		{"no queue flag", args("-envflag.enable=true"), ""},
		{"no arguments", args(), ""},
		{"no command line", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, ok := tmpDataPath(tc.cmdline)
			assert.Equal(t, tc.expected, path)
			assert.Equal(t, tc.expected != "", ok)
		})
	}
}
