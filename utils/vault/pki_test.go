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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientIssueCertificate(t *testing.T) {
	t.Parallel()

	fv, srv := newFakeVault(t)
	fv.issue = func(body map[string]any) map[string]any {
		return map[string]any{
			"certificate": "leaf of " + body["common_name"].(string),
			"private_key": "key",
			"issuing_ca":  "ca",
			"expiration":  1792000000,
		}
	}
	c, err := New(Config{Address: srv.URL, Token: "static-token", CacheTTL: time.Hour})
	require.NoError(t, err)

	req := CertificateRequest{CommonName: "server.global.nomad", AltNames: []string{"localhost", "a"}, IPSANs: []string{"127.0.0.1"}}
	for range 2 {
		cert, err := c.IssueCertificate(t.Context(), "/pki/issue/pmm-nomad/", req)
		require.NoError(t, err)
		assert.Equal(t, &Certificate{
			Certificate: "leaf of server.global.nomad",
			PrivateKey:  "key",
			IssuingCA:   "ca",
			Expiration:  time.Unix(1792000000, 0),
		}, cert)
	}

	require.Len(t, fv.writes, 2, "issued each time, never from the cache")
	assert.Equal(t, fakeWrite{path: "pki/issue/pmm-nomad", body: map[string]any{
		"common_name": "server.global.nomad",
		"alt_names":   "localhost,a",
		"ip_sans":     "127.0.0.1",
		"format":      "pem",
	}}, fv.writes[0])

	t.Run("an incomplete response is an error", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.issue = func(map[string]any) map[string]any { return map[string]any{"certificate": "leaf"} }
		c, err := New(Config{Address: srv.URL, Token: "static-token"})
		require.NoError(t, err)

		_, err = c.IssueCertificate(t.Context(), "pki/issue/pmm-nomad", CertificateRequest{CommonName: "x"})
		require.ErrorContains(t, err, "no certificate, private key, or issuing CA in the response")
	})

	t.Run("a refused request is an error", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		c, err := New(Config{Address: srv.URL, Token: "static-token"})
		require.NoError(t, err)
		fv.revokeAll()

		_, err = c.IssueCertificate(t.Context(), "pki/issue/pmm-nomad", CertificateRequest{CommonName: "x"})
		require.ErrorContains(t, err, "403")
	})
}
