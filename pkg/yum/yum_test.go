// pmm-update
// Copyright (C) 2019 Percona LLC
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

package yum

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstalled(t *testing.T) {
	v, err := Installed(context.Background(), "pmm-update")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(v.Installed.Version, "2.0.0-beta"), "%s", v.Installed.Version)
	assert.True(t, strings.HasPrefix(v.Installed.FullVersion, "2.0.0"), "%s", v.Installed.FullVersion)
	require.NotEmpty(t, v.Installed.BuildTime)
	assert.True(t, time.Since(*v.Installed.BuildTime) < 60*24*time.Hour, "InstalledTime = %s", v.Installed.BuildTime)
	assert.Equal(t, "local", v.Installed.Repo)
}

func TestCheck(t *testing.T) {
	v, err := Check(context.Background(), "pmm-update")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(v.Installed.Version, "2.0.0-beta"), "%s", v.Installed.Version)
	assert.True(t, strings.HasPrefix(v.Installed.FullVersion, "2.0.0"), "%s", v.Installed.FullVersion)
	require.NotEmpty(t, v.Installed.BuildTime)
	assert.True(t, time.Since(*v.Installed.BuildTime) < 60*24*time.Hour, "InstalledTime = %s", v.Installed.BuildTime)
	assert.Equal(t, "local", v.Installed.Repo)

	assert.True(t, strings.HasPrefix(v.Latest.Version, "2.0.0-beta"), "%s", v.Latest.Version)
	assert.True(t, strings.HasPrefix(v.Latest.FullVersion, "2.0.0"), "%s", v.Latest.FullVersion)
	require.NotEmpty(t, v.Latest.BuildTime)
	assert.True(t, time.Since(*v.Latest.BuildTime) < 60*24*time.Hour, "LatestTime = %s", v.Latest.BuildTime)
	assert.NotEmpty(t, v.Latest.Repo)

	// We assume that the latest perconalab/pmm-server:dev-latest image always contains the latest
	// pmm-update package version. That is true for Travis CI. If this test fails locally,
	// run "docker pull perconalab/pmm-server:dev-latest" and recreate devcontainer.
	if os.Getenv("PMM_SERVER_IMAGE") == "perconalab/pmm-server:dev-latest" {
		assert.Equal(t, v.Installed, v.Latest)
		assert.False(t, v.UpdateAvailable)
	} else {
		assert.NotEqual(t, v.Installed.Version, v.Latest.Version)
		assert.NotEqual(t, v.Installed.FullVersion, v.Latest.FullVersion)
		assert.NotEqual(t, *v.Installed.BuildTime, *v.Latest.BuildTime)
		assert.Equal(t, "pmm2-laboratory", v.Latest.Repo)
		assert.True(t, v.UpdateAvailable)
	}
}

func TestUpdate(t *testing.T) {
	err := Update(context.Background(), "golang")
	require.NoError(t, err)
}
