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

package dbaas

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/percona/pmm/version"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

func TestClient(t *testing.T) {
	getClient := func(t *testing.T) *Client {
		opts := []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{MaxDelay: time.Second}}),
			grpc.WithUserAgent("pmm-managed/" + version.Version),
		}

		conn, err := grpc.DialContext(context.TODO(), "127.0.0.1:20201", opts...)
		require.NoError(t, err, "Cannot connect to dbaas-controller")
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})

		c := NewClient(conn)
		return c
	}
	t.Run("InvalidKubeConfig", func(t *testing.T) {
		if os.Getenv("PERCONA_TEST_DBAAS") != "1" {
			t.Skip("PERCONA_TEST_DBAAS env variable is not passed, skipping")
		}
		kubeConfig := os.Getenv("PERCONA_TEST_DBAAS_KUBECONFIG")
		if kubeConfig == "" {
			t.Skip("PERCONA_TEST_DBAAS_KUBECONFIG env variable is not provided")
		}
		c := getClient(t)
		err := c.CheckKubernetesClusterConnection(context.TODO(), kubeConfig)
		require.NoError(t, err)
	})

	t.Run("InvalidKubeConfig", func(t *testing.T) {
		if os.Getenv("PERCONA_TEST_DBAAS") != "1" {
			t.Skip("PERCONA_TEST_DBAAS env variable is not passed, skipping")
		}

		c := getClient(t)
		err := c.CheckKubernetesClusterConnection(context.TODO(), "{}")
		require.Error(t, err)
	})
}
