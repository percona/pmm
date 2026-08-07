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

func BenchmarkCache_Get(b *testing.B) {
	c := NewCache[int]()
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

func BenchmarkCache_Set(b *testing.B) {
	c := NewCache[int]()

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

func BenchmarkCache_Delete(b *testing.B) {
	c := NewCache[int]()

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

func BenchmarkCache_Size(b *testing.B) {
	c := NewCache[int]()

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
