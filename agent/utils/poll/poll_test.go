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

package poll

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollUntilContextTimeout(t *testing.T) {
	t.Parallel()

	t.Run("immediate success", func(t *testing.T) {
		t.Parallel()

		calls := 0
		err := PollUntilContextTimeout(t.Context(), time.Millisecond, func(context.Context) (bool, error) {
			calls++
			return true, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("success after retries", func(t *testing.T) {
		t.Parallel()

		calls := 0
		err := PollUntilContextTimeout(t.Context(), time.Millisecond, func(context.Context) (bool, error) {
			calls++
			return calls == 3, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 3, calls)
	})

	t.Run("condition error", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("boom")
		err := PollUntilContextTimeout(t.Context(), time.Millisecond, func(context.Context) (bool, error) {
			return false, expected
		})
		assert.ErrorIs(t, err, expected)
	})

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := PollUntilContextTimeout(ctx, time.Millisecond, func(context.Context) (bool, error) {
			return false, nil
		})
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("context timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		err := PollUntilContextTimeout(ctx, 5*time.Millisecond, func(context.Context) (bool, error) {
			return false, nil
		})
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
