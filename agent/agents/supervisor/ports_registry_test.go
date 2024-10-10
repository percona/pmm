// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	defer l1.Close() //nolint:gosec,errcheck,nolintlint

	p, err := r.Reserve()
	assert.NoError(t, err)
	assert.EqualValues(t, 65002, p)
	_, err = r.Reserve()
	assert.Equal(t, errNoFreePort, err)

	l2, err := net.Listen("tcp", "127.0.0.1:65002")
	require.NoError(t, err)
	defer l2.Close() //nolint:errcheck,gosec,nolintlint

	err = r.Release(65000)
	assert.NoError(t, err)
	err = r.Release(65001)
	assert.Equal(t, errPortNotReserved, err)
	err = r.Release(65002)
	assert.Equal(t, errPortBusy, err)

	l1.Close() //nolint:errcheck
	l2.Close() //nolint:errcheck

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
