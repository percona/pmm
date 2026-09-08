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

package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// removerFunc adapts a function to ServiceAccountRemover.
type removerFunc func(ctx context.Context, nodeName string, force bool) (string, error)

func (f removerFunc) DeleteServiceAccount(ctx context.Context, nodeName string, force bool) (string, error) {
	return f(ctx, nodeName, force)
}

func TestRemoveNodeServiceAccount(t *testing.T) {
	t.Parallel()

	t.Run("the cleanup outlives the request and carries a deadline", func(t *testing.T) {
		t.Parallel()

		// A client which gave up right after the Node was removed must not leave the account behind.
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		called := false
		warning, err := RemoveNodeServiceAccount(ctx, removerFunc(func(ctx context.Context, nodeName string, force bool) (string, error) {
			called = true
			require.NoError(t, ctx.Err())
			_, ok := ctx.Deadline()
			assert.True(t, ok, "the cleanup has to give up on an unresponsive Grafana")
			assert.Equal(t, "test-node", nodeName)
			assert.True(t, force)

			return "the account is kept", nil
		}), "test-node", true)

		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, "the account is kept", warning)
	})

	t.Run("the failure of Grafana is reported", func(t *testing.T) {
		t.Parallel()

		errGrafana := errors.New("connection refused")
		_, err := RemoveNodeServiceAccount(t.Context(), removerFunc(func(context.Context, string, bool) (string, error) {
			return "", errGrafana
		}), "test-node", false)

		assert.ErrorIs(t, err, errGrafana)
	})
}
