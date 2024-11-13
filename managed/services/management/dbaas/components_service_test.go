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

package dbaas

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	goversion "github.com/hashicorp/go-version"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
	pmmversion "github.com/percona/pmm/version"
)

const (
	versionServiceURL = "https://check.percona.com/versions/v1"
	twoPointEighteen  = "2.18.0"
)

func TestComponentService(t *testing.T) {
	const (
		clusterName = "pxcCluster"
		kubeConfig  = "{}"
	)

	setup := func(t *testing.T) (context.Context, dbaasv1beta1.ComponentsServer, *mockDbaasClient, *mockKubernetesClient,
		*mockKubeStorageManager,
	) {
		t.Helper()

		ctx := logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		dbaasClient := &mockDbaasClient{}

		kubernetesCluster, err := models.CreateKubernetesCluster(db.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: clusterName,
			KubeConfig:            kubeConfig,
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
			assert.NoError(t, db.Delete(kubernetesCluster))
			require.NoError(t, sqlDB.Close())
		})

		vsc := NewVersionServiceClient(versionServiceURL)
		kubeStorage := &mockKubeStorageManager{}
		kubeClient := &mockKubernetesClient{}
		kubeClient.On("GetServerVersion").Return(nil, nil)
		cs := NewComponentsService(db, dbaasClient, vsc, kubeStorage)

		return ctx, cs, dbaasClient, kubeClient, kubeStorage
	}

	t.Run("PXC", func(t *testing.T) {
		t.Run("BasicGet", func(t *testing.T) {
			ctx, cs, _, kubeClient, kubeStorageClient := setup(t)
			kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
			kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)
			kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)

			pxcComponents, err := cs.GetPXCComponents(ctx, &dbaasv1beta1.GetPXCComponentsRequest{
				KubernetesClusterName: clusterName,
			})
			require.NoError(t, err)
			require.NotNil(t, pxcComponents)

			expected := map[string]*dbaasv1beta1.Component{
				"8.0.19-10.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
				"8.0.20-11.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false},
				"8.0.20-11.2": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false},
				"8.0.21-12.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false, Default: true},
			}
			require.Equal(t, 1, len(pxcComponents.Versions))
			assert.Equal(t, expected, pxcComponents.Versions[0].Matrix.Pxc)
		})

		t.Run("Change", func(t *testing.T) {
			ctx, cs, _, kubeClient, kubeStorageClient := setup(t)
			kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
			kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)
			kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)

			resp, err := cs.ChangePXCComponents(ctx, &dbaasv1beta1.ChangePXCComponentsRequest{
				KubernetesClusterName: clusterName,
				Pxc: &dbaasv1beta1.ChangeComponent{
					DefaultVersion: "8.0.19-10.1",
					Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
						Version: "8.0.20-11.1",
						Disable: true,
					}, {
						Version: "8.0.20-11.2",
						Disable: true,
					}},
				},
				Proxysql: nil,
			})
			require.NoError(t, err)
			require.NotNil(t, resp)

			pxcComponents, err := cs.GetPXCComponents(ctx, &dbaasv1beta1.GetPXCComponentsRequest{
				KubernetesClusterName: clusterName,
			})
			require.NoError(t, err)
			require.NotNil(t, pxcComponents)

			expected := map[string]*dbaasv1beta1.Component{
				"8.0.19-10.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false, Default: true},
				"8.0.20-11.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false, Disabled: true},
				"8.0.20-11.2": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false, Disabled: true},
				"8.0.21-12.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false},
			}
			require.Equal(t, 1, len(pxcComponents.Versions))
			assert.Equal(t, expected, pxcComponents.Versions[0].Matrix.Pxc)

			t.Run("Change Again", func(t *testing.T) {
				resp, err := cs.ChangePXCComponents(ctx, &dbaasv1beta1.ChangePXCComponentsRequest{
					KubernetesClusterName: clusterName,
					Pxc: &dbaasv1beta1.ChangeComponent{
						DefaultVersion: "8.0.20-11.1",
						Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
							Version: "8.0.20-11.1",
							Enable:  true,
						}},
					},
					Proxysql: nil,
				})
				require.NoError(t, err)
				require.NotNil(t, resp)

				pxcComponents, err := cs.GetPXCComponents(ctx, &dbaasv1beta1.GetPXCComponentsRequest{
					KubernetesClusterName: clusterName,
				})

				require.NoError(t, err)
				require.NotNil(t, pxcComponents)

				expected := map[string]*dbaasv1beta1.Component{
					"8.0.19-10.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
					"8.0.20-11.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false, Default: true},
					"8.0.20-11.2": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false, Disabled: true},
					"8.0.21-12.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false},
				}
				require.Equal(t, 1, len(pxcComponents.Versions))
				assert.Equal(t, expected, pxcComponents.Versions[0].Matrix.Pxc)
			})
		})

		t.Run("Don't let disable and make default same version", func(t *testing.T) {
			ctx, cs, _, _, _ := setup(t)

			resp, err := cs.ChangePXCComponents(ctx, &dbaasv1beta1.ChangePXCComponentsRequest{
				KubernetesClusterName: clusterName,
				Pxc: &dbaasv1beta1.ChangeComponent{
					DefaultVersion: "8.0.19-10.1",
					Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
						Version: "8.0.19-10.1",
						Disable: true,
						Enable:  false,
					}},
				},
				Proxysql: nil,
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, fmt.Sprintf("default version can't be disabled, cluster: %s, component: pxc", clusterName)), err)
			require.Nil(t, resp)
		})

		t.Run("enable and disable", func(t *testing.T) {
			ctx, cs, _, _, _ := setup(t)

			resp, err := cs.ChangePXCComponents(ctx, &dbaasv1beta1.ChangePXCComponentsRequest{
				KubernetesClusterName: clusterName,
				Pxc:                   nil,
				Proxysql: &dbaasv1beta1.ChangeComponent{
					Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
						Version: "8.0.19-10.1",
						Disable: true,
						Enable:  true,
					}},
				},
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, fmt.Sprintf("enable and disable for version 8.0.19-10.1 can't be passed together, cluster: %s, component: proxySQL", clusterName)), err)
			require.Nil(t, resp)
		})
	})

	t.Run("PSMDB", func(t *testing.T) {
		t.Run("BasicGet", func(t *testing.T) {
			ctx, cs, _, kubeClient, kubeStorageClient := setup(t)
			kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
			kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)
			kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)

			psmdbComponents, err := cs.GetPSMDBComponents(ctx, &dbaasv1beta1.GetPSMDBComponentsRequest{
				KubernetesClusterName: clusterName,
			})
			require.NoError(t, err)
			require.NotNil(t, psmdbComponents)

			expected := map[string]*dbaasv1beta1.Component{
				"4.2.7-7":   {ImagePath: "percona/percona-server-mongodb:4.2.7-7", ImageHash: "1d8a0859b48a3e9cadf9ad7308ec5aa4b278a64ca32ff5d887156b1b46146b13", Status: "available", Critical: false},
				"4.2.8-8":   {ImagePath: "percona/percona-server-mongodb:4.2.8-8", ImageHash: "a66e889d3e986413e41083a9c887f33173da05a41c8bd107cf50eede4588a505", Status: "available", Critical: false},
				"4.2.11-12": {ImagePath: "percona/percona-server-mongodb:4.2.11-12", ImageHash: "1909cb7a6ecea9bf0535b54aa86b9ae74ba2fa303c55cf4a1a54262fb0edbd3c", Status: "recommended", Critical: false},
				"4.4.2-4":   {ImagePath: "percona/percona-server-mongodb:4.4.2-4", ImageHash: "991d6049059e5eb1a74981290d829a5fb4ab0554993748fde1e67b2f46f26bf0", Status: "recommended", Critical: false, Default: true},
			}
			require.Equal(t, 1, len(psmdbComponents.Versions))
			assert.Equal(t, expected, psmdbComponents.Versions[0].Matrix.Mongod)
		})

		t.Run("Change", func(t *testing.T) {
			ctx, cs, _, kubeClient, kubeStorageClient := setup(t)
			kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
			kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)
			kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)

			resp, err := cs.ChangePSMDBComponents(ctx, &dbaasv1beta1.ChangePSMDBComponentsRequest{
				KubernetesClusterName: clusterName,
				Mongod: &dbaasv1beta1.ChangeComponent{
					DefaultVersion: "4.2.8-8",
					Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
						Version: "4.2.7-7",
						Disable: true,
					}, {
						Version: "4.4.2-4",
						Disable: true,
					}},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, resp)

			psmdbComponents, err := cs.GetPSMDBComponents(ctx, &dbaasv1beta1.GetPSMDBComponentsRequest{
				KubernetesClusterName: clusterName,
			})
			require.NoError(t, err)
			require.NotNil(t, psmdbComponents)

			expected := map[string]*dbaasv1beta1.Component{
				"4.2.7-7":   {ImagePath: "percona/percona-server-mongodb:4.2.7-7", ImageHash: "1d8a0859b48a3e9cadf9ad7308ec5aa4b278a64ca32ff5d887156b1b46146b13", Status: "available", Critical: false, Disabled: true},
				"4.2.8-8":   {ImagePath: "percona/percona-server-mongodb:4.2.8-8", ImageHash: "a66e889d3e986413e41083a9c887f33173da05a41c8bd107cf50eede4588a505", Status: "available", Critical: false, Default: true},
				"4.2.11-12": {ImagePath: "percona/percona-server-mongodb:4.2.11-12", ImageHash: "1909cb7a6ecea9bf0535b54aa86b9ae74ba2fa303c55cf4a1a54262fb0edbd3c", Status: "recommended", Critical: false},
				"4.4.2-4":   {ImagePath: "percona/percona-server-mongodb:4.4.2-4", ImageHash: "991d6049059e5eb1a74981290d829a5fb4ab0554993748fde1e67b2f46f26bf0", Status: "recommended", Critical: false, Disabled: true},
			}
			require.Equal(t, 1, len(psmdbComponents.Versions))
			assert.Equal(t, expected, psmdbComponents.Versions[0].Matrix.Mongod)

			t.Run("Change Again", func(t *testing.T) {
				resp, err := cs.ChangePSMDBComponents(ctx, &dbaasv1beta1.ChangePSMDBComponentsRequest{
					KubernetesClusterName: clusterName,
					Mongod: &dbaasv1beta1.ChangeComponent{
						DefaultVersion: "4.2.11-12",
						Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
							Version: "4.4.2-4",
							Enable:  true,
						}, {
							Version: "4.2.8-8",
							Disable: true,
						}},
					},
				})
				require.NoError(t, err)
				require.NotNil(t, resp)

				psmdbComponents, err := cs.GetPSMDBComponents(ctx, &dbaasv1beta1.GetPSMDBComponentsRequest{
					KubernetesClusterName: clusterName,
				})
				require.NoError(t, err)
				require.NotNil(t, psmdbComponents)

				expected := map[string]*dbaasv1beta1.Component{
					"4.2.7-7":   {ImagePath: "percona/percona-server-mongodb:4.2.7-7", ImageHash: "1d8a0859b48a3e9cadf9ad7308ec5aa4b278a64ca32ff5d887156b1b46146b13", Status: "available", Critical: false, Disabled: true},
					"4.2.8-8":   {ImagePath: "percona/percona-server-mongodb:4.2.8-8", ImageHash: "a66e889d3e986413e41083a9c887f33173da05a41c8bd107cf50eede4588a505", Status: "available", Critical: false, Disabled: true},
					"4.2.11-12": {ImagePath: "percona/percona-server-mongodb:4.2.11-12", ImageHash: "1909cb7a6ecea9bf0535b54aa86b9ae74ba2fa303c55cf4a1a54262fb0edbd3c", Status: "recommended", Critical: false, Default: true},
					"4.4.2-4":   {ImagePath: "percona/percona-server-mongodb:4.4.2-4", ImageHash: "991d6049059e5eb1a74981290d829a5fb4ab0554993748fde1e67b2f46f26bf0", Status: "recommended", Critical: false},
				}
				require.Equal(t, 1, len(psmdbComponents.Versions))
				assert.Equal(t, expected, psmdbComponents.Versions[0].Matrix.Mongod)
			})
		})

		t.Run("Don't let disable and make default same version", func(t *testing.T) {
			ctx, cs, _, _, _ := setup(t)

			resp, err := cs.ChangePSMDBComponents(ctx, &dbaasv1beta1.ChangePSMDBComponentsRequest{
				KubernetesClusterName: clusterName,
				Mongod: &dbaasv1beta1.ChangeComponent{
					DefaultVersion: "4.2.11-12",
					Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
						Version: "4.2.11-12",
						Disable: true,
						Enable:  false,
					}},
				},
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, fmt.Sprintf("default version can't be disabled, cluster: %s, component: mongod", clusterName)), err)
			require.Nil(t, resp)
		})

		t.Run("enable and disable", func(t *testing.T) {
			ctx, cs, _, _, _ := setup(t)

			resp, err := cs.ChangePSMDBComponents(ctx, &dbaasv1beta1.ChangePSMDBComponentsRequest{
				KubernetesClusterName: clusterName,
				Mongod: &dbaasv1beta1.ChangeComponent{
					Versions: []*dbaasv1beta1.ChangeComponent_ComponentVersion{{
						Version: "4.2.11-12",
						Disable: true,
						Enable:  true,
					}},
				},
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, fmt.Sprintf("enable and disable for version 4.2.11-12 can't be passed together, cluster: %s, component: mongod", clusterName)), err)
			require.Nil(t, resp)
		})
	})
}

