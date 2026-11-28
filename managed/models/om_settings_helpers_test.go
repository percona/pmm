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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

func intValidate(v json.RawMessage) error {
	var n int
	return json.Unmarshal(v, &n)
}

func settingsRegistry(t *testing.T) *models.OmSettingsRegistry {
	t.Helper()
	reg := models.NewOmSettingsRegistry()
	require.NoError(t, reg.Register(models.OmSettingDef{Key: "retention_days", Default: json.RawMessage(`7`), Scopes: []models.OmSettingScope{models.OmSettingGlobal, models.OmSettingCluster}, Validate: intValidate}))
	require.NoError(t, reg.Register(models.OmSettingDef{Key: "telemetry", Default: json.RawMessage(`true`), Scopes: []models.OmSettingScope{models.OmSettingGlobal}}))
	require.NoError(t, reg.Register(models.OmSettingDef{Key: "api_token", Scopes: []models.OmSettingScope{models.OmSettingGlobal, models.OmSettingCluster}, Secret: true}))
	return reg
}

func effective(settings []models.OmEffectiveSetting, key string) models.OmEffectiveSetting {
	for _, s := range settings {
		if s.Key == key {
			return s
		}
	}
	return models.OmEffectiveSetting{}
}

func TestOmSettingsResolution(t *testing.T) {
	db := omTestDB(t)
	reg := settingsRegistry(t)
	clusterID := newCluster(t, db.Querier)

	g, err := models.GetOmSettings(db.Querier, reg, nil)
	require.NoError(t, err)
	ret := effective(g, "retention_days")
	assert.Equal(t, models.OmSettingSourceDefault, ret.Source)
	assert.JSONEq(t, `7`, string(ret.Value))
	assert.False(t, ret.HasOverride)

	require.NoError(t, models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{"retention_days": json.RawMessage(`30`)}))
	g, err = models.GetOmSettings(db.Querier, reg, nil)
	require.NoError(t, err)
	ret = effective(g, "retention_days")
	assert.Equal(t, models.OmSettingSourceGlobal, ret.Source)
	assert.JSONEq(t, `30`, string(ret.Value))
	assert.True(t, ret.HasOverride)

	require.NoError(t, models.UpdateOmSettings(t.Context(), db, reg, &clusterID, nil, "admin", map[string]json.RawMessage{"retention_days": json.RawMessage(`90`)}))
	c, err := models.GetOmSettings(db.Querier, reg, &clusterID)
	require.NoError(t, err)
	ret = effective(c, "retention_days")
	assert.Equal(t, models.OmSettingSourceCluster, ret.Source)
	assert.JSONEq(t, `90`, string(ret.Value))

	t.Run("cluster inherits the global value when it has no override", func(t *testing.T) {
		other := newCluster(t, db.Querier)
		c, err := models.GetOmSettings(db.Querier, reg, &other)
		require.NoError(t, err)
		assert.Equal(t, models.OmSettingSourceGlobal, effective(c, "retention_days").Source)
	})
}

func TestOmSettingsValidation(t *testing.T) {
	db := omTestDB(t)
	reg := settingsRegistry(t)
	clusterID := newCluster(t, db.Querier)

	t.Run("unknown key is refused", func(t *testing.T) {
		err := models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{"nope": json.RawMessage(`1`)})
		require.Error(t, err)
	})
	t.Run("invalid value is refused", func(t *testing.T) {
		err := models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{"retention_days": json.RawMessage(`"not-an-int"`)})
		require.Error(t, err)
	})
	t.Run("a global-only key rejects a cluster override", func(t *testing.T) {
		err := models.UpdateOmSettings(t.Context(), db, reg, &clusterID, nil, "admin", map[string]json.RawMessage{"telemetry": json.RawMessage(`false`)})
		require.Error(t, err)
	})
	t.Run("empty batch is refused", func(t *testing.T) {
		err := models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{})
		require.Error(t, err)
	})
}

func TestOmSettingsSecret(t *testing.T) {
	db := omTestDB(t)
	reg := settingsRegistry(t)

	const token = "tok-s3cret-xyz"
	require.NoError(t, models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{"api_token": json.RawMessage(`"` + token + `"`)}))

	g, err := models.GetOmSettings(db.Querier, reg, nil)
	require.NoError(t, err)
	tok := effective(g, "api_token")
	assert.True(t, tok.Secret)
	assert.True(t, tok.IsSet)
	assert.Equal(t, models.OmSettingSourceGlobal, tok.Source)
	assert.Nil(t, tok.Value)

	var payload, settingValue string
	require.NoError(t, db.QueryRow("SELECT payload FROM om_secrets WHERE purpose = 'setting' AND discriminator = 'api_token' AND generation = 'current'").Scan(&payload))
	assert.NotContains(t, payload, token)
	require.NoError(t, db.QueryRow("SELECT value::text FROM om_settings WHERE key = 'api_token' AND cluster_id IS NULL").Scan(&settingValue))
	assert.Contains(t, settingValue, "secret_id")
	assert.NotContains(t, settingValue, token)

	t.Run("a second write retires the old secret and keeps one current", func(t *testing.T) {
		require.NoError(t, models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{"api_token": json.RawMessage(`"tok-rotated"`)}))
		var current int
		require.NoError(t, db.QueryRow("SELECT count(*) FROM om_secrets WHERE purpose = 'setting' AND discriminator = 'api_token' AND generation = 'current' AND cluster_id IS NULL").Scan(&current))
		assert.Equal(t, 1, current)
	})
}

func TestOmSettingsChangeAudit(t *testing.T) {
	db := omTestDB(t)
	reg := settingsRegistry(t)

	require.NoError(t, models.UpdateOmSettings(t.Context(), db, reg, nil, nil, "admin", map[string]json.RawMessage{"api_token": json.RawMessage(`"tok-audit"`)}))

	ops, err := db.SelectAllFrom(models.OmOperationTable, "WHERE kind = $1", "settings_change")
	require.NoError(t, err)
	require.Len(t, ops, 1)
	op := ops[0].(*models.OmOperation)
	assert.Equal(t, models.OmOperationSucceeded, op.Status)
	assert.NotContains(t, string(op.Params), "tok-audit")
	assert.Contains(t, string(op.Params), "update")

	t.Run("deleting an absent override records nothing", func(t *testing.T) {
		before := countOps(t, db.Querier)
		removed, err := models.DeleteOmSettingOverride(t.Context(), db, reg, nil, "admin", nil, "retention_days")
		require.NoError(t, err)
		assert.False(t, removed)
		assert.Equal(t, before, countOps(t, db.Querier))
	})

	t.Run("deleting a present override records one change", func(t *testing.T) {
		before := countOps(t, db.Querier)
		removed, err := models.DeleteOmSettingOverride(t.Context(), db, reg, nil, "admin", nil, "api_token")
		require.NoError(t, err)
		assert.True(t, removed)
		assert.Equal(t, before+1, countOps(t, db.Querier))
	})
}

func countOps(t *testing.T, q *reform.Querier) int {
	t.Helper()
	var n int
	require.NoError(t, q.QueryRow("SELECT count(*) FROM om_operations WHERE kind = 'settings_change'").Scan(&n))
	return n
}
