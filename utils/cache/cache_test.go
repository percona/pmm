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

func TestNew_ReturnsErrorForInvalidInputs(t *testing.T) {
	t.Parallel()

	_, err := New[string, int](nil, time.Second, time.Second) //nolint:staticcheck
	require.Error(t, err)

	_, err = New[string, int](t.Context(), time.Second, 0)
	require.Error(t, err)
}

func TestNew_ReturnsCacheForValidInputs(t *testing.T) {
	t.Parallel()

	c, err := New[string, int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)
	if c == nil {
		t.Fatal("expected cache instance")
	}
}

func TestSetGetDelete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c, err := New[string, int](t.Context(), time.Second, 10*time.Millisecond)
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

func TestGet_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := New[string, int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	got, ok := c.Get("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestGet_ReturnsMissAfterTTLExpiration(t *testing.T) {
	t.Parallel()

	c, err := New[string, int](t.Context(), 10*time.Millisecond, time.Second)
	require.NoError(t, err)

	c.Set("k", 7)
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get("k")
	if ok {
		t.Fatal("expected expired key to miss")
	}
}

func TestSize_TracksInsertUpdateDeleteAndMissingDelete(t *testing.T) {
	t.Parallel()

	c, err := New[string, int](t.Context(), time.Second, 10*time.Millisecond)
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

func TestEvictionWorker_RemovesExpiredItemsAndUpdatesSize(t *testing.T) {
	t.Parallel()

	c, err := New[string, int](t.Context(), 15*time.Millisecond, 5*time.Millisecond)
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
