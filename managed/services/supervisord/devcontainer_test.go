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

package supervisord

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// TODO move tests to other files and remove this one.
func TestDevContainer(t *testing.T) {
	t.Run("UpdateConfiguration", func(t *testing.T) {
		// logrus.SetLevel(logrus.DebugLevel)
		vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		s := New("/etc/supervisord.d", &models.Params{VMParams: vmParams, PGParams: &models.PGParams{}, HAParams: &models.HAParams{}})
		require.NotEmpty(t, s.supervisorctlPath)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go s.Run(ctx)

		// restore original files after test
		originals := make(map[string][]byte)
		matches, err := filepath.Glob("/etc/supervisord.d/*.ini")
		require.NoError(t, err)
		for _, m := range matches {
			b, err := os.ReadFile(m) //nolint:gosec
			require.NoError(t, err)
			originals[m] = b
		}
		defer func() {
			for name, b := range originals {
				err = os.WriteFile(name, b, 0)
				assert.NoError(t, err)
			}
			// force update supervisor config
			_, err := s.supervisorctl("update")
			assert.NoError(t, err)
		}()

		settings := &models.Settings{
			DataRetention: 3600 * time.Hour,
		}

		b, err := s.marshalConfig(templates.Lookup("victoriametrics"), settings, nil)
		require.NoError(t, err)
		changed, err := s.saveConfigAndReload("victoriametrics", b)
		require.NoError(t, err)
		assert.True(t, changed)
		changed, err = s.saveConfigAndReload("victoriametrics", b)
		require.NoError(t, err)
		assert.False(t, changed)

		err = s.UpdateConfiguration(settings, nil)
		require.NoError(t, err)
	})

	//t.Run("Update", func(t *testing.T) {
	//	// This test can be run only once as it breaks assumptions of other tests.
	//	// It also should be the last test in devcontainer.
	//	if ok, _ := strconv.ParseBool(os.Getenv("PMM_TEST_RUN_UPDATE")); !ok {
	//		t.Skip("skipping update test")
	//	}
	//
	//	// logrus.SetLevel(logrus.DebugLevel)
	//	checker := NewPMMUpdateChecker()
	//	vmParams := &models.VictoriaMetricsParams{}
	//	s := New("/etc/supervisord.d", &models.Params{VMParams: vmParams, PGParams: &models.PGParams{}, HAParams: &models.HAParams{}}, gRPCMessageMaxSize)
	//	require.NotEmpty(t, s.supervisorctlPath)
	//
	//	ctx, cancel := context.WithCancel(context.Background())
	//	defer cancel()
	//	go s.Run(ctx)
	//
	//	require.Equal(t, false, s.UpdateRunning())
	//
	//	offset, err := s.StartUpdate()
	//	require.NoError(t, err)
	//	assert.Zero(t, offset)
	//
	//	assert.True(t, s.UpdateRunning())
	//
	//	_, err = s.StartUpdate()
	//	assert.Equal(t, status.Errorf(codes.FailedPrecondition, "Update is already running."), err)
	//
	//	// get logs as often as possible to increase a chance for race detector to spot something
	//	var lastLine string
	//	for {
	//		done := s.UpdateRunning()
	//		if done {
	//			// give supervisord a second to flush logs to file
	//			time.Sleep(time.Second)
	//		}
	//
	//		lines, newOffset, err := s.UpdateLog(offset)
	//		require.NoError(t, err)
	//		if newOffset == offset {
	//			assert.Empty(t, lines, "lines:\n%s", strings.Join(lines, "\n"))
	//			if done {
	//				continue
	//			}
	//			break
	//		}
	//
	//		assert.NotEmpty(t, lines)
	//		t.Logf("%s", strings.Join(lines, "\n"))
	//		lastLine = lines[len(lines)-1]
	//
	//		assert.NotZero(t, newOffset)
	//		assert.True(t, newOffset > offset, "expected newOffset = %d > offset = %d", newOffset, offset)
	//		offset = newOffset
	//	}
	//
	//	t.Logf("lastLine = %q", lastLine)
	//	assert.Contains(t, lastLine, "Waiting for Grafana dashboards update to finish...")
	//
	//	// extra checks that we did not miss `pmp-update -perform` self-update and restart by supervisord
	//	const wait = 3 * time.Second
	//	const delay = 200 * time.Millisecond
	//	for i := 0; i < int(wait/delay); i++ {
	//		time.Sleep(delay)
	//		require.False(t, s.UpdateRunning())
	//		lines, newOffset, err := s.UpdateLog(offset)
	//		require.NoError(t, err)
	//		require.Empty(t, lines, "lines:\n%s", strings.Join(lines, "\n"))
	//		require.Equal(t, offset, newOffset, "offset = %d, newOffset = %d", offset, newOffset)
	//	}
	//})
}
