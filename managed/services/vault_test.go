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

package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/vault"
)

type fakeStorer struct {
	pmmAgentID, path string
	data             map[string]string
	err              error
}

func (f *fakeStorer) StoreSecret(_ context.Context, _ *reform.Querier, pmmAgentID, path string, data map[string]string) error {
	f.pmmAgentID, f.path, f.data = pmmAgentID, path, data
	return f.err
}

type fakeChecker struct {
	checked *models.Agent
	err     error
}

func (f *fakeChecker) CheckConnectionToService(_ context.Context, _ *reform.Querier, _ *models.Service, agent *models.Agent) error {
	f.checked = agent
	return f.err
}

func TestStoreCredentialsInVault(t *testing.T) {
	t.Parallel()

	// No Service, so that no check needs the database; the check is covered with one below.
	agent := &models.Agent{
		AgentID:    "agent-id",
		AgentType:  models.MySQLdExporterType,
		PMMAgentID: new("pmm-agent-id"),
		Username:   new("fallback"),
		VaultPath:  new("secret/data/pmm/mysql"),
	}

	t.Run("NothingGiven", func(t *testing.T) {
		t.Parallel()

		storer := &fakeStorer{}
		stored, err := StoreCredentialsInVault(t.Context(), nil, storer, &fakeChecker{}, agent, "", "", false)
		require.NoError(t, err)
		assert.False(t, stored)
		assert.Empty(t, storer.path)
	})

	t.Run("Stored", func(t *testing.T) {
		t.Parallel()

		storer := &fakeStorer{}
		stored, err := StoreCredentialsInVault(t.Context(), nil, storer, &fakeChecker{}, agent, "pmm", "s3cr3t", true)
		require.NoError(t, err)
		assert.True(t, stored)
		assert.Equal(t, "pmm-agent-id", storer.pmmAgentID)
		assert.Equal(t, "secret/data/pmm/mysql", storer.path)
		assert.Equal(t, map[string]string{"username": "pmm", "password": "s3cr3t"}, storer.data)
	})

	t.Run("StoreFails", func(t *testing.T) {
		t.Parallel()

		storer := &fakeStorer{err: errors.New("vault is down")}
		stored, err := StoreCredentialsInVault(t.Context(), nil, storer, &fakeChecker{}, agent, "", "s3cr3t", true)
		require.EqualError(t, err, "vault is down")
		assert.False(t, stored)
	})
}

func TestWithCheckCredentials(t *testing.T) {
	t.Parallel()

	agent := &models.Agent{AgentType: models.MySQLdExporterType, Username: new("fallback"), VaultPath: new("secret/data/pmm/mysql")}

	// The new password is checked in plaintext, while the username stays the one in Vault.
	username, password := agent.WithCheckCredentials(nil, new("new")).DBCredentials(vault.EncodingRaw)
	assert.Equal(t, "new", password)
	assert.Equal(t, vault.TokenWithFallback("secret/data/pmm/mysql", "username", vault.EncodingRaw, "fallback"), username)

	username, password = agent.WithCheckCredentials(new("pmm"), nil).DBCredentials(vault.EncodingUserinfo)
	assert.Equal(t, "pmm", username)
	assert.Equal(t, vault.Token("secret/data/pmm/mysql", "password", vault.EncodingUserinfo), password)

	// The Agent itself is not changed.
	username, password = agent.DBCredentials(vault.EncodingRaw)
	assert.True(t, vault.HasTokens(username))
	assert.True(t, vault.HasTokens(password))
}
