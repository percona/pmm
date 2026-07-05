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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestClient_Models(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/model", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model_name": ["model-a", "model-b"]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	models, err := client.Models(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"model-a", "model-b"}, models)
}

func TestClient_Models_LegacyEncodedString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/model", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model_name":"[\"model-a\",\"model-b\"]"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	models, err := client.Models(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"model-a", "model-b"}, models)
}

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"analysis": "Hello!", "conversation_history": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Chat(context.Background(), &ChatRequest{Ask: "Hi"})
	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Analysis)
}

func TestClient_Reload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/admin/reload", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","component":"all","detail":"12 toolsets (10 enabled), 3 skills, 2 models","counts":{"toolsets_total":12,"toolsets_enabled":10,"skills":3,"models_loaded":2}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	res, err := client.Reload(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, 12, res.Counts["toolsets_total"])
	assert.Equal(t, 2, res.Counts["models_loaded"])
}

// TestClient_Reload_Unsupported covers an older Holmes / ENABLE_ADMIN_API unset: a 404 maps to the
// typed ErrReloadUnsupported so callers can fall back to requiring a restart.
func TestClient_Reload_Unsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Reload(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrReloadUnsupported))
}

func TestClient_Reload_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"reload failed"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Reload(context.Background())
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrReloadUnsupported))
}

func TestClient_TLSVerifyFails(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model_name": ["test"]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Models(context.Background())
	require.Error(t, err)
}

func TestClientFromSettings_TLSSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model_name": ["test"]}`))
	}))
	defer server.Close()

	enabled := true
	settings := &models.Settings{}
	settings.Adre.Enabled = &enabled
	settings.Adre.URL = server.URL
	settings.Adre.TLSSkipVerify = true

	client := NewClientFromSettings(settings)
	modelsList, err := client.Models(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"test"}, modelsList)
}
