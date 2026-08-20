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
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
)

func TestReadyz(t *testing.T) {
	t.Parallel()
	paths := []string{
		"ping",
		"v1/server/readyz",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			// make a BaseURL without authentication
			baseURL, err := url.Parse(pmmapitests.BaseURL.String())
			require.NoError(t, err)
			baseURL.User = nil

			uri := baseURL.ResolveReference(&url.URL{
				Path: path,
			})

			var lastStatus int
			var lastBody []byte
			require.Eventually(t, func() bool {
				req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
				if err != nil {
					return false
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return false
				}
				t.Cleanup(func() {
					assert.NoError(t, resp.Body.Close())
				})

				b, err := io.ReadAll(resp.Body)
				if err != nil {
					return false
				}
				lastStatus = resp.StatusCode
				lastBody = append([]byte(nil), b...)
				return resp.StatusCode == http.StatusOK && string(b) == "{}"
			}, 30*time.Second, 200*time.Millisecond,
				"GET %s expected HTTP 200 and body {}; last status=%d body=%s",
				uri.String(), lastStatus, lastBody)
		})
	}
}
