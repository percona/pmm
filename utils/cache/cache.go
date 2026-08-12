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

package cache

import (
	"hash/maphash"
	"iter"
)

// Cache is a high-performance, sharded, generic cache.
type Cache[V any] struct {
	shards [shardCount]*shard[V]
	seed   maphash.Seed
}

// NewCache initializes the cache.
func NewCache[V any]() *Cache[V] {
	c := &Cache[V]{
		seed: maphash.MakeSeed(),
	}

	for i := range shardCount {
		c.shards[i] = &shard[V]{
			items: make(map[uint64]item[V]),
		}
	}

	return c
}

// calculateKeyHash builds a deterministic cache key directly from passed key.
func (c *Cache[V]) calculateKeyHash(key string) uint64 {
	return maphash.String(c.seed, key)
}

// getShard returns the shard for a precomputed key hash.
func (c *Cache[V]) getShard(keyHash uint64) *shard[V] {
	shardKey := maphash.Comparable(c.seed, keyHash)
	return c.shards[shardKey&shardMask]
}

// Load retrieves an item. Zero-allocation on the hot path.
// If the key is not found, the second return value will be false.
func (c *Cache[V]) Load(key string) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.RLock()
	itm, found := shard.items[keyHash]
	shard.mu.RUnlock()

	if !found {
		var zero V
		return zero, false
	}

	return itm.value, true
}

// IterAll returns an iterator over all items in the cache.
// Deadlock Risk: Because the yield function executes while the shard's read lock is held,
// you must not write to the cache inside the loop.
// Calling a method that acquires a Lock() on the same shard will cause a deadlock.
func (c *Cache[V]) IterAll() iter.Seq2[uint64, V] {
	return func(yield func(uint64, V) bool) {
		for i := range shardCount {
			s := c.shards[i]

			s.mu.RLock()
			for k, item := range s.items {
				if !yield(k, item.value) {
					s.mu.RUnlock()
					return
				}
			}
			s.mu.RUnlock()
		}
	}
}

// Store inserts or updates an item. If the key already exists, it will be overwritten.
func (c *Cache[V]) Store(key string, value V) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	if _, exists := shard.items[keyHash]; !exists {
		shard.size++
	}
	shard.items[keyHash] = item[V]{
		value: value,
	}
	shard.mu.Unlock()
}

// LoadOrStore returns the existing value for a key if present.
// Otherwise, it stores and returns the given value. The loaded result is true when the key already existed.
func (c *Cache[V]) LoadOrStore(key string, value V) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	itm, exists := shard.items[keyHash]
	if exists {
		shard.mu.Unlock()
		return itm.value, true
	}

	shard.size++
	shard.items[keyHash] = item[V]{
		value: value,
	}
	shard.mu.Unlock()

	return value, false
}

// Delete removes an item explicitly.
func (c *Cache[V]) Delete(key string) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	if _, exists := shard.items[keyHash]; exists {
		delete(shard.items, keyHash)
		shard.size--
	}
	shard.mu.Unlock()
}

// LoadAndDelete atomically retrieves and removes an item.
func (c *Cache[V]) LoadAndDelete(key string) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	itm, exists := shard.items[keyHash]
	if exists {
		delete(shard.items, keyHash)
		shard.size--
	}
	shard.mu.Unlock()

	if !exists {
		var zero V
		return zero, false
	}

	return itm.value, true
}

// CompareAndDelete atomically removes an item only when match returns true for the current value.
func (c *Cache[V]) CompareAndDelete(key string, match func(V) bool) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	shard := c.getShard(keyHash)

	shard.mu.Lock()
	itm, exists := shard.items[keyHash]
	if !exists || !match(itm.value) {
		shard.mu.Unlock()
		var zero V
		return zero, false
	}
	delete(shard.items, keyHash)
	shard.size--
	shard.mu.Unlock()

	return itm.value, true
}

// Size returns the total number of items across all cache shards.
func (c *Cache[V]) Size() int64 {
	var total int64
	for i := range shardCount {
		shard := c.shards[i]
		shard.mu.RLock()
		total += shard.size
		shard.mu.RUnlock()
	}

	return total
}
