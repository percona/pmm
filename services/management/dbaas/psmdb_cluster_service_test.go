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

const kubeconfTest = `
	{
		"apiVersion": "v1",
		"kind": "Config",
		"users": [
			{
				"name": "percona-server-mongodb-operator",
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
					"user": "percona-server-mongodb-operator"
				},
				"name": "svcs-acct-context"
			}
		],
		"current-context": "svcs-acct-context"
	}
`
const kubernetesClusterNameTest = "test-k8s-cluster-name"

func TestPSMDBClusterService(t *testing.T) {
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
	dbaasClient.On("CheckKubernetesClusterConnection", ctx, kubeconfTest).Return(&controllerv1beta1.CheckKubernetesClusterConnectionResponse{
		Operators: &controllerv1beta1.Operators{
			Xtradb: &controllerv1beta1.Operator{Status: controllerv1beta1.OperatorsStatus_OPERATORS_STATUS_NOT_INSTALLED},
			Psmdb:  &controllerv1beta1.Operator{Status: controllerv1beta1.OperatorsStatus_OPERATORS_STATUS_OK},
		},
		Status: controllerv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK,
	}, nil)

	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: kubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: kubeconfTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	t.Run("BasicListPSMDBClusters", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)
		mockResp := controllerv1beta1.ListPSMDBClustersResponse{
			Clusters: []*controllerv1beta1.ListPSMDBClustersResponse_Cluster{
				{
					Name: "first-psmdb-test",
					Params: &controllerv1beta1.PSMDBClusterParams{
						ClusterSize: 5,
						Replicaset: &controllerv1beta1.PSMDBClusterParams_ReplicaSet{
							ComputeResources: &controllerv1beta1.ComputeResources{
								CpuM:        3,
								MemoryBytes: 256,
							},
						},
					},
				},
			},
		}

		dbaasClient.On("ListPSMDBClusters", ctx, mock.Anything).Return(&mockResp, nil)

		resp, err := s.ListPSMDBClusters(ctx, &dbaasv1beta1.ListPSMDBClustersRequest{KubernetesClusterName: kubernetesClusterNameTest})
		assert.NoError(t, err)
		require.NotNil(t, resp.Clusters[0])
		assert.Equal(t, resp.Clusters[0].Name, "first-psmdb-test")
		assert.Equal(t, int32(5), resp.Clusters[0].Params.ClusterSize)
		assert.Equal(t, int32(3), resp.Clusters[0].Params.Replicaset.ComputeResources.CpuM)
		assert.Equal(t, int64(256), resp.Clusters[0].Params.Replicaset.ComputeResources.MemoryBytes)
	})

	//nolint:dupl
	t.Run("BasicCreatePSMDBClusters", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.CreatePSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third-psmdb-test",
			Params: &controllerv1beta1.PSMDBClusterParams{
				ClusterSize: 5,
				Replicaset: &controllerv1beta1.PSMDBClusterParams_ReplicaSet{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        3,
						MemoryBytes: 256,
					},
					DiskSize: 1024 * 1024 * 1024,
				},
			},
		}

		dbaasClient.On("CreatePSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.CreatePSMDBClusterResponse{}, nil)

		in := dbaasv1beta1.CreatePSMDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
			Params: &dbaasv1beta1.PSMDBClusterParams{
				ClusterSize: 5,
				Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        3,
						MemoryBytes: 256,
					},
					DiskSize: 1024 * 1024 * 1024,
				},
			},
		}

		_, err := s.CreatePSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	//nolint:dupl
	t.Run("BasicUpdatePSMDBCluster", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.UpdatePSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third-psmdb-test",
			Params: &controllerv1beta1.UpdatePSMDBClusterRequest_UpdatePSMDBClusterParams{
				ClusterSize: 8,
				Replicaset: &controllerv1beta1.UpdatePSMDBClusterRequest_UpdatePSMDBClusterParams_ReplicaSet{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
				},
			},
		}

		dbaasClient.On("UpdatePSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.UpdatePSMDBClusterResponse{}, nil)

		in := dbaasv1beta1.UpdatePSMDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
			Params: &dbaasv1beta1.UpdatePSMDBClusterRequest_UpdatePSMDBClusterParams{
				ClusterSize: 8,
				Replicaset: &dbaasv1beta1.UpdatePSMDBClusterRequest_UpdatePSMDBClusterParams_ReplicaSet{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
				},
			},
		}

		_, err := s.UpdatePSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetPSMDBCluster", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)

		mockReq := controllerv1beta1.GetPSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third-psmdb-test",
		}

		dbaasClient.On("GetPSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.GetPSMDBClusterResponse{
			Credentials: &controllerv1beta1.PSMDBCredentials{
				Username:   "userAdmin",
				Password:   "userAdmin123",
				Host:       "hostname",
				Port:       27017,
				Replicaset: "rs0",
			},
		}, nil)

		in := dbaasv1beta1.GetPSMDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
		}

		cluster, err := s.GetPSMDBCluster(ctx, &in)

		assert.NoError(t, err)
		assert.Equal(t, "hostname", cluster.ConnectionCredentials.Host)
	})

	t.Run("BasicGetPSMDBClusterWithHost", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)
		name := "another-third-psmdb-test"

		mockReq := controllerv1beta1.GetPSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: name,
		}

		resp := controllerv1beta1.GetPSMDBClusterResponse{
			Credentials: &controllerv1beta1.PSMDBCredentials{
				Host: "host",
			},
		}
		dbaasClient.On("GetPSMDBCluster", ctx, &mockReq).Return(&resp, nil)

		in := dbaasv1beta1.GetPSMDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  name,
		}

		cluster, err := s.GetPSMDBCluster(ctx, &in)

		assert.NoError(t, err)
		assert.Equal(t, resp.Credentials.Host, cluster.ConnectionCredentials.Host)
	})

	t.Run("BasicRestartPSMDBCluster", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.RestartPSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third-psmdb-test",
		}

		dbaasClient.On("RestartPSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.RestartPSMDBClusterResponse{}, nil)

		in := dbaasv1beta1.RestartPSMDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
		}

		_, err := s.RestartPSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicDeletePSMDBCluster", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient)
		mockReq := controllerv1beta1.DeletePSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: kubeconfTest,
			},
			Name: "third-psmdb-test",
		}

		dbaasClient.On("DeletePSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.DeletePSMDBClusterResponse{}, nil)

		in := dbaasv1beta1.DeletePSMDBClusterRequest{
			KubernetesClusterName: kubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
		}

		_, err := s.DeletePSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})
}
