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
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// This test checks if all (even empty) fields are present in json responses.
func TestSerialization(t *testing.T) {
	// Get json filed names from settings model
	var settings server_service.GetSettingsOKBodySettings
	jsonFields := extractJSONTagNames(settings)
	require.NotEmpty(t, jsonFields)

	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.Path = "/v1/server/settings"

	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, u.String(), nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:gosec,errcheck

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var data map[string]interface{}
	err = json.Unmarshal(b, &data)
	require.NoError(t, err)

	// Check if all expected fields present in the json response.
	for _, field := range jsonFields {
		assert.Contains(t, data["settings"], field)
	}
}

func extractJSONTagNames(v any) []string {
	var res []string
	t := reflect.ValueOf(v).Type()
	for i := 0; i < t.NumField(); i++ {
		if tag, ok := t.Field(i).Tag.Lookup("json"); ok {
			s := strings.Split(tag, ",")
			res = append(res, s[0])
		}
	}

	return res
}
