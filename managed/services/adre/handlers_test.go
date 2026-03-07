// Copyright (C) 2025 Percona LLC
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

package adre

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

type mockGrafanaAlertsFetcher struct {
	alerts []byte
	err    error
}

func (m *mockGrafanaAlertsFetcher) GetAlertmanagerAlerts(_ context.Context, _ http.Header) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.alerts != nil {
		return m.alerts, nil
	}
	return []byte("[]"), nil
}

func TestHandlers_GetSettings(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	h := NewHandlers(db, &mockGrafanaAlertsFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/v1/adre/settings", nil)
	rec := httptest.NewRecorder()
	h.GetSettings(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.False(t, body.Enabled)
	assert.Empty(t, body.URL)
}

func TestHandlers_PostSettings_Validation(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	h := NewHandlers(db, &mockGrafanaAlertsFetcher{})

	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/adre/settings", bytes.NewReader([]byte("{}")))
		rec := httptest.NewRecorder()
		h.PostSettings(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errBody map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&errBody))
		assert.Contains(t, errBody["error"], "No changes provided")
	})

	t.Run("InvalidURLScheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/adre/settings", bytes.NewReader([]byte(`{"url":"ftp://x"}`)))
		rec := httptest.NewRecorder()
		h.PostSettings(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errBody map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&errBody))
		assert.Contains(t, errBody["error"], "http:// or https://")
	})

	t.Run("InvalidURLNoHost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/adre/settings", bytes.NewReader([]byte(`{"url":"http://"}`)))
		rec := httptest.NewRecorder()
		h.PostSettings(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errBody map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&errBody))
		assert.Contains(t, errBody["error"], "valid host")
	})

	t.Run("Valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/adre/settings", bytes.NewReader([]byte(`{"enabled":true,"url":"http://holmes:8080"}`)))
		rec := httptest.NewRecorder()
		h.PostSettings(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Enabled bool   `json:"enabled"`
			URL     string `json:"url"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.True(t, body.Enabled)
		assert.Equal(t, "http://holmes:8080", body.URL)
	})
}

func TestHandlers_GetModels_AdreDisabled(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
		EnableAdre: pointer.ToBool(false),
	})
	require.NoError(t, err)

	h := NewHandlers(db, &mockGrafanaAlertsFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/v1/adre/models", nil)
	rec := httptest.NewRecorder()
	h.GetModels(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errBody map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&errBody))
	assert.Contains(t, errBody["error"], "ADRE is disabled")
}

func TestHandlers_GetModels_AdreEnabled_NoURL(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
		EnableAdre: pointer.ToBool(true),
		AdreURL:    pointer.ToString(""),
	})
	require.NoError(t, err)

	h := NewHandlers(db, &mockGrafanaAlertsFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/v1/adre/models", nil)
	rec := httptest.NewRecorder()
	h.GetModels(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errBody map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&errBody))
	assert.Contains(t, errBody["error"], "HolmesGPT URL")
}

func TestHandlers_GetAlerts(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() { require.NoError(t, sqlDB.Close()) }()
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{
		EnableAdre: pointer.ToBool(true),
		AdreURL:    pointer.ToString("http://holmes:8080"),
	})
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		alerts := []byte(`[{"labels":{"alertname":"test"},"annotations":{"summary":"Test"}}]`)
		h := NewHandlers(db, &mockGrafanaAlertsFetcher{alerts: alerts})
		req := httptest.NewRequest(http.MethodGet, "/v1/adre/alerts", nil)
		rec := httptest.NewRecorder()
		h.GetAlerts(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Alerts []interface{} `json:"alerts"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		require.Len(t, body.Alerts, 1)
	})

	t.Run("EmptyAlerts", func(t *testing.T) {
		h := NewHandlers(db, &mockGrafanaAlertsFetcher{})
		req := httptest.NewRequest(http.MethodGet, "/v1/adre/alerts", nil)
		rec := httptest.NewRecorder()
		h.GetAlerts(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Alerts []interface{} `json:"alerts"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Empty(t, body.Alerts)
	})
}
