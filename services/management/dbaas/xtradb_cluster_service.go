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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// XtraDBClusterService implements XtraDBClusterServer methods.
type XtraDBClusterService struct {
	db               *reform.DB
	l                *logrus.Entry
	controllerClient dbaasClient
}

// NewXtraDBClusterService creates XtraDB Service.
func NewXtraDBClusterService(db *reform.DB, client dbaasClient) dbaasv1beta1.XtraDBClusterServer {
	l := logrus.WithField("component", "xtradb_cluster")
	return &XtraDBClusterService{db: db, l: l, controllerClient: client}
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

	clusters := make([]*dbaasv1beta1.ListXtraDBClustersResponse_Cluster, len(out.Clusters))
	for i, c := range out.Clusters {
		cluster := dbaasv1beta1.ListXtraDBClustersResponse_Cluster{
			Name: c.Name,
			Params: &dbaasv1beta1.XtraDBClusterParams{
				ClusterSize: c.Params.ClusterSize,
			},
			State: pxcStates()[c.State],
			Operation: &dbaasv1beta1.RunningOperation{
				TotalSteps:    c.Operation.TotalSteps,
				FinishedSteps: c.Operation.FinishedSteps,
				Message:       c.Operation.Message,
			},
		}

		if c.Params.Pxc != nil {
			cluster.Params.Pxc = &dbaasv1beta1.XtraDBClusterParams_PXC{
				DiskSize: c.Params.Pxc.DiskSize,
			}
			if c.Params.Pxc.ComputeResources != nil {
				cluster.Params.Pxc.ComputeResources = &dbaasv1beta1.ComputeResources{
					CpuM:        c.Params.Pxc.ComputeResources.CpuM,
					MemoryBytes: c.Params.Pxc.ComputeResources.MemoryBytes,
				}
			}
		}

		if c.Params.Proxysql.ComputeResources != nil {
			cluster.Params.Proxysql = &dbaasv1beta1.XtraDBClusterParams_ProxySQL{
				DiskSize: c.Params.Proxysql.DiskSize,
			}
			cluster.Params.Proxysql.ComputeResources = &dbaasv1beta1.ComputeResources{
				CpuM:        c.Params.Proxysql.ComputeResources.CpuM,
				MemoryBytes: c.Params.Proxysql.ComputeResources.MemoryBytes,
			}
		}

		clusters[i] = &cluster
	}

	return &dbaasv1beta1.ListXtraDBClustersResponse{Clusters: clusters}, nil
}

// GetXtraDBClusterCredentials returns a XtraDB cluster credentials.
func (s XtraDBClusterService) GetXtraDBClusterCredentials(ctx context.Context, req *dbaasv1beta1.GetXtraDBClusterCredentialsRequest) (*dbaasv1beta1.GetXtraDBClusterCredentialsResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := &dbaascontrollerv1beta1.GetXtraDBClusterCredentialsRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
	}

	cluster, err := s.controllerClient.GetXtraDBClusterCredentials(ctx, in)
	if err != nil {
		return nil, err
	}

	_ = kubernetesCluster
	resp := dbaasv1beta1.GetXtraDBClusterCredentialsResponse{
		ConnectionCredentials: &dbaasv1beta1.XtraDBClusterConnectionCredentials{
			Username: cluster.Credentials.Username,
			Password: cluster.Credentials.Password,
			Host:     cluster.Credentials.Host,
			Port:     cluster.Credentials.Port,
		},
	}

	return &resp, nil
}

// CreateXtraDBCluster creates XtraDB cluster with given parameters.
//nolint:dupl
func (s XtraDBClusterService) CreateXtraDBCluster(ctx context.Context, req *dbaasv1beta1.CreateXtraDBClusterRequest) (*dbaasv1beta1.CreateXtraDBClusterResponse, error) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.CreateXtraDBClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name:             req.Name,
		PmmPublicAddress: settings.PMMPublicAddress,
		Params: &dbaascontrollerv1beta1.XtraDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Pxc: &dbaascontrollerv1beta1.XtraDBClusterParams_PXC{
				ComputeResources: new(dbaascontrollerv1beta1.ComputeResources),
				DiskSize:         req.Params.Pxc.DiskSize,
			},
			Proxysql: &dbaascontrollerv1beta1.XtraDBClusterParams_ProxySQL{
				ComputeResources: new(dbaascontrollerv1beta1.ComputeResources),
				DiskSize:         req.Params.Proxysql.DiskSize,
			},
		},
	}

	if req.Params.Pxc.ComputeResources != nil {
		in.Params.Pxc.ComputeResources = &dbaascontrollerv1beta1.ComputeResources{
			CpuM:        req.Params.Pxc.ComputeResources.CpuM,
			MemoryBytes: req.Params.Pxc.ComputeResources.MemoryBytes,
		}
	}

	if req.Params.Proxysql.ComputeResources != nil {
		in.Params.Proxysql.ComputeResources = &dbaascontrollerv1beta1.ComputeResources{
			CpuM:        req.Params.Proxysql.ComputeResources.CpuM,
			MemoryBytes: req.Params.Proxysql.ComputeResources.MemoryBytes,
		}
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
	}

	if req.Params != nil {
		if req.Params.Suspend && req.Params.Resume {
			return nil, status.Error(codes.InvalidArgument, "resume and suspend cannot be set together")
		}

		in.Params = &dbaascontrollerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Suspend:     req.Params.Suspend,
			Resume:      req.Params.Resume,
		}

		if req.Params.Pxc != nil && req.Params.Pxc.ComputeResources != nil {
			in.Params.Pxc = &dbaascontrollerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams_PXC{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Pxc.ComputeResources.CpuM,
					MemoryBytes: req.Params.Pxc.ComputeResources.MemoryBytes,
				},
			}
		}

		if req.Params.Proxysql != nil && req.Params.Proxysql.ComputeResources != nil {
			in.Params.Proxysql = &dbaascontrollerv1beta1.UpdateXtraDBClusterRequest_UpdateXtraDBClusterParams_ProxySQL{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Proxysql.ComputeResources.CpuM,
					MemoryBytes: req.Params.Proxysql.ComputeResources.MemoryBytes,
				},
			}
		}
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

// RestartXtraDBCluster restarts XtraDB cluster by given name.
func (s XtraDBClusterService) RestartXtraDBCluster(ctx context.Context, req *dbaasv1beta1.RestartXtraDBClusterRequest) (*dbaasv1beta1.RestartXtraDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.RestartXtraDBClusterRequest{
		Name: req.Name,
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}

	_, err = s.controllerClient.RestartXtraDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.RestartXtraDBClusterResponse{}, nil
}

func pxcStates() map[dbaascontrollerv1beta1.XtraDBClusterState]dbaasv1beta1.XtraDBClusterState {
	return map[dbaascontrollerv1beta1.XtraDBClusterState]dbaasv1beta1.XtraDBClusterState{
		dbaascontrollerv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_INVALID:  dbaasv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_INVALID,
		dbaascontrollerv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_CHANGING: dbaasv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_CHANGING,
		dbaascontrollerv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_READY:    dbaasv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_READY,
		dbaascontrollerv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_FAILED:   dbaasv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_FAILED,
		dbaascontrollerv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_DELETING: dbaasv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_DELETING,
		dbaascontrollerv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_PAUSED:   dbaasv1beta1.XtraDBClusterState_XTRA_DB_CLUSTER_STATE_PAUSED,
	}
}
