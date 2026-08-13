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
	rev     uint64
}

// shard contains the actual map and a lock, padded to prevent false sharing.
type shard[V any] struct {
	mu    sync.RWMutex
	items map[uint64]item[V]
	size  int64
	rev   uint64
	// Pad to CPU Arch dependant bytes to prevent false sharing in L1 CPU cache.
	_ cpu.CacheLinePad
}
