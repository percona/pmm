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

	t.Run("reads the target password from a file", func(t *testing.T) {
		t.Parallel()

		file := filepath.Join(t.TempDir(), "vm-password")
		require.NoError(t, os.WriteFile(file, []byte("p@ss:word\n"), 0o600))
		target, err := url.Parse("http://vmuser@vmauth:8427/")
		require.NoError(t, err)

		var cfg proxy.Config
		err = runProxy(flags{TargetURL: target, TargetPasswordFile: file}, func(c proxy.Config) error {
			cfg = c
			return nil
		})
		require.NoError(t, err)
		password, ok := cfg.TargetURL.User.Password()
		require.True(t, ok)
		assert.Equal(t, "vmuser", cfg.TargetURL.User.Username())
		assert.Equal(t, "p@ss:word", password)
		_, ok = target.User.Password()
		assert.False(t, ok, "the flag's URL stays as it was")

		err = runProxy(flags{TargetURL: target, TargetUsername: "other", TargetPasswordFile: file}, func(c proxy.Config) error {
			cfg = c
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, "other", cfg.TargetURL.User.Username())
	})

	t.Run("refuses a missing password file", func(t *testing.T) {
		t.Parallel()

		target, err := url.Parse("http://vmuser@vmauth:8427/")
		require.NoError(t, err)
		err = runProxy(flags{TargetURL: target, TargetPasswordFile: filepath.Join(t.TempDir(), "missing")}, func(proxy.Config) error {
			return nil
		})
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}
