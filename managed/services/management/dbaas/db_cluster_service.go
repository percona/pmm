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
)

type DBClusterService struct {
	db                     *reform.DB
	l                      *logrus.Entry
	controllerClient       dbaasClient
	grafanaClient          grafanaClient
	versionServiceClient   *VersionServiceClient
	dbClustersSynchronizer dbClusterSynchronizer

	dbaasv1beta1.UnimplementedDBClustersServer
}

// NewDBClusterService creates DB Clusters Service.
func NewDBClusterService(db *reform.DB, controllerClient dbaasClient, grafanaClient grafanaClient, versionServiceClient *VersionServiceClient, dbClustersSynchronizer dbClusterSynchronizer) dbaasv1beta1.DBClustersServer {
	l := logrus.WithField("component", "dbaas_db_cluster")
	service := &DBClusterService{
		db:                     db,
		l:                      l,
		controllerClient:       controllerClient,
		grafanaClient:          grafanaClient,
		versionServiceClient:   versionServiceClient,
		dbClustersSynchronizer: dbClustersSynchronizer,
	}
	return service
}

// ListDBClusters returns a list of all DB clusters.
func (s *DBClusterService) ListDBClusters(ctx context.Context, req *dbaasv1beta1.ListDBClustersRequest) (*dbaasv1beta1.ListDBClustersResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	checkResponse, err := s.controllerClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
	if err != nil {
		return nil, err
	}

	clusters, err := models.FindDBClustersForKubernetesCluster(s.db.Querier, kubernetesCluster.ID)
	if err != nil {
		return nil, err
	}

	dbClusters := make([]*dbaasv1beta1.DBCluster, len(clusters))
	for i, cluster := range clusters {
		var operatorVersion string
		var operator string
		var clusterType dbaasv1beta1.DBClusterType
		switch cluster.ClusterType {
		case models.PXCType:
			clusterType = dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC
			operatorVersion = checkResponse.Operators.PxcOperatorVersion
			operator = pxcOperator
		case models.PSMDBType:
			clusterType = dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB
			operatorVersion = checkResponse.Operators.PsmdbOperatorVersion
			operator = psmdbOperator
		default:
			panic("unexpected cluster type")
		}
		imageAndTag := strings.Split(cluster.InstalledImage, ":")
		if len(imageAndTag) != 2 {
			return nil, errors.Errorf("failed to parse PSMDB version out of %q", cluster.InstalledImage)
		}
		currentDBVersion := imageAndTag[1]

		nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, operator, operatorVersion, currentDBVersion)
		if err != nil {
			return nil, err
		}
		dbClusters[i] = &dbaasv1beta1.DBCluster{
			Id:             cluster.ID,
			Name:           cluster.Name,
			ClusterType:    clusterType,
			InstalledImage: currentDBVersion,
			AvailableImage: nextVersionImage,
		}
	}

	return &dbaasv1beta1.ListDBClustersResponse{
		DbClusters: dbClusters,
	}, nil
}

// GetDBCluster returns an information about the cluster of the certain type
func (s *DBClusterService) GetDBCluster(ctx context.Context, req *dbaasv1beta1.GetDBClusterRequest) (*dbaasv1beta1.GetDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	dbCluster, err := models.FindDBCluster(s.db.Querier, kubernetesCluster.ID, req.Name, dbTypes()[req.ClusterType])
	if err != nil {
		return nil, err
	}

	switch req.ClusterType {
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		pxcCluster, err := s.controllerClient.GetPXCCluster(ctx, kubernetesCluster.KubeConfig, req.Name)
		if err != nil {
			statusErr, ok := status.FromError(err)
			if ok {
				s.l.Errorf("couldn't get a PXC cluster: %q", err)
				if statusErr.Code() == codes.NotFound {
					go func() {
						_ = s.dbClustersSynchronizer.RemoveDBCluster(dbCluster)
					}()
				}
				return nil, status.Errorf(statusErr.Code(), "couldn't get a PXC cluster: %q", statusErr.Message())
			}
			return nil, err
		}

		return s.convertPXCCluster(pxcCluster.Cluster)
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
		cluster, err := s.controllerClient.GetPSMDBCluster(ctx, kubernetesCluster.KubeConfig, req.Name)
		if err != nil {
			statusErr, ok := status.FromError(err)
			if ok {
				s.l.Errorf("couldn't get a PSMDB cluster: %q", err)
				if statusErr.Code() == codes.NotFound {
					go func() {
						_ = s.dbClustersSynchronizer.RemoveDBCluster(dbCluster)
					}()
				}
				return nil, status.Errorf(statusErr.Code(), "couldn't get a PSMDB cluster: %q", statusErr.Message())
			}
			return nil, err
		}

		return s.convertPSMDBCluster(cluster.Cluster)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unexpected db cluster type")
	}
}

