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

package dbaas

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	dbaasClient "github.com/percona/pmm/api/managementpb/dbaas/json/client"
	dbclusters "github.com/percona/pmm/api/managementpb/dbaas/json/client/db_clusters"
	"github.com/percona/pmm/api/managementpb/dbaas/json/client/kubernetes"
	psmdbclusters "github.com/percona/pmm/api/managementpb/dbaas/json/client/psmdb_clusters"
)

func TestKubernetesServer(t *testing.T) {
	if os.Getenv("PERCONA_TEST_DBAAS") != "1" {
		t.Skip("PERCONA_TEST_DBAAS env variable is not passed, skipping")
	}
	kubeConfig := os.Getenv("PERCONA_TEST_DBAAS_KUBECONFIG")
	if kubeConfig == "" {
		t.Skip("PERCONA_TEST_DBAAS_KUBECONFIG env variable is not provided")
	}
	t.Run("Basic", func(t *testing.T) {
		kubernetesClusterName := pmmapitests.TestString(t, "api-test-cluster")
		clusters, err := dbaasClient.Default.Kubernetes.ListKubernetesClusters(nil)
		require.NoError(t, err)
		require.NotContains(t, clusters.Payload.KubernetesClusters,
			&kubernetes.ListKubernetesClustersOKBodyKubernetesClustersItems0{KubernetesClusterName: kubernetesClusterName})

		registerKubernetesCluster(t, kubernetesClusterName, kubeConfig)
		clusters, err = dbaasClient.Default.Kubernetes.ListKubernetesClusters(nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(clusters.Payload.KubernetesClusters), 1)
		assert.Contains(t, clusters.Payload.KubernetesClusters,
			&kubernetes.ListKubernetesClustersOKBodyKubernetesClustersItems0{KubernetesClusterName: kubernetesClusterName})

		unregisterKubernetesClusterResponse, err := dbaasClient.Default.Kubernetes.UnregisterKubernetesCluster(
			&kubernetes.UnregisterKubernetesClusterParams{
				Body:    kubernetes.UnregisterKubernetesClusterBody{KubernetesClusterName: kubernetesClusterName},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.NotNil(t, unregisterKubernetesClusterResponse)

		clusters, err = dbaasClient.Default.Kubernetes.ListKubernetesClusters(nil)
		assert.NoError(t, err)
		require.NotContains(t, clusters.Payload.KubernetesClusters,
			&kubernetes.ListKubernetesClustersOKBodyKubernetesClustersItems0{KubernetesClusterName: kubernetesClusterName})
	})

	t.Run("DuplicateClusterName", func(t *testing.T) {
		kubernetesClusterName := pmmapitests.TestString(t, "api-test-cluster-duplicate")

		registerKubernetesCluster(t, kubernetesClusterName, kubeConfig)
		registerKubernetesClusterResponse, err := dbaasClient.Default.Kubernetes.RegisterKubernetesCluster(
			&kubernetes.RegisterKubernetesClusterParams{
				Body: kubernetes.RegisterKubernetesClusterBody{
					KubernetesClusterName: kubernetesClusterName,
					KubeAuth:              &kubernetes.RegisterKubernetesClusterParamsBodyKubeAuth{Kubeconfig: kubeConfig},
				},
				Context: pmmapitests.Context,
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, fmt.Sprintf("Kubernetes Cluster with Name %q already exists.", kubernetesClusterName))
		require.Nil(t, registerKubernetesClusterResponse)
	})

	t.Run("EmptyKubernetesClusterName", func(t *testing.T) {
		registerKubernetesClusterResponse, err := dbaasClient.Default.Kubernetes.RegisterKubernetesCluster(
			&kubernetes.RegisterKubernetesClusterParams{
				Body: kubernetes.RegisterKubernetesClusterBody{
					KubernetesClusterName: "",
					KubeAuth:              &kubernetes.RegisterKubernetesClusterParamsBodyKubeAuth{Kubeconfig: kubeConfig},
				},
				Context: pmmapitests.Context,
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field KubernetesClusterName: value '' must not be an empty string")
		require.Nil(t, registerKubernetesClusterResponse)
	})

	t.Run("EmptyKubeConfig", func(t *testing.T) {
		registerKubernetesClusterResponse, err := dbaasClient.Default.Kubernetes.RegisterKubernetesCluster(
			&kubernetes.RegisterKubernetesClusterParams{
				Body: kubernetes.RegisterKubernetesClusterBody{
					KubernetesClusterName: "empty-kube-config",
					KubeAuth:              &kubernetes.RegisterKubernetesClusterParamsBodyKubeAuth{},
				},
				Context: pmmapitests.Context,
			},
		)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field KubeAuth.Kubeconfig: value '' must not be an empty string")
		require.Nil(t, registerKubernetesClusterResponse)
	})

	t.Run("GetKubernetesCluster", func(t *testing.T) {
		kubernetesClusterName := pmmapitests.TestString(t, "api-test-cluster")
		registerKubernetesCluster(t, kubernetesClusterName, kubeConfig)

		cluster, err := dbaasClient.Default.Kubernetes.GetKubernetesCluster(&kubernetes.GetKubernetesClusterParams{
			Body: kubernetes.GetKubernetesClusterBody{
				KubernetesClusterName: kubernetesClusterName,
			},
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
		assert.NotNil(t, cluster)
		assert.NotNil(t, cluster.Payload.KubeAuth)
		assert.Equal(t, kubeConfig, cluster.Payload.KubeAuth.Kubeconfig)
	})

	t.Run("GetResources", func(t *testing.T) {
		kubernetesClusterName := pmmapitests.TestString(t, "api-test-cluster")

		resources, err := dbaasClient.Default.Kubernetes.GetResources(&kubernetes.GetResourcesParams{
			Body: kubernetes.GetResourcesBody{
				KubernetesClusterName: kubernetesClusterName,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, resources)
		require.NotNil(t, resources.Payload.All)
		require.NotNil(t, resources.Payload.Available)
		assert.Greater(t, resources.Payload.All.CPUm, resources.Payload.Available.CPUm)
		assert.Greater(t, resources.Payload.All.MemoryBytes, resources.Payload.Available.MemoryBytes)
		assert.Greater(t, resources.Payload.All.DiskSize, resources.Payload.Available.DiskSize)
		assert.Greater(t, resources.Payload.Available.CPUm, uint64(0))
		assert.Greater(t, resources.Payload.Available.MemoryBytes, uint64(0))
		assert.Greater(t, resources.Payload.Available.DiskSize, uint64(0))
	})

	t.Run("UnregisterNotExistCluster", func(t *testing.T) {
		unregisterKubernetesClusterOK, err := unregisterKubernetesCluster("not-exist-cluster")
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Kubernetes Cluster with name \"not-exist-cluster\" not found.")
		require.Nil(t, unregisterKubernetesClusterOK)
	})

	t.Run("UnregisterEmptyClusterName", func(t *testing.T) {
		unregisterKubernetesClusterOK, err := unregisterKubernetesCluster("")
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid field KubernetesClusterName: value '' must not be an empty string")
		require.Nil(t, unregisterKubernetesClusterOK)
	})

	t.Run("UnregisterWithoutAndWithForce", func(t *testing.T) {
		kubernetesClusterName := pmmapitests.TestString(t, "api-test-cluster")
		dbClusterName := "first-psmdb-test"
		clusters, err := dbaasClient.Default.Kubernetes.ListKubernetesClusters(nil)
		require.NoError(t, err)
		require.NotContains(t, clusters.Payload.KubernetesClusters,
			&kubernetes.ListKubernetesClustersOKBodyKubernetesClustersItems0{KubernetesClusterName: kubernetesClusterName})
		registerKubernetesCluster(t, kubernetesClusterName, kubeConfig)

		paramsFirstPSMDB := psmdbclusters.CreatePSMDBClusterParams{
			Context: pmmapitests.Context,
			Body: psmdbclusters.CreatePSMDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  dbClusterName,
				Params: &psmdbclusters.CreatePSMDBClusterParamsBodyParams{
					ClusterSize: 3,
					Replicaset: &psmdbclusters.CreatePSMDBClusterParamsBodyParamsReplicaset{
						ComputeResources: &psmdbclusters.CreatePSMDBClusterParamsBodyParamsReplicasetComputeResources{
							CPUm:        500,
							MemoryBytes: "1000000000",
						},
						DiskSize: "1000000000",
					},
				},
			},
		}
		_, err = dbaasClient.Default.PSMDBClusters.CreatePSMDBCluster(&paramsFirstPSMDB)
		assert.NoError(t, err)

		clusters, err = dbaasClient.Default.Kubernetes.ListKubernetesClusters(nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(clusters.Payload.KubernetesClusters), 1)
		assert.Contains(t, clusters.Payload.KubernetesClusters,
			&kubernetes.ListKubernetesClustersOKBodyKubernetesClustersItems0{KubernetesClusterName: kubernetesClusterName})

		_, err = dbaasClient.Default.Kubernetes.UnregisterKubernetesCluster(
			&kubernetes.UnregisterKubernetesClusterParams{
				Body: kubernetes.UnregisterKubernetesClusterBody{
					KubernetesClusterName: kubernetesClusterName,
				},
				Context: pmmapitests.Context,
			},
		)
		require.Error(t, err)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, fmt.Sprintf(`Kubernetes cluster %s has PSMDB clusters`, kubernetesClusterName))

		unregisterKubernetesClusterResponse, err := dbaasClient.Default.Kubernetes.UnregisterKubernetesCluster(
			&kubernetes.UnregisterKubernetesClusterParams{
				Body: kubernetes.UnregisterKubernetesClusterBody{
					KubernetesClusterName: kubernetesClusterName,
					Force:                 true,
				},
				Context: pmmapitests.Context,
			},
		)
		require.NoError(t, err)
		assert.NotNil(t, unregisterKubernetesClusterResponse)

		_, err = dbaasClient.Default.Kubernetes.UnregisterKubernetesCluster(
			&kubernetes.UnregisterKubernetesClusterParams{
				Body: kubernetes.UnregisterKubernetesClusterBody{
					KubernetesClusterName: kubernetesClusterName,
				},
				Context: pmmapitests.Context,
			},
		)
		require.Error(t, err)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf(`Kubernetes Cluster with name "%s" not found.`, kubernetesClusterName))

		registerKubernetesCluster(t, kubernetesClusterName, kubeConfig)
		deletePSMDBClusterParamsParam := dbclusters.DeleteDBClusterParams{
			Context: pmmapitests.Context,
			Body: dbclusters.DeleteDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  dbClusterName,
			},
		}
		_, err = dbaasClient.Default.DBClusters.DeleteDBCluster(&deletePSMDBClusterParamsParam)
		assert.NoError(t, err)

		listPSMDBClustersParamsParam := dbclusters.ListDBClustersParams{
			Context: pmmapitests.Context,
			Body: dbclusters.ListDBClustersBody{
				KubernetesClusterName: kubernetesClusterName,
			},
		}

		for {
			psmDBClusters, err := dbaasClient.Default.DBClusters.ListDBClusters(&listPSMDBClustersParamsParam)
			assert.NoError(t, err)
			if len(psmDBClusters.Payload.PSMDBClusters) == 0 {
				break
			}
			time.Sleep(1 * time.Second)
		}

		unregisterKubernetesClusterResponse, err = dbaasClient.Default.Kubernetes.UnregisterKubernetesCluster(
			&kubernetes.UnregisterKubernetesClusterParams{
				Body: kubernetes.UnregisterKubernetesClusterBody{
					KubernetesClusterName: kubernetesClusterName,
				},
				Context: pmmapitests.Context,
			},
		)
		assert.NoError(t, err)
		assert.NotNil(t, unregisterKubernetesClusterResponse)
	})
}
