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
	"fmt"
	"testing"

	"github.com/google/uuid"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

const dbKubeconfigTest = `
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
const dbKubernetesClusterNameTest = "test-k8s-db-cluster-name"

func TestDBClusterService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, dbaasClient *mockDbaasClient, grafanaClient *mockGrafanaClient, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = &mockDbaasClient{}
		grafanaClient = &mockGrafanaClient{}

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
		}

		return
	}

	ctx, db, dbaasClient, grafanaClient, teardown := setup(t)
	defer teardown(t)

	versionService := NewVersionServiceClient(versionServiceURL)

	ks := NewKubernetesServer(db, dbaasClient, versionService, grafanaClient)
	dbaasClient.On("CheckKubernetesClusterConnection", ctx, dbKubeconfigTest).Return(&controllerv1beta1.CheckKubernetesClusterConnectionResponse{
		Operators: &controllerv1beta1.Operators{
			PxcOperatorVersion:   "",
			PsmdbOperatorVersion: "",
		},
		Status: controllerv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK,
	}, nil)
	dbaasClient.On("InstallOLMOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallOLMOperatorResponse{}, nil)
	dbaasClient.On("InstallOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallOperatorResponse{}, nil)
	mockGetSubscriptionResponse := &controllerv1beta1.GetSubscriptionResponse{
		Subscription: &controllerv1beta1.Subscription{
			InstallPlanName: "mocked-install-plan",
		},
	}
	dbaasClient.On("GetSubscription", mock.Anything, mock.Anything).Return(mockGetSubscriptionResponse, nil)
	dbaasClient.On("ApproveInstallPlan", mock.Anything, mock.Anything).Return(&controllerv1beta1.ApproveInstallPlanResponse{}, nil)
	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: dbKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: dbKubeconfigTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	t.Run("BasicListPXCClusters", func(t *testing.T) {
		s := NewDBClusterService(db, dbaasClient, grafanaClient, versionService)
		mockPXCResp := controllerv1beta1.ListPXCClustersResponse{
			Clusters: []*controllerv1beta1.ListPXCClustersResponse_Cluster{
				{
					Name: "first-pxc-test",
					Params: &controllerv1beta1.PXCClusterParams{
						ClusterSize: 5,
						Pxc: &controllerv1beta1.PXCClusterParams_PXC{
							ComputeResources: &controllerv1beta1.ComputeResources{
								CpuM:        3,
								MemoryBytes: 256,
							},
							DiskSize: 1024 * 1024 * 1024,
						},
						Proxysql: &controllerv1beta1.PXCClusterParams_ProxySQL{
							ComputeResources: &controllerv1beta1.ComputeResources{
								CpuM:        2,
								MemoryBytes: 124,
							},
							DiskSize: 1024 * 1024 * 1024,
						},
					},
					Operation: &controllerv1beta1.RunningOperation{
						TotalSteps:    int32(15),
						FinishedSteps: int32(15),
					},
				},
			},
		}
		dbaasClient.On("ListPXCClusters", ctx, mock.Anything).Return(&mockPXCResp, nil)

		mockPSMDBResp := controllerv1beta1.ListPSMDBClustersResponse{
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
					Operation: &controllerv1beta1.RunningOperation{
						TotalSteps:    int32(10),
						FinishedSteps: int32(10),
					},
				},
			},
		}
		dbaasClient.On("ListPSMDBClusters", ctx, mock.Anything).Return(&mockPSMDBResp, nil)

		resp, err := s.ListDBClusters(ctx, &dbaasv1beta1.ListDBClustersRequest{KubernetesClusterName: dbKubernetesClusterNameTest})
		assert.NoError(t, err)
		assert.Len(t, resp.PxcClusters, 1)
		require.NotNil(t, resp.PxcClusters[0])
		assert.Equal(t, resp.PxcClusters[0].Name, "first-pxc-test")
		assert.Equal(t, int32(5), resp.PxcClusters[0].Params.ClusterSize)
		assert.Equal(t, int32(3), resp.PxcClusters[0].Params.Pxc.ComputeResources.CpuM)
		assert.Equal(t, int64(256), resp.PxcClusters[0].Params.Pxc.ComputeResources.MemoryBytes)
		assert.Equal(t, int32(2), resp.PxcClusters[0].Params.Proxysql.ComputeResources.CpuM)
		assert.Equal(t, int64(124), resp.PxcClusters[0].Params.Proxysql.ComputeResources.MemoryBytes)
		assert.Equal(t, int32(15), resp.PxcClusters[0].Operation.TotalSteps)
		assert.Equal(t, int32(15), resp.PxcClusters[0].Operation.FinishedSteps)

		assert.Len(t, resp.PsmdbClusters, 1)
		require.NotNil(t, resp.PsmdbClusters[0])
		assert.Equal(t, resp.PsmdbClusters[0].Name, "first-psmdb-test")
		assert.Equal(t, int32(5), resp.PsmdbClusters[0].Params.ClusterSize)
		assert.Equal(t, int32(3), resp.PsmdbClusters[0].Params.Replicaset.ComputeResources.CpuM)
		assert.Equal(t, int64(256), resp.PsmdbClusters[0].Params.Replicaset.ComputeResources.MemoryBytes)
		assert.Equal(t, int32(10), resp.PsmdbClusters[0].Operation.TotalSteps)
		assert.Equal(t, int32(10), resp.PsmdbClusters[0].Operation.FinishedSteps)
	})

	t.Run("BasicRestartPXCCluster", func(t *testing.T) {
		s := NewDBClusterService(db, dbaasClient, grafanaClient, versionService)
		mockReq := controllerv1beta1.RestartPXCClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: dbKubeconfigTest,
			},
			Name: "third-pxc-test",
		}

		dbaasClient.On("RestartPXCCluster", ctx, &mockReq).Return(&controllerv1beta1.RestartPXCClusterResponse{}, nil)

		in := dbaasv1beta1.RestartDBClusterRequest{
			KubernetesClusterName: dbKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
			ClusterType:           dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC,
		}

		_, err := s.RestartDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicRestartPSMDBCluster", func(t *testing.T) {
		s := NewDBClusterService(db, dbaasClient, grafanaClient, versionService)
		mockReq := controllerv1beta1.RestartPSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: dbKubeconfigTest,
			},
			Name: "third-psmdb-test",
		}

		dbaasClient.On("RestartPSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.RestartPSMDBClusterResponse{}, nil)

		in := dbaasv1beta1.RestartDBClusterRequest{
			KubernetesClusterName: dbKubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
			ClusterType:           dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB,
		}

		_, err := s.RestartDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicDeletePXCCluster", func(t *testing.T) {
		s := NewDBClusterService(db, dbaasClient, grafanaClient, versionService)
		dbClusterName := "delete-pxc-test"
		mockReq := controllerv1beta1.DeletePXCClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: dbKubeconfigTest,
			},
			Name: dbClusterName,
		}

		dbaasClient.On("DeletePXCCluster", ctx, &mockReq).Return(&controllerv1beta1.DeletePXCClusterResponse{}, nil)
		grafanaClient.On("DeleteAPIKeysWithPrefix", ctx, fmt.Sprintf("pxc-%s-%s", dbKubernetesClusterNameTest, dbClusterName)).Return(nil)

		in := dbaasv1beta1.DeleteDBClusterRequest{
			KubernetesClusterName: dbKubernetesClusterNameTest,
			Name:                  dbClusterName,
			ClusterType:           dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC,
		}

		_, err := s.DeleteDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicDeletePSMDBCluster", func(t *testing.T) {
		s := NewDBClusterService(db, dbaasClient, grafanaClient, versionService)
		dbClusterName := "delete-psmdb-test"
		mockReq := controllerv1beta1.DeletePSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: dbKubeconfigTest,
			},
			Name: dbClusterName,
		}

		dbaasClient.On("DeletePSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.DeletePSMDBClusterResponse{}, nil)
		grafanaClient.On("DeleteAPIKeysWithPrefix", ctx, fmt.Sprintf("psmdb-%s-%s", dbKubernetesClusterNameTest, dbClusterName)).Return(nil)

		in := dbaasv1beta1.DeleteDBClusterRequest{
			KubernetesClusterName: dbKubernetesClusterNameTest,
			Name:                  dbClusterName,
			ClusterType:           dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB,
		}

		_, err := s.DeleteDBCluster(ctx, &in)
		assert.NoError(t, err)
	})
}
