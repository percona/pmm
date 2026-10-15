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

package commands

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/utils/secrets"
	"github.com/percona/pmm/utils/vault"
)

type mapVault map[string]string

func (m mapVault) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

func (m mapVault) Write(_ context.Context, path string, data map[string]string) error {
	for k, v := range data {
		m[path+"#"+k] = v
	}
	return nil
}

func tokenConfig(password string) *config.Config {
	return &config.Config{ID: testAgentID, Server: config.Server{Address: "127.0.0.1:1", Username: "service_token", Password: password}}
}

// Not parallel: it configures the process-wide Vault client.
func TestSetupWithVault(t *testing.T) {
	const (
		ref      = "vault:secret/data/pmm/agents/node1#token"
		otherRef = "vault:secret/data/pmm/agents/other#token"
		missing  = "vault:secret/data/pmm/agents/missing#token"
	)
	secrets.Configure(mapVault{
		"secret/data/pmm/agents/node1#token": "glsa_token",
		"secret/data/pmm/agents/other#token": "glsa_other",
	})
	t.Cleanup(func() { secrets.Configure(nil) })
	l := logrus.WithField("test", t.Name())

	t.Run("a reference counts as the token it stands for", func(t *testing.T) {
		assert.True(t, sameCredentials(tokenConfig(ref), tokenConfig("glsa_token")))
		assert.Empty(t, unappliedCredentials(tokenConfig(ref), tokenConfig("glsa_token")))
		assert.False(t, sameCredentials(tokenConfig(missing), tokenConfig("glsa_token")), "an unreadable reference is not the token")
	})

	t.Run("a reference to the token the Agent runs with moves it to Vault", func(t *testing.T) {
		cfg := tokenConfig(ref)
		keepRegistration(cfg, tokenConfig("glsa_token"))
		assert.Equal(t, ref, cfg.Server.Password)
	})

	t.Run("a reference to another token is not adopted", func(t *testing.T) {
		cfg := tokenConfig(otherRef)
		assert.Equal(t, []string{"--server-username", "--server-password"}, unappliedCredentials(cfg, tokenConfig("glsa_token")))
		keepRegistration(cfg, tokenConfig("glsa_token"))
		assert.Equal(t, "glsa_token", cfg.Server.Password)
	})

	t.Run("where the token issued at registration goes", func(t *testing.T) {
		asked := &config.Config{Setup: config.Setup{ServerTokenVaultPath: "/secret/data/pmm/agents/node1/"}}
		assert.Equal(t, ref, tokenRef(asked, nil))

		// The reference the Agent reads its token from is replaced in place, given or in the file.
		assert.Equal(t, ref, tokenRef(tokenConfig(ref), nil))
		admin := &config.Config{Server: config.Server{Username: "admin", Password: "vault:secret/data/pmm#admin_password"}}
		assert.Equal(t, ref, tokenRef(admin, tokenConfig(ref)))

		// The credentials setup registers with are not where the token goes.
		assert.Empty(t, tokenRef(admin, nil))
		assert.Empty(t, tokenRef(tokenConfig("glsa_token"), tokenConfig("glsa_token")))
	})

	t.Run("an unreadable reference leaves the registration unverified", func(t *testing.T) {
		assert.Equal(t, registrationUnverified, checkRegistrationOnServer(tokenConfig(missing), tokenConfig(missing), l))
	})

	t.Run("the Vault settings come from the file pmm-admin config does not load", func(t *testing.T) {
		fileCfg := &config.Config{Vault: vault.Config{Address: "https://vault:8200", Token: "hvs.file"}}
		args := []string{"setup", "127.0.0.1", "generic", "node1"}

		assert.Equal(t, "https://vault:8200", vaultConfig(&config.Config{}, fileCfg, false, args, l).Vault.Address)

		flags := append([]string{"setup", "--vault-address=https://other:8200"}, args[1:]...)
		got := vaultConfig(&config.Config{}, fileCfg, false, flags, l).Vault
		assert.Equal(t, "https://other:8200", got.Address)
		assert.Equal(t, "hvs.file", got.Token)

		cfg := &config.Config{Vault: vault.Config{Address: "https://loaded:8200"}}
		assert.Same(t, cfg, vaultConfig(cfg, fileCfg, true, args, l))
	})
}

func TestWithStoredVaultCredentials(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
	require.NoError(t, config.SaveToFile(path, &config.Config{
		ID:    testAgentID,
		Vault: vault.Config{Address: "https://vault:8200", AuthMethod: vault.AuthMethodAppRole, RoleID: "role", SecretID: "from-file"},
	}, ""))

	// Given by the environment, which the operator keeps them in to keep them off the disk.
	cfg := &config.Config{ID: testAgentID, Vault: vault.Config{
		Address: "https://vault:8200", AuthMethod: vault.AuthMethodAppRole, RoleID: "role", SecretID: "from-env", Token: "hvs.env",
	}}

	stored := withStoredVaultCredentials(cfg, path)
	assert.Equal(t, "from-file", stored.Vault.SecretID)
	assert.Empty(t, stored.Vault.Token)
	assert.Equal(t, "from-env", cfg.Vault.SecretID, "the running configuration keeps them")
}

func TestWithoutEnvironmentVaultCredentials(t *testing.T) {
	t.Setenv("PMM_AGENT_VAULT_TOKEN", "hvs.env")
	t.Setenv("PMM_AGENT_VAULT_SECRET_ID", "from-env")

	cfg := &config.Config{ID: testAgentID, Vault: vault.Config{Address: "https://vault:8200", Token: "hvs.env", SecretID: "from-env"}}
	fileCfg := &config.Config{Vault: vault.Config{SecretID: "from-file"}}

	stored := withoutEnvironmentVaultCredentials(cfg, fileCfg, []string{"setup", "--vault-address=https://vault:8200"})
	assert.Empty(t, stored.Vault.Token)
	assert.Equal(t, "from-file", stored.Vault.SecretID)
	assert.Equal(t, "https://vault:8200", stored.Vault.Address)
	assert.Equal(t, "hvs.env", cfg.Vault.Token, "the configuration setup runs with keeps them")

	// Those given by flags are stored, even when the environment holds the same.
	stored = withoutEnvironmentVaultCredentials(cfg, nil, []string{"setup", "--vault-token=hvs.env", "--vault-secret-id", "from-env"})
	assert.Equal(t, "hvs.env", stored.Vault.Token)
	assert.Equal(t, "from-env", stored.Vault.SecretID)
}
