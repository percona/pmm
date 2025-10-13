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

package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/vmproxy/proxy"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	t.Run("shall run proxy with no error", func(t *testing.T) {
		t.Parallel()
		err := runProxy(flags{}, func(_ proxy.Config) error {
			return nil
		})

		require.NoError(t, err)
	})
}
