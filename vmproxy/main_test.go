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
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/vmproxy/proxy"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	t.Run("shall run proxy with no error", func(t *testing.T) {
		t.Parallel()
		err := runProxy(flags{}, func(_ proxy.Config) error {
			return nil
		})

		require.NoError(t, err)
	})
}

func TestCredentialsFromFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	usernameFile := filepath.Join(dir, "username")
	passwordFile := filepath.Join(dir, "password")
	require.NoError(t, os.WriteFile(usernameFile, []byte("vmadmin"), 0o600))
	require.NoError(t, os.WriteFile(passwordFile, []byte("p@ss:w/rd\n"), 0o600))

	target := func(t *testing.T, s string) *url.URL {
		t.Helper()
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	t.Run("the files give the credentials", func(t *testing.T) {
		t.Parallel()

		var cfg proxy.Config
		opts := flags{
			TargetURL:          target(t, "http://127.0.0.1:9090/prometheus"),
			TargetUsernameFile: usernameFile,
			TargetPasswordFile: passwordFile,
		}
		err := runProxy(opts, func(c proxy.Config) error {
			cfg = c
			return nil
		})
		require.NoError(t, err)

		assert.Equal(t, "vmadmin", cfg.TargetURL.User.Username())
		password, _ := cfg.TargetURL.User.Password()
		assert.Equal(t, "p@ss:w/rd", password)
		assert.Nil(t, opts.TargetURL.User, "the given URL is left as it is")
	})

	t.Run("without files, the target URL is used as it is", func(t *testing.T) {
		t.Parallel()

		var cfg proxy.Config
		err := runProxy(flags{TargetURL: target(t, "http://u:p@127.0.0.1:9090")}, func(c proxy.Config) error {
			cfg = c
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, "http://u:p@127.0.0.1:9090", cfg.TargetURL.String())
	})

	t.Run("credentials in both the URL and files are refused", func(t *testing.T) {
		t.Parallel()

		opts := flags{TargetURL: target(t, "http://u:p@127.0.0.1:9090"), TargetPasswordFile: passwordFile}
		err := runProxy(opts, func(proxy.Config) error { return nil })
		require.ErrorContains(t, err, "files with credentials are given too")
	})

	t.Run("a missing file is an error", func(t *testing.T) {
		t.Parallel()

		opts := flags{TargetURL: target(t, "http://127.0.0.1:9090"), TargetPasswordFile: filepath.Join(dir, "missing")}
		err := runProxy(opts, func(proxy.Config) error { return nil })
		require.ErrorContains(t, err, "cannot read the credentials of the target")
	})
}