func (s *DBClusterService) convertPXCCluster(c *dbaascontrollerv1beta1.PXCCluster) (*dbaasv1beta1.GetDBClusterResponse, error) {
	pxcCluster := dbaasv1beta1.PXCCluster{
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
		pxcCluster.Params.Pxc = &dbaasv1beta1.PXCClusterParams_PXC{
			DiskSize: c.Params.Pxc.DiskSize,
		}
		if c.Params.Pxc.ComputeResources != nil {
			pxcCluster.Params.Pxc.ComputeResources = &dbaasv1beta1.ComputeResources{
				CpuM:        c.Params.Pxc.ComputeResources.CpuM,
				MemoryBytes: c.Params.Pxc.ComputeResources.MemoryBytes,
			}
		}
	}

	if c.Params.Haproxy != nil {
		if c.Params.Haproxy.ComputeResources != nil {
			pxcCluster.Params.Haproxy = &dbaasv1beta1.PXCClusterParams_HAProxy{
				ComputeResources: &dbaasv1beta1.ComputeResources{
					CpuM:        c.Params.Haproxy.ComputeResources.CpuM,
					MemoryBytes: c.Params.Haproxy.ComputeResources.MemoryBytes,
				},
			}
		}
	} else if c.Params.Proxysql != nil {
		if c.Params.Proxysql.ComputeResources != nil {
			pxcCluster.Params.Proxysql = &dbaasv1beta1.PXCClusterParams_ProxySQL{
				DiskSize: c.Params.Proxysql.DiskSize,
				ComputeResources: &dbaasv1beta1.ComputeResources{
					CpuM:        c.Params.Proxysql.ComputeResources.CpuM,
					MemoryBytes: c.Params.Proxysql.ComputeResources.MemoryBytes,
				},
			}
		}
	}
	return &dbaasv1beta1.GetDBClusterResponse{
		Cluster: &dbaasv1beta1.GetDBClusterResponse_PxcCluster{
			PxcCluster: &pxcCluster,
		},
	}, nil
}

func (s *DBClusterService) convertPSMDBCluster(c *dbaascontrollerv1beta1.PSMDBCluster) (*dbaasv1beta1.GetDBClusterResponse, error) {
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
	psmdbCluster := dbaasv1beta1.PSMDBCluster{
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
	return &dbaasv1beta1.GetDBClusterResponse{
		Cluster: &dbaasv1beta1.GetDBClusterResponse_PsmdbCluster{
			PsmdbCluster: &psmdbCluster,
		},
	}, nil
}

// RestartDBCluster restarts DB cluster by given name and type.
func (s *DBClusterService) RestartDBCluster(ctx context.Context, req *dbaasv1beta1.RestartDBClusterRequest) (*dbaasv1beta1.RestartDBClusterResponse, error) {
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
func (s *DBClusterService) DeleteDBCluster(ctx context.Context, req *dbaasv1beta1.DeleteDBClusterRequest) (*dbaasv1beta1.DeleteDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	cluster, err := models.FindDBCluster(s.db.Querier, kubernetesCluster.ID, req.Name, dbTypes()[req.ClusterType])
	if err != nil {
		return nil, err
	}

	var clusterType string
	switch req.ClusterType {
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		in := dbaascontrollerv1beta1.DeletePXCClusterRequest{
			Name: req.Name,
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		}

		_, err = s.controllerClient.DeletePXCCluster(ctx, &in)
		if err != nil {
			return nil, err
		}
		clusterType = "pxc"
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
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
		clusterType = "psmdb"
	default:
		return nil, status.Error(codes.InvalidArgument, "unexpected DB cluster type")
	}

	err = s.grafanaClient.DeleteAPIKeysWithPrefix(ctx, fmt.Sprintf("%s-%s-%s", clusterType, req.KubernetesClusterName, req.Name))
	if err != nil {
		// ignore if API Key is not deleted.
		s.l.Warnf("Couldn't delete API key: %s", err)
	}

	go func() {
		s.dbClustersSynchronizer.WatchDBClusterDeletion(cluster)
	}()

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

func dbTypes() map[dbaasv1beta1.DBClusterType]models.DBClusterType {
	return map[dbaasv1beta1.DBClusterType]models.DBClusterType{
		dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:   models.PXCType,
		dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB: models.PSMDBType,
	}
}
