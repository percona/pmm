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
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCache_ReturnsCacheForValidInputs(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	require.NotNil(t, c, "expected cache instance")
}

func TestCache_CalculateCacheKey_ReturnsSameValueForSameInput(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	key := "Authorization:Bearer token"
	first := c.calculateKeyHash(key)
	second := c.calculateKeyHash(key)

	require.Equal(t, first, second, "expected stable key hash")
}

func TestCache_CalculateCacheKey_ReturnsDifferentValuesForDifferentInputs(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	first := c.calculateKeyHash("Authorization:Bearer token-a")
	second := c.calculateKeyHash("Authorization:Bearer token-b")

	require.NotEqual(t, first, second, "expected different key hashes for different inputs")
}

func TestCache_Store_Load_Delete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	c.Store("k", 42)

	got, ok := c.Load("k")
	require.True(t, ok, "expected key to exist")
	require.Equal(t, 42, got, "unexpected value")

	c.Delete("k")

	_, ok = c.Load("k")
	require.False(t, ok, "expected key to be deleted")
}

func TestCache_Load_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	got, ok := c.Load("missing")
	require.False(t, ok, "expected missing key")
	require.Zero(t, got, "unexpected zero value")
}

func TestCache_Load_ReturnsStoredZeroValue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	c.Store("k", 0)

	got, ok := c.Load("k")
	require.True(t, ok, "expected key to exist")
	require.Zero(t, got, "unexpected value")
}

func TestCache_Size_TracksInsertUpdateDeleteAndMissingDelete(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	require.Zero(t, c.Size(), "unexpected size")

	c.Store("a", 1)
	require.EqualValues(t, 1, c.Size(), "unexpected size after first insert")

	c.Store("a", 2)
	require.EqualValues(t, 1, c.Size(), "unexpected size after update")

	c.Store("b", 3)
	require.EqualValues(t, 2, c.Size(), "unexpected size after second insert")

	c.Delete("missing")
	require.EqualValues(t, 2, c.Size(), "unexpected size after deleting missing key")

	c.Delete("a")
	require.EqualValues(t, 1, c.Size(), "unexpected size after deleting existing key")
}

func TestCache_LoadAndDelete_ReturnsValueAndRemovesKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 42)

	got, ok := c.LoadAndDelete("k")
	require.True(t, ok, "expected key to exist")
	require.Equal(t, 42, got, "unexpected value")

	require.Zero(t, c.Size(), "unexpected size after LoadAndDelete")

	_, exists := c.Load("k")
	require.False(t, exists, "expected key to be deleted")
}

func TestCache_LoadAndDelete_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	got, ok := c.LoadAndDelete("missing")
	require.False(t, ok, "expected missing key")
	require.Zero(t, got, "unexpected zero value")
}

func TestCache_LoadAndDelete_ReturnsStoredZeroValueAndTrue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 0)

	got, ok := c.LoadAndDelete("k")
	require.True(t, ok, "expected key to exist")
	require.Zero(t, got, "unexpected value")
}

func TestCache_LoadAndDelete_ReturnsMissOnSecondCallForSameKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 7)

	_, ok := c.LoadAndDelete("k")
	require.True(t, ok, "expected key to exist on first call")

	got, ok := c.LoadAndDelete("k")
	require.False(t, ok, "expected key to be missing on second call")
	require.Zero(t, got, "unexpected zero value")
}

func TestCache_CompareAndDelete_PanickingPredicateDoesNotLeaveLockHeld(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 42)

	require.Panics(t, func() {
		c.CompareAndDelete("k", func(_ int) bool {
			panic("boom")
		})
	}, "expected predicate panic to propagate")

	got, ok := c.Load("k")
	require.True(t, ok, "entry should remain after panic")
	require.Equal(t, 42, got, "unexpected value after panic")

	deleted, removed := c.CompareAndDelete("k", func(v int) bool {
		return v == 42
	})
	require.True(t, removed, "expected follow-up delete to succeed")
	require.Equal(t, 42, deleted, "unexpected deleted value")
}

func TestCache_CompareAndDelete_ReentrantPredicateDoesNotDeadlockAndPreventsStaleDelete(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 1)

	type result struct {
		value int
		ok    bool
	}

	resultCh := make(chan result, 1)
	go func() {
		value, ok := c.CompareAndDelete("k", func(_ int) bool {
			c.Store("k", 2)
			return true
		})
		resultCh <- result{value: value, ok: ok}
	}()

	select {
	case r := <-resultCh:
		require.False(t, r.ok, "delete must fail when value changed after predicate evaluation")
		require.Zero(t, r.value, "unexpected value when delete fails")
	case <-time.After(2 * time.Second):
		t.Fatal("CompareAndDelete deadlocked with re-entrant predicate")
	}

	got, ok := c.Load("k")
	require.True(t, ok, "entry should still exist")
	require.Equal(t, 2, got, "re-entrant update should win")
}

func TestCache_LoadOrStore_StoresValueForMissingKeyAndReturnsLoadedFalse(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()

	got, loaded := c.LoadOrStore("k", 42)
	require.False(t, loaded, "expected loaded to be false for missing key")
	require.Equal(t, 42, got, "unexpected value")

	stored, ok := c.Load("k")
	require.True(t, ok, "expected key to exist after LoadOrStore")
	require.Equal(t, 42, stored, "unexpected stored value")
}

func TestCache_LoadOrStore_ReturnsExistingValueAndDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 7)

	got, loaded := c.LoadOrStore("k", 99)
	require.True(t, loaded, "expected loaded to be true for existing key")
	require.Equal(t, 7, got, "unexpected value")

	stored, ok := c.Load("k")
	require.True(t, ok, "expected key to exist")
	require.Equal(t, 7, stored, "unexpected stored value")
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
		require.Equal(t, firstValue, r.value, "all calls should observe the same stored value")
	}

	require.Equal(t, 1, loadedFalse, "expected exactly one store operation")
	require.EqualValues(t, 1, c.Size(), "unexpected size after concurrent LoadOrStore")
}

func TestCache_LoadOrStore_ConcurrentCallsOnExistingKeyReturnExistingValue(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 42)

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

	for r := range results {
		require.True(t, r.loaded, "all calls should report loaded=true when key exists")
		require.Equal(t, 42, r.value, "all calls should return the pre-existing value")
	}

	require.EqualValues(t, 1, c.Size(), "size should remain unchanged for existing key")
}

func TestCache_LoadOrStore_ReturnsStoredZeroValueForExistingKey(t *testing.T) {
	t.Parallel()

	c := NewCache[int]()
	c.Store("k", 0)

	got, loaded := c.LoadOrStore("k", 99)
	require.True(t, loaded, "expected loaded to be true for existing key")
	require.Zero(t, got, "unexpected value")
}
