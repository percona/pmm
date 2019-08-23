// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package supervisord

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestService(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)

	if os.Getenv("DEVCONTAINER") == "" {
		t.Skip("can be tested only inside devcontainer")
	}

	s := New()
	require.NotEmpty(t, s.supervisorctlPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx)

	assert.Equal(t, false, s.UpdateRunning())

	t.Run("StartUpdate", func(t *testing.T) {
		offset, err := s.StartUpdate()
		require.NoError(t, err)
		assert.Zero(t, offset)

		assert.True(t, s.UpdateRunning())

		_, err = s.StartUpdate()
		assert.Equal(t, status.Errorf(codes.FailedPrecondition, "Update is already running."), err)

		// get logs as often as possible to increase a chance for race detector to spot something
		for {
			lines, newOffset, err := s.UpdateLog(offset)
			require.NoError(t, err)
			if newOffset == offset {
				assert.Empty(t, lines, "lines:\n%s", strings.Join(lines, "\n"))
				if s.UpdateRunning() {
					continue
				}
				break
			}

			assert.NotEmpty(t, lines)
			t.Logf("%s", strings.Join(lines, "\n"))

			assert.NotZero(t, newOffset)
			assert.True(t, newOffset > offset, "expected newOffset = %d > offset = %d", newOffset, offset)
			offset = newOffset
		}

		// extra checks that we did not miss `pmp-update -perform` self-update and restart by supervisord
		const delay = 50 * time.Millisecond
		const wait = 2 * time.Second
		for i := 0; i < int(delay/wait); i++ {
			time.Sleep(200 * time.Millisecond)
			assert.False(t, s.UpdateRunning())
			lines, newOffset, err := s.UpdateLog(offset)
			require.NoError(t, err)
			assert.Empty(t, lines, "lines:\n%s", strings.Join(lines, "\n"))
			assert.Equal(t, offset, newOffset, "offset = %d, newOffset = %d", offset, newOffset)
		}
	})
}
