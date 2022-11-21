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
	"strings"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

type DBClusterService struct {
	db                   *reform.DB
	l                    *logrus.Entry
	controllerClient     dbaasClient
	grafanaClient        grafanaClient
	versionServiceClient *VersionServiceClient

	dbaasv1beta1.UnimplementedDBClustersServer
}

// NewDBClusterService creates DB Clusters Service.
func NewDBClusterService(db *reform.DB, controllerClient dbaasClient, grafanaClient grafanaClient, versionServiceClient *VersionServiceClient) dbaasv1beta1.DBClustersServer {
	l := logrus.WithField("component", "dbaas_db_cluster")
	return &DBClusterService{
		db:                   db,
		l:                    l,
		controllerClient:     controllerClient,
		grafanaClient:        grafanaClient,
		versionServiceClient: versionServiceClient,
	}
}

// ListDBClusters returns a list of all DB clusters.
func (s DBClusterService) ListDBClusters(ctx context.Context, req *dbaasv1beta1.ListDBClustersRequest) (*dbaasv1beta1.ListDBClustersResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	checkResponse, err := s.controllerClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}

	pxcClusters, err := s.listPXCClusters(ctx, kubernetesCluster.KubeConfig, checkResponse.Operators.PxcOperatorVersion)
	if err != nil {
		return nil, err
	}

	psmdbClusters, err := s.listPSMDBClusters(ctx, kubernetesCluster.KubeConfig, checkResponse.Operators.PsmdbOperatorVersion)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.ListDBClustersResponse{
		PxcClusters:   pxcClusters,
		PsmdbClusters: psmdbClusters,
	}, nil
}

func (s DBClusterService) listPSMDBClusters(ctx context.Context, kubeConfig string, operatorVersion string) ([]*dbaasv1beta1.PSMDBCluster, error) {
	in := dbaascontrollerv1beta1.ListPSMDBClustersRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubeConfig,
		},
	}

	out, err := s.controllerClient.ListPSMDBClusters(ctx, &in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Can't get list of PSMDB clusters: %s", err.Error())
	}

	clusters := make([]*dbaasv1beta1.PSMDBCluster, len(out.Clusters))
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

		cluster := dbaasv1beta1.PSMDBCluster{
			Name: c.Name,
			Params: &dbaasv1beta1.PSMDBClusterParams{
				ClusterSize: c.Params.ClusterSize,
				Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
					ComputeResources: computeResources,
					DiskSize:         diskSize,
				},
			},
			State: dbClusterStates()[c.State],
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

	return clusters, nil
}

