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

package dir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDataDir(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		path string
		perm os.FileMode
		err  string
	}{{
		name: "valid params",
		path: "/tmp/testdir_valid",
		perm: os.FileMode(0o775),
		err:  ``,
	}, {
		name: "invalid path",
		path: "",
		perm: os.FileMode(0o775),
		err:  `cannot create path "": mkdir : no such file or directory`,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.path != "" {
				t.Cleanup(func() {
					assert.NoError(t, os.Remove(tc.path))
				})
			}

			err := CreateDataDir(tc.path, tc.perm)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			stat, err := os.Stat(tc.path)
			require.NoError(t, err)
			assert.True(t, stat.IsDir())
			assert.Equal(t, tc.perm, stat.Mode().Perm())
		})
	}
}

func TestFindFilesWithExtensions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	createTemp := func(pattern string) {
		_, err := os.CreateTemp(tmpDir, t.Name()+pattern)
		require.NoError(t, err)
	}

	createTemp("*.yaml")
	createTemp("*.yaml")
	createTemp("*.yml")
	createTemp("*")

	testcases := []struct {
		name       string
		extensions []string
		expected   int
	}{
		{
			name:       "only yml",
			extensions: []string{"yml"},
			expected:   1,
		},
		{
			name:       "only yaml",
			extensions: []string{"yaml"},
			expected:   2,
		},
		{
			name:       "yml and yaml",
			extensions: []string{"yml", "yaml"},
			expected:   3,
		},
		{
			name:       "non existing - bin",
			extensions: []string{"bin"},
			expected:   0,
		},
		{
			name:       "empty",
			extensions: []string{""},
			expected:   0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			files, err := FindFilesWithExtensions(tmpDir, tc.extensions...)
			require.NoError(t, err)
			assert.Len(t, files, tc.expected)
		})
	}
}

func TestWriteFileAtomic(t *testing.T) {
	t.Parallel()

	t.Run("creates a new file with the requested permissions", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "provisioning.json")
		require.NoError(t, WriteFileAtomic(path, []byte("first"), 0o664))

		content, err := os.ReadFile(path) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "first", string(content))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o664), info.Mode().Perm())
	})

	t.Run("replaces an existing file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "provisioning.json")
		require.NoError(t, WriteFileAtomic(path, []byte("first"), 0o664))
		require.NoError(t, WriteFileAtomic(path, []byte("second"), 0o664))

		content, err := os.ReadFile(path) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "second", string(content))
	})

	t.Run("leaves no temporary files behind", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "provisioning.json")
		require.NoError(t, WriteFileAtomic(path, []byte("content"), 0o664))

		// Grafana reads every file in its provisioning directory, so a leftover temporary file
		// would be parsed as a second provisioning file.
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "provisioning.json", entries[0].Name())
	})

	t.Run("leaves the previous content in place when the write fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "provisioning.json")
		require.NoError(t, WriteFileAtomic(path, []byte("good"), 0o664))

		require.Error(t, WriteFileAtomic(filepath.Join(dir, "missing", "provisioning.json"), []byte("bad"), 0o664))

		content, err := os.ReadFile(path) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "good", string(content))
	})
}
