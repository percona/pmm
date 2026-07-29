// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tailog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeLines(t *testing.T, s *Store, lines ...string) {
	t.Helper()
	for _, line := range lines {
		n, err := s.Write([]byte(line))
		require.NoError(t, err)
		assert.Equal(t, len(line), n)
	}
}

func TestStoreWriteAndGetLogs(t *testing.T) {
	t.Parallel()

	s := NewStore(3)
	writeLines(t, s, "a", "b", "c")

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(3), capacity)
	assert.Equal(t, []string{"a", "b", "c"}, logs)
}

func TestStoreRingOverflowKeepsMostRecent(t *testing.T) {
	t.Parallel()

	s := NewStore(3)
	// Writing more entries than the capacity should drop the oldest ones
	// while preserving insertion order of the survivors.
	writeLines(t, s, "a", "b", "c", "d", "e")

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(3), capacity)
	assert.Equal(t, []string{"c", "d", "e"}, logs)
}

func TestStorePartiallyFilled(t *testing.T) {
	t.Parallel()

	s := NewStore(5)
	writeLines(t, s, "a", "b")

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(5), capacity)
	// Only the written entries are returned, unused slots are skipped.
	assert.Equal(t, []string{"a", "b"}, logs)
}

func TestStoreZeroCapacity(t *testing.T) {
	t.Parallel()

	s := NewStore(0)

	n, err := s.Write([]byte("ignored"))
	require.NoError(t, err)
	assert.Equal(t, len("ignored"), n)

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(0), capacity)
	assert.Nil(t, logs)
}

func TestStoreStripsColorCodes(t *testing.T) {
	t.Parallel()

	s := NewStore(5)
	writeLines(
		t, s,
		"\x1b[31mred\x1b[0m",
		"\x1b[33myellow\x1b[0m",
		"\x1b[36mblue\x1b[0m",
		"\x1b[37mgray\x1b[0m",
	)

	logs, _ := s.GetLogs()
	assert.Equal(t, []string{"red", "yellow", "blue", "gray"}, logs)
}

func TestStoreResizeLarger(t *testing.T) {
	t.Parallel()

	s := NewStore(3)
	writeLines(t, s, "a", "b", "c")

	s.Resize(5)

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(5), capacity)
	assert.Equal(t, []string{"a", "b", "c"}, logs)

	// New capacity is honoured for subsequent writes.
	writeLines(t, s, "d", "e", "f")
	logs, _ = s.GetLogs()
	assert.Equal(t, []string{"b", "c", "d", "e", "f"}, logs)
}

func TestStoreResizeSmaller(t *testing.T) {
	t.Parallel()

	s := NewStore(4)
	writeLines(t, s, "a", "b", "c", "d")

	s.Resize(2)

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(2), capacity)
	// Shrinking keeps only the most recent entries that fit.
	assert.Equal(t, []string{"c", "d"}, logs)
}

func TestStoreResizeSameCapacityIsNoOp(t *testing.T) {
	t.Parallel()

	s := NewStore(3)
	writeLines(t, s, "a", "b", "c")

	s.Resize(3)

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(3), capacity)
	assert.Equal(t, []string{"a", "b", "c"}, logs)
}

func TestStoreResizeToZero(t *testing.T) {
	t.Parallel()

	s := NewStore(3)
	writeLines(t, s, "a", "b", "c")

	s.Resize(0)

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(0), capacity)
	assert.Nil(t, logs)

	// Writes are silently dropped once capacity is zero.
	n, err := s.Write([]byte("x"))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestStoreResizeFromZero(t *testing.T) {
	t.Parallel()

	s := NewStore(0)
	writeLines(t, s, "dropped")

	s.Resize(2)

	logs, capacity := s.GetLogs()
	assert.Equal(t, uint(2), capacity)
	assert.Empty(t, logs)

	writeLines(t, s, "a", "b")
	logs, _ = s.GetLogs()
	assert.Equal(t, []string{"a", "b"}, logs)
}
