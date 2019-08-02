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

func TestCheckVersions(t *testing.T) {
	v, err := CheckVersions(context.Background(), "pmm-update")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(v.InstalledRPMVersion, "2.0.0"), "%s", v.InstalledRPMVersion)
	assert.True(t, strings.HasPrefix(v.InstalledRPMNiceVersion, "2.0.0-beta"), "%s", v.InstalledRPMNiceVersion)
	assert.True(t, strings.HasPrefix(v.LatestRPMVersion, "2.0.0"), "%s", v.LatestRPMVersion)
	assert.True(t, strings.HasPrefix(v.LatestRPMNiceVersion, "2.0.0-beta"), "%s", v.LatestRPMNiceVersion)
	assert.NotEmpty(t, v.LatestRepo)
	require.NotEmpty(t, v.InstalledTime)
	assert.True(t, time.Since(*v.InstalledTime) < 60*24*time.Hour, "InstalledTime = %s", v.InstalledTime)
	require.NotEmpty(t, v.LatestTime)
	assert.True(t, time.Since(*v.LatestTime) < 60*24*time.Hour, "LatestTime = %s", v.LatestTime)

	// We assume that the latest perconalab/pmm-server:dev-latest image always contains the latest
	// pmm-update package version. That is true for Travis CI. If this test fails locally,
	// run "docker pull perconalab/pmm-server:dev-latest" and recreate devcontainer.
	if os.Getenv("PMM_SERVER_IMAGE") == "perconalab/pmm-server:dev-latest" {
		assert.Equal(t, v.InstalledRPMVersion, v.LatestRPMVersion)
		assert.Equal(t, v.InstalledRPMNiceVersion, v.LatestRPMNiceVersion)
		assert.False(t, v.UpdateAvailable)
		assert.Equal(t, "local", v.LatestRepo)
		assert.Equal(t, *v.InstalledTime, *v.LatestTime)
	} else {
		assert.NotEqual(t, v.InstalledRPMVersion, v.LatestRPMVersion)
		assert.NotEqual(t, v.InstalledRPMNiceVersion, v.LatestRPMNiceVersion)
		assert.True(t, v.UpdateAvailable)
		assert.Equal(t, "pmm2-laboratory", v.LatestRepo)
		assert.NotEqual(t, *v.InstalledTime, *v.LatestTime)
	}
}

func TestUpdatePackage(t *testing.T) {
	err := UpdatePackage(context.Background(), "golang")
	require.NoError(t, err)
}
