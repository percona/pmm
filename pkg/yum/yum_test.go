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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckVersions(t *testing.T) {
	v, err := CheckVersions(context.Background(), "pmm-update")
	require.NoError(t, err)
	assert.NotEmpty(t, v.InstalledRPMVersion)
	assert.Empty(t, v.LatestTime)
	assert.Equal(t, "pmm2-laboratory", v.LatestRepo)

	// the latest perconalab/pmm-server:dev-latest image always contains the latest pmm-update package version
	if os.Getenv("PMM_SERVER_IMAGE") == "perconalab/pmm-server:dev-latest" {
		assert.Equal(t, v.InstalledRPMVersion, v.LatestRPMVersion)
		assert.False(t, v.UpdateAvailable)
	} else {
		assert.NotEqual(t, v.InstalledRPMVersion, v.LatestRPMVersion)
		assert.True(t, v.UpdateAvailable)
		// TODO assert.True(t, v.InstalledTime.Before(v.LatestTime), "expected %s < %s", v.InstalledTime, v.LatestTime)
	}
}

func TestUpdatePackage(t *testing.T) {
	err := UpdatePackage(context.Background(), "golang")
	require.NoError(t, err)
}
