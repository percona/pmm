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
	"math/rand"
	"strings"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// PSMDBClusterService implements PSMDBClusterServer methods.
type PSMDBClusterService struct {
	db                   *reform.DB
	l                    *logrus.Entry
	controllerClient     dbaasClient
	grafanaClient        grafanaClient
	versionServiceClient versionService
}

// NewPSMDBClusterService creates PSMDB Service.
func NewPSMDBClusterService(db *reform.DB, dbaasClient dbaasClient, grafanaClient grafanaClient, versionServiceClient versionService) dbaasv1beta1.PSMDBClusterServer {
	l := logrus.WithField("component", "xtradb_cluster")
	return &PSMDBClusterService{
		db:                   db,
		l:                    l,
		controllerClient:     dbaasClient,
		grafanaClient:        grafanaClient,
		versionServiceClient: versionServiceClient,
	}
}

// Enabled returns if service is enabled and can be used.
func (s *PSMDBClusterService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.DBaaS.Enabled
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

	checkResponse, err := s.controllerClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}
	operatorVersion := checkResponse.Operators.PsmdbOperatorVersion

	clusters := make([]*dbaasv1beta1.ListPSMDBClustersResponse_Cluster, len(out.Clusters))
	for i, c := range out.Clusters {
		var computeResources *dbaasv1beta1.ComputeResources
		var diskSize int64
		if c.Params.Replicaset != nil {
			diskSize = c.Params.Replicaset.DiskSize
			if c.Params.Replicaset.ComputeResources != nil {
				computeResources = &dbaasv1beta1.ComputeResources{
					CpuM:        c.Params.Replicaset.ComputeResources.CpuM,
					MemoryBytes: c.Params.Replicaset.ComputeResources.MemoryBytes,
				}
			}
		}

		cluster := dbaasv1beta1.ListPSMDBClustersResponse_Cluster{
			Name: c.Name,
			Params: &dbaasv1beta1.PSMDBClusterParams{
				ClusterSize: c.Params.ClusterSize,
				Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
					ComputeResources: computeResources,
					DiskSize:         diskSize,
				},
			},
			State: psmdbStates()[c.State],
			Operation: &dbaasv1beta1.RunningOperation{
				TotalSteps:    c.Operation.TotalSteps,
				FinishedSteps: c.Operation.FinishedSteps,
				Message:       c.Operation.Message,
			},
			Exposed: c.Exposed,
		}

		if c.Params.Image != "" {
			imageAndTag := strings.Split(c.Params.Image, ":")
			if len(imageAndTag) != 2 {
				return nil, errors.Errorf("failed to parse PSMDB version out of %q", c.Params.Image)
			}
			currentDBVersion := imageAndTag[1]

			nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, psmdbOperator, operatorVersion, currentDBVersion)
			if err != nil {
				return nil, err
			}
			cluster.AvailableImage = nextVersionImage
			cluster.InstalledImage = c.Params.Image
		}

		clusters[i] = &cluster
	}

	return &dbaasv1beta1.ListPSMDBClustersResponse{Clusters: clusters}, nil
}

// GetPSMDBClusterCredentials returns a PSMDB cluster credentials by cluster name.
func (s PSMDBClusterService) GetPSMDBClusterCredentials(ctx context.Context, req *dbaasv1beta1.GetPSMDBClusterCredentialsRequest) (*dbaasv1beta1.GetPSMDBClusterCredentialsResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := &dbaascontrollerv1beta1.GetPSMDBClusterCredentialsRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
	}

	cluster, err := s.controllerClient.GetPSMDBClusterCredentials(ctx, in)
	if err != nil {
		return nil, err
	}

	resp := dbaasv1beta1.GetPSMDBClusterCredentialsResponse{
		ConnectionCredentials: &dbaasv1beta1.GetPSMDBClusterCredentialsResponse_PSMDBCredentials{
			Username:   cluster.Credentials.Username,
			Password:   cluster.Credentials.Password,
			Host:       cluster.Credentials.Host,
			Port:       cluster.Credentials.Port,
			Replicaset: cluster.Credentials.Replicaset,
		},
	}

	return &resp, nil
}

