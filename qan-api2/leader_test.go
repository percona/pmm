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

	// Only 200 and 400 are answers about leadership. Anything else read as "follower" would
	// stop retention on every node with nothing above debug level in the log.
	t.Run("unexpected status is undetermined", func(t *testing.T) {
		t.Parallel()

		for _, status := range []int{
			http.StatusNotFound,            // route renamed
			http.StatusUnauthorized,        // a proxy interposed on the port
			http.StatusInternalServerError, // pmm-managed broken
			http.StatusServiceUnavailable,
		} {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))

			leader, err := shouldApplyRetention(context.Background(), client, srv.URL)
			require.Error(t, err, "status %d must not be reported as a follower verdict", status)
			assert.False(t, leader)
			assert.Contains(t, err.Error(), "unexpected status")

			srv.Close()
		}
	})

	// A 400 is not specific to leadership: the gateway renders several gRPC codes as one, and
	// anything else on the port can answer 400 too. Reading those as "follower" would stop
	// retention on every node at once.
	t.Run("a 400 that is not FailedPrecondition is undetermined", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name string
			body string
		}{
			{"another code", `{"code": 3, "message": "invalid argument"}`},
			{"no body at all", ""},
			{"not even JSON", "Bad Request"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(tc.body))
				}))
				t.Cleanup(srv.Close)

				leader, err := shouldApplyRetention(t.Context(), client, srv.URL)
				require.Error(t, err, "a 400 without FailedPrecondition must not be a follower verdict")
				assert.False(t, leader)
			})
		}
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
