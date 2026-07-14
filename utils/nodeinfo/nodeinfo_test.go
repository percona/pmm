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

package nodeinfo

import (
	"net"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	info := Get()
	require.False(t, info.Container, "not expected to be run inside a container")
	assert.Equal(t, runtime.GOOS, info.Distro)

	// all our test environments have IPv4 addresses
	ip := net.ParseIP(info.PublicAddress)
	require.NotNil(t, ip)
	assert.NotNil(t, ip.To4())

	assert.False(t, strings.HasSuffix(info.MachineID, "\n"), "%q", info.MachineID)
}
