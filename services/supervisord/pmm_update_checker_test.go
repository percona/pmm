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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPMMUpdateChecker(t *testing.T) {
	if os.Getenv("DEVCONTAINER") == "" {
		t.Skip("can be tested only inside devcontainer")
	}

	c := newPMMUpdateChecker(logrus.WithField("test", t.Name()))

	t.Run("InstalledPackageInfo", func(t *testing.T) {
		info := c.installed()
		require.NotNil(t, info)

		assert.True(t, strings.HasPrefix(info.Version, "2.0.0-beta"), "%s", info.Version)
		assert.True(t, strings.HasPrefix(info.FullVersion, "2.0.0"), "%s", info.FullVersion)
		require.NotEmpty(t, info.BuildTime)
		assert.True(t, time.Since(*info.BuildTime) < 60*24*time.Hour, "InstalledTime = %s", info.BuildTime)
		assert.Equal(t, "local", info.Repo)

		info2 := c.installed()
		assert.Equal(t, info, info2)
	})

	t.Run("Check", func(t *testing.T) {
		res, resT := c.checkResult()
		assert.WithinDuration(t, time.Now(), resT, time.Second)

		assert.True(t, strings.HasPrefix(res.Installed.Version, "2.0.0-beta"), "%s", res.Installed.Version)
		assert.True(t, strings.HasPrefix(res.Installed.FullVersion, "2.0.0"), "%s", res.Installed.FullVersion)
		require.NotEmpty(t, res.Installed.BuildTime)
		assert.True(t, time.Since(*res.Installed.BuildTime) < 60*24*time.Hour, "InstalledTime = %s", res.Installed.BuildTime)
		assert.Equal(t, "local", res.Installed.Repo)

		assert.True(t, strings.HasPrefix(res.Latest.Version, "2.0.0-beta"), "%s", res.Latest.Version)
		assert.True(t, strings.HasPrefix(res.Latest.FullVersion, "2.0.0"), "%s", res.Latest.FullVersion)
		require.NotEmpty(t, res.Latest.BuildTime)
		assert.True(t, time.Since(*res.Latest.BuildTime) < 60*24*time.Hour, "LatestTime = %s", res.Latest.BuildTime)
		assert.NotEmpty(t, res.Latest.Repo)

		// We assume that the latest perconalab/pmm-server:dev-latest image always contains the latest
		// pmm-update package version. That is true for Travis CI. If this test fails locally,
		// run "docker pull perconalab/pmm-server:dev-latest" and recreate devcontainer.
		if os.Getenv("PMM_SERVER_IMAGE") == "perconalab/pmm-server:dev-latest" {
			assert.Equal(t, res.Installed, res.Latest)
			assert.False(t, res.UpdateAvailable)
		} else {
			assert.NotEqual(t, res.Installed.Version, res.Latest.Version)
			assert.NotEqual(t, res.Installed.FullVersion, res.Latest.FullVersion)
			assert.NotEqual(t, *res.Installed.BuildTime, *res.Latest.BuildTime)
			assert.Equal(t, "pmm2-laboratory", res.Latest.Repo)
			assert.True(t, res.UpdateAvailable)
		}

		// cached result
		res2, resT2 := c.checkResult()
		assert.Equal(t, res, res2)
		assert.Equal(t, resT, resT2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		go c.run(ctx)
		time.Sleep(100 * time.Millisecond)

		// should block and wait for run to finish one iteration
		res3, resT3 := c.checkResult()
		assert.Equal(t, res2, res3)
		assert.NotEqual(t, resT2, resT3, "%s", resT2)
		assert.WithinDuration(t, resT2, resT3, 2*time.Second)
	})
}
