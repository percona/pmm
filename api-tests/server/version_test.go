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

package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	paths := []string{
		"v1/server/version",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			uri := pmmapitests.BaseURL.ResolveReference(&url.URL{
				Path: path,
			})

			t.Logf("URI: %s", uri)
			req, _ := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			t.Logf("Response: %s", b)
			assert.Equal(t, 200, resp.StatusCode)

			var res server.VersionOKBody
			err = json.Unmarshal(b, &res)
			require.NoError(t, err)

			require.True(t, strings.HasPrefix(res.Version, "3."),
				"version = %q must have '3.' prefix for PMM 1.x's and 2.x's pmm-client compatibility checking", res.Version)

			require.NotEmpty(t, res.Managed)
			assert.True(t, strings.HasPrefix(res.Managed.Version, "3."),
				"managed.version = %q must have '3.' prefix ", res.Managed.Version)
			assert.NotEmpty(t, res.Managed.FullVersion)

			// check that timestamp is not XX:00:00
			require.NotEmpty(t, res.Managed.Timestamp)
			ts := time.Time(res.Managed.Timestamp)
			_, min, sec := ts.Clock()
			assert.True(t, min != 0 || sec != 0, "managed timestamp should not contain only date: %s", ts)

			if res.Server == nil || res.Server.Version == "" {
				t.Skip("skipping the rest of the test in developer's environment")
			}

			require.NotEmpty(t, res.Server)
			assert.True(t, strings.HasPrefix(res.Server.Version, res.Version),
				"server.version = %q should have %q prefix", res.Server.Version, res.Version)
			assert.NotEmpty(t, res.Server.FullVersion)

			// check that timestamp is not XX:00:00
			require.NotEmpty(t, res.Server.Timestamp)
			ts = time.Time(res.Server.Timestamp)
			_, min, sec = ts.Clock()
			assert.True(t, min != 0 || sec != 0, "server timestamp should not contain only date: %s", ts)
		})
	}
}
