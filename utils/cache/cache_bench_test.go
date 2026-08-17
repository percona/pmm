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
	"strconv"
	"testing"
)

func BenchmarkCache_Load(b *testing.B) {
	c := NewCache[int]()
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

func BenchmarkCache_Store(b *testing.B) {
	c := NewCache[int]()

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

func BenchmarkCache_Delete(b *testing.B) {
	c := NewCache[int]()

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

func BenchmarkCache_Size(b *testing.B) {
	c := NewCache[int]()

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

func BenchmarkCache_LoadAndDelete(b *testing.B) {
	c := NewCache[int]()

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

func BenchmarkCache_LoadOrStore_Miss(b *testing.B) {
	c := NewCache[int]()

	b.ResetTimer()
	for i := range b.N {
		c.LoadOrStore("k-"+strconv.Itoa(i), i)
	}
}

func BenchmarkCache_LoadOrStore_Hit(b *testing.B) {
	c := NewCache[int]()
	c.Store("k", 1)

	b.ResetTimer()
	for i := range b.N {
		c.LoadOrStore("k", i)
	}
}

func BenchmarkCache_LoadOrStore_ParallelContention(b *testing.B) {
	c := NewCache[int]()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.LoadOrStore("k", 1)
		}
	})
}
