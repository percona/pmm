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
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	pmmversion "github.com/percona/pmm/version"
)

const postgresqlKubeconfigTest = `
{
	"apiVersion": "v1",
	"kind": "Config",
	"users": [
		{
			"name": "percona-postgresql-cluster-operator",
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
				"user": "percona-postgresql-cluster-operator"
			},
			"name": "svcs-acct-context"
		}
	],
	"current-context": "svcs-acct-context"
}
`
const postgresqlKubernetesClusterNameTest = "test-k8s-cluster-name"

func TestPostgresqlClusterService(t *testing.T) {
	// This is for local testing. When running local tests, if pmmversion.PMMVersion is empty
	// these lines in kubernetes_server.go will throw an error and tests won't finish.
	//
	//     pmmVersion, err := goversion.NewVersion(pmmversion.PMMVersion)
	//     if err != nil {
	//     	return nil, status.Error(codes.Internal, err.Error())
	//     }
	//
	if pmmversion.PMMVersion == "" {
		pmmversion.PMMVersion = "2.37.0"
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
	defer teardown(t)

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
	kubeClient.On("GetPGOperatorVersion", mock.Anything, mock.Anything).Return("2.0.0", nil)

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
		postgresqlKubernetesClusterNameTest: kubeClient,
	}
	s := ks.(*kubernetesServer)
	s.kubeStorage.clients = clients
	ks = s
	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: postgresqlKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: postgresqlKubeconfigTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)

	wg.Wait()

	kubeClient.On("SetKubeconfig", mock.Anything).Return(nil)
	kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("GetPGOperatorVersion", mock.Anything, mock.Anything).Return("2.0.0", nil)
	kubeClient.On("GetDefaultStorageClassName", mock.Anything).Return("", nil)
	kubeClient.On("GetClusterType", ctx).Return(kubernetes.ClusterTypeGeneric, nil)
	kubeClient.On("CreatePMMSecret", mock.Anything, mock.Anything).Return(nil, nil)

	//nolint:dupl
	t.Run("BasicCreatePostgresqlClusters", func(t *testing.T) {
		cs := NewPostgresqlClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PostgresqlClustersService)
		s.kubeStorage.clients = clients
		kubeClient.On("CreateDatabaseCluster", mock.Anything).Return(nil)

		in := dbaasv1beta1.CreatePostgresqlClusterRequest{
			KubernetesClusterName: postgresqlKubernetesClusterNameTest,
			Name:                  "third-postgresql-test",
			Params: &dbaasv1beta1.PostgresqlClusterParams{
				ClusterSize: 5,
				Instance: &dbaasv1beta1.PostgresqlClusterParams_Instance{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        3,
						MemoryBytes: 256,
					},
					DiskSize: 1024 * 1024 * 1024,
				},
				Pgbouncer: &dbaasv1beta1.PostgresqlClusterParams_PGBouncer{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        2,
						MemoryBytes: 124,
					},
					DiskSize: 1024 * 1024 * 1024,
				},
			},
		}

		_, err := s.CreatePostgresqlCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("CreatePostgresqlClusterMinimumParams", func(t *testing.T) {
		pgComponents := &dbaasv1beta1.GetPGComponentsResponse{
			Versions: []*dbaasv1beta1.OperatorVersion{
				{
					Product:  "pg-operator",
					Operator: "1.10.0",
					Matrix: &dbaasv1beta1.Matrix{
						Postgresql: map[string]*dbaasv1beta1.Component{
							"14.7": {
								ImagePath: "percona/percona-postgresql-operator:2.0.0-ppg14-postgres",
								ImageHash: "bf47531669ab49a26479f46efc78ed42b9393325cfac1b00c3e340987c8869f0",
								Status:    "recommended",
							},
						},
						Pgbouncer: map[string]*dbaasv1beta1.Component{
							"14.7": {
								ImagePath: "percona/percona-postgresql-operator:2.0.0-ppg14-pgbouncer",
								ImageHash: "64de9cd659e2d6f75bea9263b23a72e5aa9b00560ae403249c92a3439a2fd527",
								Status:    "recommended",
							},
						},
						Pgbackrest: map[string]*dbaasv1beta1.Component{
							"14.7": {
								ImagePath: "percona/percona-postgresql-operator:2.0.0-ppg14-pgbackrest",
								ImageHash: "9bcac75e97204eb78296f4befff555cad1600373ed5fd76576e0401a8c8eb4e6",
								Status:    "recommended",
							},
						},
					},
				},
			},
		}
		componentsClient.On("GetPGComponents", ctx, mock.Anything).Return(pgComponents, nil)
		kubeClient.On("CreateDatabaseCluster", mock.Anything).Return(nil)

		cs := NewPostgresqlClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PostgresqlClustersService)
		s.kubeStorage.clients = clients

		in := dbaasv1beta1.CreatePostgresqlClusterRequest{
			KubernetesClusterName: postgresqlKubernetesClusterNameTest,
			Name:                  "fourth-postgresql-test",
		}

		_, err := s.CreatePostgresqlCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetPostgresqlClusterCredentials", func(t *testing.T) {
		name := "third-postgresql-test"
		cs := NewPostgresqlClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PostgresqlClustersService)
		s.kubeStorage.clients = clients

		mockReq := &corev1.Secret{
			Data: map[string][]byte{
				"user":     []byte("some_username"),
				"password": []byte("user_password"),
				"host":     []byte("amazing.com"),
				"port":     []byte("5432"),
			},
		}
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: dbaasv1.DatabaseSpec{
				SecretsName: fmt.Sprintf(postgresqlSecretNameTmpl, name, name),
			},
		}

		kubeClient.On("GetDatabaseCluster", ctx, name).Return(dbMock, nil)

		kubeClient.On("GetSecret", ctx, fmt.Sprintf(postgresqlSecretNameTmpl, name, name)).Return(mockReq, nil)

		in := dbaasv1beta1.GetPostgresqlClusterCredentialsRequest{
			KubernetesClusterName: postgresqlKubernetesClusterNameTest,
			Name:                  name,
		}

		actual, err := s.GetPostgresqlClusterCredentials(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, actual.ConnectionCredentials.Username, "some_username")
		assert.Equal(t, actual.ConnectionCredentials.Password, "user_password")
		assert.Equal(t, actual.ConnectionCredentials.Host, "amazing.com", name)
		assert.Equal(t, actual.ConnectionCredentials.Port, int32(5432))
	})

	//nolint:dupl
	t.Run("BasicUpdatePostgresqlCluster", func(t *testing.T) {
		cs := NewPostgresqlClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PostgresqlClustersService)
		s.kubeStorage.clients = clients

		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "first-postgresql-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "postgresql",
				DatabaseImage: "percona/percona-postgresql-operator:2.0.0-ppg14-postgres",
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
		kubeClient.On("GetDatabaseCluster", ctx, "first-postgresql-test").Return(dbMock, nil)
		kubeClient.On("PatchDatabaseCluster", mock.Anything).Return(nil)
		in := dbaasv1beta1.UpdatePostgresqlClusterRequest{
			KubernetesClusterName: postgresqlKubernetesClusterNameTest,
			Name:                  "third-postgresql-test",
			Params: &dbaasv1beta1.UpdatePostgresqlClusterRequest_UpdatePostgresqlClusterParams{
				ClusterSize: 8,
				Instance: &dbaasv1beta1.UpdatePostgresqlClusterRequest_UpdatePostgresqlClusterParams_Instance{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 256,
					},
					Image: "path",
				},
				Pgbouncer: &dbaasv1beta1.UpdatePostgresqlClusterRequest_UpdatePostgresqlClusterParams_PGBouncer{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        1,
						MemoryBytes: 124,
					},
				},
			},
		}

		_, err := s.UpdatePostgresqlCluster(ctx, &in)
		assert.NoError(t, err)
	})

	//nolint:dupl
	t.Run("BasicSuspendResumePostgresqlCluster", func(t *testing.T) {
		cs := NewPostgresqlClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
		s := cs.(*PostgresqlClustersService)
		s.kubeStorage.clients = clients
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "forth-postgresql-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "postgresql",
				DatabaseImage: "percona/percona-postgresql-operator:2.0.0-ppg14-postgres",
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
		kubeClient.On("GetDatabaseCluster", ctx, "forth-postgresql-test").Return(dbMock, nil)
		kubeClient.On("PatchDatabaseCluster", mock.Anything).Return(nil)

		in := dbaasv1beta1.UpdatePostgresqlClusterRequest{
			KubernetesClusterName: postgresqlKubernetesClusterNameTest,
			Name:                  "forth-postgresql-test",
			Params: &dbaasv1beta1.UpdatePostgresqlClusterRequest_UpdatePostgresqlClusterParams{
				Suspend: true,
			},
		}
		_, err := s.UpdatePostgresqlCluster(ctx, &in)
		assert.NoError(t, err)

		in = dbaasv1beta1.UpdatePostgresqlClusterRequest{
			KubernetesClusterName: postgresqlKubernetesClusterNameTest,
			Name:                  "forth-postgresql-test",
			Params: &dbaasv1beta1.UpdatePostgresqlClusterRequest_UpdatePostgresqlClusterParams{
				Resume: true,
			},
		}
		_, err = s.UpdatePostgresqlCluster(ctx, &in)
		assert.NoError(t, err)
	})

	t.Run("BasicGetPostgresqlClusterResources", func(t *testing.T) {
		t.Parallel()
		t.Run("ProxySQL", func(t *testing.T) {
			t.Parallel()
			cs := NewPostgresqlClusterService(db, grafanaClient, componentsClient, versionService.GetVersionServiceURL())
			s := cs.(*PostgresqlClustersService)
			s.kubeStorage.clients = clients
			v := int64(1000000000)
			r := int64(2000000000)

			in := dbaasv1beta1.GetPostgresqlClusterResourcesRequest{
				Params: &dbaasv1beta1.PostgresqlClusterParams{
					ClusterSize: 1,
					Instance: &dbaasv1beta1.PostgresqlClusterParams_Instance{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
					Pgbouncer: &dbaasv1beta1.PostgresqlClusterParams_PGBouncer{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        1000,
							MemoryBytes: v,
						},
						DiskSize: v,
					},
				},
			}

			actual, err := s.GetPostgresqlClusterResources(ctx, &in)
			assert.NoError(t, err)
			assert.Equal(t, uint64(r), actual.Expected.MemoryBytes)
			assert.Equal(t, uint64(2000), actual.Expected.CpuM)
			assert.Equal(t, uint64(r), actual.Expected.DiskSize)
		})
	})
}