func TestComponentServiceMatrix(t *testing.T) {
	input := map[string]componentVersion{
		"5.7.26-31.37":   {ImagePath: "percona/percona-xtradb-cluster:5.7.26-31.37", ImageHash: "9d43d8e435e4aca5c694f726cc736667cb938158635c5f01a0e9412905f1327f", Status: "available", Critical: false},
		"5.7.27-31.39":   {ImagePath: "percona/percona-xtradb-cluster:5.7.27-31.39", ImageHash: "7d8eb4d2031c32c6e96451655f359d8e5e8e047dc95bada9a28c41c158876c26", Status: "available", Critical: false},
		"5.7.28-31.41.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.28-31.41.2", ImageHash: "fccd6525aaeedb5e436e9534e2a63aebcf743c043526dd05dba8519ebddc8b30", Status: "available", Critical: true},
		"5.7.29-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.29-31.43", ImageHash: "85fb479de073770280ae601cf3ec22dc5c8cca4c8b0dc893b09503767338e6f9", Status: "available", Critical: false},
		"5.7.30-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.30-31.43", ImageHash: "b03a060e9261b37288a2153c78f86dcfc53367c36e1bcdcae046dd2d0b0721af", Status: "available", Critical: false},
		"5.7.31-31.45":   {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45", ImageHash: "3852cef43cc0c6aa791463ba6279e59dcdac3a4fb1a5616c745c1b3c68041dc2", Status: "available", Critical: false},
		"5.7.31-31.45.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45.2", ImageHash: "0decf85c7c7afacc438f5fe355dc8320ea7ffc7018ca2cb6bda3ac0c526ae172", Status: "available", Critical: false},
		"5.7.32-31.47":   {ImagePath: "percona/percona-xtradb-cluster:5.7.32-31.47", ImageHash: "7b095019ad354c336494248d6080685022e2ed46e3b53fc103b25cd12c95952b", Status: "recommended", Critical: false},
		"8.0.19-10.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
		"8.0.20-11.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false},
		"8.0.20-11.2":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false},
		"8.0.21-12.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false},
	}

	t.Run("All", func(t *testing.T) {
		cs := &ComponentsService{}
		m := cs.matrix(input, nil, nil)

		expected := map[string]*dbaasv1beta1.Component{
			"5.7.26-31.37":   {ImagePath: "percona/percona-xtradb-cluster:5.7.26-31.37", ImageHash: "9d43d8e435e4aca5c694f726cc736667cb938158635c5f01a0e9412905f1327f", Status: "available", Critical: false},
			"5.7.27-31.39":   {ImagePath: "percona/percona-xtradb-cluster:5.7.27-31.39", ImageHash: "7d8eb4d2031c32c6e96451655f359d8e5e8e047dc95bada9a28c41c158876c26", Status: "available", Critical: false},
			"5.7.28-31.41.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.28-31.41.2", ImageHash: "fccd6525aaeedb5e436e9534e2a63aebcf743c043526dd05dba8519ebddc8b30", Status: "available", Critical: true},
			"5.7.29-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.29-31.43", ImageHash: "85fb479de073770280ae601cf3ec22dc5c8cca4c8b0dc893b09503767338e6f9", Status: "available", Critical: false},
			"5.7.30-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.30-31.43", ImageHash: "b03a060e9261b37288a2153c78f86dcfc53367c36e1bcdcae046dd2d0b0721af", Status: "available", Critical: false},
			"5.7.31-31.45":   {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45", ImageHash: "3852cef43cc0c6aa791463ba6279e59dcdac3a4fb1a5616c745c1b3c68041dc2", Status: "available", Critical: false},
			"5.7.31-31.45.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45.2", ImageHash: "0decf85c7c7afacc438f5fe355dc8320ea7ffc7018ca2cb6bda3ac0c526ae172", Status: "available", Critical: false},
			"5.7.32-31.47":   {ImagePath: "percona/percona-xtradb-cluster:5.7.32-31.47", ImageHash: "7b095019ad354c336494248d6080685022e2ed46e3b53fc103b25cd12c95952b", Status: "recommended", Critical: false},
			"8.0.19-10.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
			"8.0.20-11.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false},
			"8.0.20-11.2":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false},
			"8.0.21-12.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false, Default: true},
		}

		assert.Equal(t, expected, m)
	})

	t.Run("Disabled and Default Components", func(t *testing.T) {
		cs := &ComponentsService{}

		m := cs.matrix(input, nil, &models.Component{
			DisabledVersions: []string{"8.0.20-11.2", "8.0.20-11.1"},
			DefaultVersion:   "8.0.19-10.1",
		})

		expected := map[string]*dbaasv1beta1.Component{
			"5.7.26-31.37":   {ImagePath: "percona/percona-xtradb-cluster:5.7.26-31.37", ImageHash: "9d43d8e435e4aca5c694f726cc736667cb938158635c5f01a0e9412905f1327f", Status: "available", Critical: false},
			"5.7.27-31.39":   {ImagePath: "percona/percona-xtradb-cluster:5.7.27-31.39", ImageHash: "7d8eb4d2031c32c6e96451655f359d8e5e8e047dc95bada9a28c41c158876c26", Status: "available", Critical: false},
			"5.7.28-31.41.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.28-31.41.2", ImageHash: "fccd6525aaeedb5e436e9534e2a63aebcf743c043526dd05dba8519ebddc8b30", Status: "available", Critical: true},
			"5.7.29-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.29-31.43", ImageHash: "85fb479de073770280ae601cf3ec22dc5c8cca4c8b0dc893b09503767338e6f9", Status: "available", Critical: false},
			"5.7.30-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.30-31.43", ImageHash: "b03a060e9261b37288a2153c78f86dcfc53367c36e1bcdcae046dd2d0b0721af", Status: "available", Critical: false},
			"5.7.31-31.45":   {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45", ImageHash: "3852cef43cc0c6aa791463ba6279e59dcdac3a4fb1a5616c745c1b3c68041dc2", Status: "available", Critical: false},
			"5.7.31-31.45.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45.2", ImageHash: "0decf85c7c7afacc438f5fe355dc8320ea7ffc7018ca2cb6bda3ac0c526ae172", Status: "available", Critical: false},
			"5.7.32-31.47":   {ImagePath: "percona/percona-xtradb-cluster:5.7.32-31.47", ImageHash: "7b095019ad354c336494248d6080685022e2ed46e3b53fc103b25cd12c95952b", Status: "recommended", Critical: false},
			"8.0.19-10.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false, Default: true},
			"8.0.20-11.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false, Disabled: true},
			"8.0.20-11.2":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false, Disabled: true},
			"8.0.21-12.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false},
		}

		assert.Equal(t, expected, m)
	})

	t.Run("Skip unsupported Components", func(t *testing.T) {
		cs := &ComponentsService{}

		minimumSupportedVersion, err := goversion.NewVersion("8.0.0")
		require.NoError(t, err)
		m := cs.matrix(input, minimumSupportedVersion, &models.Component{
			DisabledVersions: []string{"8.0.21-12.1", "8.0.20-11.1"},
			DefaultVersion:   "8.0.20-11.2",
		})

		expected := map[string]*dbaasv1beta1.Component{
			"8.0.19-10.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
			"8.0.20-11.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false, Disabled: true},
			"8.0.20-11.2": {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false, Default: true},
			"8.0.21-12.1": {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false, Disabled: true},
		}

		assert.Equal(t, expected, m)
	})

	t.Run("EmptyMatrix", func(t *testing.T) {
		cs := &ComponentsService{}
		m := cs.matrix(make(map[string]componentVersion), nil, nil)
		assert.Equal(t, make(map[string]*dbaasv1beta1.Component), m)
	})
}

