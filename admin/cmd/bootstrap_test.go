// Copyright 2019 Percona LLC
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

package cmd

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func TestFindSocketOrPath(t *testing.T) {
	const socketPath = "/tmp/pmm-agent-find-socket-test.sock"
	const localPort = 18485

	t.Run("finds socket", func(t *testing.T) {
		l, err := net.Listen("unix", socketPath)
		require.NoError(t, err)
		defer l.Close() //nolint:errcheck

		socket, port := findSocketOrPort(socketPath, 0)
		require.Equal(t, socket, socketPath)
		require.Equal(t, port, uint32(0))
	})

	t.Run("finds port", func(t *testing.T) {
		l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort)))
		require.NoError(t, err)
		defer l.Close() //nolint:errcheck

		socket, port := findSocketOrPort(socketPath, localPort)
		require.Equal(t, socket, "")
		require.Equal(t, port, uint32(localPort))
	})

	t.Run("finds socket even if port is available", func(t *testing.T) {
		l, err := net.Listen("unix", socketPath)
		require.NoError(t, err)
		defer l.Close() //nolint:errcheck

		lp, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort)))
		require.NoError(t, err)
		defer lp.Close() //nolint:errcheck

		socket, port := findSocketOrPort(socketPath, localPort)
		require.Equal(t, socket, socketPath)
		require.Equal(t, port, uint32(0))
	})

	t.Run("defaults to socket", func(t *testing.T) {
		socket, port := findSocketOrPort(socketPath, 0)
		require.NotEqual(t, socket, "")
		require.Equal(t, port, uint32(0))
	})
}
