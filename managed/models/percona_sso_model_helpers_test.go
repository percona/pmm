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
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setupDB(t *testing.T) (*reform.DB, func()) {
	t.Helper()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	cleanup := func() {
		require.NoError(t, sqlDB.Close())
	}
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	return db, cleanup
}

func TestPerconaSSODetails(t *testing.T) {
	ctx := context.Background()
	issuerURL := "https://id-dev.percona.com/oauth2/aus15pi5rjdtfrcH51d7/v1"
	wrongIssuerURL := "https://id-dev.percona.com/wrong"
	orgID := uuid.NewString()

	t.Run("CorrectCredentials", func(t *testing.T) {
		clientID, clientSecret := os.Getenv("PMM_DEV_OAUTH_CLIENT_ID"), os.Getenv("PMM_DEV_OAUTH_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			t.Skip("Environment variables PMM_DEV_OAUTH_CLIENT_ID / PMM_DEV_OAUTH_CLIENT_SECRET are not defined, skipping test")
		}

		db, cleanup := setupDB(t)
		defer cleanup()

		expectedSSODetails := &models.PerconaSSODetails{
			IssuerURL:              issuerURL,
			PMMManagedClientID:     clientID,
			PMMManagedClientSecret: clientSecret,
			Scope:                  "percona",
			OrganizationID:         orgID,
		}
		insertSSODetails := &models.PerconaSSODetailsInsert{
			IssuerURL:              expectedSSODetails.IssuerURL,
			PMMManagedClientID:     expectedSSODetails.PMMManagedClientID,
			PMMManagedClientSecret: expectedSSODetails.PMMManagedClientSecret,
			Scope:                  expectedSSODetails.Scope,
			OrganizationID:         expectedSSODetails.OrganizationID,
		}
		err := models.InsertPerconaSSODetails(db.Querier, insertSSODetails)
		require.NoError(t, err)
		ssoDetails, err := models.GetPerconaSSODetails(ctx, db.Querier)
		require.NoError(t, err)

		assert.NotNil(t, ssoDetails)
		assert.Equal(t, expectedSSODetails.PMMManagedClientID, ssoDetails.PMMManagedClientID)
		assert.Equal(t, expectedSSODetails.PMMManagedClientSecret, ssoDetails.PMMManagedClientSecret)
		assert.Equal(t, expectedSSODetails.IssuerURL, ssoDetails.IssuerURL)
		assert.Equal(t, expectedSSODetails.Scope, ssoDetails.Scope)
		assert.Equal(t, expectedSSODetails.OrganizationID, ssoDetails.OrganizationID)

		assert.NotNil(t, ssoDetails.AccessToken)
		assert.NotNil(t, ssoDetails.AccessToken.AccessToken)
		assert.NotNil(t, ssoDetails.AccessToken.ExpiresAt)
		assert.NotNil(t, ssoDetails.AccessToken.ExpiresIn)
		assert.NotNil(t, ssoDetails.AccessToken.Scope)
		assert.NotNil(t, ssoDetails.AccessToken.TokenType)

		err = models.DeletePerconaSSODetails(db.Querier)
		require.NoError(t, err)
		ssoDetails, err = models.GetPerconaSSODetails(ctx, db.Querier)
		assert.Error(t, err)
		assert.Nil(t, ssoDetails)
		// See https://github.com/percona/pmm/managed/pull/852#discussion_r738178192
		ssoDetails, err = models.GetPerconaSSODetails(ctx, db.Querier)
		assert.Error(t, err)
		assert.Nil(t, ssoDetails)
	})

	t.Run("WrongCredentials", func(t *testing.T) {
		db, cleanup := setupDB(t)
		defer cleanup()

		InsertSSODetails := &models.PerconaSSODetailsInsert{
			IssuerURL:              issuerURL,
			PMMManagedClientID:     "wrongClientID",
			PMMManagedClientSecret: "wrongClientSecret",
			Scope:                  "percona",
			OrganizationID:         "org-id",
		}
		err := models.InsertPerconaSSODetails(db.Querier, InsertSSODetails)
		require.NoError(t, err)
		_, err = models.GetPerconaSSODetails(ctx, db.Querier)
		require.Error(t, err)
	})

	t.Run("WrongURL", func(t *testing.T) {
		clientID, clientSecret := os.Getenv("PMM_DEV_OAUTH_CLIENT_ID"), os.Getenv("PMM_DEV_OAUTH_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			t.Skip("Environment variables PMM_DEV_OAUTH_CLIENT_ID / PMM_DEV_OAUTH_CLIENT_SECRET are not defined, skipping test")
		}

		db, cleanup := setupDB(t)
		defer cleanup()

		InsertSSODetails := &models.PerconaSSODetailsInsert{
			IssuerURL:              wrongIssuerURL,
			PMMManagedClientID:     clientID,
			PMMManagedClientSecret: clientSecret,
			Scope:                  "percona",
		}
		err := models.InsertPerconaSSODetails(db.Querier, InsertSSODetails)
		require.NoError(t, err)
		_, err = models.GetPerconaSSODetails(ctx, db.Querier)
		require.Error(t, err)
	})
}
