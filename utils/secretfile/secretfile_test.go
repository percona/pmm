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

package secretfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "pw")
	require.NoError(t, os.WriteFile(file, []byte(" p@ss word\n"), 0o600))
	v, err := Read(file)
	require.NoError(t, err)
	assert.Equal(t, "p@ss word", v)

	empty := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(empty, []byte(" \n"), 0o600))
	_, err = Read(empty)
	require.EqualError(t, err, empty+" is empty")

	_, err = Read(filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
