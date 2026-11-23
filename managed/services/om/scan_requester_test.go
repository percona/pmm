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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The requester fulfils both the driver's rescan seam and the scheduler's per-cluster seam.
var (
	_ ScanRequester        = (*inventoryScanRequester)(nil)
	_ clusterScanRequester = (*inventoryScanRequester)(nil)
)

func scanRequesterFor(t *testing.T, handler http.HandlerFunc) *inventoryScanRequester {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &extensionsClient{baseURL: server.URL, token: "test-token", http: server.Client()}
	return newInventoryScanRequester(client)
}

func TestRequestScanPostsOperationRequest(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody scanRequest
	requester := scanRequesterFor(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	})

	err := requester.RequestScan("op-1", scanProfileImportCheck, "op-1:3")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/apps/om_inventory/scans", gotPath)
	assert.Equal(t, scanProfileImportCheck, gotBody.Profile)
	assert.Equal(t, "op-1:3", gotBody.IdempotencyKey)
	assert.Equal(t, "op-1", gotBody.OperationID)
	assert.Empty(t, gotBody.ClusterID)
}

func TestRequestClusterScanPostsClusterRequest(t *testing.T) {
	var gotBody scanRequest
	requester := scanRequesterFor(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	})

	err := requester.RequestClusterScan(t.Context(), "cl-1", scanProfileManaged, "cl-1:managed:42")
	require.NoError(t, err)
	assert.Equal(t, scanProfileManaged, gotBody.Profile)
	assert.Equal(t, "cl-1", gotBody.ClusterID)
	assert.Empty(t, gotBody.OperationID)
}

func TestRequestScanSurfacesAnExtensionsFailure(t *testing.T) {
	requester := scanRequesterFor(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	err := requester.RequestScan("op-1", scanProfileManaged, "op-1:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "managed")
}
