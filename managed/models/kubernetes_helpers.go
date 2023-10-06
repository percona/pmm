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

package models

import (
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
	err := q.Reload(cluster)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Kubernetes Cluster with ID %q already exists.", id)
}

func checkUniqueKubernetesClusterName(q *reform.Querier, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "empty Kubernetes Cluster Name.")
	}

	_, err := q.FindOneFrom(KubernetesClusterTable, "kubernetes_cluster_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Kubernetes Cluster with Name %q already exists.", name)
}

// FindAllKubernetesClusters returns all Kubernetes clusters.
func FindAllKubernetesClusters(q *reform.Querier) ([]*KubernetesCluster, error) {
	structs, err := q.SelectAllFrom(KubernetesClusterTable, "ORDER BY id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	clusters := make([]*KubernetesCluster, len(structs))
	for i, s := range structs {
		clusters[i] = s.(*KubernetesCluster) //nolint:forcetypeassert
	}

	return clusters, nil
}

// FindKubernetesClusterByName finds a Kubernetes cluster with provided name.
func FindKubernetesClusterByName(q *reform.Querier, name string) (*KubernetesCluster, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Kubernetes Cluster Name.")
	}

	cluster, err := q.FindOneFrom(KubernetesClusterTable, "kubernetes_cluster_name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Kubernetes Cluster with name %q not found.", name)
		}
		return nil, errors.WithStack(err)
	}

	return cluster.(*KubernetesCluster), nil //nolint:forcetypeassert
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
		IsReady:               false,
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// ChangeKubernetesClusterToReady changes k8s cluster to ready state once provisioning is finished.
func ChangeKubernetesClusterToReady(q *reform.Querier, name string) error {
	c, err := FindKubernetesClusterByName(q, name)
	if err != nil {
		return err
	}
	c.IsReady = true
	if err = q.Update(c); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// RemoveKubernetesCluster removes Kubernetes cluster with provided name.
func RemoveKubernetesCluster(q *reform.Querier, name string) error {
	c, err := FindKubernetesClusterByName(q, name)
	if err != nil {
		return err
	}

	return errors.Wrap(q.Delete(c), "failed to delete Kubernetes Cluster")
}
