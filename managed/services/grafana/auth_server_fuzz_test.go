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

package grafana

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzExtractOriginalRequest(f *testing.F) {
	for _, seed := range []struct {
		origMethod string
		origURI    string
	}{
		{origMethod: http.MethodPost, origURI: "/v1/server/version"},
		{origMethod: http.MethodDelete, origURI: "/v1/server/AWSInstanceCheck/..%2f..%2fmanaged/logs.zip?foo=bar"},
		{origMethod: http.MethodGet, origURI: "/"},
		{origMethod: http.MethodGet, origURI: "v1/server/version"},
		{origMethod: http.MethodGet, origURI: "/v1/server/%zz/logs.zip"},
		{origMethod: http.MethodGet, origURI: string([]byte{'/', 'b', 'a', 'd', 0xff})},
	} {
		f.Add("", seed.origMethod, seed.origURI)
	}

	f.Fuzz(func(t *testing.T, _ string, origMethod, origURI string) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth_request", nil)
		req.Header.Set("X-Original-Method", origMethod)
		req.Header.Set("X-Original-Uri", origURI)

		beforeMethod := req.Method
		beforePath := req.URL.Path
		beforeCleanedPath, err := cleanPath(origURI)
		if err != nil {
			return
		}

		err = extractOriginalRequest(req)
		if err != nil {
			require.Equal(t, beforeMethod, req.Method)
			require.Equal(t, beforePath, req.URL.Path)
			return
		}

		require.Equal(t, origMethod, req.Method)
		require.Equal(t, beforeCleanedPath, req.URL.Path)
		require.NotContains(t, req.URL.Path, "?")
	})
}