func TestFilteringOutOfUnsupportedVersions(t *testing.T) {
	t.Parallel()
	c := &ComponentsService{
		l:                    logrus.WithField("component", "components_service"),
		versionServiceClient: NewVersionServiceClient(versionServiceURL),
	}

	t.Run("mongod", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		params := componentsParams{
			product:        psmdbOperator,
			productVersion: onePointSix,
		}
		versions, err := c.versions(ctx, params, nil)
		require.NoError(t, err)
		parsedSupportedVersion, err := goversion.NewVersion("4.2.0")
		require.NoError(t, err)
		for _, v := range versions {
			for version := range v.Matrix.Mongod {
				parsedVersion, err := goversion.NewVersion(version)
				require.NoError(t, err)
				assert.Truef(t, parsedVersion.GreaterThanOrEqual(parsedSupportedVersion), "%s is not greater or equal to 4.2.0", version)
			}
		}
	})

	t.Run("pxc", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		params := componentsParams{
			product:        pxcOperator,
			productVersion: onePointSeven,
		}
		versions, err := c.versions(ctx, params, nil)
		require.NoError(t, err)
		parsedSupportedVersion, err := goversion.NewVersion("8.0.0")
		require.NoError(t, err)
		for _, v := range versions {
			for version := range v.Matrix.Pxc {
				parsedVersion, err := goversion.NewVersion(version)
				require.NoError(t, err)
				assert.True(t, parsedVersion.GreaterThanOrEqual(parsedSupportedVersion), "%s is not greater or equal to 8.0.0", version)
			}
		}
	})
}

