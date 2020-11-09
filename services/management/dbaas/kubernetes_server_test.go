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

package dbaas

import (
	"context"
	"testing"

	"github.com/google/uuid"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestKubernetesServer(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, ks dbaasv1beta1.KubernetesServer, dbaasClient *mockDbaasClient, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(new(tests.IDReader))

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = new(mockDbaasClient)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
		}

		ks = NewKubernetesServer(db, dbaasClient)

		return
	}
	t.Run("Basic", func(t *testing.T) {
		ctx, ks, dc, teardown := setup(t)
		defer teardown(t)
		kubeconfig := "{}"

		dc.On("CheckKubernetesClusterConnection", ctx, kubeconfig).Return(nil)
		clusters, err := ks.ListKubernetesClusters(ctx, new(dbaasv1beta1.ListKubernetesClustersRequest))
		require.NoError(t, err)
		require.Empty(t, clusters.KubernetesClusters)

		kubernetesClusterName := "test-cluster"
		registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
			KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: kubeconfig},
		})
		require.NoError(t, err)
		assert.NotNil(t, registerKubernetesClusterResponse)

		clusters, err = ks.ListKubernetesClusters(ctx, new(dbaasv1beta1.ListKubernetesClustersRequest))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(clusters.KubernetesClusters))
		expected := []*dbaasv1beta1.ListKubernetesClustersResponse_Cluster{
			{KubernetesClusterName: kubernetesClusterName},
		}
		assert.Equal(t, expected, clusters.KubernetesClusters)

		listXtraDBClustersMock := dc.On("ListXtraDBClusters", ctx, mock.Anything)
		listPSMDBClustersMock := dc.On("ListPSMDBClusters", ctx, mock.Anything)
		listXtraDBClustersMock.Return(&controllerv1beta1.ListXtraDBClustersResponse{
			Clusters: []*controllerv1beta1.ListXtraDBClustersResponse_Cluster{
				{Name: "first-xtradb-cluster"},
			},
		}, nil)
		_, err = ks.UnregisterKubernetesCluster(ctx, &dbaasv1beta1.UnregisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		tests.AssertGRPCError(t, status.Newf(codes.FailedPrecondition, "Kubernetes cluster %s has XtraDB clusters", kubernetesClusterName), err)

		listPSMDBClustersMock.Return(&controllerv1beta1.ListPSMDBClustersResponse{
			Clusters: []*controllerv1beta1.ListPSMDBClustersResponse_Cluster{
				{Name: "first-xtradb-cluster"},
			}}, nil)
		listXtraDBClustersMock.Return(&controllerv1beta1.ListXtraDBClustersResponse{}, nil)
		_, err = ks.UnregisterKubernetesCluster(ctx, &dbaasv1beta1.UnregisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		tests.AssertGRPCError(t, status.Newf(codes.FailedPrecondition, "Kubernetes cluster %s has PSMDB clusters", kubernetesClusterName), err)

		listPSMDBClustersMock.Return(&controllerv1beta1.ListPSMDBClustersResponse{}, nil)
		unregisterKubernetesClusterResponse, err := ks.UnregisterKubernetesCluster(ctx, &dbaasv1beta1.UnregisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		require.NoError(t, err)
		assert.NotNil(t, unregisterKubernetesClusterResponse)

		clusters, err = ks.ListKubernetesClusters(ctx, new(dbaasv1beta1.ListKubernetesClustersRequest))
		assert.NoError(t, err)
		assert.Empty(t, clusters.KubernetesClusters)
	})
}
