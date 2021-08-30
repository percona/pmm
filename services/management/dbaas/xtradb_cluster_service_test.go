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
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, dbaasClient *mockDbaasClient, grafanaClient *mockGrafanaClient, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = new(mockDbaasClient)
		grafanaClient = new(mockGrafanaClient)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
		}

		return
	}

	ctx, db, dbaasClient, grafanaClient, teardown := setup(t)
	defer teardown(t)

	ks := NewKubernetesServer(db, dbaasClient, grafanaClient, NewVersionServiceClient(versionServiceURL))
	dbaasClient.On("CheckKubernetesClusterConnection", ctx, pxcKubeconfigTest).Return(&controllerv1beta1.CheckKubernetesClusterConnectionResponse{
		Operators: &controllerv1beta1.Operators{
			XtradbOperatorVersion: "",
			PsmdbOperatorVersion:  onePointEight,
		},
		Status: controllerv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK,
	}, nil)

	dbaasClient.On("InstallXtraDBOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallXtraDBOperatorResponse{}, nil)
	dbaasClient.On("InstallPSMDBOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallPSMDBOperatorResponse{}, nil)

	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: pxcKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: pxcKubeconfigTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	t.Run("BasicListXtraDBClusters", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
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
							DiskSize: 1024 * 1024 * 1024,
						},
						Proxysql: &controllerv1beta1.XtraDBClusterParams_ProxySQL{
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
		assert.Equal(t, int32(15), resp.Clusters[0].Operation.TotalSteps)
		assert.Equal(t, int32(15), resp.Clusters[0].Operation.FinishedSteps)
	})

	//nolint:dupl
	t.Run("BasicCreateXtraDBClusters", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
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
					DiskSize: 1024 * 1024 * 1024,
				},
				Proxysql: &controllerv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        2,
						MemoryBytes: 124,
					},
					DiskSize: 1024 * 1024 * 1024,
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
					DiskSize: 1024 * 1024 * 1024,
				},
				Proxysql: &dbaasv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        2,
						MemoryBytes: 124,
					},
					DiskSize: 1024 * 1024 * 1024,
				},
			},
		}

		_, err := s.CreateXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetXtraDBClusterCredentials", func(t *testing.T) {
		name := "third-pxc-test"
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
		mockReq := controllerv1beta1.GetXtraDBClusterCredentialsRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: name,
		}

		dbaasClient.On("GetXtraDBClusterCredentials", ctx, &mockReq).Return(&controllerv1beta1.GetXtraDBClusterCredentialsResponse{
			Credentials: &controllerv1beta1.XtraDBCredentials{
				Username: "root",
				Password: "root_password",
				Host:     "hostname",
				Port:     3306,
			},
		}, nil)

		in := dbaasv1beta1.GetXtraDBClusterCredentialsRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  name,
		}

		actual, err := s.GetXtraDBClusterCredentials(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, actual.ConnectionCredentials.Username, "root")
		assert.Equal(t, actual.ConnectionCredentials.Password, "root_password")
		assert.Equal(t, actual.ConnectionCredentials.Host, "hostname", name)
		assert.Equal(t, actual.ConnectionCredentials.Port, int32(3306))
	})

	t.Run("BasicGetXtraDBClusterCredentialsWithHost", func(t *testing.T) { // Real kubernetes will have ingress
		name := "another-third-pxc-test"
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
		mockReq := controllerv1beta1.GetXtraDBClusterCredentialsRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: name,
		}

		mockCluster := &controllerv1beta1.GetXtraDBClusterCredentialsResponse{
			Credentials: &controllerv1beta1.XtraDBCredentials{
				Username: "root",
				Password: "root_password",
				Host:     "amazing.com",
				Port:     3306,
			},
		}

		dbaasClient.On("GetXtraDBClusterCredentials", ctx, &mockReq).Return(mockCluster, nil)

		in := dbaasv1beta1.GetXtraDBClusterCredentialsRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  name,
		}

		actual, err := s.GetXtraDBClusterCredentials(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, "root", actual.ConnectionCredentials.Username)
		assert.Equal(t, "root_password", actual.ConnectionCredentials.Password)
		assert.Equal(t, mockCluster.Credentials.Host, actual.ConnectionCredentials.Host)
		assert.Equal(t, int32(3306), actual.ConnectionCredentials.Port)
	})

	//nolint:dupl
	t.Run("BasicUpdateXtraDBCluster", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
		mockReq := controllerv1beta1.UpdateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "third-pxc-test",
			Params: &controllerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
				ClusterSize: 8,
				Pxc: &controllerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams_PXC{
					ComputeResources: &controllerv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
				},
				Proxysql: &controllerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams_ProxySQL{
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
			Params: &dbaasv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
				ClusterSize: 8,
				Pxc: &dbaasv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams_PXC{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
				},
				Proxysql: &dbaasv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams_ProxySQL{
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

	//nolint:dupl
	t.Run("BasicSuspendResumeXtraDBCluster", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
		mockReqSuspend := controllerv1beta1.UpdateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "forth-pxc-test",
			Params: &controllerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
				Suspend: true,
			},
		}

		dbaasClient.On("UpdateXtraDBCluster", ctx, &mockReqSuspend).Return(&controllerv1beta1.UpdateXtraDBClusterResponse{}, nil)

		in := dbaasv1beta1.UpdateXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "forth-pxc-test",
			Params: &dbaasv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
				Suspend: true,
			},
		}
		_, err := s.UpdateXtraDBCluster(ctx, &in)
		assert.NoError(t, err)

		mockReqResume := controllerv1beta1.UpdateXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "forth-pxc-test",
			Params: &controllerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
				Resume: true,
			},
		}
		dbaasClient.On("UpdateXtraDBCluster", ctx, &mockReqResume).Return(&controllerv1beta1.UpdateXtraDBClusterResponse{}, nil)

		in = dbaasv1beta1.UpdateXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "forth-pxc-test",
			Params: &dbaasv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
				Resume: true,
			},
		}
		_, err = s.UpdateXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicRestartXtraDBCluster", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
		mockReq := controllerv1beta1.RestartXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: "third-pxc-test",
		}

		dbaasClient.On("RestartXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.RestartXtraDBClusterResponse{}, nil)

		in := dbaasv1beta1.RestartXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
		}

		_, err := s.RestartXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicDeleteXtraDBCluster", func(t *testing.T) {
		s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
		dbClusterName := "delete-pxc-test"
		mockReq := controllerv1beta1.DeleteXtraDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: pxcKubeconfigTest,
			},
			Name: dbClusterName,
		}

		dbaasClient.On("DeleteXtraDBCluster", ctx, &mockReq).Return(&controllerv1beta1.DeleteXtraDBClusterResponse{}, nil)
		grafanaClient.On("DeleteAPIKeysWithPrefix", ctx, fmt.Sprintf("pxc-%s-%s", kubernetesClusterNameTest, dbClusterName)).Return(nil)

		in := dbaasv1beta1.DeleteXtraDBClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  dbClusterName,
		}

		_, err := s.DeleteXtraDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetXtraDBClusterResources", func(t *testing.T) {
		t.Parallel()
		t.Run("ProxySQL", func(t *testing.T) {
			t.Parallel()
			s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
			v := int64(1000000000)
			r := int64(2000000000)

			in := dbaasv1beta1.GetXtraDBClusterResourcesRequest{
				Params: &dbaasv1beta1.XtraDBClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.XtraDBClusterParams_PXC{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
					Proxysql: &dbaasv1beta1.XtraDBClusterParams_ProxySQL{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
				},
			}

			actual, err := s.GetXtraDBClusterResources(ctx, &in)
			assert.NoError(t, err)
			assert.Equal(t, uint64(r), actual.Expected.MemoryBytes)
			assert.Equal(t, uint64(2000), actual.Expected.CpuM)
			assert.Equal(t, uint64(r), actual.Expected.DiskSize)
		})

		t.Run("HAProxy", func(t *testing.T) {
			t.Parallel()
			s := NewXtraDBClusterService(db, dbaasClient, grafanaClient)
			v := int64(1000000000)

			in := dbaasv1beta1.GetXtraDBClusterResourcesRequest{
				Params: &dbaasv1beta1.XtraDBClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.XtraDBClusterParams_PXC{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
					Haproxy: &dbaasv1beta1.XtraDBClusterParams_HAProxy{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
					},
				},
			}

			actual, err := s.GetXtraDBClusterResources(ctx, &in)
			assert.NoError(t, err)
			assert.Equal(t, uint64(2000000000), actual.Expected.MemoryBytes)
			assert.Equal(t, uint64(2000), actual.Expected.CpuM)
			assert.Equal(t, uint64(v), actual.Expected.DiskSize)
		})
	})
}
