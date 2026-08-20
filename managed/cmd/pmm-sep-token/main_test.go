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
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Captured from Grafana's own encoding for this secret. The point of this
// program is to agree with it, so a change here means the tokens it mints stop
// authenticating - not that the expectation needs updating.
const (
	goldenSecret   = "AbC123xyzAbC123xyzAbC123xyzAbC12"
	goldenChecksum = "1de4e29c"
	goldenHash     = "04b88d1cf73811f69ad14ec35d15e2c1c80a06f9aa8cbf5b1266ad979635e21b" +
		"c377f2b1c507fd839f756482ac30e05c4fff"
)

var (
	tokenRE = regexp.MustCompile(`^glsa_[A-Za-z0-9]{32}_[0-9a-f]{8}$`)
	hashRE  = regexp.MustCompile(`^[0-9a-f]{100}$`)
	saltRE  = regexp.MustCompile(`^[A-Za-z0-9]{10}$`)
	uidRE   = regexp.MustCompile(`^[a-z0-9]{14}$`)
)

func TestEncodingMatchesGrafana(t *testing.T) {
	t.Parallel()

	assert.Equal(t, goldenChecksum, checksumFor(goldenSecret))

	hashed, err := hashSecret(goldenSecret, goldenChecksum)
	require.NoError(t, err)
	assert.Equal(t, goldenHash, hashed)
}

func TestGenerateShape(t *testing.T) {
	t.Parallel()

	parts, err := generate()
	require.NoError(t, err)
	require.Len(t, parts, 5)

	assert.Regexp(t, tokenRE, parts[0])
	assert.Regexp(t, hashRE, parts[1])
	assert.Regexp(t, saltRE, parts[2])
	assert.Regexp(t, saltRE, parts[3])
	assert.Regexp(t, uidRE, parts[4])
}

// The token Grafana hands out and the hash Grafana stores have to describe the
// same secret, so the emitted pair is re-derived from the token itself.
func TestGeneratedPairAgrees(t *testing.T) {
	t.Parallel()

	parts, err := generate()
	require.NoError(t, err)
	require.Len(t, parts, 5)

	trimmed := strings.TrimPrefix(parts[0], tokenPrefix)
	secret, checksum, found := strings.Cut(trimmed, "_")
	require.True(t, found)

	assert.Equal(t, checksumFor(secret), checksum)

	hashed, err := hashSecret(secret, checksum)
	require.NoError(t, err)
	assert.Equal(t, parts[1], hashed)
}

func TestGenerateDoesNotRepeat(t *testing.T) {
	t.Parallel()

	first, err := generate()
	require.NoError(t, err)
	second, err := generate()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestRandomString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		length   int
		alphabet string
	}{
		{"secret", secretLength, alphanumeric},
		{"uid", uidLength, lowerAlnum},
		{"single character alphabet", 8, "a"},
		{"zero length", 0, alphanumeric},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := randomString(tc.length, tc.alphabet)
			require.NoError(t, err)
			assert.Len(t, out, tc.length)

			for _, r := range out {
				assert.Contains(t, tc.alphabet, string(r))
			}
		})
	}
}
