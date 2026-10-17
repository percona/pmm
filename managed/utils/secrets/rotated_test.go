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

package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRotatedEnv(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pmm-secrets", "rotated-env.json")

	env, err := ReadRotatedEnv(path)
	require.NoError(t, err)
	assert.Nil(t, env, "none rotated yet")

	require.NoError(t, WriteRotatedEnv(path, map[string]string{"GF_DATABASE_PASSWORD": "new\nline=and quotes'\""}))
	fi, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	fi, err = os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), fi.Mode().Perm())

	env, err = ReadRotatedEnv(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"GF_DATABASE_PASSWORD": "new\nline=and quotes'\""}, env)

	assert.ElementsMatch(t, []string{"PATH=/bin", "GF_DATABASE_PASSWORD=new\nline=and quotes'\""},
		WithRotatedEnv([]string{"PATH=/bin", "GF_DATABASE_PASSWORD=old"}, env))

	require.NoError(t, os.WriteFile(path, []byte("not JSON"), 0o600))
	_, err = ReadRotatedEnv(path)
	require.Error(t, err)
}
