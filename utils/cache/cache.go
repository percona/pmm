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

package cache

import (
	"hash/maphash"
)

// Cache is a high-performance, sharded, generic cache.
type Cache[V any] struct {
	shards [shardCount]*shard[uint64, V]
	seed   maphash.Seed
}

// NewCache initializes the cache.
func NewCache[V any]() *Cache[V] {
	c := &Cache[V]{
		seed: maphash.MakeSeed(),
	}

	for i := range shardCount {
		c.shards[i] = &shard[uint64, V]{
			items: make(map[uint64]item[V]),
		}
	}

	return c
}

// calculateKeyHash builds a deterministic cache key directly from passed key.
func (c *Cache[V]) calculateKeyHash(key string) uint64 {
	return maphash.String(c.seed, key)
}

// Get retrieves an item. Zero-allocation on the hot path.
func (c *Cache[V]) Get(key string) (V, bool) {
	keyHash := c.calculateKeyHash(key)
	// Zero-allocation hash via maphash
	shardKey := maphash.Comparable(c.seed, keyHash)
	shard := c.shards[shardKey&shardMask]

	shard.mu.RLock()
	itm, found := shard.items[keyHash]
	shard.mu.RUnlock()

	if !found {
		var zero V
		return zero, false
	}

	return itm.value, true
}

// Set inserts or updates an item with a specific TTL.
func (c *Cache[V]) Set(key string, value V) {
	keyHash := c.calculateKeyHash(key)
	shardKey := maphash.Comparable(c.seed, keyHash)
	shard := c.shards[shardKey&shardMask]

	shard.mu.Lock()
	if _, exists := shard.items[keyHash]; !exists {
		shard.size++
	}
	shard.items[keyHash] = item[V]{
		value: value,
	}
	shard.mu.Unlock()
}

// Delete removes an item explicitly.
func (c *Cache[V]) Delete(key string) {
	keyHash := c.calculateKeyHash(key)
	shardKey := maphash.Comparable(c.seed, keyHash)
	shard := c.shards[shardKey&shardMask]

	shard.mu.Lock()
	if _, exists := shard.items[keyHash]; exists {
		delete(shard.items, keyHash)
		shard.size--
	}
	shard.mu.Unlock()
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
