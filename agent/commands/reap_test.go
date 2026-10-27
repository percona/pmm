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

func TestIsLeftoverVMAgent(t *testing.T) {
	t.Parallel()

	const (
		vmagent = "/usr/local/percona/pmm/exporters/vmagent"
		queue   = "/usr/local/percona/pmm/tmp/vmagent-temp-dir"
	)
	args := func(exe string, rest ...string) []string { return append([]string{exe}, rest...) }

	for _, tc := range []struct {
		name     string
		exe      string
		cmdline  []string
		expected bool
	}{
		{"vmagent holding this queue", vmagent, args(vmagent, "-envflag.enable=true", "-remoteWrite.tmpDataPath="+queue), true},
		{"double-dash flag", vmagent, args(vmagent, "--remoteWrite.tmpDataPath="+queue), true},
		{"queue path not clean", vmagent, args(vmagent, "-remoteWrite.tmpDataPath=/usr/local/percona/pmm/tmp//vmagent-temp-dir/"), true},

		// An upgrade replaces the binary under a vmagent that keeps running from before it, the
		// case PMM-15773 was found in.
		{"binary replaced since it started", vmagent + " (deleted)", args(vmagent, "-remoteWrite.tmpDataPath="+queue), true},

		// Only one vmagent can use a queue; another queue belongs to somebody else.
		{"vmagent with another queue", vmagent, args(vmagent, "-remoteWrite.tmpDataPath=/opt/other/vmagent-temp-dir"), false},
		{"queue sharing the prefix", vmagent, args(vmagent, "-remoteWrite.tmpDataPath="+queue+"-old"), false},
		{"queue only as a value of another flag", vmagent, args(vmagent, "-promscrape.config="+queue), false},
		{"queue as a positional argument", vmagent, args(vmagent, "remoteWrite.tmpDataPath="+queue), false},
		{"vmagent without a queue flag", vmagent, args(vmagent, "-envflag.enable=true"), false},

		{"another binary holding this queue", "/usr/bin/vmagent", args("/usr/bin/vmagent", "-remoteWrite.tmpDataPath="+queue), false},
		{"another exporter", "/usr/local/percona/pmm/exporters/node_exporter", args("node_exporter", "-remoteWrite.tmpDataPath="+queue), false},
		{"no executable", "", args(vmagent, "-remoteWrite.tmpDataPath="+queue), false},
		{"no arguments", vmagent, []string{vmagent}, false},
		{"no command line", vmagent, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, isLeftoverVMAgent(tc.exe, tc.cmdline, vmagent, queue))
		})
	}

	t.Run("nothing configured", func(t *testing.T) {
		t.Parallel()

		cmdline := args(vmagent, "-remoteWrite.tmpDataPath="+queue)
		assert.False(t, isLeftoverVMAgent(vmagent, cmdline, "", queue))
		assert.False(t, isLeftoverVMAgent(vmagent, cmdline, vmagent, ""))
	})
}
