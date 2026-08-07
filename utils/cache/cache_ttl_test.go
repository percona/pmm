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
	if c == nil {
		t.Fatal("expected cache instance")
	}
}

func TestCacheTTL_CalculateCacheKey_ReturnsSameValueForSameInput(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	key := "Authorization:Bearer token"
	first := c.calculateKeyHash(key)
	second := c.calculateKeyHash(key)

	if first != second {
		t.Fatalf("expected stable key hash: first=%d second=%d", first, second)
	}
}

func TestCacheTTL_CalculateCacheKey_ReturnsDifferentValuesForDifferentInputs(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	first := c.calculateKeyHash("Authorization:Bearer token-a")
	second := c.calculateKeyHash("Authorization:Bearer token-b")

	if first == second {
		t.Fatal("expected different key hashes for different inputs")
	}
}

func TestCacheTTL_Set_Get_Delete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

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

func TestCacheTTL_Get_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	got, ok := c.Get("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCacheTTL_Get_ReturnsMissAfterTTLExpiration(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 10*time.Millisecond, time.Second)
	require.NoError(t, err)

	c.Set("k", 7)
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get("k")
	if ok {
		t.Fatal("expected expired key to miss")
	}
}

func TestCacheTTL_Size_TracksInsertUpdateDeleteAndMissingDelete(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

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

func TestCacheTTL_EvictionWorker_RemovesExpiredItemsAndUpdatesSize(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 15*time.Millisecond, 5*time.Millisecond)
	require.NoError(t, err)

	c.Set("a", 1)
	c.Set("b", 2)
	if got := c.Size(); got != 2 {
		t.Fatalf("unexpected initial size: got %d, want %d", got, 2)
	}

	eventually(t, 300*time.Millisecond, 5*time.Millisecond, func() bool {
		return c.Size() == 0
	})

	if _, ok := c.Get("a"); ok {
		t.Fatal("expected key a to be evicted")
	}
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected key b to be evicted")
	}
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

	t.Fatal("condition was not met before timeout")
}
