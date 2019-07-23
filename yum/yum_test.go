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
	installed, remote, err := CheckVersions(context.Background(), "pmm-update")
	require.NoError(t, err)
	assert.NotEmpty(t, installed)
	require.Len(t, remote, 1)
	labsRemote := remote["pmm2-laboratory"]
	require.NotEmpty(t, labsRemote)

	devLatest := os.Getenv("PMM_SERVER_IMAGE") == "perconalab/pmm-server:dev-latest"
	assert.Equal(t, devLatest, installed == labsRemote, "installed: %q\nremote: %s", installed, labsRemote)
}
