// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
	agentconfig "github.com/percona/pmm/agent/config"
)

func writeAgentConfig(t *testing.T, cfg *agentconfig.Config) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pmm-agent.yaml")
	require.NoError(t, agentconfig.SaveToFile(path, cfg, t.Name()))
	return path
}

func writeRSAKey(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	path := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(path, keyPEM, 0o600))
	return path
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func assertCreds(t *testing.T, u *url.URL, wantUser, wantPassword string) {
	t.Helper()
	require.NotNil(t, u.User)
	assert.Equal(t, wantUser, u.User.Username())
	gotPassword, _ := u.User.Password()
	assert.Equal(t, wantPassword, gotPassword)
}

func TestMergeCredsFromAgentConfig(t *testing.T) {
	t.Run("short-circuits when URL already has full credentials", func(t *testing.T) {
		// Pass a non-existent config path; if the function tried to read it, the URL would
		// remain unchanged, but we additionally assert nothing about the URL was rewritten.
		gf := &flags.GlobalFlags{
			ServerURL:       mustParseURL(t, "https://user:secret@host:8443/"),
			AgentConfigFile: "/does/not/exist.yaml",
		}
		mergeCredsFromAgentConfig(gf, "/also/missing.yaml")
		assertCreds(t, gf.ServerURL, "user", "secret")
	})

	t.Run("no config file path leaves URL unchanged", func(t *testing.T) {
		gf := &flags.GlobalFlags{
			ServerURL: mustParseURL(t, "https://user:***@host:8443/"),
		}
		mergeCredsFromAgentConfig(gf, "")
		assertCreds(t, gf.ServerURL, "user", "***")
	})

	t.Run("plaintext config from status path merges credentials", func(t *testing.T) {
		path := writeAgentConfig(t, &agentconfig.Config{
			ID: "agent-id",
			Server: agentconfig.Server{
				Address:  "host:8443",
				Username: "real-user",
				Password: "real-secret",
			},
		})
		gf := &flags.GlobalFlags{
			ServerURL: mustParseURL(t, "https://user:***@host:8443/"),
		}
		mergeCredsFromAgentConfig(gf, path)
		assertCreds(t, gf.ServerURL, "real-user", "real-secret")
	})

	t.Run("AgentConfigFile flag overrides status path", func(t *testing.T) {
		flagPath := writeAgentConfig(t, &agentconfig.Config{
			Server: agentconfig.Server{
				Address:  "host:8443",
				Username: "from-flag",
				Password: "flag-secret",
			},
		})
		statusPath := writeAgentConfig(t, &agentconfig.Config{
			Server: agentconfig.Server{
				Address:  "host:8443",
				Username: "from-status",
				Password: "status-secret",
			},
		})
		gf := &flags.GlobalFlags{
			ServerURL:       mustParseURL(t, "https://user:***@host:8443/"),
			AgentConfigFile: flagPath,
		}
		mergeCredsFromAgentConfig(gf, statusPath)
		assertCreds(t, gf.ServerURL, "from-flag", "flag-secret")
	})

	t.Run("config read error leaves URL unchanged", func(t *testing.T) {
		gf := &flags.GlobalFlags{
			ServerURL: mustParseURL(t, "https://user:***@host:8443/"),
		}
		mergeCredsFromAgentConfig(gf, "/no/such/file.yaml")
		assertCreds(t, gf.ServerURL, "user", "***")
	})

	t.Run("config without server address leaves URL unchanged", func(t *testing.T) {
		path := writeAgentConfig(t, &agentconfig.Config{ID: "agent-id"})
		gf := &flags.GlobalFlags{
			ServerURL: mustParseURL(t, "https://user:***@host:8443/"),
		}
		mergeCredsFromAgentConfig(gf, path)
		assertCreds(t, gf.ServerURL, "user", "***")
	})

	t.Run("encrypted config with matching key merges credentials", func(t *testing.T) {
		keyPath := writeRSAKey(t)
		path := writeAgentConfig(t, &agentconfig.Config{
			Server: agentconfig.Server{
				Address:  "host:8443",
				Username: "real-user",
				Password: "real-secret",
			},
			Encryption: agentconfig.Encryption{KeyFile: keyPath},
		})
		gf := &flags.GlobalFlags{
			ServerURL:          mustParseURL(t, "https://user:***@host:8443/"),
			AgentConfigKeyFile: keyPath,
		}
		mergeCredsFromAgentConfig(gf, path)
		assertCreds(t, gf.ServerURL, "real-user", "real-secret")
	})

	t.Run("encrypted config without key leaves URL unchanged", func(t *testing.T) {
		keyPath := writeRSAKey(t)
		path := writeAgentConfig(t, &agentconfig.Config{
			Server: agentconfig.Server{
				Address:  "host:8443",
				Username: "real-user",
				Password: "real-secret",
			},
			Encryption: agentconfig.Encryption{KeyFile: keyPath},
		})
		gf := &flags.GlobalFlags{
			ServerURL: mustParseURL(t, "https://user:***@host:8443/"),
		}
		mergeCredsFromAgentConfig(gf, path)
		assertCreds(t, gf.ServerURL, "user", "***")
	})

	t.Run("URL with username only still reads config", func(t *testing.T) {
		path := writeAgentConfig(t, &agentconfig.Config{
			Server: agentconfig.Server{
				Address:  "host:8443",
				Username: "real-user",
				Password: "real-secret",
			},
		})
		gf := &flags.GlobalFlags{
			ServerURL: mustParseURL(t, "https://user@host:8443/"),
		}
		mergeCredsFromAgentConfig(gf, path)
		assertCreds(t, gf.ServerURL, "real-user", "real-secret")
	})
}
