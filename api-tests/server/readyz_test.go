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

	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
)

// Under full-suite load, /ping and /v1/server/readyz can briefly return 503/500 while components catch up.
const readinessProbeTimeout = 30 * time.Second
const readinessProbeInterval = 200 * time.Millisecond

func TestReadyz(t *testing.T) {
	t.Parallel()
	paths := []string{
		"ping",
		"v1/server/readyz",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			// Copy BaseURL to avoid race conditions when accessing it concurrently
			baseURL := *pmmapitests.BaseURL
			baseURL.User = nil

			uri := baseURL.ResolveReference(&url.URL{
				Path: path,
			})

			// Use a dedicated client to avoid interference from other parallel tests
			client := &http.Client{}

			var lastStatus int
			var lastBody []byte
			require.Eventually(t, func() bool {
				req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, uri.String(), nil)
				if err != nil {
					return false
				}
				resp, err := client.Do(req)
				if err != nil {
					return false
				}
				defer func() { _ = resp.Body.Close() }() //nolint:errcheck
				b, err := io.ReadAll(resp.Body)
				if err != nil {
					return false
				}
				lastStatus = resp.StatusCode
				lastBody = append([]byte(nil), b...)
				return resp.StatusCode == http.StatusOK && string(b) == "{}"
			}, readinessProbeTimeout, readinessProbeInterval,
				"GET %s expected HTTP 200 and body {}; last status=%d body=%s",
				uri.String(), lastStatus, lastBody)
		})
	}
}
