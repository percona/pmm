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
	"testing"

	dbaasClient "github.com/percona/pmm/api/managementpb/dbaas/json/client"
	"github.com/percona/pmm/api/managementpb/dbaas/json/client/xtra_db_cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm-managed/api-tests"
)

const (
	kubernetesClusterName = "api-test-k8s-cluster"
)

//nolint:funlen
func TestXtraDBClusterServer(t *testing.T) {
	if pmmapitests.Kubeconfig == "" {
		t.Skip("Skip tests of XtraDBClusterServer without kubeconfig")
	}
	registerKubernetesCluster(t, kubernetesClusterName, pmmapitests.Kubeconfig)

	t.Run("BasicXtraDBCluster", func(t *testing.T) {
		paramsFirstPXC := xtra_db_cluster.CreateXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.CreateXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "first-pxc-test",
				Params: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParams{
					ClusterSize: 3,
					Haproxy: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsHaproxy{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsHaproxyComputeResources{
							CPUm:        500,
							MemoryBytes: "1000000000",
						},
					},
					Pxc: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxc{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxcComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
						DiskSize: "1000000000",
					},
				},
			},
		}

		_, err := dbaasClient.Default.XtraDBCluster.CreateXtraDBCluster(&paramsFirstPXC)
		assert.NoError(t, err)

		// Create one more XtraDB Cluster.
		paramsSecondPXC := xtra_db_cluster.CreateXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.CreateXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "second-pxc-test",
				Params: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsProxysql{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        500,
							MemoryBytes: "1000000000",
						},
						DiskSize: "1000000000",
					},
					Pxc: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxc{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxcComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
						DiskSize: "1000000000",
					},
				},
			},
		}
		_, err = dbaasClient.Default.XtraDBCluster.CreateXtraDBCluster(&paramsSecondPXC)
		assert.NoError(t, err)

		listXtraDBClustersParamsParam := xtra_db_cluster.ListXtraDBClustersParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.ListXtraDBClustersBody{
				KubernetesClusterName: kubernetesClusterName,
			},
		}
		xtraDBClusters, err := dbaasClient.Default.XtraDBCluster.ListXtraDBClusters(&listXtraDBClustersParamsParam)
		assert.NoError(t, err)

		for _, name := range []string{"first-pxc-test", "second-pxc-test"} {
			foundPXC := false
			for _, pxc := range xtraDBClusters.Payload.Clusters {
				if name == pxc.Name {
					foundPXC = true

					break
				}
			}
			assert.True(t, foundPXC, "Cannot find PXC with name %s in cluster list", name)
		}

		getXtraDBClusterParamsParam := xtra_db_cluster.GetXtraDBClusterCredentialsParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.GetXtraDBClusterCredentialsBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "first-pxc-test",
			},
		}
		xtraDBCluster, err := dbaasClient.Default.XtraDBCluster.GetXtraDBClusterCredentials(&getXtraDBClusterParamsParam)
		assert.NoError(t, err)
		assert.Equal(t, xtraDBCluster.Payload.ConnectionCredentials.Username, "root")
		assert.Equal(t, xtraDBCluster.Payload.ConnectionCredentials.Host, "first-pxc-test-haproxy")
		assert.Equal(t, xtraDBCluster.Payload.ConnectionCredentials.Port, int32(3306))
		assert.NotEmpty(t, xtraDBCluster.Payload.ConnectionCredentials.Password)

		paramsUpdatePXC := xtra_db_cluster.UpdateXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.UpdateXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "second-pxc-test",
				Params: &xtra_db_cluster.UpdateXtraDBClusterParamsBodyParams{
					ClusterSize: 2,
					Proxysql: &xtra_db_cluster.UpdateXtraDBClusterParamsBodyParamsProxysql{
						ComputeResources: &xtra_db_cluster.UpdateXtraDBClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        2,
							MemoryBytes: "128",
						},
					},
					Pxc: &xtra_db_cluster.UpdateXtraDBClusterParamsBodyParamsPxc{
						ComputeResources: &xtra_db_cluster.UpdateXtraDBClusterParamsBodyParamsPxcComputeResources{
							CPUm:        2,
							MemoryBytes: "128",
						},
					},
				},
			},
		}

		_, err = dbaasClient.Default.XtraDBCluster.UpdateXtraDBCluster(&paramsUpdatePXC)
		pmmapitests.AssertAPIErrorf(t, err, 500, codes.Internal, `state is Error: XtraDB cluster is not ready`)

		for _, pxc := range xtraDBClusters.Payload.Clusters {
			if pxc.Name == "" {
				continue
			}
			deleteXtraDBClusterParamsParam := xtra_db_cluster.DeleteXtraDBClusterParams{
				Context: pmmapitests.Context,
				Body: xtra_db_cluster.DeleteXtraDBClusterBody{
					KubernetesClusterName: kubernetesClusterName,
					Name:                  pxc.Name,
				},
			}
			_, err := dbaasClient.Default.XtraDBCluster.DeleteXtraDBCluster(&deleteXtraDBClusterParamsParam)
			assert.NoError(t, err)
		}

		t.Skip("Skip restart till better implementation. https://jira.percona.com/browse/PMM-6980")
		restartXtraDBClusterParamsParam := xtra_db_cluster.RestartXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.RestartXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "first-pxc-test",
			},
		}
		_, err = dbaasClient.Default.XtraDBCluster.RestartXtraDBCluster(&restartXtraDBClusterParamsParam)
		assert.NoError(t, err)
	})

	t.Run("CreateXtraDBClusterEmptyName", func(t *testing.T) {
		paramsPXCEmptyName := xtra_db_cluster.CreateXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.CreateXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "",
				Params: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsProxysql{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
					Pxc: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxc{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxcComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
				},
			},
		}
		_, err := dbaasClient.Default.XtraDBCluster.CreateXtraDBCluster(&paramsPXCEmptyName)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `invalid field Name: value '' must be a string conforming to regex "^[a-z]([-a-z0-9]*[a-z0-9])?$"`)
	})

	t.Run("CreateXtraDBClusterInvalidName", func(t *testing.T) {
		paramsPXCInvalidName := xtra_db_cluster.CreateXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.CreateXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "123_asd",
				Params: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsProxysql{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsProxysqlComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
					Pxc: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxc{
						ComputeResources: &xtra_db_cluster.CreateXtraDBClusterParamsBodyParamsPxcComputeResources{
							CPUm:        1,
							MemoryBytes: "64",
						},
					},
				},
			},
		}
		_, err := dbaasClient.Default.XtraDBCluster.CreateXtraDBCluster(&paramsPXCInvalidName)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `invalid field Name: value '123_asd' must be a string conforming to regex "^[a-z]([-a-z0-9]*[a-z0-9])?$"`)
	})

	t.Run("ListUnknownCluster", func(t *testing.T) {
		listXtraDBClustersParamsParam := xtra_db_cluster.ListXtraDBClustersParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.ListXtraDBClustersBody{
				KubernetesClusterName: "Unknown-kubernetes-cluster-name",
			},
		}
		_, err := dbaasClient.Default.XtraDBCluster.ListXtraDBClusters(&listXtraDBClustersParamsParam)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Kubernetes Cluster with name "Unknown-kubernetes-cluster-name" not found.`)
	})

	t.Run("RestartUnknownXtraDBCluster", func(t *testing.T) {
		restartXtraDBClusterParamsParam := xtra_db_cluster.RestartXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.RestartXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "Unknown-pxc-name",
			},
		}
		_, err := dbaasClient.Default.XtraDBCluster.RestartXtraDBCluster(&restartXtraDBClusterParamsParam)
		require.Error(t, err)
		assert.Equal(t, 500, err.(pmmapitests.ErrorResponse).Code())
	})

	t.Run("DeleteUnknownXtraDBCluster", func(t *testing.T) {
		deleteXtraDBClusterParamsParam := xtra_db_cluster.DeleteXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.DeleteXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "Unknown-pxc-name",
			},
		}
		_, err := dbaasClient.Default.XtraDBCluster.DeleteXtraDBCluster(&deleteXtraDBClusterParamsParam)
		require.Error(t, err)
		assert.Equal(t, 500, err.(pmmapitests.ErrorResponse).Code())
	})

	t.Run("SuspendResumeCluster", func(t *testing.T) {
		paramsUpdatePXC := xtra_db_cluster.UpdateXtraDBClusterParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.UpdateXtraDBClusterBody{
				KubernetesClusterName: kubernetesClusterName,
				Name:                  "second-pxc-test",
				Params: &xtra_db_cluster.UpdateXtraDBClusterParamsBodyParams{
					Suspend: true,
					Resume:  true,
				},
			},
		}
		_, err := dbaasClient.Default.XtraDBCluster.UpdateXtraDBCluster(&paramsUpdatePXC)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, `resume and suspend cannot be set together`)
	})

	t.Run("GetXtraDBClusterResources", func(t *testing.T) {
		paramsXtraDBClusterResources := xtra_db_cluster.GetXtraDBClusterResourcesParams{
			Context: pmmapitests.Context,
			Body: xtra_db_cluster.GetXtraDBClusterResourcesBody{
				Params: &xtra_db_cluster.GetXtraDBClusterResourcesParamsBodyParams{
					ClusterSize: 1,
					Proxysql: &xtra_db_cluster.GetXtraDBClusterResourcesParamsBodyParamsProxysql{
						ComputeResources: &xtra_db_cluster.GetXtraDBClusterResourcesParamsBodyParamsProxysqlComputeResources{
							CPUm:        1000,
							MemoryBytes: "1000000000",
						},
					},
					Pxc: &xtra_db_cluster.GetXtraDBClusterResourcesParamsBodyParamsPxc{
						ComputeResources: &xtra_db_cluster.GetXtraDBClusterResourcesParamsBodyParamsPxcComputeResources{
							CPUm:        1000,
							MemoryBytes: "1000000000",
						},
					},
				},
			},
		}
		resources, err := dbaasClient.Default.XtraDBCluster.GetXtraDBClusterResources(&paramsXtraDBClusterResources)
		assert.NoError(t, err)
		assert.Equal(t, resources.Payload.Expected.MemoryBytes, 2000000000)
		assert.Equal(t, resources.Payload.Expected.CPUm, 2000)
		assert.Equal(t, resources.Payload.Expected.DiskSize, 2000000000)
	})
}
