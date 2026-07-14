// Copyright (C) 2023 Percona LLC
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
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	goversion "github.com/hashicorp/go-version"
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
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
	pmmversion "github.com/percona/pmm/version"
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

func TestPXCClusterService(t *testing.T) { //nolint:tparallel
	// This is for local testing. When running local tests, if pmmversion.PMMVersion is empty
	// these lines in kubernetes_server.go will throw an error and tests won't finish.
	//
	//     pmmVersion, err := goversion.NewVersion(pmmversion.PMMVersion)
	//     if err != nil {
	//     	return nil, status.Error(codes.Internal, err.Error())
	//     }
	//
	if pmmversion.PMMVersion == "" {
		pmmversion.PMMVersion = "2.30.0"
	}
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, dbaasClient *mockDbaasClient, grafanaClient *mockGrafanaClient,
		componentsService *mockComponentsService, kubeClient *mockKubernetesClient, teardown func(t *testing.T),
	) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, nil)
		dbaasClient = &mockDbaasClient{}
		grafanaClient = &mockGrafanaClient{}
		kubeClient = &mockKubernetesClient{}
		componentsService = &mockComponentsService{}

		teardown = func(t *testing.T) {
			t.Helper()
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
		}

		return
	}

	ctx, db, dbaasClient, grafanaClient, componentsClient, kubeClient, teardown := setup(t)
	t.Cleanup(func() { teardown(t) })

	versionService := &mockVersionService{}
	v1120, _ := goversion.NewVersion("1.12.0")
	versionService.On("LatestOperatorVersion", mock.Anything, mock.Anything).Return(v1120, v1120, nil)
	versionService.On("GetVersionServiceURL", mock.Anything).Return("", nil)

	ks := NewKubernetesServer(db, dbaasClient, versionService, grafanaClient)

	grafanaClient.On("CreateAdminAPIKey", mock.Anything, mock.Anything).Return(int64(123456), "api-key", nil)
	kubeClient.On("InstallOLMOperator", mock.Anything, mock.Anything).Return(nil)
	kubeClient.On("InstallOperator", mock.Anything, mock.Anything).Return(nil)
	kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	dbaasClient.On("StartMonitoring", mock.Anything, mock.Anything).WaitUntil(time.After(15*time.Second)).
		Return(&controllerv1beta1.StartMonitoringResponse{}, nil).Run(func(a mock.Arguments) {
		// StartMonitoring if being called in a go-routine. Since we cannot forsee when the runtime scheduler
		// is going to assing some time to this go-routine, the waitgroup is being used to signal than the test
		// can continue.
		wg.Done()
	})

	kubeClient.On("GetServerVersion").Return(nil, nil)
	clients := map[string]kubernetesClient{
		pxcKubernetesClusterNameTest: kubeClient,
	}
	s := ks.(*kubernetesServer)
	s.kubeStorage.clients = clients
	ks = s
	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: pxcKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: pxcKubeconfigTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	wg.Wait()

	kubeClient.On("SetKubeconfig", mock.Anything).Return(nil)
	kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("GetDefaultStorageClassName", mock.Anything).Return("", nil)
	kubeClient.On("GetClusterType", ctx).Return(kubernetes.ClusterTypeGeneric, nil)
	kubeClient.On("CreatePMMSecret", mock.Anything, mock.Anything).Return(nil, nil)

	//nolint:dupl
	t.Run("BasicCreatePXCClusters", func(t *testing.T) {
		cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PXCClustersService)
		s.kubeStorage.clients = clients
		kubeClient.On("CreateDatabaseCluster", mock.Anything).Return(nil)

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
		kubeClient.On("CreateDatabaseCluster", mock.Anything).Return(nil)

		cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PXCClustersService)
		s.kubeStorage.clients = clients

		in := dbaasv1beta1.CreatePXCClusterRequest{
			KubernetesClusterName: pxcKubernetesClusterNameTest,
			Name:                  "fourth-pxc-test",
		}

		_, err := s.CreatePXCCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetPXCClusterCredentials", func(t *testing.T) {
		name := "third-pxc-test"
		cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PXCClustersService)
		s.kubeStorage.clients = clients

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
				Host: "hostname",
			},
			Spec: dbaasv1.DatabaseSpec{
				SecretsName: fmt.Sprintf(pxcSecretNameTmpl, name),
			},
		}

		kubeClient.On("GetDatabaseCluster", ctx, name).Return(dbMock, nil)

		kubeClient.On("GetSecret", ctx, mock.Anything).Return(mockReq, nil)

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
		cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PXCClustersService)
		s.kubeStorage.clients = clients
		mockReq := &corev1.Secret{
			Data: map[string][]byte{
				"root": []byte("root_password"),
			},
		}
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: dbaasv1.DatabaseSpec{
				SecretsName: fmt.Sprintf(pxcSecretNameTmpl, name),
			},
			Status: dbaasv1.DatabaseClusterStatus{
				Host: "amazing.com",
			},
		}

		kubeClient.On("GetDatabaseCluster", ctx, name).Return(dbMock, nil)

		kubeClient.On("GetSecret", ctx, fmt.Sprintf(pxcSecretNameTmpl, name)).Return(mockReq, nil)

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
		cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PXCClustersService)
		s.kubeStorage.clients = clients

		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "first-pxc-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "pxc",
				DatabaseImage: "percona/percona-xtradb-cluster:8.0.27-18.1",
				ClusterSize:   5,
				DBInstance: dbaasv1.DBInstanceSpec{
					CPU:      resource.MustParse("3m"),
					Memory:   resource.MustParse("256"),
					DiskSize: resource.MustParse("1073741824"),
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
		kubeClient.On("GetDatabaseCluster", ctx, "first-pxc-test").Return(dbMock, nil)
		kubeClient.On("PatchDatabaseCluster", mock.Anything).Return(nil)
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
		cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PXCClustersService)
		s.kubeStorage.clients = clients
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "forth-pxc-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "pxc",
				DatabaseImage: "percona/percona-xtradb-cluster:8.0.27-18.1",
				ClusterSize:   5,
				DBInstance: dbaasv1.DBInstanceSpec{
					CPU:      resource.MustParse("3m"),
					Memory:   resource.MustParse("256"),
					DiskSize: resource.MustParse("1073741824"),
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
		kubeClient.On("GetDatabaseCluster", ctx, "forth-pxc-test").Return(dbMock, nil)
		kubeClient.On("PatchDatabaseCluster", mock.Anything).Return(nil)

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
			cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
			s := cs.(*PXCClustersService)
			s.kubeStorage.clients = clients
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
			cs := NewPXCClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
			s := cs.(*PXCClustersService)
			s.kubeStorage.clients = clients
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
