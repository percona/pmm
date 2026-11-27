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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forgetServer answers a DELETE on the om_inventory services route with status, recording the path it saw.
func forgetServer(t *testing.T, status int) (*httptest.Server, *[]string) {
	t.Helper()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, &paths
}

func TestInventoryForgetterDeletesTheServiceRow(t *testing.T) {
	srv, paths := forgetServer(t, http.StatusNoContent)
	forgetter := inventoryForgetter{app: newExtensionsClient(srv.URL, "test-token").app(probeAppModule)}

	err := forgetter.forgetService(asService(t.Context()), "svc-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"DELETE /api/apps/om_inventory/services/svc-1"}, *paths)
}

func TestInventoryForgetterTreatsNotFoundAsForgotten(t *testing.T) {
	srv, _ := forgetServer(t, http.StatusNotFound)
	forgetter := inventoryForgetter{app: newExtensionsClient(srv.URL, "test-token").app(probeAppModule)}

	err := forgetter.forgetService(asService(t.Context()), "svc-1")

	require.NoError(t, err, "a service om_inventory already forgot must not fail a resumed forget step")
}

func TestInventoryForgetterFailsOnServerError(t *testing.T) {
	srv, _ := forgetServer(t, http.StatusInternalServerError)
	forgetter := inventoryForgetter{app: newExtensionsClient(srv.URL, "test-token").app(probeAppModule)}

	err := forgetter.forgetService(asService(t.Context()), "svc-1")

	require.Error(t, err, "a failed forget must retry the step, not report the host clean")
}