const (
	onePointTen         = "1.10.0"
	onePointNine        = "1.9.0"
	onePointEight       = "1.8.0"
	onePointSeven       = "1.7.0"
	onePointSix         = "1.6.0"
	defaultPXCVersion   = "5.7.26-31.37"
	latestPXCVersion    = "8.0.0"
	defaultPSMDBVersion = "3.6.18-5.0"
	latestPSMDBVersion  = "4.5.0"
	port                = "5497"
	clusterName         = "installoperator"
)

func setup(t *testing.T, clusterName string, response *VersionServiceResponse, port string) (
	*reform.Querier, dbaasv1beta1.ComponentsServer, *mockKubernetesClient,
	*mockKubeStorageManager,
) {
	t.Helper()

	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	dbaasClient := &mockDbaasClient{}

	kubernetesCluster, err := models.CreateKubernetesCluster(db.Querier, &models.CreateKubernetesClusterParams{
		KubernetesClusterName: clusterName,
		KubeConfig:            "{}",
	})
	require.NoError(t, err)
	kubernetesCluster.Mongod = &models.Component{
		DefaultVersion: defaultPSMDBVersion,
	}
	kubernetesCluster.PXC = &models.Component{
		DefaultVersion: defaultPXCVersion,
	}
	require.NoError(t, db.Save(kubernetesCluster))
	kubeClient := &mockKubernetesClient{}

	vsc, cleanup := newFakeVersionService(response, port, pxcOperator, psmdbOperator, "pmm-server")

	t.Cleanup(func() {
		cleanup(t)
		uuid.SetRand(nil)
		dbaasClient.AssertExpectations(t)
		assert.NoError(t, db.Delete(kubernetesCluster))
		require.NoError(t, sqlDB.Close())
	})

	kubeStorage := &mockKubeStorageManager{}

	return db.Querier, NewComponentsService(db, dbaasClient, vsc, kubeStorage), kubeClient, kubeStorage
}

