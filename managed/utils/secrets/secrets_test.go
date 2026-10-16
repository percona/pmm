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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/utils/vault"
)

type mapStore map[string]string

func (m mapStore) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

// Not parallel: it configures the process-wide Vault client.
func TestResolve(t *testing.T) {
	t.Cleanup(func() { Configure(nil, 0) })

	Configure(nil, 0)
	assert.False(t, Configured())
	assert.Equal(t, vault.DefaultCacheTTL, CacheTTL())

	v, err := Resolve(t.Context(), "plain")
	require.NoError(t, err)
	assert.Equal(t, "plain", v, "a secret stored in PMM Server needs no Vault")

	_, err = Resolve(t.Context(), "vault:secret/data/pmm#key")
	require.ErrorIs(t, err, ErrNotConfigured)

	Configure(mapStore{"secret/data/pmm#key": "s3cr3t"}, 15*time.Second)
	assert.True(t, Configured())
	assert.Equal(t, 15*time.Second, CacheTTL())

	v, err = Resolve(t.Context(), "vault:secret/data/pmm#key")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", v)

	_, err = Resolve(t.Context(), "vault:secret/data/pmm#missing")
	require.EqualError(t, err, "cannot read vault:secret/data/pmm#missing from Vault: no such secret")

	_, err = Resolve(t.Context(), "vault:secret/data/pmm")
	require.ErrorContains(t, err, "vault:<path>#<key>")

	a, b := "vault:secret/data/pmm#key", "plain"
	require.NoError(t, ResolveAll(t.Context(), &a, &b))
	assert.Equal(t, []string{"s3cr3t", "plain"}, []string{a, b})
}

type issuingStore struct {
	mapStore

	issue func(path string, req vault.CertificateRequest) (*vault.Certificate, error)
}

func (s issuingStore) IssueCertificate(_ context.Context, path string, req vault.CertificateRequest) (*vault.Certificate, error) {
	return s.issue(path, req)
}

// Not parallel: it configures the process-wide Vault client.
func TestIssueCertificate(t *testing.T) {
	t.Cleanup(func() { Configure(nil, 0) })
	req := vault.CertificateRequest{CommonName: "server.global.nomad"}

	Configure(nil, 0)
	_, err := IssueCertificate(t.Context(), "pki/issue/pmm-nomad", req)
	require.ErrorIs(t, err, ErrNotConfigured)

	Configure(mapStore{}, 0)
	_, err = IssueCertificate(t.Context(), "pki/issue/pmm-nomad", req)
	require.EqualError(t, err, "cannot issue a certificate from pki/issue/pmm-nomad: Vault does not issue certificates")

	Configure(issuingStore{issue: func(path string, r vault.CertificateRequest) (*vault.Certificate, error) {
		assert.Equal(t, "pki/issue/pmm-nomad", path)
		assert.Equal(t, req, r)
		return &vault.Certificate{Certificate: "leaf"}, nil
	}}, 0)
	cert, err := IssueCertificate(t.Context(), "pki/issue/pmm-nomad", req)
	require.NoError(t, err)
	assert.Equal(t, "leaf", cert.Certificate)

	Configure(issuingStore{issue: func(string, vault.CertificateRequest) (*vault.Certificate, error) {
		return nil, errors.New("permission denied")
	}}, 0)
	_, err = IssueCertificate(t.Context(), "pki/issue/pmm-nomad", req)
	require.EqualError(t, err, "cannot issue a certificate from pki/issue/pmm-nomad: permission denied")
}

func TestConfigFromEnv(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"PMM_VAULT_ADDRESS":         "https://vault:8200",
		"PMM_VAULT_AUTH_METHOD":     "approle",
		"PMM_VAULT_ROLE_ID":         "role",
		"PMM_VAULT_SECRET_ID_FILE":  "/run/secret-id",
		"PMM_VAULT_TLS_SKIP_VERIFY": "true",
		"PMM_VAULT_CACHE_TTL":       "30s",
	}
	cfg, err := ConfigFromEnv(func(name string) string { return env[name] })
	require.NoError(t, err)
	assert.Equal(t, vault.Config{
		Address:       "https://vault:8200",
		AuthMethod:    "approle",
		RoleID:        "role",
		SecretIDFile:  "/run/secret-id",
		TLSSkipVerify: true,
		CacheTTL:      30 * time.Second,
	}, cfg)

	env["PMM_VAULT_CACHE_TTL"] = "soon"
	_, err = ConfigFromEnv(func(name string) string { return env[name] })
	require.ErrorContains(t, err, "invalid PMM_VAULT_CACHE_TTL")

	azure := map[string]string{
		"PMM_VAULT_ADDRESS":         "https://vault:8200",
		"PMM_VAULT_AUTH_METHOD":     "azure",
		"PMM_VAULT_ROLE":            "pmm-server",
		"PMM_VAULT_AZURE_RESOURCE":  "https://vault.example.com",
		"PMM_VAULT_AZURE_CLIENT_ID": "6d0c1f2e",
	}
	cfg, err = ConfigFromEnv(func(name string) string { return azure[name] })
	require.NoError(t, err)
	assert.Equal(t, vault.Config{
		Address:       "https://vault:8200",
		AuthMethod:    "azure",
		Role:          "pmm-server",
		AzureResource: "https://vault.example.com",
		AzureClientID: "6d0c1f2e",
	}, cfg)
}
