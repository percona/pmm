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

package vault

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSealed(t *testing.T) {
	t.Parallel()

	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)

	tlsKey := "-----BEGIN PRIVATE KEY-----\n" + strings.Repeat("MIIEvQIBADANBgkqhkiG9w0BAQEFAASC", 50) + "\n-----END PRIVATE KEY-----\n"
	for _, value := range []string{"s3cr3t", "", "p@ss:w/rd#?%", tlsKey} {
		ref, err := Seal(key.PublicKey().Bytes(), "password", value)
		require.NoError(t, err)
		assert.True(t, IsSealedRef(ref))
		assert.True(t, ToStore(ref), "a secret to store")
		if value != "" {
			assert.NotContains(t, ref, value, "not in plaintext")
		}

		r, err := ParseRef(ref)
		require.NoError(t, err, "a valid reference")
		assert.Equal(t, "password", r.Key)

		opened, err := Open(key, r.Path)
		require.NoError(t, err)
		assert.Equal(t, value, opened)
	}

	ref, err := Seal(key.PublicKey().Bytes(), "password", "s3cr3t")
	require.NoError(t, err)
	r, err := ParseRef(ref)
	require.NoError(t, err)

	other, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	_, err = Open(other, r.Path)
	require.ErrorContains(t, err, "sealed to another key")

	// Changed on the way, it does not open.
	tampered := r.Path[:len(r.Path)-2] + map[bool]string{true: "AA", false: "BB"}[!strings.HasSuffix(r.Path, "AA")]
	_, err = Open(key, tampered)
	require.Error(t, err)

	_, err = Open(key, SealedPathPrefix+"not-base64!")
	require.ErrorContains(t, err, "malformed sealed secret")
	_, err = Open(key, "secret/data/pmm")
	require.Error(t, err)

	_, err = Seal([]byte("not a key"), "password", "s3cr3t")
	require.ErrorContains(t, err, "invalid sealing key")
}

// TestOpenWebCrypto opens a secret that the PMM web interface sealed, with WebCrypto in Node.js, to the public key
// of a fixed private key: the browser and pmm-agent must agree on every step of the sealing.
func TestOpenWebCrypto(t *testing.T) {
	t.Parallel()

	d := sha256.Sum256([]byte("pmm-agent-sealed test vector"))
	key, err := ecdh.P256().NewPrivateKey(d[:])
	require.NoError(t, err)

	const sealed = "vault:pmm-agent-sealed/BKCuioVM3JJD2lW5mv7l7amPpDsETrQItMyGiK1-ZDL-K0agP8yHFYDRDWR5Pe7Xye5ytivEXIRcE0l1" +
		"pdq2NqD433NqYyy15zWzOsX0jcygoHv1mMR_LVj99Z3fr6k3jQH8krUS9YCMFSFHJ5XhuzIQwgeS7Nc#password"
	r, err := ParseRef(sealed)
	require.NoError(t, err)
	value, err := Open(key, r.Path)
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t sealed by WebCrypto", value)
}
