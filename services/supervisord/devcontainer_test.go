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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/models"
)

func TestDevContainer(t *testing.T) {
	if os.Getenv("DEVCONTAINER") == "" {
		t.Skip("can be tested only inside devcontainer")
	}

	t.Run("Installed", func(t *testing.T) {
		checker := newPMMUpdateChecker(logrus.WithField("test", t.Name()))

		info := checker.installed()
		require.NotNil(t, info)

		assert.True(t, strings.HasPrefix(info.Version, "2.0."), "%s", info.Version)
		assert.True(t, strings.HasPrefix(info.FullVersion, "2.0."), "%s", info.FullVersion)
		require.NotEmpty(t, info.BuildTime)
		assert.True(t, time.Since(*info.BuildTime) < 60*24*time.Hour, "InstalledTime = %s", info.BuildTime)
		assert.Equal(t, "local", info.Repo)

		info2 := checker.installed()
		assert.Equal(t, info, info2)
	})

	t.Run("Check", func(t *testing.T) {
		checker := newPMMUpdateChecker(logrus.WithField("test", t.Name()))

		res, resT := checker.checkResult()
		assert.WithinDuration(t, time.Now(), resT, time.Second)

		assert.True(t, strings.HasPrefix(res.Installed.Version, "2.0."), "%s", res.Installed.Version)
		assert.True(t, strings.HasPrefix(res.Installed.FullVersion, "2.0."), "%s", res.Installed.FullVersion)
		require.NotEmpty(t, res.Installed.BuildTime)
		assert.True(t, time.Since(*res.Installed.BuildTime) < 60*24*time.Hour, "InstalledTime = %s", res.Installed.BuildTime)
		assert.Equal(t, "local", res.Installed.Repo)

		assert.True(t, strings.HasPrefix(res.Latest.Version, "2.0."), "%s", res.Latest.Version)
		assert.True(t, strings.HasPrefix(res.Latest.FullVersion, "2.0."), "%s", res.Latest.FullVersion)
		require.NotEmpty(t, res.Latest.BuildTime)
		assert.True(t, time.Since(*res.Latest.BuildTime) < 60*24*time.Hour, "LatestTime = %s", res.Latest.BuildTime)
		assert.NotEmpty(t, res.Latest.Repo)

		// We assume that the latest percona/pmm-server:2 and perconalab/pmm-server:dev-latest images
		// always contains the latest pmm-update package versions.
		// If this test fails, re-pull them and recreate devcontainer.
		var updateAvailable bool
		image := os.Getenv("PMM_SERVER_IMAGE")
		require.NotEmpty(t, image)
		if image != "percona/pmm-server:2" && image != "perconalab/pmm-server:dev-latest" {
			updateAvailable = true
		}
		if updateAvailable {
			t.Log("Assuming pmm-update update is available.")
			assert.True(t, res.UpdateAvailable, "update should be available")
			assert.NotEqual(t, res.Installed.Version, res.Latest.Version, "versions should not be the same")
			assert.NotEqual(t, res.Installed.FullVersion, res.Latest.FullVersion, "versions should not be the same")
			assert.NotEqual(t, *res.Installed.BuildTime, *res.Latest.BuildTime, "build times should not be the same (%s)", *res.Installed.BuildTime)
			assert.Equal(t, "pmm2-server", res.Latest.Repo)
		} else {
			t.Log("Assuming the latest pmm-update version.")
			assert.False(t, res.UpdateAvailable, "update should not be available")
			assert.Equal(t, res.Installed, res.Latest, "version should be the same (latest)")
			assert.Equal(t, *res.Installed.BuildTime, *res.Latest.BuildTime, "build times should be the same")
			assert.Equal(t, "local", res.Latest.Repo)
		}

		// cached result
		res2, resT2 := checker.checkResult()
		assert.Equal(t, res, res2)
		assert.Equal(t, resT, resT2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		go checker.run(ctx)
		time.Sleep(100 * time.Millisecond)

		// should block and wait for run to finish one iteration
		res3, resT3 := checker.checkResult()
		assert.Equal(t, res2, res3)
		assert.NotEqual(t, resT2, resT3, "%s", resT2)
		assert.WithinDuration(t, resT2, resT3, 5*time.Second)
	})

	t.Run("UpdateConfiguration", func(t *testing.T) {
		// logrus.SetLevel(logrus.DebugLevel)

		s := New("/etc/supervisord.d")
		require.NotEmpty(t, s.supervisorctlPath)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go s.Run(ctx)

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

	t.Run("Update", func(t *testing.T) {
		// This test can be run only once as it breaks assumptions of other tests.
		// It also should be the last test in devcontainer.
		if os.Getenv("TEST_RUN_UPDATE") == "" {
			t.Skip("skipping update test")
		}

		// logrus.SetLevel(logrus.DebugLevel)

		s := New("/etc/supervisord.d")
		require.NotEmpty(t, s.supervisorctlPath)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go s.Run(ctx)

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
		const wait = 3 * time.Second
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
