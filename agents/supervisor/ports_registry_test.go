// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package supervisor

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	// 65000 is marked as reserved, 65001 is busy, 65002 is free
	r := newPortsRegistry(65000, 65002, []uint16{65000})
	l1, err := net.Listen("tcp", "127.0.0.1:65001")
	require.NoError(t, err)
	defer l1.Close()

	p, err := r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65002, p)
	_, err = r.Reserve()
	assert.Equal(t, errNoFreePort, err)

	l2, err := net.Listen("tcp", "127.0.0.1:65002")
	require.NoError(t, err)
	defer l2.Close()

	err = r.Release(65000)
	assert.NoError(t, err)
	err = r.Release(65001)
	assert.Equal(t, errPortNotReserved, err)
	err = r.Release(65002)
	assert.Equal(t, errPortBusy, err)

	l1.Close()
	l2.Close()

	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65000, p)
	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65001, p)
	_, err = r.Reserve()
	assert.Equal(t, errNoFreePort, err)

	err = r.Release(65002)
	assert.NoError(t, err)

	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65002, p)
	_, err = r.Reserve()
	assert.Equal(t, errNoFreePort, err)
}

func TestPreferNewPort(t *testing.T) {
	r := newPortsRegistry(65000, 65002, nil)

	p, err := r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65000, p)

	err = r.Release(p)
	assert.NoError(t, err)

	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65001, p)

	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65002, p)

	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65000, p)
}

func TestSinglePort(t *testing.T) {
	r := newPortsRegistry(65000, 65000, nil)

	p, err := r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65000, p)

	_, err = r.Reserve()
	assert.Equal(t, errNoFreePort, err)

	err = r.Release(p)
	assert.NoError(t, err)

	p, err = r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65000, p)
}