// CreatePSMDBCluster creates PSMDB cluster with given parameters.
//nolint:dupl
func (s PSMDBClusterService) CreatePSMDBCluster(ctx context.Context, req *dbaasv1beta1.CreatePSMDBClusterRequest) (*dbaasv1beta1.CreatePSMDBClusterResponse, error) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	var pmmParams *dbaascontrollerv1beta1.PMMParams
	var apiKeyID int64
	if settings.PMMPublicAddress != "" {
		var apiKey string
		apiKeyName := fmt.Sprintf("psmdb-%s-%s-%d", req.KubernetesClusterName, req.Name, rand.Int63())
		apiKeyID, apiKey, err = s.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
		if err != nil {
			return nil, err
		}
		pmmParams = &dbaascontrollerv1beta1.PMMParams{
			PublicAddress: settings.PMMPublicAddress,
			Login:         "api_key",
			Password:      apiKey,
		}
	}

	in := dbaascontrollerv1beta1.CreatePSMDBClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
		Params: &dbaascontrollerv1beta1.PSMDBClusterParams{
			Image:       req.Params.Image,
			ClusterSize: req.Params.ClusterSize,
			Replicaset: &dbaascontrollerv1beta1.PSMDBClusterParams_ReplicaSet{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Replicaset.ComputeResources.CpuM,
					MemoryBytes: req.Params.Replicaset.ComputeResources.MemoryBytes,
				},
				DiskSize: req.Params.Replicaset.DiskSize,
			},
			VersionServiceUrl: s.versionServiceClient.GetVersionServiceURL(),
		},
		Pmm:    pmmParams,
		Expose: req.Expose,
	}

	_, err = s.controllerClient.CreatePSMDBCluster(ctx, &in)
	if err != nil {
		if apiKeyID != 0 {
			e := s.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID)
			if e != nil {
				s.l.Warnf("couldn't delete created API Key %v: %s", apiKeyID, e)
			}
		}
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
	}

	if req.Params != nil {
		if req.Params.Suspend && req.Params.Resume {
			return nil, status.Error(codes.InvalidArgument, "resume and suspend cannot be set together")
		}

		in.Params = &dbaascontrollerv1beta1.UpdatePSMDBClusterRequest_UpdatePSMDBClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Suspend:     req.Params.Suspend,
			Resume:      req.Params.Resume,
		}

		if req.Params.Replicaset != nil && req.Params.Replicaset.ComputeResources != nil {
			in.Params.Replicaset = &dbaascontrollerv1beta1.UpdatePSMDBClusterRequest_UpdatePSMDBClusterParams_ReplicaSet{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Replicaset.ComputeResources.CpuM,
					MemoryBytes: req.Params.Replicaset.ComputeResources.MemoryBytes,
				},
			}
		}
		in.Params.Image = req.Params.Image
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

	err = s.grafanaClient.DeleteAPIKeysWithPrefix(ctx, fmt.Sprintf("psmdb-%s-%s", req.KubernetesClusterName, req.Name))
	if err != nil {
		// ignore if API Key is not deleted.
		s.l.Warnf("Couldn't delete API key: %s", err)
	}

	return &dbaasv1beta1.DeletePSMDBClusterResponse{}, nil
}

// RestartPSMDBCluster restarts PSMDB cluster by given name.
func (s PSMDBClusterService) RestartPSMDBCluster(ctx context.Context, req *dbaasv1beta1.RestartPSMDBClusterRequest) (*dbaasv1beta1.RestartPSMDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.RestartPSMDBClusterRequest{
		Name: req.Name,
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
	}

	_, err = s.controllerClient.RestartPSMDBCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.RestartPSMDBClusterResponse{}, nil
}

// GetPSMDBClusterResources returns expected resources to be consumed by the cluster.
func (s PSMDBClusterService) GetPSMDBClusterResources(ctx context.Context, req *dbaasv1beta1.GetPSMDBClusterResourcesRequest) (*dbaasv1beta1.GetPSMDBClusterResourcesResponse, error) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	clusterSize := uint64(req.Params.ClusterSize)
	memory := uint64(req.Params.Replicaset.ComputeResources.MemoryBytes) * 2 * clusterSize
	cpu := uint64(req.Params.Replicaset.ComputeResources.CpuM) * 2 * clusterSize
	disk := uint64(req.Params.Replicaset.DiskSize)*3 + uint64(req.Params.Replicaset.DiskSize)*clusterSize

	if settings.PMMPublicAddress != "" {
		memory += (3 + 2*clusterSize) * 500000000
		cpu += (3 + 2*clusterSize) * 500
	}

	return &dbaasv1beta1.GetPSMDBClusterResourcesResponse{
		Expected: &dbaasv1beta1.Resources{
			CpuM:        cpu,
			MemoryBytes: memory,
			DiskSize:    disk,
		},
	}, nil
}

func psmdbStates() map[dbaascontrollerv1beta1.PSMDBClusterState]dbaasv1beta1.PSMDBClusterState {
	return map[dbaascontrollerv1beta1.PSMDBClusterState]dbaasv1beta1.PSMDBClusterState{
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_INVALID:   dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_INVALID,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_CHANGING:  dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_CHANGING,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_READY:     dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_READY,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_FAILED:    dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_FAILED,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_DELETING:  dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_DELETING,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_PAUSED:    dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_PAUSED,
		dbaascontrollerv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_UPGRADING: dbaasv1beta1.PSMDBClusterState_PSMDB_CLUSTER_STATE_UPGRADING,
	}
}
