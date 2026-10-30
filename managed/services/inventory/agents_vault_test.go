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

package inventory

import (
	"errors"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/vault"
)

func TestVaultChange(t *testing.T) {
	t.Parallel()

	const ref = "vault:secret/data/pmm/mysql#tls_key"
	vaultBacked := &models.Agent{VaultPath: new("secret/data/pmm/mysql")}
	stored := &models.Agent{Password: new("old")}

	for name, tc := range map[string]struct {
		current       *models.Agent
		params        models.ChangeAgentParams
		secrets       map[string]string
		keepsPassword bool
		tlsKey        *string
	}{
		"PasswordOfVaultBackedAgentGoesToVault": {
			current: vaultBacked, params: models.ChangeAgentParams{Password: new("new")}, secrets: map[string]string{"password": "new"},
		},
		"UsernameGoesToVaultToo": {
			current: vaultBacked, params: models.ChangeAgentParams{Username: new("pmm"), Password: new("new")},
			secrets: map[string]string{"username": "pmm", "password": "new"},
		},
		"UsernameAlone": {
			current: vaultBacked, params: models.ChangeAgentParams{Username: new("pmm")}, secrets: map[string]string{"username": "pmm"},
		},
		"TLSKeyGoesToVault": {
			current: vaultBacked, params: models.ChangeAgentParams{MySQLOptions: &models.ChangeMySQLOptions{TLSKey: new("KEY")}},
			secrets: map[string]string{"tls_key": "KEY"}, tlsKey: new(ref),
		},
		"TLSReferenceIsKept": {
			current: vaultBacked, params: models.ChangeAgentParams{MySQLOptions: &models.ChangeMySQLOptions{TLSKey: new("vault:secret/data/x#k")}},
			tlsKey: new("vault:secret/data/x#k"),
		},
		"MovedToVaultWithItsPassword": {
			current: stored, params: models.ChangeAgentParams{VaultPath: new("secret/data/pmm/mysql"), Password: new("new")},
			secrets: map[string]string{"password": "new"},
		},
		"TakenOffVault": {
			current: vaultBacked, params: models.ChangeAgentParams{VaultPath: new(""), Password: new("new")}, keepsPassword: true,
		},
		"NotInVault": {
			current: stored, params: models.ChangeAgentParams{Password: new("new"), MySQLOptions: &models.ChangeMySQLOptions{TLSKey: new("KEY")}},
			keepsPassword: true, tlsKey: new("KEY"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var givenKey string
			if tc.params.MySQLOptions != nil {
				givenKey = *tc.params.MySQLOptions.TLSKey
			}

			params, secrets := vaultChange(tc.current, &tc.params)
			assert.Equal(t, tc.secrets, secrets)
			assert.Equal(t, tc.keepsPassword, params.Password != nil, "the password is applied to the row")
			if tc.tlsKey != nil {
				assert.Equal(t, *tc.tlsKey, *params.MySQLOptions.TLSKey)
				assert.Equal(t, givenKey, *tc.params.MySQLOptions.TLSKey, "the caller's options are kept")
			}
		})
	}
}

func TestAgentsCredentialsInVault(t *testing.T) {
	t.Parallel()

	const path = "secret/data/pmm/mysql"
	// checkedWith matches an Agent that connects with the given password, which is how new credentials
	// are checked before they are stored in Vault.
	checkedWith := func(password string) any {
		return mock.MatchedBy(func(a *models.Agent) bool {
			_, p := a.DBCredentials(vault.EncodingRaw)
			return p == password
		})
	}

	_, as, _, teardown, ctx, _ := setup(t) //nolint:dogsled
	t.Cleanup(func() { teardown(t) })
	r := as.r.(*mockAgentsRegistry)
	cc := as.cc.(*mockConnectionChecker)
	state := as.state.(*mockAgentsStateUpdater)

	pmmAgent, err := models.CreatePMMAgent(as.db.Querier, models.PMMServerNodeID, nil)
	require.NoError(t, err)
	pmmAgent.Version = new("3.11.0")
	require.NoError(t, models.UpdateAgent(as.db.Querier, pmmAgent))
	service, err := models.AddNewService(as.db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "vault-mysql",
		NodeID:      models.PMMServerNodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(3306)),
	})
	require.NoError(t, err)

	state.On("RequestStateUpdate", ctx, pmmAgent.AgentID)

	// Added with a password: it goes to Vault, and only the path to PMM Server.
	r.On("StoreSecret", mock.Anything, mock.Anything, pmmAgent.AgentID, path,
		map[string]string{"username": "pmm", "password": "s3cr3t"}).Return(nil).Once()
	added, err := as.AddMySQLdExporter(ctx, &inventoryv1.AddMySQLdExporterParams{
		PmmAgentId: pmmAgent.AgentID, ServiceId: service.ServiceID, Username: "pmm", Password: "s3cr3t",
		VaultPath: path, SkipConnectionCheck: true,
	})
	require.NoError(t, err)
	agentID := added.GetMysqldExporter().AgentId
	assert.Equal(t, path, added.GetMysqldExporter().VaultPath)

	row, err := models.FindAgentByID(as.db.Querier, agentID)
	require.NoError(t, err)
	assert.Nil(t, row.Password)
	assert.Equal(t, "pmm", pointer.GetString(row.Username))

	// A new password is checked with the plaintext, then stored in Vault; the Agent keeps its path.
	cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, checkedWith("rotated")).Return(nil).Once()
	r.On("StoreSecret", mock.Anything, mock.Anything, pmmAgent.AgentID, path,
		map[string]string{"password": "rotated"}).Return(nil).Once()
	changed, err := as.ChangeMySQLdExporter(ctx, agentID, &inventoryv1.ChangeMySQLdExporterParams{Password: new("rotated")})
	require.NoError(t, err)
	assert.Equal(t, path, changed.GetMysqldExporter().VaultPath)

	row, err = models.FindAgentByID(as.db.Querier, agentID)
	require.NoError(t, err)
	assert.Nil(t, row.Password)
	assert.Equal(t, path, pointer.GetString(row.VaultPath))

	// A password the database refuses never reaches Vault.
	cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, checkedWith("wrong")).
		Return(errors.New("access denied")).Once()
	_, err = as.ChangeMySQLdExporter(ctx, agentID, &inventoryv1.ChangeMySQLdExporterParams{Password: new("wrong")})
	require.EqualError(t, err, "access denied")

	// An empty Vault path takes the Agent off Vault, with the password stored in PMM Server again.
	cc.On("CheckConnectionToService", mock.Anything, mock.Anything, mock.Anything, checkedWith("plain")).Return(nil).Once()
	changed, err = as.ChangeMySQLdExporter(ctx, agentID, &inventoryv1.ChangeMySQLdExporterParams{
		Password: new("plain"), VaultPath: new(""),
	})
	require.NoError(t, err)
	assert.Empty(t, changed.GetMysqldExporter().VaultPath)

	row, err = models.FindAgentByID(as.db.Querier, agentID)
	require.NoError(t, err)
	assert.Equal(t, "plain", pointer.GetString(row.Password))
	assert.Nil(t, row.VaultPath)

	r.AssertNumberOfCalls(t, "StoreSecret", 2)
}
