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
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkCacheTTL_Get(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	c.Set("hit", 42)

	b.Run("returns value for existing key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, ok := c.Get("hit")
			if !ok || v != 42 {
				b.Fatalf("unexpected get result: ok=%v value=%d", ok, v)
			}
		}
	})

	b.Run("returns miss for unknown key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, ok := c.Get("missing"); ok {
				b.Fatal("expected cache miss")
			}
		}
	})
}

func BenchmarkCacheTTL_Set(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := NewCacheTTL[int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	b.Run("updates same key", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			c.Set("stable", 1)
		}
	})

	b.Run("inserts unique keys", func(b *testing.B) {
		b.ReportAllocs()
		n := 0
		for b.Loop() {
			c.Set(strconv.Itoa(n), n)
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
			c.Set(k, n)
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
		c.Set(strconv.Itoa(i), i)
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
			c.Set(strconv.Itoa(i), i)
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
