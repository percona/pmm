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

package supervisord

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestConfigVMCredentials(t *testing.T) {
	t.Parallel()

	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "https://vmuser:vm%40pass@vm.example.com:8428/prometheus")
	require.NoError(t, err)
	s := New("/etc/supervisord.d", &models.Params{VMParams: vmParams, HAParams: &models.HAParams{}})
	settings := &models.Settings{DataRetention: 30 * 24 * time.Hour}

	for name, expected := range map[string][]string{
		"vmalert": {
			"--external.url=https://vm.example.com:8428/prometheus/\n",
			"--datasource.url=https://vm.example.com:8428/prometheus/\n",
			"--remoteRead.url=https://vm.example.com:8428/prometheus/\n",
			"--remoteWrite.url=https://vm.example.com:8428/prometheus/\n",
			"--datasource.basicAuth.usernameFile=/tmp/pmm-vm-credentials/username\n",
			"--datasource.basicAuth.passwordFile=/tmp/pmm-vm-credentials/password\n",
			"--remoteRead.basicAuth.usernameFile=/tmp/pmm-vm-credentials/username\n",
			"--remoteRead.basicAuth.passwordFile=/tmp/pmm-vm-credentials/password\n",
			"--remoteWrite.basicAuth.usernameFile=/tmp/pmm-vm-credentials/username\n",
			"--remoteWrite.basicAuth.passwordFile=/tmp/pmm-vm-credentials/password\n",
		},
		"vmproxy": {
			"--target-url=https://vm.example.com:8428/prometheus/\n",
			"--target-username-file=/tmp/pmm-vm-credentials/username\n",
			"--target-password-file=/tmp/pmm-vm-credentials/password\n",
		},
	} {
		b, err := s.marshalConfig(templates.Lookup(name), settings)
		require.NoError(t, err)
		for _, e := range expected {
			assert.Contains(t, string(b), e, name)
		}
		assert.NotContains(t, string(b), "vmuser", name)
		assert.NotContains(t, string(b), "vm%40pass", name)
		assert.NotContains(t, string(b), "vm@pass", name)
	}
}

func TestSaveVMCredentials(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "pmm-vm-credentials")
	read := func(t *testing.T, name string) string {
		t.Helper()
		fi, err := os.Stat(filepath.Join(dir, name))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm(), name)
		b, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		return string(b)
	}
	parse := func(t *testing.T, s string) *url.URL {
		t.Helper()
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	require.NoError(t, saveVMCredentials(dir, parse(t, "https://vmuser:vm%40pass@vm.example.com:8428/")))
	fi, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), fi.Mode().Perm())
	assert.Equal(t, "vmuser", read(t, "username"))
	assert.Equal(t, "vm@pass", read(t, "password"), "decoded, as vmalert sends it")

	// Changed credentials replace the files, and nothing else is left in the directory.
	require.NoError(t, saveVMCredentials(dir, parse(t, "https://other:rotated@vm.example.com:8428/")))
	assert.Equal(t, "other", read(t, "username"))
	assert.Equal(t, "rotated", read(t, "password"))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// A URL without credentials removes them.
	require.NoError(t, saveVMCredentials(dir, parse(t, "https://vm.example.com:8428/")))
	assert.NoDirExists(t, dir)
	require.NoError(t, saveVMCredentials(dir, parse(t, "https://vm.example.com:8428/")))
}
