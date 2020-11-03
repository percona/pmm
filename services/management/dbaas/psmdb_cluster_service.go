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
	"fmt"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// PSMDBClusterService implements PSMDBClusterServer methods.
type PSMDBClusterService struct {
	db               *reform.DB
	l                *logrus.Entry
	controllerClient dbaasClient
}

// NewPSMDBClusterService creates PSMDB Service.
func NewPSMDBClusterService(db *reform.DB, client dbaasClient) dbaasv1beta1.PSMDBClusterServer {
	l := logrus.WithField("component", "xtradb_cluster")
	return &PSMDBClusterService{db: db, l: l, controllerClient: client}
}

// ListPSMDBClusters returns a list of all PSMDB clusters.
func (s PSMDBClusterService) ListPSMDBClusters(ctx context.Context, req *dbaasv1beta1.ListPSMDBClustersRequest) (*dbaasv1beta1.ListPSMDBClustersResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.ListPSMDBClustersRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}

	out, err := s.controllerClient.ListPSMDBClusters(ctx, &in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Can't get list of PSMDB clusters: %s", err.Error())
	}

	clusters := make([]*dbaasv1beta1.ListPSMDBClustersResponse_Cluster, len(out.Clusters))
	for i, c := range out.Clusters {
		var computeResources *dbaasv1beta1.ComputeResources
		if c.Params.Replicaset != nil && c.Params.Replicaset.ComputeResources != nil {
			computeResources = &dbaasv1beta1.ComputeResources{
				CpuM:        c.Params.Replicaset.ComputeResources.CpuM,
				MemoryBytes: c.Params.Replicaset.ComputeResources.MemoryBytes,
			}
		}
		cluster := dbaasv1beta1.ListPSMDBClustersResponse_Cluster{
			Name: c.Name,
			Params: &dbaasv1beta1.PSMDBClusterParams{
				ClusterSize: c.Params.ClusterSize,
				Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
					ComputeResources: computeResources,
				},
			},
			State: psmdbStates()[c.State],
		}

		clusters[i] = &cluster
	}

	return &dbaasv1beta1.ListPSMDBClustersResponse{Clusters: clusters}, nil
}

// GetPSMDBCluster returns a PSMDB cluster by name.
func (s PSMDBClusterService) GetPSMDBCluster(ctx context.Context, req *dbaasv1beta1.GetPSMDBClusterRequest) (*dbaasv1beta1.GetPSMDBClusterResponse, error) {
	_, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	// TODO: implement on dbaas-controller side:
	// 1. Get psmdb host and status
	//  - Ex.: kubectl get -o=json PerconaServerMongoDB/<cluster_name>
	// 2. Get root password:
	//   - Ex.: kubectl get secret my-cluster-name-secrets -o json  | jq -r ".data.MONGODB_USER_ADMIN_PASSWORD" | base64 -d
	resp := dbaasv1beta1.GetPSMDBClusterResponse{
		ConnectionCredentials: &dbaasv1beta1.GetPSMDBClusterResponse_PSMDBCredentials{
			Username:   "userAdmin",
			Password:   "userAdmin123456",
			Host:       fmt.Sprintf("%s-rs0.default.svc.cluster.local", req.Name),
			Port:       27017,
			Replicaset: "rs0",
		},
	}

	return &resp, nil
}

// CreatePSMDBCluster creates PSMDB cluster with given parameters.
//nolint:dupl
func (s PSMDBClusterService) CreatePSMDBCluster(ctx context.Context, req *dbaasv1beta1.CreatePSMDBClusterRequest) (*dbaasv1beta1.CreatePSMDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.CreatePSMDBClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
		Params: &dbaascontrollerv1beta1.PSMDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Replicaset: &dbaascontrollerv1beta1.PSMDBClusterParams_ReplicaSet{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Replicaset.ComputeResources.CpuM,
					MemoryBytes: req.Params.Replicaset.ComputeResources.MemoryBytes,
				},
			},
		},
	}

	_, err = s.controllerClient.CreatePSMDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.CreatePSMDBClusterResponse{}, nil
}

// UpdatePSMDBCluster updates PSMDB cluster.
//nolint:dupl
func (s PSMDBClusterService) UpdatePSMDBCluster(ctx context.Context, req *dbaasv1beta1.UpdatePSMDBClusterRequest) (*dbaasv1beta1.UpdatePSMDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.UpdatePSMDBClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
		Params: &dbaascontrollerv1beta1.PSMDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Replicaset: &dbaascontrollerv1beta1.PSMDBClusterParams_ReplicaSet{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Replicaset.ComputeResources.CpuM,
					MemoryBytes: req.Params.Replicaset.ComputeResources.MemoryBytes,
				},
			},
		},
	}

	_, err = s.controllerClient.UpdatePSMDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UpdatePSMDBClusterResponse{}, nil
}

// DeletePSMDBCluster deletes PSMDB cluster by given name.
func (s PSMDBClusterService) DeletePSMDBCluster(ctx context.Context, req *dbaasv1beta1.DeletePSMDBClusterRequest) (*dbaasv1beta1.DeletePSMDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.DeletePSMDBClusterRequest{
		Name: req.Name,
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}

	_, err = s.controllerClient.DeletePSMDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.DeletePSMDBClusterResponse{}, nil
}

func psmdbStates() map[dbaascontrollerv1beta1.PSMDBClusterState]dbaasv1beta1.PSMDBClusterState {
	return map[dbaascontrollerv1beta1.PSMDBClusterState]dbaasv1beta1.PSMDBClusterState{
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_INVALID:  dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_INVALID,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_CHANGING: dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_CHANGING,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_READY:    dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_READY,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_FAILED:   dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_FAILED,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_DELETING: dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_DELETING,
	}
}