func (s DBClusterService) listPXCClusters(ctx context.Context, kubeConfig string, operatorVersion string) ([]*dbaasv1beta1.PXCCluster, error) {
	in := dbaascontrollerv1beta1.ListPXCClustersRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubeConfig,
		},
	}

	out, err := s.controllerClient.ListPXCClusters(ctx, &in)
	if err != nil {
		return nil, err
	}

	pxcClusters := make([]*dbaasv1beta1.PXCCluster, len(out.Clusters))
	for i, c := range out.Clusters {
		cluster := dbaasv1beta1.PXCCluster{
			Name: c.Name,
			Params: &dbaasv1beta1.PXCClusterParams{
				ClusterSize: c.Params.ClusterSize,
			},
			State: dbClusterStates()[c.State],
			Operation: &dbaasv1beta1.RunningOperation{
				TotalSteps:    c.Operation.TotalSteps,
				FinishedSteps: c.Operation.FinishedSteps,
				Message:       c.Operation.Message,
			},
			Exposed: c.Exposed,
		}

		if c.Params.Pxc != nil {
			cluster.Params.Pxc = &dbaasv1beta1.PXCClusterParams_PXC{
				DiskSize: c.Params.Pxc.DiskSize,
			}
			if c.Params.Pxc.ComputeResources != nil {
				cluster.Params.Pxc.ComputeResources = &dbaasv1beta1.ComputeResources{
					CpuM:        c.Params.Pxc.ComputeResources.CpuM,
					MemoryBytes: c.Params.Pxc.ComputeResources.MemoryBytes,
				}
			}
		}

		if c.Params.Haproxy != nil {
			if c.Params.Haproxy.ComputeResources != nil {
				cluster.Params.Haproxy = &dbaasv1beta1.PXCClusterParams_HAProxy{
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        c.Params.Haproxy.ComputeResources.CpuM,
						MemoryBytes: c.Params.Haproxy.ComputeResources.MemoryBytes,
					},
				}
			}
		} else if c.Params.Proxysql != nil {
			if c.Params.Proxysql.ComputeResources != nil {
				cluster.Params.Proxysql = &dbaasv1beta1.PXCClusterParams_ProxySQL{
					DiskSize: c.Params.Proxysql.DiskSize,
					ComputeResources: &dbaasv1beta1.ComputeResources{
						CpuM:        c.Params.Proxysql.ComputeResources.CpuM,
						MemoryBytes: c.Params.Proxysql.ComputeResources.MemoryBytes,
					},
				}
			}
		}

		if c.Params.Pxc.Image != "" {
			imageAndTag := strings.Split(c.Params.Pxc.Image, ":")
			if len(imageAndTag) != 2 {
				return nil, errors.Errorf("failed to parse Xtradb Cluster version out of %q", c.Params.Pxc.Image)
			}
			currentDBVersion := imageAndTag[1]

			nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, pxcOperator, operatorVersion, currentDBVersion)
			if err != nil {
				return nil, err
			}
			cluster.AvailableImage = nextVersionImage
			cluster.InstalledImage = c.Params.Pxc.Image
		}

		pxcClusters[i] = &cluster
	}
	return pxcClusters, nil
}

// RestartDBCluster restarts DB cluster by given name and type.
func (s DBClusterService) RestartDBCluster(ctx context.Context, req *dbaasv1beta1.RestartDBClusterRequest) (*dbaasv1beta1.RestartDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	switch req.ClusterType { //nolint:exhaustive
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		in := dbaascontrollerv1beta1.RestartPXCClusterRequest{
			Name: req.Name,
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		}

		_, err = s.controllerClient.RestartPXCCluster(ctx, &in)
		if err != nil {
			return nil, err
		}
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
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
	}

	return &dbaasv1beta1.RestartDBClusterResponse{}, nil
}

// DeleteDBCluster deletes DB cluster by given name and type.
func (s DBClusterService) DeleteDBCluster(ctx context.Context, req *dbaasv1beta1.DeleteDBClusterRequest) (*dbaasv1beta1.DeleteDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.New(ctx, kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}
	dbCluster, err := kubeClient.GetDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	dbCluster.TypeMeta.APIVersion = "dbaas.percona.com/v1"
	dbCluster.TypeMeta.Kind = "DatabaseCluster"
	err = kubeClient.DeleteDatabaseCluster(ctx, dbCluster)
	if err != nil {
		return nil, err
	}

	var clusterType string
	switch req.ClusterType {
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		clusterType = "pxc"
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
		clusterType = "psmdb"
	default:
		return nil, status.Error(codes.InvalidArgument, "unexpected DB cluster type")
	}

	err = s.grafanaClient.DeleteAPIKeysWithPrefix(ctx, fmt.Sprintf("%s-%s-%s", clusterType, req.KubernetesClusterName, req.Name))
	if err != nil {
		// ignore if API Key is not deleted.
		s.l.Warnf("Couldn't delete API key: %s", err)
	}

	return &dbaasv1beta1.DeleteDBClusterResponse{}, nil
}

func dbClusterStates() map[dbaascontrollerv1beta1.DBClusterState]dbaasv1beta1.DBClusterState {
	return map[dbaascontrollerv1beta1.DBClusterState]dbaasv1beta1.DBClusterState{
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_INVALID:   dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_INVALID,
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_CHANGING:  dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_CHANGING,
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_READY:     dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_READY,
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_FAILED:    dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_FAILED,
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_DELETING:  dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_DELETING,
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_PAUSED:    dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_PAUSED,
		dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_UPGRADING: dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_UPGRADING,
	}
}
