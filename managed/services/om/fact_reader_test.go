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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func factReaderFor(t *testing.T, handler http.HandlerFunc) *factReader {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &extensionsClient{baseURL: server.URL, token: "test-token", http: server.Client()}
	return newFactReader(client)
}

func TestObservationReadsFactsAndCollectedAt(t *testing.T) {
	collected := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	var gotPath, gotKind, gotProfile string
	reader := factReaderFor(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKind = r.URL.Query().Get("subject_kind")
		gotProfile = r.URL.Query().Get("profile")
		_, _ = w.Write([]byte(`{"facts":{"os_supported":true},"collected_at":"2026-11-23T10:00:00Z"}`))
	})

	facts, at, err := reader.Observation(t.Context(), "install_node", "node-7", scanProfileInstallCheck)
	require.NoError(t, err)
	assert.Equal(t, "/api/apps/om_inventory/observations/node-7", gotPath)
	assert.Equal(t, "install_node", gotKind)
	assert.Equal(t, scanProfileInstallCheck, gotProfile)
	assert.Equal(t, map[string]any{"os_supported": true}, facts)
	assert.True(t, at.Equal(collected))
}

func TestObservationOfAnUnscannedSubjectIsEmpty(t *testing.T) {
	reader := factReaderFor(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	facts, at, err := reader.Observation(t.Context(), "member", "m-1", scanProfileManaged)
	require.NoError(t, err)
	assert.Nil(t, facts)
	assert.True(t, at.IsZero())
}

func TestFreshFactsDropsOverAgeFacts(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	collected := now.Add(-2 * time.Hour)
	facts := map[string]any{"replset": "rs0", "pbm_state": "idle", "unbounded": 1}
	maxAges := map[string]time.Duration{"replset": time.Hour, "pbm_state": 3 * time.Hour}

	fresh := freshFacts(facts, collected, maxAges, now)

	assert.NotContains(t, fresh, "replset", "a fact older than its max age reads as missing")
	assert.Contains(t, fresh, "pbm_state", "a fact within its max age is kept")
	assert.Contains(t, fresh, "unbounded", "a fact with no declared age is kept")
}
