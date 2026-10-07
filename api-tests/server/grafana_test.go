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
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
)

func TestGrafanaUsageReportingDisabled(t *testing.T) {
	t.Parallel()

	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/graph/api/admin/settings"

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, u.String(), nil)
	require.NoError(t, err)

	resp, b := doRequest(t, http.DefaultClient, req) //nolint:bodyclose
	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to get Grafana settings, response: %s", b)

	var settings map[string]map[string]string
	require.NoError(t, json.Unmarshal(b, &settings))
	assert.Equal(t, "false", settings["analytics"]["reporting_enabled"])
}
