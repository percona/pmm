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

package om

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/testdb"
)

func settingsService(t *testing.T) *Service {
	t.Helper()
	t.Setenv(encryption.CustomEncryptionKeyPathEnvVar, filepath.Join(t.TempDir(), "encryption.key"))
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	reg := models.NewOmSettingsRegistry()
	require.NoError(t, reg.Register(models.OmSettingDef{
		Key: "retention_days", Default: json.RawMessage(`7`),
		Scopes:   []models.OmSettingScope{models.OmSettingGlobal, models.OmSettingCluster},
		Validate: func(v json.RawMessage) error { var n int; return json.Unmarshal(v, &n) },
	}))
	return &Service{db: db, l: logrus.WithField("test", t.Name()), settingsRegistry: reg}
}

func findSetting(settings []*omv1.OmSetting, key string) *omv1.OmSetting {
	for _, s := range settings {
		if s.GetKey() == key {
			return s
		}
	}
	return nil
}

func TestSettingsHandler(t *testing.T) {
	svc := settingsService(t)
	ctx := t.Context()

	got, err := svc.GetSettings(ctx, &omv1.GetSettingsRequest{})
	require.NoError(t, err)
	ret := findSetting(got.GetSettings(), "retention_days")
	require.NotNil(t, ret)
	assert.Equal(t, omv1.OmSettingSource_OM_SETTING_SOURCE_DEFAULT, ret.GetSource())
	assert.InEpsilon(t, 7.0, ret.GetValue().GetNumberValue(), 0)

	values, err := structpb.NewStruct(map[string]any{"retention_days": 30})
	require.NoError(t, err)
	updated, err := svc.UpdateSettings(ctx, &omv1.UpdateSettingsRequest{Values: values})
	require.NoError(t, err)
	ret = findSetting(updated.GetSettings(), "retention_days")
	require.NotNil(t, ret)
	assert.Equal(t, omv1.OmSettingSource_OM_SETTING_SOURCE_GLOBAL, ret.GetSource())
	assert.True(t, ret.GetHasOverride())
	assert.InEpsilon(t, 30.0, ret.GetValue().GetNumberValue(), 0)

	t.Run("unknown key is rejected", func(t *testing.T) {
		bad, err := structpb.NewStruct(map[string]any{"nope": 1})
		require.NoError(t, err)
		_, err = svc.UpdateSettings(ctx, &omv1.UpdateSettingsRequest{Values: bad})
		require.Error(t, err)
	})

	t.Run("deleting the override reverts to default", func(t *testing.T) {
		resp, err := svc.DeleteSettingOverride(ctx, &omv1.DeleteSettingOverrideRequest{Key: "retention_days"})
		require.NoError(t, err)
		assert.True(t, resp.GetRemoved())
		got, err := svc.GetSettings(ctx, &omv1.GetSettingsRequest{})
		require.NoError(t, err)
		assert.Equal(t, omv1.OmSettingSource_OM_SETTING_SOURCE_DEFAULT, findSetting(got.GetSettings(), "retention_days").GetSource())
	})
}
