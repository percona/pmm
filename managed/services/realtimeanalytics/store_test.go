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

		results := store.Get("service1")
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

		// Verify shards are being used (buckets should be distributed)
		usedShards := 0
		for i := 0; i < numShards; i++ {
			if len(store.shards[i].buckets) != 0 {
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

		t.Run("get by service", func(t *testing.T) {
			results := store.Get("s1")
			require.Len(t, results, 2)
			assert.Equal(t, "q1", results[0].QueryID)
			assert.Equal(t, "q2", results[1].QueryID)
		})

		t.Run("different service", func(t *testing.T) {
			results := store.Get("s2")
			require.Len(t, results, 1)
			assert.Equal(t, "q3", results[0].QueryID)
		})

		t.Run("non-existent service returns empty", func(t *testing.T) {
			results := store.Get("nonexistent")
			require.Empty(t, results)
		})
	})

	t.Run("BucketTTL", func(t *testing.T) {
		store := NewStore()
		store.ttl = 100 * time.Millisecond // Short TTL for testing

		// Set some queries
		queries := []*QueryData{
			{QueryID: "q1", ServiceID: "service1", Timestamp: time.Now()},
			{QueryID: "q2", ServiceID: "service1", Timestamp: time.Now()},
		}
		store.Set("service1", queries)

		// Wait for bucket to expire
		time.Sleep(150 * time.Millisecond)

		// Get should return empty because bucket is expired
		results := store.Get("service1")
		require.Empty(t, results, "Should return empty for expired bucket")
	})

	t.Run("Cleanup", func(t *testing.T) {
		store := NewStore()
		store.ttl = 50 * time.Millisecond

		// Set queries that will expire
		store.Set("service1", []*QueryData{
			{QueryID: "q1", ServiceID: "service1", Timestamp: time.Now()},
		})

		// Wait for service1 bucket to expire
		time.Sleep(60 * time.Millisecond)

		// Set fresh queries for service2
		store.Set("service2", []*QueryData{
			{QueryID: "q2", ServiceID: "service2", Timestamp: time.Now()},
		})

		// Run cleanup
		store.cleanup()

		// Check that expired bucket was removed
		results := store.Get("service1")
		assert.Empty(t, results, "Expired bucket should be removed")

		// Check that fresh bucket remains
		results = store.Get("service2")
		assert.Len(t, results, 1, "Fresh bucket should remain")
	})

	t.Run("Clear", func(t *testing.T) {
		store := NewStore()

		store.Set("service1", []*QueryData{{QueryID: "q1", ServiceID: "service1", Timestamp: time.Now()}})
		store.Set("service2", []*QueryData{{QueryID: "q2", ServiceID: "service2", Timestamp: time.Now()}})

		store.Clear("service1")

		results := store.Get("service1")
		assert.Empty(t, results, "service1 queries should be cleared")

		results = store.Get("service2")
		assert.Len(t, results, 1, "service2 queries should remain")

		// Test that clearing non-existent service doesn't panic
		store.Clear("nonexistent")
		results = store.Get("nonexistent")
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

	t.Run("Run_Cleanup", func(t *testing.T) {
		store := NewStore()
		store.ttl = 50 * time.Millisecond

		// Start cleanup goroutine
		ctx := t.Context()
		go store.Run(ctx)

		// Set queries that will expire
		store.Set("service1", []*QueryData{
			{QueryID: "q1", ServiceID: "service1", Timestamp: time.Now()},
		})

		// Wait for bucket to expire
		time.Sleep(60 * time.Millisecond)

		// Manually trigger cleanup for the test
		store.cleanup()

		// Check that expired data was removed
		results := store.Get("service1")
		assert.Empty(t, results, "Cleanup should have removed expired bucket")
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
				_ = store.Get("service1")
			})
		}

		wg.Wait()

		// Should not panic and should have data
		results := store.Get("service1")
		assert.NotEmpty(t, results)
	})
}
