// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nodeinfo

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	info := Get()
	assert.Equal(t, runtime.GOOS, info.Distro)

	// all our test environments have IPv4 addresses
	ip := net.ParseIP(info.PublicAddress)
	require.NotNil(t, ip)
	assert.NotNil(t, ip.To4())

	assert.False(t, strings.HasSuffix(info.MachineID, "\n"), "%q", info.MachineID)
}

func TestCheckContainer(t *testing.T) {
	for _, tc := range []struct {
		name     string
		files    map[string]string
		env      map[string]string
		expected bool
	}{
		{
			name:  "host with cgroup v2",
			files: map[string]string{"proc/1/cgroup": "0::/init.scope\n"},
		}, {
			name:  "host with cgroup v1",
			files: map[string]string{"proc/1/cgroup": "1:name=systemd:/init.scope\n0::/init.scope\n"},
		}, {
			// the case PMM Server itself hits: cgroup v2 says nothing, only the marker file does
			name:     "docker with cgroup v2",
			files:    map[string]string{".dockerenv": "", "proc/1/cgroup": "0::/\n"},
			expected: true,
		}, {
			name:     "docker with cgroup v1",
			files:    map[string]string{"proc/1/cgroup": "1:name=systemd:/docker/dc4b1a5cb7fd\n"},
			expected: true,
		}, {
			name:     "podman",
			files:    map[string]string{"run/.containerenv": "engine=\"podman-5.4.0\"\n", "proc/1/cgroup": "0::/\n"},
			expected: true,
		}, {
			name:     "lxc",
			files:    map[string]string{"proc/1/cgroup": "0::/\n"},
			env:      map[string]string{"container": "lxc"},
			expected: true,
		}, {
			name:     "kubernetes pod with cgroup v2",
			files:    map[string]string{"proc/1/cgroup": "0::/\n"},
			env:      map[string]string{"KUBERNETES_SERVICE_HOST": "10.96.0.1"},
			expected: true,
		}, {
			name:     "kubernetes pod with cgroup v1",
			files:    map[string]string{"proc/1/cgroup": "1:name=systemd:/kubepods/besteffort/pod9f4a\n"},
			expected: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// keep the outcome independent of the environment the tests themselves run in
			t.Setenv("container", "")
			t.Setenv("KUBERNETES_SERVICE_HOST", "")
			for name, value := range tc.env {
				t.Setenv(name, value)
			}

			root := t.TempDir()
			for name, content := range tc.files {
				path := filepath.Join(root, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			}

			assert.Equal(t, tc.expected, checkContainer(root))
		})
	}
}
