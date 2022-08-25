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

// Package dbaas contains logic related to communication with dbaas-controller.
package dbaas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

func TestKubeClient(t *testing.T) {
	t.Parallel()
	expectedConf := `apiVersion: v1
clusters:
    - cluster:
        certificate-authority-data: aGVsbG8=
        server: https://localhost
      name: default-cluster
contexts:
    - context:
        cluster: default-cluster
        namespace: default
        user: pmm-service-account
      name: default
current-context: default
kind: Config
users:
    - name: pmm-service-account
      user:
        token: world
`
	_, err := NewK8sInclusterClient()
	require.Error(t, err)
	k := &k8sClient{
		conf: &rest.Config{Host: "https://localhost"},
	}
	conf, err := k.GenerateKubeConfig(&v1.Secret{
		Data: map[string][]byte{
			"ca.crt": []byte("hello"),
			"token":  []byte("world"),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedConf, string(conf))
}
