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
	setup := func(t *testing.T) (ctx context.Context, db *reform.DB, dbaasClient *mockDbaasClient, grafanaClient *mockGrafanaClient,
		componentsService *mockComponentsService, teardown func(t *testing.T),
	) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = &mockDbaasClient{}
		grafanaClient = &mockGrafanaClient{}
		componentsService = &mockComponentsService{}

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
			require.NoError(t, sqlDB.Close())
		}

		return
	}

	ctx, db, dbaasClient, grafanaClient, componentsService, teardown := setup(t)
	defer teardown(t)

	synchronizer := new(mockDbClusterSynchronizer)
	ks := NewKubernetesServer(db, dbaasClient, grafanaClient, NewVersionServiceClient(versionServiceURL), synchronizer)
	dbaasClient.On("CheckKubernetesClusterConnection", ctx, psmdbKubeconfTest).Return(&controllerv1beta1.CheckKubernetesClusterConnectionResponse{
		Operators: &controllerv1beta1.Operators{
			PxcOperatorVersion:   onePointEight,
			PsmdbOperatorVersion: "",
		},
		Status: controllerv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK,
	}, nil)
	dbaasClient.On("InstallPSMDBOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallPSMDBOperatorResponse{}, nil)

	registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
		KubernetesClusterName: psmdbKubernetesClusterNameTest,
		KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: psmdbKubeconfTest},
	})
	require.NoError(t, err)
	assert.NotNil(t, registerKubernetesClusterResponse)
	versionService := NewVersionServiceClient(versionServiceURL)

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
		s := NewPSMDBClusterService(db, dbaasClient, grafanaClient, componentsService, versionService.GetVersionServiceURL())

		mockReq := controllerv1beta1.CreatePSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: psmdbKubeconfTest,
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
				VersionServiceUrl: versionService.GetVersionServiceURL(),
				Image:             "path",
			},
		}

		componentsService.On("GetPSMDBComponents", mock.Anything, mock.Anything).Return(mockGetPSMDBComponentsResponse, nil)
		dbaasClient.On("CreatePSMDBCluster", ctx, &mockReq).Return(&controllerv1beta1.CreatePSMDBClusterResponse{}, nil)

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
		dbaasClient.On("CreatePSMDBCluster", ctx, mock.Anything).Return(&controllerv1beta1.CreatePSMDBClusterResponse{}, nil)

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

		s := NewPSMDBClusterService(db, dbaasClient, grafanaClient, componentsService, versionService.GetVersionServiceURL())

		in := dbaasv1beta1.CreatePSMDBClusterRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
		}

		_, err := s.CreatePSMDBCluster(ctx, &in)
		assert.NoError(t, err)
	})

	//nolint:dupl
	t.Run("BasicUpdatePSMDBCluster", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		mockReq := controllerv1beta1.UpdatePSMDBClusterRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: psmdbKubeconfTest,
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
		s := NewPSMDBClusterService(db, dbaasClient, grafanaClient, componentsService, versionService.GetVersionServiceURL())

		mockReq := controllerv1beta1.GetPSMDBClusterCredentialsRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: psmdbKubeconfTest,
			},
			Name: "third-psmdb-test",
		}

		dbaasClient.On("GetPSMDBClusterCredentials", ctx, &mockReq).Return(&controllerv1beta1.GetPSMDBClusterCredentialsResponse{
			Credentials: &controllerv1beta1.PSMDBCredentials{
				Username:   "userAdmin",
				Password:   "userAdmin123",
				Host:       "hostname",
				Port:       27017,
				Replicaset: "rs0",
			},
		}, nil)

		in := dbaasv1beta1.GetPSMDBClusterCredentialsRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
			Name:                  "third-psmdb-test",
		}

		cluster, err := s.GetPSMDBClusterCredentials(ctx, &in)

		assert.NoError(t, err)
		assert.Equal(t, "hostname", cluster.ConnectionCredentials.Host)
	})

	t.Run("BasicGetPSMDBClusterCredentialsWithHost", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient, grafanaClient, componentsService, versionService.GetVersionServiceURL())
		name := "another-third-psmdb-test"

		mockReq := controllerv1beta1.GetPSMDBClusterCredentialsRequest{
			KubeAuth: &controllerv1beta1.KubeAuth{
				Kubeconfig: psmdbKubeconfTest,
			},
			Name: name,
		}

		resp := controllerv1beta1.GetPSMDBClusterCredentialsResponse{
			Credentials: &controllerv1beta1.PSMDBCredentials{
				Host: "host",
			},
		}
		dbaasClient.On("GetPSMDBClusterCredentials", ctx, &mockReq).Return(&resp, nil)

		in := dbaasv1beta1.GetPSMDBClusterCredentialsRequest{
			KubernetesClusterName: psmdbKubernetesClusterNameTest,
			Name:                  name,
		}

		cluster, err := s.GetPSMDBClusterCredentials(ctx, &in)

		assert.NoError(t, err)
		assert.Equal(t, resp.Credentials.Host, cluster.ConnectionCredentials.Host)
	})

	t.Run("BasicGetPSMDBClusterResources", func(t *testing.T) {
		s := NewPSMDBClusterService(db, dbaasClient, grafanaClient, componentsService, versionService.GetVersionServiceURL())

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
