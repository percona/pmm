// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

// putDeploymentConfig saves config.yaml through the handler and returns the recorder.
func putDeploymentConfig(t *testing.T, h *Handlers, configYAML string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"config_yaml": configYAML})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPut, "/v1/adre/deployment/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.PutDeploymentConfig(rec, req)
	return rec
}

// TestHandlers_PutDeploymentConfig_HotReload verifies that saving a reload-able config renders to
// disk and hot-reloads Holmes: a successful reload clears restart_required, while an unavailable
// reload (404 — admin API off / older Holmes) keeps restart_required set as a fallback.
func TestHandlers_PutDeploymentConfig_HotReload(t *testing.T) {
	t.Run("reload success clears restart_required", func(t *testing.T) {
		var reloadCalled bool
		holmes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/admin/reload" && r.Method == http.MethodPost {
				reloadCalled = true
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"ok","component":"all","counts":{"toolsets_total":1,"models_loaded":0}}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer holmes.Close()

		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		defer func() { require.NoError(t, sqlDB.Close()) }()
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		enable, url := true, holmes.URL
		_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAdre: &enable, AdreURL: &url})
		require.NoError(t, err)

		// Seed restart_required=true so a successful reload provably clears it.
		prov, err := models.GetAdreProvisioning(db)
		require.NoError(t, err)
		prov.RestartRequired = true
		require.NoError(t, models.SaveAdreProvisioning(db, prov))

		t.Setenv("ADRE_CONFIG_DIR", t.TempDir())

		h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
		rec := putDeploymentConfig(t, h, "model: gpt-5.4\n")

		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, reloadCalled, "Holmes /api/admin/reload should be called")

		var resp struct {
			Saved  bool `json:"saved"`
			Reload struct {
				Reloaded bool `json:"reloaded"`
			} `json:"reload"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.True(t, resp.Saved)
		assert.True(t, resp.Reload.Reloaded)

		prov, err = models.GetAdreProvisioning(db)
		require.NoError(t, err)
		assert.False(t, prov.RestartRequired, "successful reload should clear restart_required")
	})

	t.Run("reload unavailable keeps restart_required", func(t *testing.T) {
		// Holmes without the admin API: every path 404s, mapping to ErrReloadUnsupported.
		holmes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer holmes.Close()

		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		defer func() { require.NoError(t, sqlDB.Close()) }()
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		enable, url := true, holmes.URL
		_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAdre: &enable, AdreURL: &url})
		require.NoError(t, err)

		t.Setenv("ADRE_CONFIG_DIR", t.TempDir())

		// Start from restart_required=false so the assertion proves the failed reload SET it.
		seed, err := models.GetAdreProvisioning(db)
		require.NoError(t, err)
		seed.RestartRequired = false
		require.NoError(t, models.SaveAdreProvisioning(db, seed))

		h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
		rec := putDeploymentConfig(t, h, "model: gpt-5.4\n")

		require.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Reload struct {
				Reloaded bool   `json:"reloaded"`
				Error    string `json:"error"`
			} `json:"reload"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.False(t, resp.Reload.Reloaded)
		assert.NotEmpty(t, resp.Reload.Error)

		prov, err := models.GetAdreProvisioning(db)
		require.NoError(t, err)
		assert.True(t, prov.RestartRequired, "failed reload must keep restart_required set")
	})

	t.Run("successful reload preserves a pending env restart", func(t *testing.T) {
		holmes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/admin/reload" && r.Method == http.MethodPost {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"ok","component":"all"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer holmes.Close()

		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		defer func() { require.NoError(t, sqlDB.Close()) }()
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		enable, url := true, holmes.URL
		_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAdre: &enable, AdreURL: &url})
		require.NoError(t, err)

		// A pending .env change (e.g. a new PMM URL) awaiting a container restart.
		prov, err := models.GetAdreProvisioning(db)
		require.NoError(t, err)
		prov.RestartRequired = true
		prov.EnvRestartRequired = true
		require.NoError(t, models.SaveAdreProvisioning(db, prov))

		t.Setenv("ADRE_CONFIG_DIR", t.TempDir())

		h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
		rec := putDeploymentConfig(t, h, "model: gpt-5.4\n")
		require.Equal(t, http.StatusOK, rec.Code)

		// The config hot-reloads fine, but the env restart must NOT be cleared by it.
		prov, err = models.GetAdreProvisioning(db)
		require.NoError(t, err)
		assert.True(t, prov.RestartRequired, "a pending env restart must survive a successful config reload")
		assert.True(t, prov.EnvRestartRequired)
	})
}

// TestHandlers_PostDeploymentRestarted verifies the operator's "restarted" acknowledgement clears both
// the restart requirement and the env restart flag.
func TestHandlers_PostDeploymentRestarted(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	prov, err := models.GetAdreProvisioning(db)
	require.NoError(t, err)
	prov.RestartRequired = true
	prov.EnvRestartRequired = true
	require.NoError(t, models.SaveAdreProvisioning(db, prov))

	h := NewHandlers(db, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
	req := httptest.NewRequest(http.MethodPost, "/v1/adre/deployment/restarted", nil)
	rec := httptest.NewRecorder()
	h.PostDeploymentRestarted(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	prov, err = models.GetAdreProvisioning(db)
	require.NoError(t, err)
	assert.False(t, prov.RestartRequired, "acknowledgement must clear restart_required")
	assert.False(t, prov.EnvRestartRequired, "acknowledgement must clear env_restart_required")
}
