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

// Package dbaas contains all logic related to dbaas services.
package dbaas

import (
	"context"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

type kubernetesServer struct {
	db *reform.DB
}

// NewKubernetesServer creates Kubernetes Server.
func NewKubernetesServer(db *reform.DB) dbaasv1beta1.KubernetesServer {
	return &kubernetesServer{db: db}
}

// ListKubernetesClusters returns a list of all registered Kubernetes clusters.
func (k kubernetesServer) ListKubernetesClusters(ctx context.Context, _ *dbaasv1beta1.ListKubernetesClustersRequest) (*dbaasv1beta1.ListKubernetesClustersResponse, error) {
	kubernetesClusters, err := models.FindAllKubernetesClusters(k.db.Querier)
	if err != nil {
		return nil, err
	}

	clusters := make([]*dbaasv1beta1.ListKubernetesClustersResponse_Cluster, len(kubernetesClusters))
	for i, cluster := range kubernetesClusters {
		clusters[i] = &dbaasv1beta1.ListKubernetesClustersResponse_Cluster{
			KubernetesClusterName: cluster.KubernetesClusterName,
		}
	}
	return &dbaasv1beta1.ListKubernetesClustersResponse{KubernetesClusters: clusters}, nil
}

// RegisterKubernetesCluster registers an existing Kubernetes cluster in PMM.
func (k kubernetesServer) RegisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.RegisterKubernetesClusterRequest) (*dbaasv1beta1.RegisterKubernetesClusterResponse, error) {
	err := k.db.InTransaction(func(t *reform.TX) error {
		_, err := models.CreateKubernetesCluster(k.db.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: req.KubernetesClusterName,
			KubeConfig:            req.KubeAuth.Kubeconfig,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.RegisterKubernetesClusterResponse{}, nil
}

// UnregisterKubernetesCluster removes a registered Kubernetes cluster from PMM.
func (k kubernetesServer) UnregisterKubernetesCluster(ctx context.Context, req *dbaasv1beta1.UnregisterKubernetesClusterRequest) (*dbaasv1beta1.UnregisterKubernetesClusterResponse, error) {
	err := k.db.InTransaction(func(t *reform.TX) error {
		err := models.RemoveKubernetesCluster(k.db.Querier, req.KubernetesClusterName)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UnregisterKubernetesClusterResponse{}, nil
}
