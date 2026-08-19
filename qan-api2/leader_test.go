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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldApplyRetention(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: leaderCheckTimeout}

	t.Run("leader", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(srv.Close)

		leader, err := shouldApplyRetention(context.Background(), client, srv.URL)
		require.NoError(t, err)
		assert.True(t, leader)
	})

	t.Run("not a leader", func(t *testing.T) {
		t.Parallel()

		// What pmm-managed answers on a follower: FailedPrecondition, rendered as 400.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code": 9, "message": "this PMM Server isn't the leader"}`))
		}))
		t.Cleanup(srv.Close)

		leader, err := shouldApplyRetention(context.Background(), client, srv.URL)
		require.NoError(t, err)
		assert.False(t, leader)
	})

	t.Run("pmm-managed unreachable", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := srv.URL
		srv.Close()

		leader, err := shouldApplyRetention(context.Background(), client, url)
		require.Error(t, err)
		assert.False(t, leader, "an unreachable pmm-managed must not authorize dropping data")
	})

	t.Run("check disabled", func(t *testing.T) {
		t.Parallel()

		leader, err := shouldApplyRetention(context.Background(), client, "")
		require.NoError(t, err)
		assert.True(t, leader)
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(srv.Close)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		leader, err := shouldApplyRetention(ctx, client, srv.URL)
		require.Error(t, err)
		assert.False(t, leader)
	})
}
