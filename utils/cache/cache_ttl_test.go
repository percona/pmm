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

func TestCacheTTL_Store_Load_Delete_StoresReadsAndRemovesValue(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

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

func TestCacheTTL_Load_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	got, ok := c.Load("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCacheTTL_Load_ReturnsMissAfterTTLExpiration(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 10*time.Millisecond, time.Second)
	require.NoError(t, err)

	c.Store("k", 7)
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Load("k")
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

func TestCacheTTL_EvictionWorker_RemovesExpiredItemsAndUpdatesSize(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 15*time.Millisecond, 5*time.Millisecond)
	require.NoError(t, err)

	c.Store("a", 1)
	c.Store("b", 2)
	if got := c.Size(); got != 2 {
		t.Fatalf("unexpected initial size: got %d, want %d", got, 2)
	}

	eventually(t, 300*time.Millisecond, 5*time.Millisecond, func() bool {
		return c.Size() == 0
	})

	if _, ok := c.Load("a"); ok {
		t.Fatal("expected key a to be evicted")
	}
	if _, ok := c.Load("b"); ok {
		t.Fatal("expected key b to be evicted")
	}
}

func TestCacheTTL_LoadAndDelete_ReturnsValueAndRemovesKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

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

func TestCacheTTL_LoadAndDelete_ReturnsMissForExpiredKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), 10*time.Millisecond, time.Second)
	require.NoError(t, err)

	c.Store("k", 42)
	time.Sleep(20 * time.Millisecond)

	got, ok := c.LoadAndDelete("k")
	if ok {
		t.Fatal("expected expired key to miss")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}

	if gotSize := c.Size(); gotSize != 0 {
		t.Fatalf("unexpected size after expired LoadAndDelete: got %d, want %d", gotSize, 0)
	}
}

func TestCacheTTL_LoadAndDelete_ReturnsMissForUnknownKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	got, ok := c.LoadAndDelete("missing")
	if ok {
		t.Fatal("expected missing key")
	}
	if got != 0 {
		t.Fatalf("unexpected zero value: got %d", got)
	}
}

func TestCacheTTL_LoadAndDelete_ReturnsStoredZeroValueAndTrue(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

	c.Store("k", 0)

	got, ok := c.LoadAndDelete("k")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != 0 {
		t.Fatalf("unexpected value: got %d, want %d", got, 0)
	}
}

func TestCacheTTL_LoadAndDelete_ReturnsMissOnSecondCallForSameKey(t *testing.T) {
	t.Parallel()

	c, err := NewCacheTTL[int](t.Context(), time.Second, 10*time.Millisecond)
	require.NoError(t, err)

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
