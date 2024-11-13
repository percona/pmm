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

package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestKubernetesHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	defer func() {
		models.Now = origNowF
	}()
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.KubernetesCluster{
				ID:                    "KC1",
				KubernetesClusterName: "Kubernetes Cluster 1",
				KubeConfig:            `{"kind": "Config", "apiVersion": "v1"}`,
				PXC: &models.Component{
					DisabledVersions: []string{"8.0.0"},
					DefaultVersion:   "8.0.1-20",
				},
				ProxySQL: &models.Component{
					DisabledVersions: []string{"8.0.0"},
					DefaultVersion:   "8.0.1-19",
				},
				HAProxy: &models.Component{
					DisabledVersions: []string{"2.0.0"},
					DefaultVersion:   "2.1.7",
				},
				Mongod: &models.Component{
					DisabledVersions: []string{"3.4.0", "3.6.0"},
					DefaultVersion:   "4.4.3-8",
				},
			},
			&models.KubernetesCluster{
				ID:                    "KC2",
				KubernetesClusterName: "Kubernetes Cluster 2",
				KubeConfig:            `{}`,
			},
		} {
			require.NoError(t, q.Insert(str), "failed to INSERT %+v", str)
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return q, teardown
	}

	t.Run("FindAllKubernetesClusters", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		expected := []*models.KubernetesCluster{
			{
				ID:                    "KC1",
				KubernetesClusterName: "Kubernetes Cluster 1",
				KubeConfig:            `{"kind": "Config", "apiVersion": "v1"}`,
				PXC: &models.Component{
					DisabledVersions: []string{"8.0.0"},
					DefaultVersion:   "8.0.1-20",
				},
				ProxySQL: &models.Component{
					DisabledVersions: []string{"8.0.0"},
					DefaultVersion:   "8.0.1-19",
				},
				HAProxy: &models.Component{
					DisabledVersions: []string{"2.0.0"},
					DefaultVersion:   "2.1.7",
				},
				Mongod: &models.Component{
					DisabledVersions: []string{"3.4.0", "3.6.0"},
					DefaultVersion:   "4.4.3-8",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:                    "KC2",
				KubernetesClusterName: "Kubernetes Cluster 2",
				KubeConfig:            `{}`,
				CreatedAt:             now,
				UpdatedAt:             now,
			},
		}

		clusters, err := models.FindAllKubernetesClusters(q)
		require.NoError(t, err)
		require.Equal(t, expected, clusters)
	})

	t.Run("CreateKubernetesCluster", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			cluster, err := models.CreateKubernetesCluster(q, &models.CreateKubernetesClusterParams{
				KubernetesClusterName: "Kubernetes Cluster 3",
				KubeConfig:            "{}",
			})
			require.NoError(t, err)
			expected := &models.KubernetesCluster{
				ID:                    cluster.ID,
				KubernetesClusterName: "Kubernetes Cluster 3",
				KubeConfig:            "{}",
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			require.Equal(t, expected, cluster)
		})

		t.Run("EmptyKubernetesClusterName", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			cluster, err := models.CreateKubernetesCluster(q, &models.CreateKubernetesClusterParams{
				KubernetesClusterName: "",
				KubeConfig:            "{}",
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "empty Kubernetes Cluster Name."), err)
			require.Nil(t, cluster)
		})

		t.Run("EmptyKubeConfig", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			cluster, err := models.CreateKubernetesCluster(q, &models.CreateKubernetesClusterParams{
				KubernetesClusterName: "Kubernetes Cluster without config",
				KubeConfig:            "",
			})
			require.EqualError(t, err, `pq: new row for relation "kubernetes_clusters" violates check constraint "kubernetes_clusters_kube_config_check"`)
			require.Nil(t, cluster)
		})

		t.Run("AlreadyExist", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			cluster, err := models.CreateKubernetesCluster(q, &models.CreateKubernetesClusterParams{
				KubernetesClusterName: "Kubernetes Cluster 1",
				KubeConfig:            `{}`,
			})

			tests.AssertGRPCError(t, status.New(codes.AlreadyExists, "Kubernetes Cluster with Name \"Kubernetes Cluster 1\" already exists."), err)
			require.Nil(t, cluster)
		})
	})

	t.Run("RemoveKubernetesCluster", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			err := models.RemoveKubernetesCluster(q, "Kubernetes Cluster 1")
			assert.NoError(t, err)
		})
		t.Run("NonExistCluster", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)
			err := models.RemoveKubernetesCluster(q, "test-cluster")
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Kubernetes Cluster with name "test-cluster" not found.`), err)
		})
	})
}
