package cache

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkCacheGet(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := New[string, int](ctx, time.Minute, time.Minute)
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

func BenchmarkCacheSet(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := New[string, int](ctx, time.Minute, time.Minute)
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

func BenchmarkCacheDelete(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := New[string, int](ctx, time.Minute, time.Minute)
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

func BenchmarkCacheSize(b *testing.B) {
	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	c, err := New[string, int](ctx, time.Minute, time.Minute)
	require.NoError(b, err)

	for i := 0; i < 10_000; i++ {
		c.Set(strconv.Itoa(i), i)
	}

	b.ReportAllocs()
	for b.Loop() {
		if got := c.Size(); got != 10_000 {
			b.Fatalf("unexpected size: got %d, want %d", got, 10_000)
		}
	}
}

func BenchmarkCacheEvictionEffectOnSize(b *testing.B) {
	b.Run("size reflects evicted entries", func(b *testing.B) {
		ctx, cancel := context.WithCancel(b.Context())
		defer cancel()

		c, err := New[string, int](ctx, 2*time.Millisecond, time.Millisecond)
		require.NoError(b, err)

		for i := 0; i < 1000; i++ {
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
