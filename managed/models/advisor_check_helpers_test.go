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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

// builtinCheck returns a minimal valid built-in advisor check model.
func builtinCheck(name string) *models.AdvisorCheck {
	return &models.AdvisorCheck{
		Name:        name,
		Source:      models.BuiltinCheckSource,
		Version:     2,
		Summary:     "Test summary",
		Description: "Test description",
		Category:    "Test",
		Subcategory: "Helpers",
		Technology:  "POSTGRESQL",
		Interval:    "standard",
		Queries:     []byte(`[{"type":"POSTGRESQL_SELECT","query":"1"}]`),
		Script:      "def check(): return []",
	}
}

func TestAdvisorCheckHelpers(t *testing.T) { //nolint:tparallel
	t.Parallel()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	tx := func(t *testing.T) *reform.Querier {
		t.Helper()
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})
		return tx.Querier
	}

	t.Run("upsert inserts and refreshes content", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		err := models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("check1"))
		require.NoError(t, err)

		c, err := models.FindAdvisorCheckByName(q, "check1")
		require.NoError(t, err)
		assert.Equal(t, models.BuiltinCheckSource, c.Source)
		assert.Equal(t, "Test summary", c.Summary)
		assert.False(t, c.Disabled)
		assert.Nil(t, c.IntervalOverride)

		updated := builtinCheck("check1")
		updated.Summary = "New summary"
		updated.Interval = "rare"
		err = models.UpsertAdvisorCheckContent(ctx, q, updated)
		require.NoError(t, err)

		c, err = models.FindAdvisorCheckByName(q, "check1")
		require.NoError(t, err)
		assert.Equal(t, "New summary", c.Summary)
		assert.Equal(t, "rare", c.Interval)
	})

	t.Run("upsert preserves settings columns", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		err := models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("check1"))
		require.NoError(t, err)

		// record user overrides
		_, err = models.ChangeAdvisorCheckInterval(ctx, q, "check1", models.Rare)
		require.NoError(t, err)
		err = models.SetAdvisorChecksDisabled(ctx, q, []string{"check1"}, true)
		require.NoError(t, err)
		_, err = models.ChangeAdvisorCheckDisabledServices(ctx, q, "check1", []string{"svc-1"})
		require.NoError(t, err)

		// a content refresh must not touch them
		err = models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("check1"))
		require.NoError(t, err)

		c, err := models.FindAdvisorCheckByName(q, "check1")
		require.NoError(t, err)
		assert.Equal(t, new("rare"), c.IntervalOverride)
		assert.True(t, c.Disabled)
		ids, err := c.GetDisabledServiceIDs()
		require.NoError(t, err)
		assert.Equal(t, []string{"svc-1"}, ids)
	})

	t.Run("prune removes only vanished built-in checks", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		err := models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("builtin1"))
		require.NoError(t, err)
		err = models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("builtin2"))
		require.NoError(t, err)

		user := builtinCheck("user1")
		user.Source = models.UserCheckSource
		_, err = models.CreateAdvisorCheck(q, user)
		require.NoError(t, err)

		err = models.RemoveAdvisorChecksNotIn(ctx, q, []string{"builtin1"})
		require.NoError(t, err)

		checks, err := models.FindAdvisorChecks(q)
		require.NoError(t, err)
		names := make([]string, 0, len(checks))
		for _, c := range checks {
			names = append(names, c.Name)
		}
		assert.ElementsMatch(t, []string{"builtin1", "user1"}, names)
	})

	t.Run("prune with empty list is a no-op", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		err := models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("builtin1"))
		require.NoError(t, err)

		err = models.RemoveAdvisorChecksNotIn(ctx, q, nil)
		require.NoError(t, err)

		_, err = models.FindAdvisorCheckByName(q, "builtin1")
		require.NoError(t, err)
	})

	t.Run("disabled names round-trip", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		err := models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("check1"))
		require.NoError(t, err)
		err = models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("check2"))
		require.NoError(t, err)

		err = models.SetAdvisorChecksDisabled(ctx, q, []string{"check1"}, true)
		require.NoError(t, err)

		names, err := models.FindDisabledAdvisorCheckNames(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, []string{"check1"}, names)

		err = models.SetAdvisorChecksDisabled(ctx, q, []string{"check1"}, false)
		require.NoError(t, err)

		names, err = models.FindDisabledAdvisorCheckNames(ctx, q)
		require.NoError(t, err)
		assert.Empty(t, names)
	})

	t.Run("disabled services round-trip", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		err := models.UpsertAdvisorCheckContent(ctx, q, builtinCheck("check1"))
		require.NoError(t, err)

		_, err = models.ChangeAdvisorCheckDisabledServices(ctx, q, "check1", []string{"svc-1", "svc-2"})
		require.NoError(t, err)

		m, err := models.FindAdvisorCheckDisabledServices(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, map[string][]string{"check1": {"svc-1", "svc-2"}}, m)

		// clearing the list removes the check from the map
		_, err = models.ChangeAdvisorCheckDisabledServices(ctx, q, "check1", nil)
		require.NoError(t, err)

		m, err = models.FindAdvisorCheckDisabledServices(ctx, q)
		require.NoError(t, err)
		assert.Empty(t, m)
	})

	t.Run("update preserves settings columns", func(t *testing.T) {
		q := tx(t)
		ctx := t.Context()

		user := builtinCheck("user1")
		user.Source = models.UserCheckSource
		_, err := models.CreateAdvisorCheck(q, user)
		require.NoError(t, err)

		_, err = models.ChangeAdvisorCheckInterval(ctx, q, "user1", models.Frequent)
		require.NoError(t, err)
		err = models.SetAdvisorChecksDisabled(ctx, q, []string{"user1"}, true)
		require.NoError(t, err)

		edited := builtinCheck("user1")
		edited.Source = models.UserCheckSource
		edited.Summary = "Edited summary"
		_, err = models.UpdateAdvisorCheck(q, edited)
		require.NoError(t, err)

		c, err := models.FindAdvisorCheckByName(q, "user1")
		require.NoError(t, err)
		assert.Equal(t, "Edited summary", c.Summary)
		assert.Equal(t, models.UserCheckSource, c.Source)
		assert.Equal(t, new("frequent"), c.IntervalOverride)
		assert.True(t, c.Disabled)
	})
}
