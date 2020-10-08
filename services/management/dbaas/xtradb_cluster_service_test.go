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
	"reflect"
	"testing"

	"github.com/google/uuid"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/sirupsen/logrus"
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

const kubeconfTest = `
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
const kubernetesClusterNameTest = "test-k8s-cluster-name"

func Test_XtraDBClusterService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
		}

		return
	}

	l := logrus.WithField("component", "xtradb_cluster_test")

	ctx, db, teardown := setup(t)
	defer teardown(t)

	ks := NewKubernetesServer(db)

	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: kubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: kubeconfTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	t.Run("BasicListXtraDBClusters", func(t *testing.T) {
		c := new(MockXtraDBClusterAPIConnector)
		c.Test(t)

		defer c.AssertExpectations(t)
		client := Client{
			XtraDBClusterAPIClient: c,
		}
		mockResp := controllerv1beta1.ListXtraDBClustersResponse{
			Clusters: []*controllerv1beta1.ListXtraDBClustersResponse_Cluster{
				{
					Name: "first.pxc.test.percona.com",
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

		c.On("ListXtraDBClusters", ctx, mock.AnythingOfType(reflect.TypeOf(&controllerv1beta1.ListXtraDBClustersRequest{}).String())).Return(&mockResp, nil)

		s := XtraDBClusterService{
			db:               db,
			l:                l,
			controllerClient: client.XtraDBClusterAPIClient,
		}

		resp, err := s.ListXtraDBClusters(ctx, &dbaasv1beta1.ListXtraDBClustersRequest{KubernetesClusterName: kubernetesClusterNameTest})
		assert.NoError(t, err)
		assert.Equal(t, resp.Clusters[0].Name, "first.pxc.test.percona.com")
		assert.Equal(t, int32(5), resp.Clusters[0].Params.ClusterSize)
		assert.Equal(t, int32(3), resp.Clusters[0].Params.Pxc.ComputeResources.CpuM)
		assert.Equal(t, int64(256), resp.Clusters[0].Params.Pxc.ComputeResources.MemoryBytes)
		assert.Equal(t, int32(2), resp.Clusters[0].Params.Proxysql.ComputeResources.CpuM)
		assert.Equal(t, int64(124), resp.Clusters[0].Params.Proxysql.ComputeResources.MemoryBytes)
	})

	//nolint:dupl
	t.Run("BasicCreateXtraDBClusters", func(t *testing.T) {
		c := new(MockXtraDBClusterAPIConnector)
		c.Test(t)

		defer c.AssertExpectations(t)
		client := Client{
			XtraDBClusterAPIClient: c,
		}
		mockReq := controllerv1beta1.CreateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third.pxc.test.percona.com",
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

		c.On("CreateXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.CreateXtraDBClusterResponse{}, nil)

		s := XtraDBClusterService{
			db:               db,
			l:                l,
			controllerClient: client.XtraDBClusterAPIClient,
		}
		in := dbaasv1beta1.CreateXtraDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third.pxc.test.percona.com",
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

	//nolint:dupl
	t.Run("BasicUpdateXtraDBCluster", func(t *testing.T) {
		c := new(MockXtraDBClusterAPIConnector)
		c.Test(t)

		defer c.AssertExpectations(t)
		client := Client{
			XtraDBClusterAPIClient: c,
		}
		mockReq := controllerv1beta1.UpdateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third.pxc.test.percona.com",
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

		c.On("UpdateXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.UpdateXtraDBClusterResponse{}, nil)

		s := XtraDBClusterService{
			db:               db,
			l:                l,
			controllerClient: client.XtraDBClusterAPIClient,
		}
		in := dbaasv1beta1.UpdateXtraDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third.pxc.test.percona.com",
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
		c := new(MockXtraDBClusterAPIConnector)
		c.Test(t)

		defer c.AssertExpectations(t)
		client := Client{
			XtraDBClusterAPIClient: c,
		}
		mockReq := controllerv1beta1.DeleteXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third.pxc.test.percona.com",
		}

		c.On("DeleteXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.DeleteXtraDBClusterResponse{}, nil)

		s := XtraDBClusterService{
			db:               db,
			l:                l,
			controllerClient: client.XtraDBClusterAPIClient,
		}
		in := dbaasv1beta1.DeleteXtraDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third.pxc.test.percona.com",
		}

		_, err := s.DeleteXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})
}
