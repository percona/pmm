// Copyright (C) 2024 Percona LLC
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

package pprof

import (
	"bytes"
	"compress/gzip"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHeap(t *testing.T) {
	t.Parallel()
	t.Run("Heap test", func(t *testing.T) {
		t.Parallel()
		heapBytes, err := Heap(true)
		assert.NoError(t, err)

		// read gzip
		reader, err := gzip.NewReader(bytes.NewBuffer(heapBytes))
		assert.NoError(t, err)

		var resB bytes.Buffer
		_, err = resB.ReadFrom(reader)
		assert.NoError(t, err)
		assert.NotEmpty(t, resB.Bytes())
	})
}

func TestProfile(t *testing.T) {
	t.Parallel()
	t.Run("Profile test", func(t *testing.T) {
		// Create a new context
		ctx := context.Background()
		profileBytes, err := Profile(ctx, 1*time.Second)

		assert.NoError(t, err)
		assert.NotEmpty(t, profileBytes)

		// read gzip
		reader, err := gzip.NewReader(bytes.NewBuffer(profileBytes))
		assert.NoError(t, err)

		var resB bytes.Buffer
		_, err = resB.ReadFrom(reader)
		assert.NoError(t, err)

		assert.NotEmpty(t, resB.Bytes())
	})

	t.Run("Profile break test", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		go func() {
			profileBytes, err := Profile(ctx, 30*time.Second)
			assert.Empty(t, profileBytes)
			assert.Error(t, err)
		}()

		go func() {
			time.Sleep(1 * time.Second)
			cancel()
		}()
	})
}

func TestTrace(t *testing.T) {
	t.Parallel()
	t.Run("Trace test", func(t *testing.T) {
		// Create a new context
		ctx := context.Background()
		traceBytes, err := Trace(ctx, 1*time.Second)

		assert.NoError(t, err)
		assert.NotEmpty(t, traceBytes)
	})

	t.Run("Trace break test", func(t *testing.T) {
		t.Parallel()
		// Create a new context
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		go func() {
			traceBytes, err := Trace(ctx, 30*time.Second)
			assert.Empty(t, traceBytes)
			assert.Error(t, err)
		}()

		go func() {
			time.Sleep(1 * time.Second)
			cancel()
		}()
	})
}