func TestInstallOperator(t *testing.T) {
	pmmversion.PMMVersion = "2.19.0"

	response := &VersionServiceResponse{
		Versions: []Version{
			{
				Product:        pxcOperator,
				ProductVersion: onePointSeven,
				Matrix: matrix{
					Pxc: map[string]componentVersion{
						defaultPXCVersion: {},
					},
				},
			},
			{
				Product:        pxcOperator,
				ProductVersion: onePointEight,
				Matrix: matrix{
					Pxc: map[string]componentVersion{
						latestPXCVersion: {},
						"5.8.0":          {},
					},
				},
			},
			{
				Product:        psmdbOperator,
				ProductVersion: onePointSeven,
				Matrix: matrix{
					Mongod: map[string]componentVersion{
						defaultPSMDBVersion: {},
					},
				},
			},
			{
				Product:        psmdbOperator,
				ProductVersion: onePointEight,
				Matrix: matrix{
					Mongod: map[string]componentVersion{
						latestPSMDBVersion: {},
						"3.7.0":            {},
					},
				},
			},
			{
				Product:        "pmm-server",
				ProductVersion: "2.19.0",
				Matrix: matrix{
					PXCOperator: map[string]componentVersion{
						onePointEight: {},
					},
					PSMDBOperator: map[string]componentVersion{
						onePointEight: {},
					},
				},
			},
		},
	}

	t.Run("Defaults not supported", func(t *testing.T) {
		_, c, kubeClient, kubeStorageClient := setup(t, clusterName, response, "5497")
		kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)
		kubeClient.On("SetKubeConfig", mock.Anything).Return(nil)
		kubeClient.On("InstallOLMOperator", mock.Anything, mock.Anything).Return(nil)
		kubeClient.On("InstallOperator", mock.Anything, mock.Anything).Return(nil)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
		defer cancel()
		resp, err := c.InstallOperator(ctx, &dbaasv1beta1.InstallOperatorRequest{
			KubernetesClusterName: clusterName,
			OperatorType:          pxcOperator,
			Version:               onePointEight,
		})
		require.Error(t, err)
		assert.Nil(t, resp)

		resp, err = c.InstallOperator(ctx, &dbaasv1beta1.InstallOperatorRequest{
			KubernetesClusterName: clusterName,
			OperatorType:          psmdbOperator,
			Version:               onePointEight,
		})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Defaults supported", func(t *testing.T) {
		db, c, kubeClient, kubeStorageClient := setup(t, clusterName, response, "5497")
		kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)
		kubeClient.On("SetKubeConfig", mock.Anything).Return(nil)
		kubeClient.On("InstallOperator", mock.Anything, mock.Anything).Return(nil)
		kubeClient.On("UpgradeOperator", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
		defer cancel()
		response.Versions[1].Matrix.Pxc[defaultPXCVersion] = componentVersion{}
		response.Versions[3].Matrix.Mongod[defaultPSMDBVersion] = componentVersion{}

		kubernetesCluster, err := models.FindKubernetesClusterByName(db, clusterName)
		require.NoError(t, err)
		kubernetesCluster.Mongod.DefaultVersion = defaultPSMDBVersion
		kubernetesCluster.PXC.DefaultVersion = defaultPXCVersion
		require.NoError(t, db.Save(kubernetesCluster))

		resp, err := c.InstallOperator(ctx, &dbaasv1beta1.InstallOperatorRequest{
			KubernetesClusterName: clusterName,
			OperatorType:          pxcOperator,
			Version:               onePointEight,
		})
		require.NoError(t, err)
		assert.Equal(t, dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK, resp.Status)

		resp, err = c.InstallOperator(ctx, &dbaasv1beta1.InstallOperatorRequest{
			KubernetesClusterName: clusterName,
			OperatorType:          psmdbOperator,
			Version:               onePointEight,
		})
		require.NoError(t, err)
		assert.Equal(t, dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_OK, resp.Status)
	})
}

