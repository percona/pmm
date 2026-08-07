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
type Cache[K comparable, V any] struct {
    shards [shardCount]*shard[K, V]
    seed   maphash.Seed
}

// NewCache initializes the cache.
func NewCache[K comparable, V any]() *Cache[K, V] {
    c := &Cache[K, V]{
        seed: maphash.MakeSeed(),
    }

    for i := range shardCount {
        c.shards[i] = &shard[K, V]{
            items: make(map[K]item[V]),
        }
    }

    return c
}

// CalculateCacheKey builds a deterministic cache key directly from passed key.
func (c *Cache[K, V]) CalculateCacheKey(key string) uint64 {
    return maphash.String(c.seed, key)
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

    return itm.value, true
}

// Set inserts or updates an item with a specific TTL.
func (c *Cache[K, V]) Set(key K, value V) {
    hash := maphash.Comparable(c.seed, key)
    shard := c.shards[hash&shardMask]

    shard.mu.Lock()
    if _, exists := shard.items[key]; !exists {
        shard.size++
    }
    shard.items[key] = item[V]{
        value: value,
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
