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
	"sync"
	"testing"
)

func TestNewCache_ReturnsCacheForValidInputs(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	if c == nil {
		t.Fatal("expected cache instance")
	}
}

func TestCache_CalculateCacheKey_ReturnsSameValueForSameInput(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	key := "Authorization:Bearer token"
	first := c.calculateKeyHash(key)
	second := c.calculateKeyHash(key)

	if first != second {
		t.Fatalf("expected stable key hash: first=%d second=%d", first, second)
	}
}

func TestCache_CalculateCacheKey_ReturnsDifferentValuesForDifferentInputs(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	first := c.calculateKeyHash("Authorization:Bearer token-a")
	second := c.calculateKeyHash("Authorization:Bearer token-b")

	if first == second {
		t.Fatal("expected different key hashes for different inputs")
	}
}

func TestCache_Store_Load_Delete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	c.Store("k", 42)

	got, ok := c.Load("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 42 {
		t.Fatalf("unexpected value: got %d, want %d", got, 42)
	}

	c.Delete("k")

	_, ok = c.Load("k")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestCache_Load_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	got, ok := c.Load("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCache_Load_ReturnsStoredZeroValue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	c.Store("k", 0)

	got, ok := c.Load("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 0 {
		t.Fatalf("unexpected value: got %d, want %d", got, 0)
	}
}

func TestCache_Size_TracksInsertUpdateDeleteAndMissingDelete(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	if got := c.Size(); got != 0 {
		t.Fatalf("unexpected size: got %d, want %d", got, 0)
	}

	c.Store("a", 1)
	if got := c.Size(); got != 1 {
		t.Fatalf("unexpected size after first insert: got %d, want %d", got, 1)
	}

	c.Store("a", 2)
	if got := c.Size(); got != 1 {
		t.Fatalf("unexpected size after update: got %d, want %d", got, 1)
	}

	c.Store("b", 3)
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

func TestCache_LoadAndDelete_ReturnsValueAndRemovesKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 42)

	got, ok := c.LoadAndDelete("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 42 {
		t.Fatalf("unexpected value: got %d, want %d", got, 42)
	}

	if gotSize := c.Size(); gotSize != 0 {
		t.Fatalf("unexpected size after LoadAndDelete: got %d, want %d", gotSize, 0)
	}

	_, exists := c.Load("k")
	if exists {
		t.Fatal("expected key to be deleted")
	}
}

func TestCache_LoadAndDelete_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	got, ok := c.LoadAndDelete("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCache_LoadAndDelete_ReturnsStoredZeroValueAndTrue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 0)

	got, ok := c.LoadAndDelete("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 0 {
		t.Fatalf("unexpected value: got %d, want %d", got, 0)
	}
}

func TestCache_LoadAndDelete_ReturnsMissOnSecondCallForSameKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 7)

	_, ok := c.LoadAndDelete("k")
	if !ok {
		t.Fatal("expected key to exist on first call")
	}

	got, ok := c.LoadAndDelete("k")
	if ok {
		t.Fatal("expected key to be missing on second call")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCache_LoadOrStore_StoresValueForMissingKeyAndReturnsLoadedFalse(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	got, loaded := c.LoadOrStore("k", 42)
	if loaded {
		t.Fatal("expected loaded to be false for missing key")
	}
	if got != 42 {
		t.Fatalf("unexpected value: got %d, want %d", got, 42)
	}

	stored, ok := c.Load("k")
	if !ok {
		t.Fatal("expected key to exist after LoadOrStore")
	}
	if stored != 42 {
		t.Fatalf("unexpected stored value: got %d, want %d", stored, 42)
	}
}

func TestCache_LoadOrStore_ReturnsExistingValueAndDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 7)

	got, loaded := c.LoadOrStore("k", 99)
	if !loaded {
		t.Fatal("expected loaded to be true for existing key")
	}
	if got != 7 {
		t.Fatalf("unexpected value: got %d, want %d", got, 7)
	}

	stored, ok := c.Load("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if stored != 7 {
		t.Fatalf("unexpected stored value: got %d, want %d", stored, 7)
	}
}

func TestCache_LoadOrStore_ConcurrentCallsOnSameKeyStoreOnlyOnce(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	const goroutines = 64

	var wg sync.WaitGroup
	results := make(chan struct {
		value  int
		loaded bool
	}, goroutines)

	for i := range goroutines {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			actual, loaded := c.LoadOrStore("k", v)
			results <- struct {
				value  int
				loaded bool
			}{
				value:  actual,
				loaded: loaded,
			}
		}(i)
	}

	wg.Wait()
	close(results)

	loadedFalse := 0
	var firstValue int
	firstStore := false
	for r := range results {
		if !r.loaded {
			loadedFalse++
		}
		if !firstStore {
			firstValue = r.value
			firstStore = true
			continue
		}
		if r.value != firstValue {
			t.Fatalf("all calls should observe the same stored value: got %d and %d", r.value, firstValue)
		}
	}

	if loadedFalse != 1 {
		t.Fatalf("expected exactly one store operation, got %d", loadedFalse)
	}
	if gotSize := c.Size(); gotSize != 1 {
		t.Fatalf("unexpected size after concurrent LoadOrStore: got %d, want %d", gotSize, 1)
	}
}

func TestCache_LoadOrStore_ReturnsStoredZeroValueForExistingKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 0)

	got, loaded := c.LoadOrStore("k", 99)
	if !loaded {
		t.Fatal("expected loaded to be true for existing key")
	}
	if got != 0 {
		t.Fatalf("unexpected value: got %d, want %d", got, 0)
	}
}