func TestCheckForOperatorUpdate(t *testing.T) {
	response := &VersionServiceResponse{
		Versions: []Version{
			{
				ProductVersion: onePointSix,
				Product:        pxcOperator,
			},
			{
				ProductVersion: onePointSeven,
				Product:        pxcOperator,
			},
			{
				ProductVersion: onePointEight,
				Product:        pxcOperator,
			},

			{
				ProductVersion: onePointSix,
				Product:        psmdbOperator,
			},
			{
				ProductVersion: onePointSeven,
				Product:        psmdbOperator,
			},
			{
				ProductVersion: onePointEight,
				Product:        psmdbOperator,
			},

			{
				ProductVersion: twoPointEighteen,
				Product:        "pmm-server",
				Matrix: matrix{
					PSMDBOperator: map[string]componentVersion{
						onePointEight: {Status: "recommended"},
						onePointSeven: {},
					},
					PXCOperator: map[string]componentVersion{
						onePointEight: {Status: "recommended"},
						onePointSeven: {},
					},
				},
			},
		},
	}

	pmmversion.PMMVersion = twoPointEighteen
	ctx := context.Background()
	t.Run("Update available", func(t *testing.T) {
		clusterName := "update-available"
		_, cs, kubeClient, kubeStorageClient := setup(t, clusterName, response, "9873")
		kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)
		kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
		kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)

		mockSubscriptions := &v1alpha1.SubscriptionList{
			Items: []v1alpha1.Subscription{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "space-x",
						Name:      "psmdb-operator",
					},
					Spec: &v1alpha1.SubscriptionSpec{
						Package:       "percona-server-mongodb-operator",
						CatalogSource: "src",
						Channel:       "nat-geo",
					},
					Status: v1alpha1.SubscriptionStatus{
						CurrentCSV:   "percona-server-mongodb-operator-v1.8.0",
						InstalledCSV: "percona-server-mongodb-operator-v1.2.2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "space-x",
						Name:      "pxc-operator",
					},
					Spec: &v1alpha1.SubscriptionSpec{
						Package:       "percona-xtradb-cluster-operator",
						CatalogSource: "src",
						Channel:       "nat-geo",
					},
					Status: v1alpha1.SubscriptionStatus{
						CurrentCSV:   "percona-xtradb-cluster-operator-v1.8.0",
						InstalledCSV: "percona-xtradb-cluster-operator-v1.2.2",
					},
				},
			},
		}
		kubeClient.On("ListSubscriptions", mock.Anything, mock.Anything).WaitUntil(time.After(time.Second)).Return(mockSubscriptions, nil)
		kubeClient.On("SetKubeConfig", mock.Anything).Return(nil)
		resp, err := cs.CheckForOperatorUpdate(ctx, &dbaasv1beta1.CheckForOperatorUpdateRequest{})
		require.NoError(t, err)
		cluster := resp.ClusterToComponents[clusterName]
		require.NotNil(t, cluster)
		require.NotNil(t, cluster.ComponentToUpdateInformation)
		require.NotNil(t, cluster.ComponentToUpdateInformation[psmdbOperator])
		require.NotNil(t, cluster.ComponentToUpdateInformation[pxcOperator])
		assert.Equal(t, onePointEight, cluster.ComponentToUpdateInformation[psmdbOperator].AvailableVersion)
		assert.Equal(t, onePointEight, cluster.ComponentToUpdateInformation[pxcOperator].AvailableVersion)
	})
	t.Run("Update NOT available", func(t *testing.T) {
		clusterName := "update-not-available"
		_, cs, kubeClient, kubeStorageClient := setup(t, clusterName, response, "7895")
		kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)
		kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
		kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)

		mockSubscriptions := &v1alpha1.SubscriptionList{
			Items: []v1alpha1.Subscription{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "space-x",
						Name:      "psmdb-operator",
					},
					Spec: &v1alpha1.SubscriptionSpec{
						Package:       "percona-server-mongodb-operator",
						CatalogSource: "src",
						Channel:       "nat-geo",
					},
					Status: v1alpha1.SubscriptionStatus{
						CurrentCSV:   "percona-server-mongodb-operator-v1.8.0",
						InstalledCSV: "percona-server-mongodb-operator-v1.8.0",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "space-x",
						Name:      "pxc-operator",
					},
					Spec: &v1alpha1.SubscriptionSpec{
						Package:       "percona-xtradb-cluster-operator",
						CatalogSource: "src",
						Channel:       "nat-geo",
					},
					Status: v1alpha1.SubscriptionStatus{
						CurrentCSV:   "percona-xtradb-cluster-operator-v1.8.0",
						InstalledCSV: "percona-xtradb-cluster-operator-v1.8.0",
					},
				},
			},
		}
		kubeClient.On("ListSubscriptions", mock.Anything, mock.Anything).WaitUntil(time.After(time.Second)).Return(mockSubscriptions, nil)
		kubeClient.On("SetKubeConfig", mock.Anything).Return(nil)

		resp, err := cs.CheckForOperatorUpdate(ctx, &dbaasv1beta1.CheckForOperatorUpdateRequest{})
		require.NoError(t, err)
		cluster := resp.ClusterToComponents[clusterName]
		require.NotNil(t, cluster)
		require.NotNil(t, cluster.ComponentToUpdateInformation)
		require.NotNil(t, cluster.ComponentToUpdateInformation[psmdbOperator])
		require.NotNil(t, cluster.ComponentToUpdateInformation[pxcOperator])
		assert.Equal(t, "", cluster.ComponentToUpdateInformation[psmdbOperator].AvailableVersion)
		assert.Equal(t, "", cluster.ComponentToUpdateInformation[pxcOperator].AvailableVersion)
	})
	t.Run("User's operators version is ahead of version service", func(t *testing.T) {
		clusterName := "update-available-pmm-update"
		_, cs, kubeClient, kubeStorageClient := setup(t, clusterName, response, "5863")
		kubeStorageClient.On("GetOrSetClient", mock.Anything).Return(kubeClient, nil)
		kubeClient.On("GetPXCOperatorVersion", mock.Anything, mock.Anything).Return("1.7.0", nil)
		kubeClient.On("GetPSMDBOperatorVersion", mock.Anything, mock.Anything).Return("1.6.0", nil)
		mockSubscriptions := &v1alpha1.SubscriptionList{
			Items: []v1alpha1.Subscription{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "space-x",
						Name:      "psmdb-operator",
					},
					Spec: &v1alpha1.SubscriptionSpec{
						Package:       "percona-server-mongodb-operator",
						CatalogSource: "src",
						Channel:       "nat-geo",
					},
					Status: v1alpha1.SubscriptionStatus{
						CurrentCSV:   "percona-server-mongodb-operator-v1.8.0",
						InstalledCSV: "percona-server-mongodb-operator-v1.8.0",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "space-x",
						Name:      "pxc-operator",
					},
					Spec: &v1alpha1.SubscriptionSpec{
						Package:       "percona-xtradb-cluster-operator",
						CatalogSource: "src",
						Channel:       "nat-geo",
					},
					Status: v1alpha1.SubscriptionStatus{
						CurrentCSV:   "percona-xtradb-cluster-operator-v1.8.0",
						InstalledCSV: "percona-xtradb-cluster-operator-v1.8.0",
					},
				},
			},
		}
		kubeClient.On("ListSubscriptions", mock.Anything, mock.Anything).WaitUntil(time.After(time.Second)).Return(mockSubscriptions, nil)
		kubeClient.On("SetKubeConfig", mock.Anything).Return(nil)
		resp, err := cs.CheckForOperatorUpdate(ctx, &dbaasv1beta1.CheckForOperatorUpdateRequest{})
		require.NoError(t, err)
		cluster := resp.ClusterToComponents[clusterName]
		require.NotNil(t, cluster)
		require.NotNil(t, cluster.ComponentToUpdateInformation)
		require.NotNil(t, cluster.ComponentToUpdateInformation[psmdbOperator])
		require.NotNil(t, cluster.ComponentToUpdateInformation[pxcOperator])
		assert.Equal(t, "", cluster.ComponentToUpdateInformation[psmdbOperator].AvailableVersion)
		assert.Equal(t, "", cluster.ComponentToUpdateInformation[pxcOperator].AvailableVersion)
	})
}
