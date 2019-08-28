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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/models"
)

func TestService(t *testing.T) {
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	s := New(configDir)
	settings := &models.Settings{
		DataRetention: 3 * 24 * time.Hour,
	}

	for _, tmpl := range templates.Templates() {
		if tmpl.Name() == "" {
			continue
		}

		tmpl := tmpl
		t.Run(tmpl.Name(), func(t *testing.T) {
			expected, err := ioutil.ReadFile(filepath.Join(configDir, tmpl.Name()+".ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}

func TestServiceDevContainer(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)

	if os.Getenv("DEVCONTAINER") == "" {
		t.Skip("can be tested only inside devcontainer")
	}

	s := New("/etc/supervisord.d")
	require.NotEmpty(t, s.supervisorctlPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx)

	t.Run("UpdateConfiguration", func(t *testing.T) {
		// restore original files after test
		originals := make(map[string][]byte)
		matches, err := filepath.Glob("/etc/supervisord.d/*.ini")
		require.NoError(t, err)
		for _, m := range matches {
			b, err := ioutil.ReadFile(m) //nolint:gosec
			require.NoError(t, err)
			originals[m] = b
		}
		defer func() {
			for name, b := range originals {
				err = ioutil.WriteFile(name, b, 0)
				assert.NoError(t, err)
			}
		}()

		settings := &models.Settings{
			DataRetention: 24 * time.Hour,
		}

		b, err := s.marshalConfig(templates.Lookup("prometheus"), settings)
		require.NoError(t, err)
		changed, err := s.saveConfigAndReload("prometheus", b)
		require.NoError(t, err)
		assert.True(t, changed)
		changed, err = s.saveConfigAndReload("prometheus", b)
		require.NoError(t, err)
		assert.False(t, changed)

		err = s.UpdateConfiguration(settings)
		require.NoError(t, err)
	})

	t.Run("StartUpdate", func(t *testing.T) {
		require.Equal(t, false, s.UpdateRunning())

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
