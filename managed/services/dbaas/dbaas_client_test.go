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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	v := os.Getenv("ENABLE_DBAAS")
	dbaasEnabled, err := strconv.ParseBool(v)
	if err != nil {
		t.Skipf("Invalid value %q for environment variable ENABLE_DBAAS", v)
	}
	if !dbaasEnabled {
		t.Skip("DBaaS is not enabled")
	}

	getClient := func(t *testing.T) *Client {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		c := NewClient("127.0.0.1:20201")
		err := c.Connect(ctx)
		require.NoError(t, err, "Cannot connect to dbaas-controller")
		t.Cleanup(func() {
			require.NoError(t, c.Disconnect())
		})
		return c
	}

	t.Run("ValidKubeConfig", func(t *testing.T) {
		kubeConfig := os.Getenv("PERCONA_TEST_DBAAS_KUBECONFIG")
		if kubeConfig == "" {
			t.Skip("PERCONA_TEST_DBAAS_KUBECONFIG env variable is not provided")
		}
		c := getClient(t)
		_, err = c.CheckKubernetesClusterConnection(context.TODO(), kubeConfig)
		require.NoError(t, err)
	})

	t.Run("InvalidKubeConfig", func(t *testing.T) {
		c := getClient(t)
		_, err = c.CheckKubernetesClusterConnection(context.TODO(), "{}")
		require.Error(t, err)
	})
	t.Run("GetKubeConfig", func(t *testing.T) {
		c := getClient(t)
		kubeconfig, err := c.GetKubeConfig()
		require.Error(t, err)
		assert.Equal(t, "", kubeconfig)
	})
}
