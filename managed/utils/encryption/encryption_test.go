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

package encryption

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptionGenerateKey(t *testing.T) {
	e := &Encryption{}

	key1, err := e.GenerateKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key1)

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(key1)
	assert.NoError(t, err)

	// Generate another key and ensure they are different
	key2, err := e.GenerateKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)

	// Verify second key is also valid base64
	_, err = base64.StdEncoding.DecodeString(key2)
	assert.NoError(t, err)
}

func TestEncryptionGenerateAndPersistKey(t *testing.T) {
	// Create a temporary file path for testing
	tempFile, err := os.CreateTemp("", "encryption_test_*.key")
	require.NoError(t, err)
	tempFile.Close()

	t.Cleanup(func() {
		os.Remove(tempFile.Name())
	})

	e := &Encryption{Path: tempFile.Name()}

	err = e.generateAndPersistKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, e.Key)

	// Verify the file was written with the correct content
	content, err := os.ReadFile(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, e.Key, string(content))

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(e.Key)
	assert.NoError(t, err)
}
