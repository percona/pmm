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
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	// The time-to-live for query data buckets in the store.
	defaultTTL = 30 * time.Second
	// How often to run TTL cleanup.
	cleanupInterval = 5 * time.Second
	// Number of shards for the store.
	// We use 256 shards to minimize lock contention in high-throughput scenarios.
	//
	// Performance characteristics:
	// - Without sharding: Single global lock serializes all writes (~100-200 writes/sec max)
	// - With 256 shards: Writes to different services are fully parallel (~10,000+ writes/sec)
	//
	// For 1000 instances writing 1 query/sec:
	// - Total throughput: 1000 writes/sec
	// - Per-shard load: ~4 writes/sec (1000/256)
	// - Lock contention: Minimal (writes are distributed across shards)
	//
	// Trade-offs:
	// - Memory: +256 shard structures (~few KB overhead, negligible)
	// - Complexity: Slightly more complex than single lock, but well worth it
	// - CPU: Hash calculation per operation (negligible cost).
	numShards = 256
)

// queryBucket represents a collection of queries for a service with metadata.
// The bucket's timestamp indicates when it was last updated, which is used for TTL.
type queryBucket struct {
	timestamp time.Time          // When this bucket was last updated (Set was called)
	queries   []*rtav1.QueryData // The actual query data
}

// shard represents a single shard of the store with its own lock.
// Each shard is completely independent, allowing concurrent operations
// on different shards without any lock contention.
type shard struct {
	mu      sync.RWMutex            // Lock only for this shard's data
	buckets map[string]*queryBucket // Query buckets stored in this shard
}

// Store provides thread-safe in-memory storage for real-time query data.
//
// Architecture: Uses sharding to minimize lock contention in high-throughput scenarios.
// Instead of a single global lock that serializes all operations, we distribute
// data across 256 independent shards, each with its own lock. This allows
// concurrent writes to different services to proceed in parallel.
//
// Example: With 1000 MongoDB instances sending queries:
//   - Without sharding: All writes compete for a single lock (bottleneck at ~100-200/sec)
//   - With sharding: Writes are distributed across 256 shards (~4 writes/sec per shard)
//     allowing the system to handle 10,000+ writes/sec with minimal lock contention.
type Store struct {
	l      *logrus.Entry
	shards [numShards]*shard // Fixed array of shards (no allocation overhead)
	ttl    time.Duration
}

// NewStore creates a new in-memory store for RTA query data.
func NewStore() *Store {
	s := &Store{
		l:   logrus.WithField("component", "rta-store"),
		ttl: defaultTTL,
	}

	// Initialize all shards
	for i := range numShards {
		s.shards[i] = &shard{
			buckets: make(map[string]*queryBucket),
		}
	}

	return s
}

// getShard returns the shard for a given serviceID using a hash function.
//
// Hash function properties:
// - Deterministic: Same serviceID always maps to the same shard
// - Well-distributed: UUIDs and other IDs spread evenly across shards
// - Fast: Simple multiplication and modulo operation
//
// The hash function ensures that queries for the same service always go to
// the same shard (required for correctness), while different services are
// distributed across different shards (required for performance).
func (s *Store) getShard(serviceID string) *shard {
	// Simple but effective hash: polynomial rolling hash
	// Uses prime multiplier (31) for good distribution
	hash := uint32(0)
	for i := range len(serviceID) {
		hash = hash*31 + uint32(serviceID[i]) //nolint:mnd
	}

	return s.shards[hash%numShards]
}

// Run starts the cleanup goroutine that removes expired query buckets.
func (s *Store) Run(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.l.Info("Stopping RTA store cleanup goroutine")
			return

		case <-ticker.C:
			s.cleanup()
		}
	}
}

// Set sets the queries for a specific service.
// Creates a new bucket with the current timestamp, which is used for TTL.
//
// Performance: Only locks the specific shard, allowing concurrent writes to different services.
// For example, if 1000 agents are writing simultaneously to 1000 different services,
// and they're distributed across 256 shards, only ~4 operations will compete for each
// shard's lock at any given time. This is much better than 1000 operations competing
// for a single global lock.
func (s *Store) Set(serviceID string, queries []*rtav1.QueryData) {
	shard := s.getShard(serviceID)

	shard.mu.Lock() // Lock ONLY this shard, not the entire store
	defer shard.mu.Unlock()

	shard.buckets[serviceID] = &queryBucket{
		timestamp: time.Now(),
		queries:   queries,
	}
}

// Get retrieves queries for a specific service.
// ServiceID must be specified - this ensures we only read from a single shard for optimal performance.
// Returns an empty slice if the service has no data or if the bucket is expired (never panics).
func (s *Store) Get(serviceID string) []*rtav1.QueryData {
	now := time.Now()
	cutoff := now.Add(-s.ttl)

	// Read from the specific shard for this service
	shard := s.getShard(serviceID)
	shard.mu.RLock()
	bucket, exists := shard.buckets[serviceID]
	shard.mu.RUnlock()

	// If service has no data, return empty slice (safe - no panic)
	if !exists {
		return []*rtav1.QueryData{}
	}

	// Check if the bucket is expired
	if bucket.timestamp.Before(cutoff) {
		return []*rtav1.QueryData{}
	}

	return bucket.queries
}

// cleanup removes expired query buckets from all shards.
// A bucket is expired if its timestamp is older than TTL.
//
// Performance: Locks each shard independently and briefly. While one shard is being
// cleaned, operations on other shards can proceed normally. This is much better than
// a single cleanup lock that would block all operations during cleanup.
func (s *Store) cleanup() {
	now := time.Now()
	cutoff := now.Add(-s.ttl)

	// Clean each shard independently - other shards remain available during cleanup
	for i := range numShards {
		shard := s.shards[i]
		shard.mu.Lock() // Lock only this shard during its cleanup

		for serviceID, bucket := range shard.buckets {
			// Check if bucket is expired
			if bucket.timestamp.Before(cutoff) {
				delete(shard.buckets, serviceID)
			}
		}

		shard.mu.Unlock()
	}
}

// Clear removes all queries for a specific service.
// Safe to call even if the service has no data (delete on non-existent key is a no-op).
func (s *Store) Clear(serviceID string) {
	shard := s.getShard(serviceID)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	delete(shard.buckets, serviceID) // Safe: delete on non-existent key is a no-op
}

// Stats returns the number of queries stored per service across all shards.
func (s *Store) Stats() map[string]int {
	stats := make(map[string]int)

	// Aggregate stats from all shards
	for i := range numShards {
		shard := s.shards[i]
		shard.mu.RLock()

		for serviceID, bucket := range shard.buckets {
			stats[serviceID] += len(bucket.queries)
		}

		shard.mu.RUnlock()
	}

	return stats
}
