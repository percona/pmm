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
	"hash/maphash"
	"time"
)

// TTLCache is a high-performance, sharded, generic TTL cache.
type TTLCache[V any] struct {
	shards [shardCount]*shard[V]
	seed   maphash.Seed
	ttl    time.Duration
}

// NewCacheTTL initializes the cache with TTL support and starts the background eviction worker.
func NewCacheTTL[V any](ctx context.Context, ttl time.Duration, cleanupInterval time.Duration) (*TTLCache[V], error) {
	if ctx == nil {
		return nil, errInvalidContext
	}

	if ttl <= 0 {
		return nil, errInvalidTTLInterval
	}

	if cleanupInterval <= 0 {
		return nil, errInvalidCleanupInterval
	}

	c := &TTLCache[V]{
		seed: maphash.MakeSeed(),
		ttl:  ttl,
	}

	for i := range shardCount {
		c.shards[i] = &shard[V]{
			items: make(map[uint64]item[V]),
		}
	}

	go c.evictionWorker(ctx, cleanupInterval)
	return c, nil
}

// calculateKeyHash builds a deterministic cache key directly from passed key.
func (c *TTLCache[V]) calculateKeyHash(key string) uint64 {
	return maphash.String(c.seed, key)
}

// getShard returns the shard for a precomputed key hash.
func (c *TTLCache[V]) getShard(keyHash uint64) *shard[V] {
	shardKey := maphash.Comparable(c.seed, keyHash)
	return c.shards[shardKey&shardMask]
}

// Load retrieves an item. Zero-allocation on the hot path.
// If the key is not found or already exired, the second return value will be false.
func (c *TTLCache[V]) Load(key string) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.RLock()
	itm, found := shard.items[keyHash]
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
		if currentItm, stillExists := shard.items[keyHash]; stillExists && now.After(currentItm.expires) {
			delete(shard.items, keyHash)
			shard.size--
		}
		shard.mu.Unlock()

		var zero V
		return zero, false
	}

	return itm.value, true
}

// Store inserts or updates an item with a specific TTL.
func (c *TTLCache[V]) Store(key string, value V) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	if _, exists := shard.items[keyHash]; !exists {
		shard.size++
	}
	shard.items[keyHash] = item[V]{
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
	shard.mu.Unlock()
}

// Delete removes an item explicitly.
func (c *TTLCache[V]) Delete(key string) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	if _, exists := shard.items[keyHash]; exists {
		delete(shard.items, keyHash)
		shard.size--
	}
	shard.mu.Unlock()
}

// LoadAndDelete atomically retrieves and removes a non-expired item.
func (c *TTLCache[V]) LoadAndDelete(key string) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	now := time.Now()

	shard.mu.Lock()
	itm, exists := shard.items[keyHash]
	if !exists {
		shard.mu.Unlock()
		var zero V
		return zero, false
	}

	delete(shard.items, keyHash)
	shard.size--
	shard.mu.Unlock()

	if now.After(itm.expires) {
		var zero V
		return zero, false
	}

	return itm.value, true
}

// Size returns the total number of items across all cache shards.
func (c *TTLCache[V]) Size() int64 {
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
func (c *TTLCache[V]) evictionWorker(ctx context.Context, interval time.Duration) {
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
