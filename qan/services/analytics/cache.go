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

package analytics

import (
	"hash/fnv"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	resultCacheTTL  = 10 * time.Second
	maxCacheEntries = 10000
)

type cacheEntry struct {
	val any
	exp time.Time
}

// resultCache is a small TTL cache for read responses. The short TTL bounds
// staleness; "now"-relative windows naturally vary the key and so are not served stale.
type resultCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func newResultCache(ttl time.Duration) *resultCache {
	return &resultCache{entries: make(map[string]cacheEntry), ttl: ttl}
}

func (c *resultCache) get(key string) (any, bool) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || !e.exp.After(now) {
		return nil, false
	}
	return e.val, true
}

func (c *resultCache) set(key string, val any) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= maxCacheEntries {
		for k, e := range c.entries {
			if !e.exp.After(now) {
				delete(c.entries, k)
			}
		}
	}
	c.entries[key] = cacheEntry{val: val, exp: now.Add(c.ttl)}
}

// cacheKey hashes a request message into a cache key, or returns "" to skip caching.
func cacheKey(prefix string, msg proto.Message) string {
	b, err := proto.Marshal(msg)
	if err != nil {
		return ""
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return prefix + ":" + strconv.FormatUint(h.Sum64(), 16)
}
