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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

func TestVaultEnv(t *testing.T) {
	t.Parallel()

	const tricky = "it's $HOME `id` \"quoted\"\nsecond line"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "root" || r.URL.Path != "/v1/secret/data/pmm-server" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"data":     map[string]any{"admin_password": tricky, "clickhouse_password": "ch"},
				"metadata": map[string]any{"version": 1},
			},
		})
	}))
	t.Cleanup(srv.Close)

	cfg, err := secrets.ConfigFromEnv(func(name string) string {
		return map[string]string{"PMM_VAULT_ADDRESS": srv.URL, "PMM_VAULT_TOKEN": "root"}[name]
	})
	require.NoError(t, err)

	t.Run("Exports", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		err := vaultEnv(t.Context(), cfg, []string{
			"PMM_ADMIN_PASSWORD=vault:secret/data/pmm-server#admin_password",
			"GF_DATABASE_PASSWORD=vault:secret/data/pmm-server#clickhouse_password",
			"VMAGENT_remoteWrite_basicAuth_password=vault:secret/data/pmm-server#clickhouse_password",
			"PMM_DEBUG=1",
			"PMM_VAULT_TOKEN_FILE=vault:not/resolved#key",
			"PMM_AGENT_SERVER_PASSWORD=vault:not/resolved#key",
			"OTHER=vault:not/resolved#key",
		}, &out)
		require.NoError(t, err)

		script := out.String() + `printf '%s|%s|%s' "$PMM_ADMIN_PASSWORD" "$GF_DATABASE_PASSWORD" "$PMM_VAULT_REF_PMM_ADMIN_PASSWORD"`
		b, err := exec.CommandContext(t.Context(), "bash", "-c", script).Output()
		require.NoError(t, err)
		assert.Equal(t, tricky+"|ch|vault:secret/data/pmm-server#admin_password", string(b))
		assert.Equal(t, 6, strings.Count(out.String(), "export "))
		assert.Contains(t, out.String(), "export VMAGENT_remoteWrite_basicAuth_password='ch'\n")
		assert.Contains(t, out.String(),
			"export PMM_VAULT_REF_VMAGENT_remoteWrite_basicAuth_password='vault:secret/data/pmm-server#clickhouse_password'\n")
	})

	t.Run("NoReferences", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		require.NoError(t, vaultEnv(t.Context(), vault.Config{}, []string{"PMM_DEBUG=1"}, &out))
		assert.Empty(t, out.String())
	})

	t.Run("NotConfigured", func(t *testing.T) {
		t.Parallel()

		err := vaultEnv(t.Context(), vault.Config{}, []string{"PMM_ADMIN_PASSWORD=vault:secret/data/pmm-server#admin_password"}, &strings.Builder{})
		require.EqualError(t, err, "PMM_ADMIN_PASSWORD is a Vault reference, but PMM_VAULT_ADDRESS is not set")
	})

	t.Run("MissingKey", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		err := vaultEnv(t.Context(), cfg, []string{"PMM_ADMIN_PASSWORD=vault:secret/data/pmm-server#nope"}, &out)
		require.EqualError(t, err, "failed to read PMM_ADMIN_PASSWORD from Vault: vault secret secret/data/pmm-server has no key nope")
		assert.Empty(t, out.String())
	})
}

func TestVaultPut(t *testing.T) {
	t.Parallel()

	var written map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Header.Get("X-Vault-Token") != "root":
			w.WriteHeader(http.StatusForbidden)
		case r.URL.Path == "/v1/sys/internal/ui/mounts/secret/data/pmm-server":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"type": "kv", "options": map[string]any{"version": "2"}}})
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"data": map[string]any{"admin_password": "old", "other": "kept"}, "metadata": map[string]any{}},
			})
		default:
			_ = json.NewDecoder(r.Body).Decode(&written)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)

	cfg := vault.Config{Address: srv.URL, Token: "root"}
	err := vaultPut(t.Context(), cfg, "vault:secret/data/pmm-server#admin_password", strings.NewReader("n3w pass\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"data": map[string]any{"admin_password": "n3w pass", "other": "kept"}}, written)

	err = vaultPut(t.Context(), vault.Config{}, "vault:secret/data/pmm-server#admin_password", strings.NewReader("x"))
	require.EqualError(t, err, "PMM_VAULT_ADDRESS is not set")
}
