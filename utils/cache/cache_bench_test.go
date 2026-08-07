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
