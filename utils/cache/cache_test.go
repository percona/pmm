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
	"testing"
)

func TestNewCache_ReturnsCacheForValidInputs(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()
	if c == nil {
		t.Fatal("expected cache instance")
	}
}

func TestCache_CalculateCacheKey_ReturnsSameValueForSameInput(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()

	key := "Authorization:Bearer token"
	first := c.CalculateCacheKey(key)
	second := c.CalculateCacheKey(key)

	if first != second {
		t.Fatalf("expected stable key hash: first=%d second=%d", first, second)
	}
}

func TestCache_CalculateCacheKey_ReturnsDifferentValuesForDifferentInputs(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()

	first := c.CalculateCacheKey("Authorization:Bearer token-a")
	second := c.CalculateCacheKey("Authorization:Bearer token-b")

	if first == second {
		t.Fatal("expected different key hashes for different inputs")
	}
}

func TestCache_Set_Get_Delete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()

	c.Set("k", 42)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 42 {
		t.Fatalf("unexpected value: got %d, want %d", got, 42)
	}

	c.Delete("k")

	_, ok = c.Get("k")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestCache_Get_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()

	got, ok := c.Get("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCache_Get_ReturnsStoredZeroValue(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()

	c.Set("k", 0)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 0 {
		t.Fatalf("unexpected value: got %d, want %d", got, 0)
	}
}

func TestCache_Size_TracksInsertUpdateDeleteAndMissingDelete(t *testing.T) {
	t.Parallel()

	c := NewCache[string, int]()

	if got := c.Size(); got != 0 {
		t.Fatalf("unexpected size: got %d, want %d", got, 0)
	}

	c.Set("a", 1)
	if got := c.Size(); got != 1 {
		t.Fatalf("unexpected size after first insert: got %d, want %d", got, 1)
	}

	c.Set("a", 2)
	if got := c.Size(); got != 1 {
		t.Fatalf("unexpected size after update: got %d, want %d", got, 1)
	}

	c.Set("b", 3)
	if got := c.Size(); got != 2 {
		t.Fatalf("unexpected size after second insert: got %d, want %d", got, 2)
	}

	c.Delete("missing")
	if got := c.Size(); got != 2 {
		t.Fatalf("unexpected size after deleting missing key: got %d, want %d", got, 2)
	}

	c.Delete("a")
	if got := c.Size(); got != 1 {
		t.Fatalf("unexpected size after deleting existing key: got %d, want %d", got, 1)
	}
}

