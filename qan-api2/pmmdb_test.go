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

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeptUUID(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		copies []pmmCopy
		want   string
	}{
		"no pmm anywhere":    {[]pmmCopy{{}, {}}, ""},
		"most rows wins":     {[]pmmCopy{{"a", 5}, {"b", 9}, {}}, "b"},
		"tie: lowest UUID":   {[]pmmCopy{{"b", 5}, {"a", 5}}, "a"},
		"zero-row copies":    {[]pmmCopy{{"b", 0}, {"a", 0}}, "a"},
		"missing is ignored": {[]pmmCopy{{}, {"b", 0}}, "b"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, keptUUID(tc.copies))
		})
	}
}

func TestIsLeader(t *testing.T) {
	t.Parallel()

	const nodes = `{"nodes":[{"node_name":"pmm-0","role":"NODE_ROLE_LEADER","status":"alive"},` +
		`{"node_name":"pmm-1","role":"NODE_ROLE_FOLLOWER","status":"alive"}],"expected_nodes":2}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		if r.URL.Path != "/v1/ha/nodes" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(nodes))
	}))
	t.Cleanup(srv.Close)
	url := srv.URL + "/v1/ha/nodes"

	leader, err := isLeader(t.Context(), url, "pmm-0")
	require.NoError(t, err)
	assert.True(t, leader)

	leader, err = isLeader(t.Context(), url, "pmm-1")
	require.NoError(t, err)
	assert.False(t, leader)

	_, err = isLeader(t.Context(), url, "")
	require.Error(t, err)

	_, err = isLeader(t.Context(), srv.URL+"/wrong", "pmm-0")
	require.Error(t, err)
}
