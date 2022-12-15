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
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
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

func TestPXCClusterService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, dbaasClient *mockDbaasClient, grafanaClient *mockGrafanaClient, kubernetesClient *mockKubernetesClient,
		componentsService *mockComponentsService, teardown func(t *testing.T),
	) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = &mockDbaasClient{}
		grafanaClient = &mockGrafanaClient{}
		kubernetesClient = &mockKubernetesClient{}
		componentsService = &mockComponentsService{}

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
		}

		return
	}

	ctx, db, dbaasClient, grafanaClient, kubernetesClient, componentsClient, teardown := setup(t)
	defer teardown(t)
	versionService := NewVersionServiceClient(versionServiceURL)

	ks := NewKubernetesServer(db, dbaasClient, kubernetesClient, versionService, grafanaClient)
	dbaasClient.On("CheckKubernetesClusterConnection", ctx, pxcKubeconfigTest).Return(&controllerv1beta1.CheckKubernetesClusterConnectionResponse{
		Operators: &controllerv1beta1.Operators{
			PxcOperatorVersion:   "",
			PsmdbOperatorVersion: onePointEight,
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
		KubernetesClusterName: pxcKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: pxcKubeconfigTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)
	kubernetesClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubernetesClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubernetesClient.On("GetDefaultStorageClassName", mock.Anything).Return("", nil)
	kubernetesClient.On("GetClusterType", ctx).Return(kubernetes.ClusterTypeGeneric, nil)

	//nolint:dupl
	t.Run("BasicCreatePXCClusters", func(t *testing.T) {
		s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())
		kubernetesClient.On("CreateDatabaseCluster", ctx, mock.Anything).Return(nil)

		in := dbaasv1beta1.CreatePXCClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
			Params: &dbaasv1beta1.PXCClusterParams{
				ClusterSize: 5,
				Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        3,
						MemoryBytes: 256,
					},
					DiskSize: 1024 * 1024 * 1024,
					Image:    "path",
				},
				Proxysql: &dbaasv1beta1.PXCClusterParams_ProxySQL{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        2,
						MemoryBytes: 124,
					},
					DiskSize: 1024 * 1024 * 1024,
				},
			},
		}

		_, err := s.CreatePXCCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("CreatePXCClusterMinimumParams", func(t *testing.T) {
		pxcComponents := &dbaasv1beta1.GetPXCComponentsResponse{
			Versions: []*dbaasv1beta1.OperatorVersion{
				{
					Product:  "pxc-operator",
					Operator: "1.10.0",
					Matrix: &dbaasv1beta1.Matrix{
						Pxc: map[string]*dbaasv1beta1.Component{
							"8.0.19-10.1": {
								ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1",
								ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a",
								Status:    "available",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
							"8.0.20-11.1": {
								ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1",
								ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349",
								Status:    "available",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
							"8.0.25-15.1": {
								ImagePath: "percona/percona-xtradb-cluster:8.0.25-15.1",
								ImageHash: "529e979c86442429e6feabef9a2d9fc362f4626146f208fbfac704e145a492dd",
								Status:    "recommended",
								Critical:  false,
								Default:   true,
								Disabled:  false,
							},
						},
					},
				},
			},
		}
		componentsClient.On("GetPXCComponents", ctx, mock.Anything).Return(pxcComponents, nil)
		kubernetesClient.On("CreateDatabaseCluster", ctx, mock.Anything).Return(nil)

		s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())

		in := dbaasv1beta1.CreatePXCClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "fourth-pxc-test",
		}

		_, err := s.CreatePXCCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetPXCClusterCredentials", func(t *testing.T) {
		name := "third-pxc-test"
		s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())
		mockReq := &corev1.Secret{
			Data: map[string][]byte{
				"root": []byte("root_password"),
			},
		}
		dbMock := &dbaasv1.DatabaseCluster{
			Status: dbaasv1.DatabaseClusterStatus{
				Host: "hostname",
			},
		}

		kubernetesClient.On("GetDatabaseCluster", ctx, name).Return(dbMock, nil)

		kubernetesClient.On("GetSecret", ctx, mock.Anything).Return(mockReq, nil)

		in := dbaasv1beta1.GetPXCClusterCredentialsRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  name,
		}

		actual, err := s.GetPXCClusterCredentials(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, actual.ConnectionCredentials.Username, "root")
		assert.Equal(t, actual.ConnectionCredentials.Password, "root_password")
		assert.Equal(t, actual.ConnectionCredentials.Host, "hostname", name)
		assert.Equal(t, actual.ConnectionCredentials.Port, int32(3306))
	})

	t.Run("BasicGetPXCClusterCredentialsWithHost", func(t *testing.T) { // Real kubernetes will have ingress
		name := "another-third-pxc-test"
		s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())
		mockReq := &corev1.Secret{
			Data: map[string][]byte{
				"root": []byte("root_password"),
			},
		}
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Status: dbaasv1.DatabaseClusterStatus{
				Host: "amazing.com",
			},
		}

		kubernetesClient.On("GetDatabaseCluster", ctx, name).Return(dbMock, nil)

		kubernetesClient.On("GetSecret", ctx, fmt.Sprintf(pxcSecretNameTmpl, name)).Return(mockReq, nil)

		in := dbaasv1beta1.GetPXCClusterCredentialsRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  name,
		}

		actual, err := s.GetPXCClusterCredentials(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, "root", actual.ConnectionCredentials.Username)
		assert.Equal(t, "root_password", actual.ConnectionCredentials.Password)
		assert.Equal(t, "amazing.com", actual.ConnectionCredentials.Host)
		assert.Equal(t, int32(3306), actual.ConnectionCredentials.Port)
	})

	//nolint:dupl
	t.Run("BasicUpdatePXCCluster", func(t *testing.T) {
		s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())

		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "first-pxc-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "pxc",
				DatabaseImage: "percona/percona-xtradb-cluster:8.0.27-18.1",
				ClusterSize:   5,
				DBInstance: dbaasv1.DBInstanceSpec{
					CPU:      "3m",
					Memory:   "256",
					DiskSize: "1073741824",
				},
				LoadBalancer: dbaasv1.LoadBalancerSpec{
					Type: "proxysql",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2m"),
							corev1.ResourceMemory: resource.MustParse("124"),
						},
					},
				},
			},
			Status: dbaasv1.DatabaseClusterStatus{
				Ready: 15,
				Size:  15,
			},
		}
		kubernetesClient.On("GetDatabaseCluster", ctx, "first-pxc-test").Return(dbMock, nil)
		kubernetesClient.On("PatchDatabaseCluster", ctx, mock.Anything).Return(nil)
		in := dbaasv1beta1.UpdatePXCClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "third-pxc-test",
			Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
				ClusterSize: 8,
				Pxc: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_PXC{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
					Image: "path",
				},
				Proxysql: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_ProxySQL{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 124,
					},
				},
			},
		}

		_, err := s.UpdatePXCCluster(ctx, &in)
		assert.NoError(t, err)
	})

	//nolint:dupl
	t.Run("BasicSuspendResumePXCCluster", func(t *testing.T) {
		s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "forth-pxc-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "pxc",
				DatabaseImage: "percona/percona-xtradb-cluster:8.0.27-18.1",
				ClusterSize:   5,
				DBInstance: dbaasv1.DBInstanceSpec{
					CPU:      "3m",
					Memory:   "256",
					DiskSize: "1073741824",
				},
				LoadBalancer: dbaasv1.LoadBalancerSpec{
					Type: "proxysql",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2m"),
							corev1.ResourceMemory: resource.MustParse("124"),
						},
					},
				},
			},
			Status: dbaasv1.DatabaseClusterStatus{
				Ready: 15,
				Size:  15,
			},
		}
		kubernetesClient.On("GetDatabaseCluster", ctx, "forth-pxc-test").Return(dbMock, nil)
		kubernetesClient.On("PatchDatabaseCluster", ctx, mock.Anything).Return(nil)

		in := dbaasv1beta1.UpdatePXCClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "forth-pxc-test",
			Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
				Suspend: true,
			},
		}
		_, err := s.UpdatePXCCluster(ctx, &in)
		assert.NoError(t, err)

		in = dbaasv1beta1.UpdatePXCClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "forth-pxc-test",
			Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
				Resume: true,
			},
		}
		_, err = s.UpdatePXCCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetXtraDBClusterResources", func(t *testing.T) {
		t.Parallel()
		t.Run("ProxySQL", func(t *testing.T) {
			t.Parallel()
			s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())
			v := int64(1000000000)
			r := int64(2000000000)

			in := dbaasv1beta1.GetPXCClusterResourcesRequest{
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
					Proxysql: &dbaasv1beta1.PXCClusterParams_ProxySQL{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
				},
			}

			actual, err := s.GetPXCClusterResources(ctx, &in)
			assert.NoError(t, err)
			assert.Equal(t, uint64(r), actual.Expected.MemoryBytes)
			assert.Equal(t, uint64(2000), actual.Expected.CpuM)
			assert.Equal(t, uint64(r), actual.Expected.DiskSize)
		})

		t.Run("HAProxy", func(t *testing.T) {
			t.Parallel()
			s := NewPXCClusterService(db, grafanaClient, kubernetesClient, componentsClient, versionService.GetVersionServiceURL())
			v := int64(1000000000)

			in := dbaasv1beta1.GetPXCClusterResourcesRequest{
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
					},
				},
			}

			actual, err := s.GetPXCClusterResources(ctx, &in)
			assert.NoError(t, err)
			assert.Equal(t, uint64(2000000000), actual.Expected.MemoryBytes)
			assert.Equal(t, uint64(2000), actual.Expected.CpuM)
			assert.Equal(t, uint64(v), actual.Expected.DiskSize)
		})
	})
}
