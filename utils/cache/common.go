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
	"errors"
	"sync"
	"time"

	"golang.org/x/sys/cpu"
)

const (
	// ShardCount must be a power of 2 for bitwise modulo.
	// 256 is optimal for typical 16-64 core server deployments.
	shardCount = 256
	shardMask  = shardCount - 1
)

var (
	errInvalidContext         = errors.New("context must not be nil")
	errInvalidTTLInterval     = errors.New("ttl must be greater than 0")
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
	// Pad to CPU Arch dependnt bytes to prevent false sharing in L1 CPU cache.
	_ cpu.CacheLinePad
}
