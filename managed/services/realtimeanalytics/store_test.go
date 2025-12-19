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

package realtimeanalytics

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Run("Set", func(t *testing.T) {
		store := NewStore()

		queries := []*QueryData{
			{
				QueryID:     "q1",
				ServiceID:   "service1",
				ServiceName: "test-mongodb",
				Cluster:     "cluster1",
				Namespace:   "db.collection",
				Query:       `{"find": "users"}`,
				Fingerprint: "find-users",
				Duration:    10.5,
				Timestamp:   time.Now(),
			},
		}

		store.Set("service1", queries)

		results := store.Get("service1", "")
		require.Len(t, results, 1)
		assert.Equal(t, "q1", results[0].QueryID)
		assert.Equal(t, "service1", results[0].ServiceID)
	})

	t.Run("Sharding", func(t *testing.T) {
		store := NewStore()

		// Write to multiple services concurrently to test sharding
		var wg sync.WaitGroup
		numServices := 100

		for i := range numServices {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				serviceID := fmt.Sprintf("service%d", idx)
				queries := []*QueryData{
					{
						QueryID:   fmt.Sprintf("q%d", idx),
						ServiceID: serviceID,
						Timestamp: time.Now(),
					},
				}
				store.Set(serviceID, queries)
			}(i)
		}

		wg.Wait()

		// Verify all services are present
		stats := store.Stats()
		assert.Equal(t, numServices, len(stats), "Should have all services")

		// Verify shards are being used (queries should be distributed)
		usedShards := 0
		for i := 0; i < numShards; i++ {
			if len(store.shards[i].queries) > 0 {
				usedShards++
			}
		}
		assert.Greater(t, usedShards, 1, "Should use multiple shards")
	})

	t.Run("GetFiltering", func(t *testing.T) {
		store := NewStore()

		// Set queries for different services and clusters
		store.Set("s1", []*QueryData{
			{QueryID: "q1", ServiceID: "s1", Cluster: "c1", Timestamp: time.Now()},
			{QueryID: "q2", ServiceID: "s1", Cluster: "c1", Timestamp: time.Now()},
		})
		store.Set("s2", []*QueryData{
			{QueryID: "q3", ServiceID: "s2", Cluster: "c1", Timestamp: time.Now()},
		})
		store.Set("s3", []*QueryData{
			{QueryID: "q4", ServiceID: "s3", Cluster: "c2", Timestamp: time.Now()},
		})

		t.Run("filter by service", func(t *testing.T) {
			results := store.Get("s1", "")
			require.Len(t, results, 2)
			assert.Equal(t, "q1", results[0].QueryID)
			assert.Equal(t, "q2", results[1].QueryID)
		})

		t.Run("filter by service and cluster", func(t *testing.T) {
			results := store.Get("s1", "c1")
			require.Len(t, results, 2)
			assert.Equal(t, "q1", results[0].QueryID)
			assert.Equal(t, "q2", results[1].QueryID)
		})

		t.Run("different service", func(t *testing.T) {
			results := store.Get("s2", "")
			require.Len(t, results, 1)
			assert.Equal(t, "q3", results[0].QueryID)
		})

		t.Run("non-existent service returns empty", func(t *testing.T) {
			results := store.Get("nonexistent", "")
			require.Empty(t, results)
		})
	})

	t.Run("TTL", func(t *testing.T) {
		store := NewStore()
		store.ttl = 100 * time.Millisecond // Short TTL for testing

		queries := []*QueryData{
			{
				QueryID:   "old",
				ServiceID: "service1",
				Timestamp: time.Now().Add(-200 * time.Millisecond), // Already expired
			},
			{
				QueryID:   "new",
				ServiceID: "service1",
				Timestamp: time.Now(),
			},
		}

		store.Set("service1", queries)

		// Get should filter out expired queries
		results := store.Get("service1", "")
		require.Len(t, results, 1, "Should only return non-expired query")
		assert.Equal(t, "new", results[0].QueryID)
	})

	t.Run("Cleanup", func(t *testing.T) {
		store := NewStore()
		store.ttl = 50 * time.Millisecond

		// Set some queries
		queries := make([]*QueryData, 5)
		for i := range 5 {
			queries[i] = &QueryData{
				QueryID:   string(rune('a' + i)),
				ServiceID: "service1",
				Timestamp: time.Now(),
			}
		}
		store.Set("service1", queries)

		// Wait for queries to expire
		time.Sleep(100 * time.Millisecond)

		// Run cleanup
		store.cleanup()

		// Check that queries were removed
		results := store.Get("service1", "")
		assert.Empty(t, results, "Service entry should be removed after all queries expire")
	})

	t.Run("Clear", func(t *testing.T) {
		store := NewStore()

		store.Set("service1", []*QueryData{{QueryID: "q1", ServiceID: "service1", Timestamp: time.Now()}})
		store.Set("service2", []*QueryData{{QueryID: "q2", ServiceID: "service2", Timestamp: time.Now()}})

		store.Clear("service1")

		results := store.Get("service1", "")
		assert.Empty(t, results, "service1 queries should be cleared")

		results = store.Get("service2", "")
		assert.Len(t, results, 1, "service2 queries should remain")

		// Test that clearing non-existent service doesn't panic
		store.Clear("nonexistent")
		results = store.Get("nonexistent", "")
		assert.Empty(t, results, "non-existent service should return empty")
	})

	t.Run("Stats", func(t *testing.T) {
		store := NewStore()

		store.Set("s1", []*QueryData{
			{QueryID: "q1", ServiceID: "s1", Timestamp: time.Now()},
			{QueryID: "q2", ServiceID: "s1", Timestamp: time.Now()},
		})
		store.Set("s2", []*QueryData{
			{QueryID: "q3", ServiceID: "s2", Timestamp: time.Now()},
		})

		stats := store.Stats()
		assert.Equal(t, 2, stats["s1"], "Should have 2 queries for s1")
		assert.Equal(t, 1, stats["s2"], "Should have 1 query for s2")
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		store := NewStore()
		var wg sync.WaitGroup

		// Concurrent writes
		for i := range 100 {
			idx := i
			wg.Go(func() {
				queries := []*QueryData{
					{
						QueryID:   string(rune('a' + idx%26)),
						ServiceID: fmt.Sprintf("service%d", idx%10),
						Timestamp: time.Now(),
					},
				}
				store.Set(fmt.Sprintf("service%d", idx%10), queries)
			})
		}

		// Concurrent reads
		for range 50 {
			wg.Go(func() {
				_ = store.Get("service1", "")
			})
		}

		wg.Wait()

		// Should not panic and should have data
		results := store.Get("service1", "")
		assert.NotEmpty(t, results)
	})
}
