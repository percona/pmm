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
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
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

// testNode is the ClickHouse that `make test-env-up` starts, with database db dropped now and after the test.
func testNode(t *testing.T, db string) chNode {
	t.Helper()
	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	if !ok {
		dsn = "clickhouse://default:clickhouse@127.0.0.1:19000/pmm_test"
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	nodes, err := openNodes(dsn, u.Hostname())
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	node := nodes[0]

	drop := func(ctx context.Context) error {
		_, err := node.db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+db+" SYNC")
		return err
	}
	require.NoError(t, drop(t.Context()))
	t.Cleanup(func() {
		assert.NoError(t, drop(context.WithoutCancel(t.Context())))
		assert.NoError(t, node.db.Close())
	})
	return node
}

func TestFixPmmChangesNothingWhenANodeDoesNotAnswer(t *testing.T) {
	t.Parallel()

	const db = "pmm_fixpmm_test"
	node := testNode(t, db)
	down, err := sqlx.Open("clickhouse", "clickhouse://default:clickhouse@127.0.0.1:1/default")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, down.Close()) })

	// The answering node comes first, so it has been read before the pass meets the silent one.
	err = fixPmm(t.Context(), []chNode{node, {host: "127.0.0.1:1", db: down}}, db, true)
	require.ErrorContains(t, err, "127.0.0.1:1 does not answer, changing nothing")

	c, err := readCopy(t.Context(), node, db)
	require.NoError(t, err)
	assert.Empty(t, c.uuid, "nothing may be created while a node does not answer")
}

func TestReadCopyDatabaseWithoutTables(t *testing.T) {
	t.Parallel()

	const db = "pmm_readcopy_test"
	node := testNode(t, db)
	_, err := node.db.ExecContext(t.Context(), "CREATE DATABASE "+db)
	require.NoError(t, err)
	var want string
	err = node.db.GetContext(t.Context(), &want, "SELECT toString(uuid) FROM system.databases WHERE name = ?", db)
	require.NoError(t, err)

	c, err := readCopy(t.Context(), node, db)
	require.NoError(t, err)
	assert.Equal(t, pmmCopy{uuid: want, rows: 0}, c)
}
