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
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkCacheTTL_Load(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	c.Store("hit", 42)

	b.Run("returns value for existing key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, ok := c.Load("hit")
			if !ok || v != 42 {
				b.Fatalf("unexpected Load result: ok=%v value=%d", ok, v)
			}
		}
	})

	b.Run("returns miss for unknown key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, ok := c.Load("missing"); ok {
				b.Fatal("expected cache miss")
			}
		}
	})
}

func BenchmarkCacheTTL_Store(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	b.Run("updates same key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			c.Store("stable", 1)
		}
	})

	b.Run("inserts unique keys", func(b *testing.B) {
		b.ReportAllocs()
		n := 0
		for b.Loop() {
			c.Store(strconv.Itoa(n), n)
			n++
		}
	})
}

func BenchmarkCacheTTL_Delete(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	b.Run("deletes existing key", func(b *testing.B) {
		b.ReportAllocs()
		n := 0
		for b.Loop() {
			k := strconv.Itoa(n)
			c.Store(k, n)
			c.Delete(k)
			n++
		}
	})

	b.Run("deletes missing key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			c.Delete("missing")
		}
	})
}

func BenchmarkCacheTTL_Size(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	for i := range 10_000 {
		c.Store(strconv.Itoa(i), i)
	}

	b.ReportAllocs()
	for b.Loop() {
		if got := c.Size(); got != 10_000 {
			b.Fatalf("unexpected size: got %d, want %d", got, 10_000)
		}
	}
}

func BenchmarkCacheTTL_EvictionEffectOnSize(b *testing.B) {
	b.Run("size reflects evicted entries", func(b *testing.B) {
		ctx, cancel := context.WithCancel(b.Context())
		defer cancel()

		c, err := NewCacheTTL[int](ctx, 2*time.Millisecond, time.Millisecond)
		require.NoError(b, err)

		for i := range 1000 {
			c.Store(strconv.Itoa(i), i)
		}

		deadline := time.Now().Add(300 * time.Millisecond)
		for c.Size() != 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		if c.Size() != 0 {
			b.Fatalf("expected empty cache after eviction, got %d", c.Size())
		}

		b.ReportAllocs()
		for b.Loop() {
			if got := c.Size(); got != 0 {
				b.Fatalf("unexpected size after eviction: got %d", got)
			}
		}
	})
}

func BenchmarkCacheTTL_LoadAndDelete(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	b.Run("loads and deletes existing key", func(b *testing.B) {
		b.ReportAllocs()
		n := 0
		for b.Loop() {
			k := strconv.Itoa(n)
			c.Store(k, n)
			v, ok := c.LoadAndDelete(k)
			if !ok || v != n {
				b.Fatalf("unexpected load and delete result: ok=%v value=%d want=%d", ok, v, n)
			}
			n++
		}
	})

	b.Run("returns miss for unknown key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, ok := c.LoadAndDelete("missing"); ok {
				b.Fatal("expected cache miss")
			}
		}
	})
}
