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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestEnrollmentTokens(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	t.Run("an expiry is always set, defaulting to a short life", func(t *testing.T) {
		row, _, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "no expiry given",
		})
		require.NoError(t, err)
		require.NotNil(t, row.ExpiresAt, "a token must never be issued without an expiry")
		assert.WithinDuration(t, models.Now().Add(models.DefaultEnrollmentTokenTTL), *row.ExpiresAt, time.Minute)
		assert.True(t, row.Usable())

		// An explicit expiry still wins, so a long rollout can ask for one.
		longer := models.Now().Add(8 * time.Hour)
		row, _, err = models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "long rollout", ExpiresAt: &longer,
		})
		require.NoError(t, err)
		require.NotNil(t, row.ExpiresAt)
		assert.WithinDuration(t, longer, *row.ExpiresAt, time.Second)
	})

	t.Run("a token is identifiable and stored only as a hash", func(t *testing.T) {
		row, token, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "ops team",
		})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(token, models.EnrollmentTokenPrefix))
		assert.NotContains(t, row.TokenHash, token)
		assert.Equal(t, "ops team", row.Description)
		assert.True(t, row.Usable())

		found, err := models.FindEnrollmentToken(db.Querier, token)
		require.NoError(t, err)
		assert.Equal(t, row.TokenHash, found.TokenHash)
	})

	t.Run("an agent token is not an enrollment token", func(t *testing.T) {
		_, err := models.FindEnrollmentToken(db.Querier, models.AgentTokenPrefix+"something")
		assert.ErrorIs(t, err, models.ErrInvalidEnrollmentToken)
	})

	t.Run("unknown tokens are refused", func(t *testing.T) {
		_, err := models.FindEnrollmentToken(db.Querier, models.EnrollmentTokenPrefix+"nope")
		assert.ErrorIs(t, err, models.ErrInvalidEnrollmentToken)
	})

	t.Run("uses are counted and run out", func(t *testing.T) {
		_, token, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "two nodes", MaxUses: 2,
		})
		require.NoError(t, err)

		require.NoError(t, models.UseEnrollmentToken(db.Querier, token))
		require.NoError(t, models.UseEnrollmentToken(db.Querier, token))

		err = models.UseEnrollmentToken(db.Querier, token)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		row, err := models.FindEnrollmentToken(db.Querier, token)
		require.NoError(t, err)
		assert.Equal(t, 2, row.UsedCount)
		assert.True(t, row.Exhausted())
		assert.False(t, row.Usable())
	})

	t.Run("zero max uses means unlimited", func(t *testing.T) {
		_, token, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "unlimited",
		})
		require.NoError(t, err)

		for range 5 {
			require.NoError(t, models.UseEnrollmentToken(db.Querier, token))
		}

		row, err := models.FindEnrollmentToken(db.Querier, token)
		require.NoError(t, err)
		assert.True(t, row.Usable())
	})

	t.Run("an expired token is refused", func(t *testing.T) {
		// Create with a valid expiry, then move it into the past directly, since minting
		// deliberately rejects an expiry that has already passed.
		expiresAt := models.Now().Add(time.Hour)
		row, token, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "expiring", ExpiresAt: &expiresAt,
		})
		require.NoError(t, err)
		assert.True(t, row.Usable())

		past := models.Now().Add(-time.Minute)
		row.ExpiresAt = &past
		require.NoError(t, db.Update(row))

		err = models.UseEnrollmentToken(db.Querier, token)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("minting rejects nonsense", func(t *testing.T) {
		past := models.Now().Add(-time.Hour)
		for name, params := range map[string]*models.CreateEnrollmentTokenParams{
			"expiry in the past": {Description: "d", ExpiresAt: &past},
			"negative uses":      {Description: "d", MaxUses: -1},
		} {
			t.Run(name, func(t *testing.T) {
				_, _, err := models.CreateEnrollmentToken(db.Querier, params)
				require.Error(t, err)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
			})
		}
	})

	t.Run("revoking makes a token unusable", func(t *testing.T) {
		row, token, err := models.CreateEnrollmentToken(db.Querier, &models.CreateEnrollmentTokenParams{
			Description: "to be revoked",
		})
		require.NoError(t, err)

		require.NoError(t, models.RemoveEnrollmentToken(db.Querier, row.TokenHash))

		_, err = models.FindEnrollmentToken(db.Querier, token)
		require.ErrorIs(t, err, models.ErrInvalidEnrollmentToken)

		err = models.RemoveEnrollmentToken(db.Querier, row.TokenHash)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("listing returns them without any token value", func(t *testing.T) {
		tokens, err := models.FindEnrollmentTokens(db.Querier)
		require.NoError(t, err)
		require.NotEmpty(t, tokens)
		for _, row := range tokens {
			assert.NotEmpty(t, row.TokenHash)
			assert.False(t, strings.HasPrefix(row.TokenHash, models.EnrollmentTokenPrefix),
				"a stored row must never hold a usable token")
		}
	})
}
