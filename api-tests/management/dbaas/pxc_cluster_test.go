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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	dbaasClient "github.com/percona/pmm/api/managementpb/dbaas/json/client"
	dbclusters "github.com/percona/pmm/api/managementpb/dbaas/json/client/db_clusters"
	pxcclusters "github.com/percona/pmm/api/managementpb/dbaas/json/client/pxc_clusters"
)

const (
	kubernetesClusterName = "api-test-k8s-cluster"
)

//nolint:funlen
func TestPXCClusterServer(t *testing.T) {
	if pmmapitests.Kubeconfig == "" {
		t.Skip("Skip tests of PXCClusterServer without kubeconfig")
	}
	registerKubernetesCluster(t, kubernetesClusterName, pmmapitests.Kubeconfig)

	t.Run("BasicPXCCluster", func(t *testing.T) {
		paramsFirstPXC := pxcclusters.CreatePXCClusterParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.CreatePXCClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "first-pxc-test",
				Params: &pxcclusters.CreatePXCClusterParamsBodyParams{
					ClusterSize: 3,
					Haproxy: &pxcclusters.CreatePXCClusterParamsBodyParamsHaproxy{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsHaproxyComputeResources{
							CPUm:        500,
							MemoryBytes: "1000000000",
						},
					},
					PXC: &pxcclusters.CreatePXCClusterParamsBodyParamsPXC{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsPXCComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
						DiskSize: "1000000000",
					},
				},
			},
		}

		_, err := dbaasClient.Default.PXCClusters.CreatePXCCluster(&paramsFirstPXC)
		assert.NoError(t, err)

		// Create one more PXC Cluster.
		paramsSecondPXC := pxcclusters.CreatePXCClusterParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.CreatePXCClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "second-pxc-test",
				Params: &pxcclusters.CreatePXCClusterParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &pxcclusters.CreatePXCClusterParamsBodyParamsProxysql{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        500,
							MemoryBytes: "1000000000",
						},
						DiskSize: "1000000000",
					},
					PXC: &pxcclusters.CreatePXCClusterParamsBodyParamsPXC{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsPXCComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
						DiskSize: "1000000000",
					},
				},
			},
		}
		_, err = dbaasClient.Default.PXCClusters.CreatePXCCluster(&paramsSecondPXC)
		assert.NoError(t, err)

		listPXCClustersParamsParam := dbclusters.ListDBClustersParams{
			Context: pmmapitests.Context,
			Body: dbclusters.ListDBClustersBody{
				KubernetesClusterName: kubernetesClusterName,
			},
		}
		pxcClusters, err := dbaasClient.Default.DBClusters.ListDBClusters(&listPXCClustersParamsParam)
		assert.NoError(t, err)

		for _, name := range []string{"first-pxc-test", "second-pxc-test"} {
			foundPXC := false
			for _, pxc := range pxcClusters.Payload.PXCClusters {
				if name == pxc.Name {
					foundPXC = true

					break
				}
			}
			assert.True(t, foundPXC, "Cannot find PXC with name %s in cluster list", name)
		}

		getPXCClusterParamsParam := pxcclusters.GetPXCClusterCredentialsParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.GetPXCClusterCredentialsBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "first-pxc-test",
			},
		}
		pxcCluster, err := dbaasClient.Default.PXCClusters.GetPXCClusterCredentials(&getPXCClusterParamsParam)
		assert.NoError(t, err)
		assert.Equal(t, pxcCluster.Payload.ConnectionCredentials.Username, "root")
		assert.Equal(t, pxcCluster.Payload.ConnectionCredentials.Host, "first-pxc-test-haproxy")
		assert.Equal(t, pxcCluster.Payload.ConnectionCredentials.Port, int32(3306))
		assert.NotEmpty(t, pxcCluster.Payload.ConnectionCredentials.Password)

		paramsUpdatePXC := pxcclusters.UpdatePXCClusterParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.UpdatePXCClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "second-pxc-test",
				Params: &pxcclusters.UpdatePXCClusterParamsBodyParams{
					ClusterSize: 2,
					Proxysql: &pxcclusters.UpdatePXCClusterParamsBodyParamsProxysql{
						ComputeResources: &pxcclusters.UpdatePXCClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        2,
							MemoryBytes: "128",
						},
					},
					PXC: &pxcclusters.UpdatePXCClusterParamsBodyParamsPXC{
						ComputeResources: &pxcclusters.UpdatePXCClusterParamsBodyParamsPXCComputeResources{
							CPUm:        2,
							MemoryBytes: "128",
						},
					},
				},
			},
		}

		_, err = dbaasClient.Default.PXCClusters.UpdatePXCCluster(&paramsUpdatePXC)
		pmmapitests.AssertAPIErrorf(t, err, 500, codes.Internal, `state is Error: PXC cluster is not ready`)

		for _, pxc := range pxcClusters.Payload.PXCClusters {
			if pxc.Name == "" {
				continue
			}
			deletePXCClusterParamsParam := dbclusters.DeleteDBClusterParams{
				Context: pmmapitests.Context,
				Body: dbclusters.DeleteDBClusterBody{
					KubernetesClusterName: kubernetesClusterName,
					Name:                  pxc.Name,
				},
			}
			_, err := dbaasClient.Default.DBClusters.DeleteDBCluster(&deletePXCClusterParamsParam)
			assert.NoError(t, err)
		}

		t.Skip("Skip restart till better implementation. https://jira.percona.com/browse/PMM-6980")
		restartPXCClusterParamsParam := dbclusters.RestartDBClusterParams{
			Context: pmmapitests.Context,
			Body: dbclusters.RestartDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "first-pxc-test",
			},
		}
		_, err = dbaasClient.Default.DBClusters.RestartDBCluster(&restartPXCClusterParamsParam)
		assert.NoError(t, err)
	})

	t.Run("CreatePXCClusterEmptyName", func(t *testing.T) {
		paramsPXCEmptyName := pxcclusters.CreatePXCClusterParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.CreatePXCClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "",
				Params: &pxcclusters.CreatePXCClusterParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &pxcclusters.CreatePXCClusterParamsBodyParamsProxysql{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
					PXC: &pxcclusters.CreatePXCClusterParamsBodyParamsPXC{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsPXCComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
				},
			},
		}
		_, err := dbaasClient.Default.PXCClusters.CreatePXCCluster(&paramsPXCEmptyName)
		pmmapitests.AssertAPIErrorf(t, err, 400,
			codes.InvalidArgument, `invalid field Name: value '' must be a string conforming to regex "^[a-z]([-a-z0-9]*[a-z0-9])?$"`)
	})

	t.Run("CreatePXCClusterInvalidName", func(t *testing.T) {
		paramsPXCInvalidName := pxcclusters.CreatePXCClusterParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.CreatePXCClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "123_asd",
				Params: &pxcclusters.CreatePXCClusterParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &pxcclusters.CreatePXCClusterParamsBodyParamsProxysql{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
					PXC: &pxcclusters.CreatePXCClusterParamsBodyParamsPXC{
						ComputeResources: &pxcclusters.CreatePXCClusterParamsBodyParamsPXCComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
				},
			},
		}
		_, err := dbaasClient.Default.PXCClusters.CreatePXCCluster(&paramsPXCInvalidName)
		pmmapitests.AssertAPIErrorf(t, err, 400,
			codes.InvalidArgument, `invalid field Name: value '123_asd' must be a string conforming to regex "^[a-z]([-a-z0-9]*[a-z0-9])?$"`)
	})

	t.Run("ListUnknownCluster", func(t *testing.T) {
		listPXCClustersParamsParam := dbclusters.ListDBClustersParams{
			Context: pmmapitests.Context,
			Body: dbclusters.ListDBClustersBody{
				KubernetesClusterName: "Unknown-kubernetes-cluster-name",
			},
		}
		_, err := dbaasClient.Default.DBClusters.ListDBClusters(&listPXCClustersParamsParam)
		pmmapitests.AssertAPIErrorf(t, err, 404,
			codes.NotFound, `Kubernetes Cluster with name "Unknown-kubernetes-cluster-name" not found.`)
	})

	t.Run("RestartUnknownPXCCluster", func(t *testing.T) {
		restartPXCClusterParamsParam := dbclusters.RestartDBClusterParams{
			Context: pmmapitests.Context,
			Body: dbclusters.RestartDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "Unknown-pxc-name",
			},
		}
		_, err := dbaasClient.Default.DBClusters.RestartDBCluster(&restartPXCClusterParamsParam)
		require.Error(t, err)
		assert.Equal(t, 500, err.(pmmapitests.ErrorResponse).Code()) //nolint:errorlint
	})

	t.Run("DeleteUnknownPXCCluster", func(t *testing.T) {
		deletePXCClusterParamsParam := dbclusters.DeleteDBClusterParams{
			Context: pmmapitests.Context,
			Body: dbclusters.DeleteDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "Unknown-pxc-name",
			},
		}
		_, err := dbaasClient.Default.DBClusters.DeleteDBCluster(&deletePXCClusterParamsParam)
		require.Error(t, err)
		assert.Equal(t, 500, err.(pmmapitests.ErrorResponse).Code()) //nolint:errorlint
	})

	t.Run("SuspendResumeCluster", func(t *testing.T) {
		paramsUpdatePXC := pxcclusters.UpdatePXCClusterParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.UpdatePXCClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "second-pxc-test",
				Params: &pxcclusters.UpdatePXCClusterParamsBodyParams{
					Suspend: true,
					Resume:  true,
				},
			},
		}
		_, err := dbaasClient.Default.PXCClusters.UpdatePXCCluster(&paramsUpdatePXC)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `resume and suspend cannot be set together`)
	})

	t.Run("GetPXCClusterResources", func(t *testing.T) {
		paramsPXCClusterResources := pxcclusters.GetPXCClusterResourcesParams{
			Context: pmmapitests.Context,
			Body: pxcclusters.GetPXCClusterResourcesBody{
				Params: &pxcclusters.GetPXCClusterResourcesParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &pxcclusters.GetPXCClusterResourcesParamsBodyParamsProxysql{
						ComputeResources: &pxcclusters.GetPXCClusterResourcesParamsBodyParamsProxysqlComputeResources{
							CPUm:        1000,
							MemoryBytes: "1000000000",
						},
					},
					PXC: &pxcclusters.GetPXCClusterResourcesParamsBodyParamsPXC{
						ComputeResources: &pxcclusters.GetPXCClusterResourcesParamsBodyParamsPXCComputeResources{
							CPUm:        1000,
							MemoryBytes: "1000000000",
						},
					},
				},
			},
		}
		resources, err := dbaasClient.Default.PXCClusters.GetPXCClusterResources(&paramsPXCClusterResources)
		assert.NoError(t, err)
		assert.Equal(t, resources.Payload.Expected.MemoryBytes, 2000000000)
		assert.Equal(t, resources.Payload.Expected.CPUm, 2000)
		assert.Equal(t, resources.Payload.Expected.DiskSize, 2000000000)
	})
}
