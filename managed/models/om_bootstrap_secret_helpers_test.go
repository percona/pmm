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

package models_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

func newTestOmBootstrapSecret(runID string) *models.OmBootstrapSecret {
	return &models.OmBootstrapSecret{
		RunID:           runID,
		MongoDBUsername: "pmm",
		MongoDBPassword: "monitoring-secret",
		KeyFile:         "keyfile-content",
		AdminUsername:   "dba",
		AdminPassword:   "admin-secret",
	}
}

func TestOmBootstrapSecret(t *testing.T) {
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "encryption.key"))

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("stores every credential encrypted and reads it back decrypted", func(t *testing.T) {
		created := newTestOmBootstrapSecret("run-encrypted")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, created))
		assert.Equal(t, "admin-secret", created.AdminPassword)

		raw := make([]string, 5)
		row := sqlDB.QueryRowContext(t.Context(),
			`SELECT mongodb_username, mongodb_password, key_file, admin_username, admin_password
			   FROM om_bootstrap_secrets WHERE run_id = $1`, "run-encrypted")
		require.NoError(t, row.Scan(&raw[0], &raw[1], &raw[2], &raw[3], &raw[4]))
		for i, plain := range []string{"pmm", "monitoring-secret", "keyfile-content", "dba", "admin-secret"} {
			assert.NotEqual(t, plain, raw[i], "column %d is stored in plaintext", i)
		}

		found, err := models.FindOmBootstrapSecretByRunID(db.Querier, "run-encrypted")
		require.NoError(t, err)
		want := newTestOmBootstrapSecret("run-encrypted")
		want.CreatedAt = found.CreatedAt
		assert.Equal(t, want, found)
	})

	t.Run("clears only the administrator password, and only once", func(t *testing.T) {
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, newTestOmBootstrapSecret("run-clear")))

		require.NoError(t, models.ClearOmBootstrapAdminPassword(db.Querier, "run-clear"))
		require.NoError(t, models.ClearOmBootstrapAdminPassword(db.Querier, "run-clear"))

		found, err := models.FindOmBootstrapSecretByRunID(db.Querier, "run-clear")
		require.NoError(t, err)
		want := newTestOmBootstrapSecret("run-clear")
		want.AdminPassword = ""
		want.CreatedAt = found.CreatedAt
		assert.Equal(t, want, found)
	})

	t.Run("reports a run with no secret as not found", func(t *testing.T) {
		_, err := models.FindOmBootstrapSecretByRunID(db.Querier, "run-missing")
		require.ErrorIs(t, err, models.ErrNotFound)

		_, err = models.FindOmBootstrapSecretByRunID(db.Querier, "")
		require.Error(t, err)
	})
}
