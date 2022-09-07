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

package models

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkUniqueKubernetesClusterID(q *reform.Querier, id string) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "empty Kubernetes Cluster ID")
	}

	cluster := &KubernetesCluster{ID: id}
	switch err := q.Reload(cluster); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Kubernetes Cluster with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func checkUniqueKubernetesClusterName(q *reform.Querier, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "empty Kubernetes Cluster Name.")
	}

	switch _, err := q.FindOneFrom(KubernetesClusterTable, "kubernetes_cluster_name", name); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Kubernetes Cluster with Name %q already exists.", name)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

// FindAllKubernetesClusters returns all Kubernetes clusters.
func FindAllKubernetesClusters(q *reform.Querier) ([]*KubernetesCluster, error) {
	structs, err := q.SelectAllFrom(KubernetesClusterTable, "ORDER BY id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	clusters := make([]*KubernetesCluster, len(structs))
	for i, s := range structs {
		clusters[i] = s.(*KubernetesCluster)
	}

	return clusters, nil
}

// FindKubernetesClusterByName finds a Kubernetes cluster with provided name.
func FindKubernetesClusterByName(q *reform.Querier, name string) (*KubernetesCluster, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Kubernetes Cluster Name.")
	}

	switch cluster, err := q.FindOneFrom(KubernetesClusterTable, "kubernetes_cluster_name", name); err {
	case nil:
		return cluster.(*KubernetesCluster), nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Kubernetes Cluster with name %q not found.", name)
	default:
		return nil, errors.WithStack(err)
	}
}

// FindKubernetesClusterByID finds a Kubernetes cluster with provided ID.
func FindKubernetesClusterByID(q *reform.Querier, id string) (*KubernetesCluster, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Kubernetes Cluster ID.")
	}

	switch cluster, err := q.FindByPrimaryKeyFrom(KubernetesClusterTable, id); err {
	case nil:
		return cluster.(*KubernetesCluster), nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Kubernetes Cluster with id %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// CreateKubernetesClusterParams contains all params required to create Kubernetes cluster.
type CreateKubernetesClusterParams struct {
	KubernetesClusterName string
	KubeConfig            string
}

// CreateKubernetesCluster creates Kubernetes cluster with provided params.
func CreateKubernetesCluster(q *reform.Querier, params *CreateKubernetesClusterParams) (*KubernetesCluster, error) {
	id := "/kubernetes_cluster_id/" + uuid.New().String()
	if err := checkUniqueKubernetesClusterID(q, id); err != nil {
		return nil, err
	}
	if err := checkUniqueKubernetesClusterName(q, params.KubernetesClusterName); err != nil {
		return nil, err
	}

	row := &KubernetesCluster{
		ID:                    id,
		KubernetesClusterName: params.KubernetesClusterName,
		KubeConfig:            params.KubeConfig,
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// RemoveKubernetesCluster removes Kubernetes cluster with provided name.
func RemoveKubernetesCluster(q *reform.Querier, name string, mode RemoveMode) error {
	c, err := FindKubernetesClusterByName(q, name)
	if err != nil {
		return err
	}

	dbClusters, err := FindDBClustersForKubernetesCluster(q, c.ID)
	if err != nil {
		return err
	}
	if len(dbClusters) != 0 {
		switch mode {
		case RemoveRestrict:
			return status.Errorf(codes.FailedPrecondition, "Kubernetes cluster with ID %q has DB clusters.", c.ID)
		case RemoveCascade:
			for _, str := range dbClusters {
				if _, err = RemoveDBCluster(q, str.ID); err != nil {
					return err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode)) //nolint:goerr113
		}
	}

	if err = q.Delete(c); err != nil {
		return errors.Wrap(err, "failed to delete Kubernetes Cluster")
	}

	return nil
}
