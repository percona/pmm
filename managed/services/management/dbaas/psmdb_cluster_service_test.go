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

// Package dbaas contains all logic related to dbaas services.
package dbaas

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

const psmdbKubeconfTest = `
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
const psmdbKubernetesClusterNameTest = "test-k8s-cluster-name"

func TestPSMDBClusterService(t *testing.T) {
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
		kubeClient *mockKubernetesClient, componentsService *mockComponentsService, teardown func(t *testing.T),
	) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		// To enable queries logs, use:
		// db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		db = reform.NewDB(sqlDB, postgresql.Dialect, nil)
		dbaasClient = &mockDbaasClient{}
		grafanaClient = &mockGrafanaClient{}
		kubeClient = &mockKubernetesClient{}
		componentsService = &mockComponentsService{}

		teardown = func(t *testing.T) {
			t.Helper()
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
			require.NoError(t, sqlDB.Close())
		}

		return
	}

	ctx, db, dbaasClient, grafanaClient, kubeClient, componentsService, teardown := setup(t)
	defer teardown(t)
	versionService := NewVersionServiceClient(versionServiceURL)
	ks := NewKubernetesServer(db, dbaasClient, versionService, grafanaClient)

	grafanaClient.On("CreateAdminAPIKey", mock.Anything, mock.Anything).Return(int64(0), "", nil)
	kubeClient.On("CreatePMMSecret", mock.Anything, mock.Anything).Return(nil, nil)
	kubeClient.On("GetClusterType", ctx).Return(kubernetes.ClusterTypeGeneric, nil)
	kubeClient.On("GetDefaultStorageClassName", mock.Anything).Return("", nil)
	kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.11.0", nil)
	kubeClient.On("InstallOLMOperator", mock.Anything, mock.Anything).WaitUntil(time.After(time.Second)).Return(nil)
	kubeClient.On("InstallOperator", mock.Anything, mock.Anything).WaitUntil(time.After(time.Second)).Return(nil)

	kubeClient.On("GetServerVersion").Return(nil, nil)
	clients := map[string]kubernetesClient{
		psmdbKubernetesClusterNameTest: kubeClient,
	}
	s := ks.(*kubernetesServer)
	s.kubeStorage.clients = clients
	ks = s
	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: psmdbKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: psmdbKubeconfTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)
	versionService = NewVersionServiceClient(versionServiceURL)

	//nolint:dupl
	t.Run("BasicCreatePSMDBClusters", func(t *testing.T) {
		mockGetPSMDBComponentsResponse := &dbaasv1beta1.GetPSMDBComponentsResponse{
			Versions: []*dbaasv1beta1.OperatorVersion{
				{
					Product:  "psmdb-operator",
					Operator: "1.6.0",
					Matrix: &dbaasv1beta1.Matrix{
						Mongod: map[string]*dbaasv1beta1.Component{
							"4.2.11-12": {
								ImagePath: "percona/percona-server-mongodb:4.2.11-12",
								ImageHash: "1909cb7a6ecea9bf0535b54aa86b9ae74ba2fa303c55cf4a1a54262fb0edbd3c",
								Status:    "recommended",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
							"4.2.7-7": {
								ImagePath: "percona/percona-server-mongodb:4.2.7-7",
								ImageHash: "1d8a0859b48a3e9cadf9ad7308ec5aa4b278a64ca32ff5d887156b1b46146b13",
								Status:    "available",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
							"4.4.2-4": {
								ImagePath: "percona/percona-server-mongodb:4.4.2-4",
								ImageHash: "991d6049059e5eb1a74981290d829a5fb4ab0554993748fde1e67b2f46f26bf0",
								Status:    "recommended",
								Critical:  false,
								Default:   true,
								Disabled:  false,
							},
							"4.2.8-8": {
								ImagePath: "percona/percona-server-mongodb:4.2.8-8",
								ImageHash: "a66e889d3e986413e41083a9c887f33173da05a41c8bd107cf50eede4588a505",
								Status:    "available",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
						},
					},
				},
			},
		}
		cs := NewPSMDBClusterService(db, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		s := cs.(*PSMDBClusterService)
		s.kubeStorage.clients = clients

		componentsService.On("GetPSMDBComponents", mock.Anything, mock.Anything).Return(mockGetPSMDBComponentsResponse, nil)
		kubeClient.On("CreateDatabaseCluster", mock.Anything).Return(nil)

		in := dbaasv1beta1.CreatePSMDBClusterRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
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
				Image: "path",
			},
		}

		_, err := s.CreatePSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	// Pass the minimum parameters to use the defaults set by the fillDefaults function
	t.Run("CreatePSMDBClustersMinimumParams", func(t *testing.T) {
		psmdbComponents := &dbaasv1beta1.GetPSMDBComponentsResponse{
			Versions: []*dbaasv1beta1.OperatorVersion{
				{
					Product:  "psmdb-operator",
					Operator: "1.11.0",
					Matrix: &dbaasv1beta1.Matrix{
						Mongod: map[string]*dbaasv1beta1.Component{
							"4.2.11-12": {
								ImagePath: "percona/percona-server-mongodb:4.2.11-12",
								ImageHash: "1909cb7a6ecea9bf0535b54aa86b9ae74ba2fa303c55cf4a1a54262fb0edbd3c",
								Status:    "available",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
							"4.2.12-13": {
								ImagePath: "percona/percona-server-mongodb:4.2.12-13",
								ImageHash: "dda89e647ea5aa1266055ef465d66a139722d9e3f78a839a90a9f081b09ce26d",
								Status:    "available",
								Critical:  false,
								Default:   false,
								Disabled:  false,
							},
							"4.2.17-17": {
								ImagePath: "percona/percona-server-mongodb:4.2.17-17",
								ImageHash: "dde894b50568e088b28767ff18cfbdfe6b2496f12eddb14743d3d33c105e3f01",
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
		componentsService.On("GetPSMDBComponents", ctx, mock.Anything).Return(psmdbComponents, nil)

		cs := NewPSMDBClusterService(db, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		s := cs.(*PSMDBClusterService)
		s.kubeStorage.clients = clients
		kubeClient.On("CreateDatabaseCluster", mock.Anything).Return(nil)

		in := dbaasv1beta1.CreatePSMDBClusterRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
		}

		_, err := s.CreatePSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	//nolint:dupl
	t.Run("BasicUpdatePSMDBCluster", func(t *testing.T) {
		cs := NewPSMDBClusterService(db, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		s := cs.(*PSMDBClusterService)
		s.kubeStorage.clients = clients
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "third-psmdb-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				Database:      "psmdb",
				DatabaseImage: "percona/percona-server-mongodb:4.4.5-7",
				ClusterSize:   5,
				DBInstance: dbaasv1.DBInstanceSpec{
					CPU:      resource.MustParse("3m"),
					Memory:   resource.MustParse("256"),
					DiskSize: resource.MustParse("1073741824"),
				},
				LoadBalancer: dbaasv1.LoadBalancerSpec{
					Type: "mongos",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2m"),
							corev1.ResourceMemory: resource.MustParse("124"),
						},
					},
				},
			},
			Status: dbaasv1.DatabaseClusterStatus{
				Ready: 10,
				Size:  10,
			},
		}

		kubeClient.On("GetDatabaseCluster", ctx, "third-psmdb-test").Return(dbMock, nil)
		kubeClient.On("PatchDatabaseCluster", mock.Anything).Return(nil)

		in := dbaasv1beta1.UpdatePSMDBClusterRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
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

	t.Run("BasicGetPSMDBClusterCredentials", func(t *testing.T) {
		cs := NewPSMDBClusterService(db, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		s := cs.(*PSMDBClusterService)
		s.kubeStorage.clients = clients
		mockReq := &corev1.Secret{
			Data: map[string][]byte{
				"MONGODB_USER_ADMIN_USER":     []byte("userAdmin"),
				"MONGODB_USER_ADMIN_PASSWORD": []byte("userAdmin123"),
			},
		}
		dbMock := &dbaasv1.DatabaseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "another-third-psmdb-test",
			},
			Spec: dbaasv1.DatabaseSpec{
				SecretsName: fmt.Sprintf(psmdbSecretNameTmpl, "another-third-psmdb-test"),
			},
			Status: dbaasv1.DatabaseClusterStatus{
				Host: "hostname",
			},
		}

		kubeClient.On("GetDatabaseCluster", ctx, "another-third-psmdb-test").Return(dbMock, nil)

		kubeClient.On("GetSecret", ctx, fmt.Sprintf(psmdbSecretNameTmpl, "another-third-psmdb-test")).Return(mockReq, nil)

		in := dbaasv1beta1.GetPSMDBClusterCredentialsRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
			Name:                  "another-third-psmdb-test",
		}

		cluster, err := s.GetPSMDBClusterCredentials(ctx, &in)

		assert.NoError(t, err)
		assert.Equal(t, "hostname", cluster.ConnectionCredentials.Host)
	})

	t.Run("BasicGetPSMDBClusterResources", func(t *testing.T) {
		cs := NewPSMDBClusterService(db, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		s := cs.(*PSMDBClusterService)
		s.kubeStorage.clients = clients

		in := dbaasv1beta1.GetPSMDBClusterResourcesRequest{
			Params: &dbaasv1beta1.PSMDBClusterParams{
				ClusterSize: 4,
				Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        2000,
						MemoryBytes: 2000000000,
					},
					DiskSize: 2000000000,
				},
			},
		}

		actual, err := s.GetPSMDBClusterResources(ctx, &in)
		assert.NoError(t, err)
		assert.Equal(t, uint64(16000000000), actual.Expected.MemoryBytes)
		assert.Equal(t, uint64(16000), actual.Expected.CpuM)
		assert.Equal(t, uint64(14000000000), actual.Expected.DiskSize)
	})
}
