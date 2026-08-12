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
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCacheTTL_ReturnsErrorForInvalidInputs(t *testing.T) {
	t.Parallel()

	_, err := NewCacheTTL[int](nil, time.Second, time.Second) //nolint:staticcheck
	require.ErrorIs(t, err, errInvalidContext)

	_, err = NewCacheTTL[int](t.Context(), 0, time.Second)
	require.ErrorIs(t, err, errInvalidTTLInterval)

	_, err = NewCacheTTL[int](t.Context(), time.Second, 0)
	require.ErrorIs(t, err, errInvalidCleanupInterval)
}

func TestNewCacheTTL_ReturnsCacheForValidInputs(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, c, "expected cache instance")
}

func TestCacheTTL_CalculateCacheKey_ReturnsSameValueForSameInput(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	key := "Authorization:Bearer token"
	first := c.calculateKeyHash(key)
	second := c.calculateKeyHash(key)

	require.Equal(t, first, second, "expected stable key hash")
}

func TestCacheTTL_CalculateCacheKey_ReturnsDifferentValuesForDifferentInputs(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	first := c.calculateKeyHash("Authorization:Bearer token-a")
	second := c.calculateKeyHash("Authorization:Bearer token-b")

	require.NotEqual(t, first, second, "expected different key hashes for different inputs")
}

func TestCacheTTL_Store_Load_Delete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	c.Store("k", 42)

	got, ok := c.Load("k")
	require.True(t, ok, "expected key to exist")
	require.Equal(t, 42, got, "unexpected value")

	c.Delete("k")

	_, ok = c.Load("k")
	require.False(t, ok, "expected key to be deleted")
}

func TestCacheTTL_Load_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	got, ok := c.Load("missing")
	require.False(t, ok, "expected missing key")
	require.Zero(t, got, "unexpected zero value")
}

func TestCacheTTL_Load_ReturnsMissAfterTTLExpiration(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 10*time.Millisecond, time.Second)
	require.NoError(t, err)

	c.Store("k", 7)
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Load("k")
	require.False(t, ok, "expected expired key to miss")
}

func TestCacheTTL_Size_TracksInsertUpdateDeleteAndMissingDelete(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

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

func TestCacheTTL_EvictionWorker_RemovesExpiredItemsAndUpdatesSize(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 15*time.Millisecond, 5*time.Millisecond)
	require.NoError(t, err)

	c.Store("a", 1)
	c.Store("b", 2)
	require.EqualValues(t, 2, c.Size(), "unexpected initial size")

	eventually(t, 300*time.Millisecond, 5*time.Millisecond, func() bool {
		return c.Size() == 0
	})

	_, okA := c.Load("a")
	require.False(t, okA, "expected key a to be evicted")
	_, okB := c.Load("b")
	require.False(t, okB, "expected key b to be evicted")
}

func TestCacheTTL_LoadAndDelete_ReturnsValueAndRemovesKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	c.Store("k", 42)

	got, ok := c.LoadAndDelete("k")
	require.True(t, ok, "expected key to exist")
	require.Equal(t, 42, got, "unexpected value")

	require.Zero(t, c.Size(), "unexpected size after LoadAndDelete")

	_, exists := c.Load("k")
	require.False(t, exists, "expected key to be deleted")
}

func TestCacheTTL_LoadAndDelete_ReturnsMissForExpiredKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 10*time.Millisecond, time.Second)
	require.NoError(t, err)

	c.Store("k", 42)
	time.Sleep(20 * time.Millisecond)

	got, ok := c.LoadAndDelete("k")
	require.False(t, ok, "expected expired key to miss")
	require.Zero(t, got, "unexpected zero value")

	require.Zero(t, c.Size(), "unexpected size after expired LoadAndDelete")
}

func TestCacheTTL_LoadAndDelete_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	got, ok := c.LoadAndDelete("missing")
	require.False(t, ok, "expected missing key")
	require.Zero(t, got, "unexpected zero value")
}

func TestCacheTTL_LoadAndDelete_ReturnsStoredZeroValueAndTrue(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	c.Store("k", 0)

	got, ok := c.LoadAndDelete("k")
	require.True(t, ok, "expected key to exist")
	require.Zero(t, got, "unexpected value")
}

func TestCacheTTL_LoadAndDelete_ReturnsMissOnSecondCallForSameKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	c.Store("k", 7)

	_, ok := c.LoadAndDelete("k")
	require.True(t, ok, "expected key to exist on first call")

	got, ok := c.LoadAndDelete("k")
	require.False(t, ok, "expected key to be missing on second call")
	require.Zero(t, got, "unexpected zero value")
}

func eventually(t *testing.T, timeout, interval time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(interval)
	}

	require.FailNow(t, "condition was not met before timeout")
}
