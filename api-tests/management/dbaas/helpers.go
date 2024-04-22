// Copyright (C) 2024 Percona LLC
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

// Package dbaas contains DBaaS API tests.
package dbaas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	dbaasClient "github.com/percona/pmm/api/managementpb/dbaas/json/client"
	"github.com/percona/pmm/api/managementpb/dbaas/json/client/kubernetes"
)

func registerKubernetesCluster(t *testing.T, kubernetesClusterName string, kubeconfig string) {
	t.Helper()
	registerKubernetesClusterResponse, err := dbaasClient.Default.Kubernetes.RegisterKubernetesCluster(
		&kubernetes.RegisterKubernetesClusterParams{
			Body: kubernetes.RegisterKubernetesClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				KubeAuth:              &kubernetes.RegisterKubernetesClusterParamsBodyKubeAuth{Kubeconfig: kubeconfig},
			},
			Context: pmmapitests.Context,
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)
	t.Cleanup(func() {
		_, _ = unregisterKubernetesCluster(kubernetesClusterName)
	})
}

func unregisterKubernetesCluster(kubernetesClusterName string) (*kubernetes.UnregisterKubernetesClusterOK, error) {
	return dbaasClient.Default.Kubernetes.UnregisterKubernetesCluster(
		&kubernetes.UnregisterKubernetesClusterParams{
			Body:    kubernetes.UnregisterKubernetesClusterBody{KubernetesClusterName: kubernetesClusterName},
			Context: pmmapitests.Context,
		},
	)
}
