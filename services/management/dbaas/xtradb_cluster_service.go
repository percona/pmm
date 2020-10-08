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

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// XtraDBClusterService implements XtraDBClusterServer methods.
type XtraDBClusterService struct {
	db               *reform.DB
	l                *logrus.Entry
	controllerClient XtraDBClusterAPIConnector
}

// NewXtraDBClusterService creates XtraDB Service.
func NewXtraDBClusterService(db *reform.DB, client *Client) dbaasv1beta1.XtraDBClusterServer {
	l := logrus.WithField("component", "xtradb_cluster")
	return &XtraDBClusterService{db: db, l: l, controllerClient: client.XtraDBClusterAPIClient}
}

// ListXtraDBClusters returns a list of all XtraDB clusters.
func (s XtraDBClusterService) ListXtraDBClusters(ctx context.Context, req *dbaasv1beta1.ListXtraDBClustersRequest) (*dbaasv1beta1.ListXtraDBClustersResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.ListXtraDBClustersRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}

	out, err := s.controllerClient.ListXtraDBClusters(ctx, &in)
	if err != nil {
		return nil, err
	}

	clusters := []*dbaasv1beta1.ListXtraDBClustersResponse_Cluster{}
	for _, c := range out.Clusters {
		cluster := dbaasv1beta1.ListXtraDBClustersResponse_Cluster{
			Name: c.Name,
			Params: &dbaasv1beta1.XtraDBClusterParams{
				ClusterSize: c.Params.ClusterSize,
				Pxc: &dbaasv1beta1.XtraDBClusterParams_PXC{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        c.Params.Pxc.ComputeResources.CpuM,
						MemoryBytes: c.Params.Pxc.ComputeResources.MemoryBytes,
					},
				},
				Proxysql: &dbaasv1beta1.XtraDBClusterParams_ProxySQL{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        c.Params.Proxysql.ComputeResources.CpuM,
						MemoryBytes: c.Params.Proxysql.ComputeResources.MemoryBytes,
					},
				},
			},
		}

		clusters = append(clusters, &cluster)
	}

	return &dbaasv1beta1.ListXtraDBClustersResponse{Clusters: clusters}, nil
}

// CreateXtraDBCluster creates XtraDB cluster with given parameters.
//nolint:dupl
func (s XtraDBClusterService) CreateXtraDBCluster(ctx context.Context, req *dbaasv1beta1.CreateXtraDBClusterRequest) (*dbaasv1beta1.CreateXtraDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.CreateXtraDBClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
		Params: &dbaascontrollerv1beta1.XtraDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Pxc: &dbaascontrollerv1beta1.XtraDBClusterParams_PXC{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Pxc.ComputeResources.CpuM,
					MemoryBytes: req.Params.Pxc.ComputeResources.MemoryBytes,
				},
			},
			Proxysql: &dbaascontrollerv1beta1.XtraDBClusterParams_ProxySQL{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Proxysql.ComputeResources.CpuM,
					MemoryBytes: req.Params.Proxysql.ComputeResources.MemoryBytes,
				},
			},
		},
	}

	_, err = s.controllerClient.CreateXtraDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.CreateXtraDBClusterResponse{}, nil
}

// UpdateXtraDBCluster updates XtraDB cluster.
//nolint:dupl
func (s XtraDBClusterService) UpdateXtraDBCluster(ctx context.Context, req *dbaasv1beta1.UpdateXtraDBClusterRequest) (*dbaasv1beta1.UpdateXtraDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.UpdateXtraDBClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
		Params: &dbaascontrollerv1beta1.XtraDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Pxc: &dbaascontrollerv1beta1.XtraDBClusterParams_PXC{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Pxc.ComputeResources.CpuM,
					MemoryBytes: req.Params.Pxc.ComputeResources.MemoryBytes,
				},
			},
			Proxysql: &dbaascontrollerv1beta1.XtraDBClusterParams_ProxySQL{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Proxysql.ComputeResources.CpuM,
					MemoryBytes: req.Params.Proxysql.ComputeResources.MemoryBytes,
				},
			},
		},
	}

	_, err = s.controllerClient.UpdateXtraDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UpdateXtraDBClusterResponse{}, nil
}

// DeleteXtraDBCluster deletes XtraDB cluster by given name.
func (s XtraDBClusterService) DeleteXtraDBCluster(ctx context.Context, req *dbaasv1beta1.DeleteXtraDBClusterRequest) (*dbaasv1beta1.DeleteXtraDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.DeleteXtraDBClusterRequest{
		Name: req.Name,
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}

	_, err = s.controllerClient.DeleteXtraDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.DeleteXtraDBClusterResponse{}, nil
}
