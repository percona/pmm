// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cache provides a high-performance cache implementation with sharding
// and TTL support. It is designed to minimize lock contention and maximize
// throughput in concurrent environments.
package cache

import (
	"context"
	"errors"
	"hash/maphash"
	"sync"
	"time"
)

const (
	// ShardCount must be a power of 2 for bitwise modulo.
	// 256 is optimal for typical 16-64 core server deployments.
	shardCount = 256
	shardMask  = shardCount - 1
)

var (
	errInvalidContext         = errors.New("context must not be nil")
	errInvalidCleanupInterval = errors.New("cleanupInterval must be greater than 0")
)

// item wraps the value and its expiration timestamp.
type item[V any] struct {
	value   V
	expires time.Time
}

// shard contains the actual map and a lock, padded to prevent false sharing.
type shard[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]item[V]
	size  int64
	// Pad to 64 bytes to prevent false sharing in L1 CPU cache.
	// RWMutex (24B) + map pointer (8B) + int64 (8B) = 40B. We pad remaining 24B.
	_ [24]byte
}

// Cache is a high-performance, sharded, generic TTL cache.
type Cache[K comparable, V any] struct {
	shards [shardCount]*shard[K, V]
	seed   maphash.Seed
	ttl    time.Duration
}

// New initializes the cache and starts the background eviction worker.
func New[K comparable, V any](ctx context.Context, ttl time.Duration, cleanupInterval time.Duration) (*Cache[K, V], error) {
	if ctx == nil {
		return nil, errInvalidContext
	}

	if cleanupInterval <= 0 {
		return nil, errInvalidCleanupInterval
	}

	c := &Cache[K, V]{
		seed: maphash.MakeSeed(),
		ttl:  ttl,
	}

	for i := range shardCount {
		c.shards[i] = &shard[K, V]{
			items: make(map[K]item[V]),
		}
	}

	go c.evictionWorker(ctx, cleanupInterval)
	return c, nil
}

// Get retrieves an item. Zero-allocation on the hot path.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	// Zero-allocation hash via maphash
	hash := maphash.Comparable(c.seed, key)
	shard := c.shards[hash&shardMask]

	shard.mu.RLock()
	itm, found := shard.items[key]
	shard.mu.RUnlock()

	if !found {
		var zero V
		return zero, false
	}

	// Lazy eviction check.
	// We read time.Now() outside the lock to minimize critical section time.
	now := time.Now()
	if now.After(itm.expires) {
		// 3. Cold Path: Upgrade to Write Lock to physically remove an element from the map,
		// so that the GC can instantly remove V from memory.
		shard.mu.Lock()
		// Double-checking: check if another goroutine has overwritten the key while we were switching locks.
		if currentItm, stillExists := shard.items[key]; stillExists && now.After(currentItm.expires) {
			delete(shard.items, key)
			shard.size--
		}
		shard.mu.Unlock()

		var zero V
		return zero, false
	}

	return itm.value, true
}

// Set inserts or updates an item with a specific TTL.
func (c *Cache[K, V]) Set(key K, value V) {
	hash := maphash.Comparable(c.seed, key)
	shard := c.shards[hash&shardMask]

	expires := time.Now().Add(c.ttl)

	shard.mu.Lock()
	if _, exists := shard.items[key]; !exists {
		shard.size++
	}
	shard.items[key] = item[V]{
		value:   value,
		expires: expires,
	}
	shard.mu.Unlock()
}

// Delete removes an item explicitly.
func (c *Cache[K, V]) Delete(key K) {
	hash := maphash.Comparable(c.seed, key)
	shard := c.shards[hash&shardMask]

	shard.mu.Lock()
	if _, exists := shard.items[key]; exists {
		delete(shard.items, key)
		shard.size--
	}
	shard.mu.Unlock()
}

// Size returns the total number of items across all cache shards.
func (c *Cache[K, V]) Size() int64 {
	var total int64
	for i := range shardCount {
		shard := c.shards[i]
		shard.mu.RLock()
		total += shard.size
		shard.mu.RUnlock()
	}

	return total
}

// evictionWorker periodically sweeps shards to remove expired items.
func (c *Cache[K, V]) evictionWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			// Iterate through shards. We lock one shard at a time to ensure
			// we don't stall the entire cache during the sweep.
			for i := range shardCount {
				shard := c.shards[i]

				shard.mu.Lock()
				for key, itm := range shard.items {
					if now.After(itm.expires) {
						delete(shard.items, key)
						shard.size--
					}
				}
				shard.mu.Unlock()
			}
		}
	}
}
