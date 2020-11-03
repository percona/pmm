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

// Package dbaas contains all logic related to dbaas services.
package dbaas

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

const pxcKubeconfigTest = `
	{
		"apiVersion": "v1",
		"kind": "Config",
		"users": [
			{
				"name": "percona-xtradb-cluster-operator",
				"user": {
					"token": "some-token"
				}
			}
		],
		"clusters": [
			{
				"cluster": {
					"certificate-authority-data": "some-certificate-authority-data",
					"server": "https://192.168.0.42:8443"
				},
				"name": "self-hosted-cluster"
			}
		],
		"contexts": [
			{
				"context": {
					"cluster": "self-hosted-cluster",
					"user": "percona-xtradb-cluster-operator"
				},
				"name": "svcs-acct-context"
			}
		],
		"current-context": "svcs-acct-context"
	}
`
const pxcKubernetesClusterNameTest = "test-k8s-cluster-name"

func TestXtraDBClusterService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, dbaasClient *mockDbaasClient, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = new(mockDbaasClient)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
		}

		return
	}

	ctx, db, dbaasClient, teardown := setup(t)
	defer teardown(t)

	ks := NewKubernetesServer(db, dbaasClient)
	dbaasClient.On("CheckKubernetesClusterConnection", ctx, pxcKubeconfigTest).Return(nil)

	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: pxcKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: pxcKubeconfigTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	t.Run("BasicListXtraDBClusters", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient)
		mockResp := controllerv1beta1.ListXtraDBClustersResponse{
			Clusters: []*controllerv1beta1.ListXtraDBClustersResponse_Cluster{
				{
					Name: "first-pxc-test",
					Params: &controllerv1beta1.XtraDBClusterParams{
						ClusterSize: 5,
						Pxc: &controllerv1beta1.XtraDBClusterParams_PXC{
							ComputeResources: &controllerv1beta1.ComputeResources{
								CpuM:        3,
								MemoryBytes: 256,
							},
						},
						Proxysql: &controllerv1beta1.XtraDBClusterParams_ProxySQL{
							ComputeResources: &controllerv1beta1.ComputeResources{
								CpuM:        2,
								MemoryBytes: 124,
							},
						},
					},
				},
			},
		}

		dbaasClient.On("ListXtraDBClusters", ctx, mock.Anything).Return(&mockResp, nil)

		resp, err := s.ListXtraDBClusters(ctx, &dbaasv1beta1.ListXtraDBClustersRequest{KubernetesClusterName: pxcKubernetesClusterNameTest})
		assert.NoError(t, err)
		require.NotNil(t, resp.Clusters[0])
		assert.Equal(t, resp.Clusters[0].Name, "first-pxc-test")
		assert.Equal(t, int32(5), resp.Clusters[0].Params.ClusterSize)
		assert.Equal(t, int32(3), resp.Clusters[0].Params.Pxc.ComputeResources.CpuM)
		assert.Equal(t, int64(256), resp.Clusters[0].Params.Pxc.ComputeResources.MemoryBytes)
		assert.Equal(t, int32(2), resp.Clusters[0].Params.Proxysql.ComputeResources.CpuM)
		assert.Equal(t, int64(124), resp.Clusters[0].Params.Proxysql.ComputeResources.MemoryBytes)
	})

	//nolint:dupl
	t.Run("BasicCreateXtraDBClusters", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.CreateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "third-pxc-test",
			Params: &controllerv1beta1.XtraDBClusterParams{
				ClusterSize: 5,
				Pxc: &controllerv1beta1.XtraDBClusterParams_PXC{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        3,
						MemoryBytes: 256,
					},
				},
				Proxysql: &controllerv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        2,
						MemoryBytes: 124,
					},
				},
			},
		}

		dbaasClient.On("CreateXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.CreateXtraDBClusterResponse{}, nil)

		in := dbaasv1beta1.CreateXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
			Params: &dbaasv1beta1.XtraDBClusterParams{
				ClusterSize: 5,
				Pxc: &dbaasv1beta1.XtraDBClusterParams_PXC{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        3,
						MemoryBytes: 256,
					},
				},
				Proxysql: &dbaasv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        2,
						MemoryBytes: 124,
					},
				},
			},
		}

		_, err := s.CreateXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetXtraDBCluster", func(t *testing.T) {
		name := "third-pxc-test"
		s := NewXtraDBClusterService(db, dbaasClient)
		in := dbaasv1beta1.GetXtraDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  name,
		}

		actual, err := s.GetXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, actual.ConnectionCredentials.Username, "root")
		assert.Equal(t, actual.ConnectionCredentials.Password, "root_password")
		assert.Equal(t, actual.ConnectionCredentials.Host, fmt.Sprintf("%s-proxysql", name))
		assert.Equal(t, actual.ConnectionCredentials.Port, int32(3306))
	})

	//nolint:dupl
	t.Run("BasicUpdateXtraDBCluster", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.UpdateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "third-pxc-test",
			Params: &controllerv1beta1.XtraDBClusterParams{
				ClusterSize: 8,
				Pxc: &controllerv1beta1.XtraDBClusterParams_PXC{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
				},
				Proxysql: &controllerv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 124,
					},
				},
			},
		}

		dbaasClient.On("UpdateXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.UpdateXtraDBClusterResponse{}, nil)

		in := dbaasv1beta1.UpdateXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
			Params: &dbaasv1beta1.XtraDBClusterParams{
				ClusterSize: 8,
				Pxc: &dbaasv1beta1.XtraDBClusterParams_PXC{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
				},
				Proxysql: &dbaasv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 124,
					},
				},
			},
		}

		_, err := s.UpdateXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicDeleteXtraDBCluster", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.DeleteXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "third-pxc-test",
		}

		dbaasClient.On("DeleteXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.DeleteXtraDBClusterResponse{}, nil)

		in := dbaasv1beta1.DeleteXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
		}

		_, err := s.DeleteXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})
}
