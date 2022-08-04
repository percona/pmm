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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
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
	service := &DBClusterService{
		db:                   db,
		l:                    l,
		controllerClient:     controllerClient,
		grafanaClient:        grafanaClient,
		versionServiceClient: versionServiceClient,
	}
	go func() {
		err := service.ImportDBClusters(context.TODO())
		if err != nil {
			l.Errorf("couldn't import db clusters: %q", err)
		}
	}()
	return service
}

func (s *DBClusterService) ImportDBClusters(ctx context.Context) error {
	clusters, err := models.FindAllKubernetesClusters(s.db.Querier)
	if err != nil {
		return err
	}
	g, ctx := errgroup.WithContext(ctx)
	for _, k := range clusters {
		kubernetesCluster := k
		g.Go(func() error {
			pxc, err := s.controllerClient.ListPXCClusters(ctx, &dbaascontrollerv1beta1.ListPXCClustersRequest{
				KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
					Kubeconfig: kubernetesCluster.KubeConfig,
				},
			})
			if err != nil {
				return err
			}

			psmdb, err := s.controllerClient.ListPSMDBClusters(ctx, &dbaascontrollerv1beta1.ListPSMDBClustersRequest{
				KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
					Kubeconfig: kubernetesCluster.KubeConfig,
				},
			})
			if err != nil {
				return err
			}
			tx, err := s.db.Begin()
			if err != nil {
				return err
			}

			for _, c := range pxc.Clusters {
				_, err := models.CreateOrUpdateDBCluster(tx.Querier, models.PXCType, &models.DBClusterParams{
					KubernetesClusterID: kubernetesCluster.ID,
					Name:                c.Name,
					InstalledImage:      c.Params.Pxc.Image,
				})
				if err != nil {
					tx.Rollback()
					return err
				}
			}

			for _, c := range psmdb.Clusters {
				_, err = models.CreateOrUpdateDBCluster(s.db.Querier, models.PSMDBType, &models.DBClusterParams{
					KubernetesClusterID: kubernetesCluster.ID,
					Name:                c.Name,
					InstalledImage:      c.Params.Image,
				})
				if err != nil {
					tx.Rollback()
					return err
				}
			}
			tx.Commit()

			return nil
		})
	}
	return g.Wait()
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

	clusters, err := models.FindDBClusters(s.db.Querier, models.DBClusterFilters{KubernetesClusterID: kubernetesCluster.ID})
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
			InstalledImage: cluster.InstalledImage,
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

	switch req.ClusterType {
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		pxcCluster, err := s.controllerClient.GetPXCCluster(ctx, kubernetesCluster.KubeConfig, req.Name)
		if err != nil {
			s.l.Errorf("couldn't get a PXC cluster")
			return nil, status.Errorf(codes.Internal, "couldn't get a PXC cluster: %q", err)
		}

		return s.convertPXCCluster(pxcCluster.Cluster)
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
		cluster, err := s.controllerClient.GetPSMDBCluster(ctx, kubernetesCluster.KubeConfig, req.Name)
		if err != nil {
			s.l.Errorf("couldn't get a PSMDB cluster")
			return nil, status.Errorf(codes.Internal, "couldn't get a PSMDB cluster: %q", err)
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

//
//func (s *DBClusterService) listPSMDBClusters(ctx context.Context, kubeConfig string, operatorVersion string) ([]*dbaasv1beta1.PSMDBCluster, error) {
//	in := dbaascontrollerv1beta1.ListPSMDBClustersRequest{
//		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
//			Kubeconfig: kubeConfig,
//		},
//	}
//
//	out, err := s.controllerClient.ListPSMDBClusters(ctx, &in)
//	if err != nil {
//		return nil, status.Errorf(codes.Internal, "Can't get list of PSMDB clusters: %s", err.Error())
//	}
//
//	clusters := make([]*dbaasv1beta1.PSMDBCluster, len(out.Clusters))
//	for i, c := range out.Clusters {
//		var computeResources *dbaasv1beta1.ComputeResources
//		var diskSize int64
//		if c.Params.Replicaset != nil {
//			diskSize = c.Params.Replicaset.DiskSize
//			if c.Params.Replicaset.ComputeResources != nil {
//				computeResources = &dbaasv1beta1.ComputeResources{
//					CpuM:        c.Params.Replicaset.ComputeResources.CpuM,
//					MemoryBytes: c.Params.Replicaset.ComputeResources.MemoryBytes,
//				}
//			}
//		}
//
//		cluster := dbaasv1beta1.PSMDBCluster{
//			Name: c.Name,
//			Params: &dbaasv1beta1.PSMDBClusterParams{
//				ClusterSize: c.Params.ClusterSize,
//				Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
//					ComputeResources: computeResources,
//					DiskSize:         diskSize,
//				},
//			},
//			State: dbClusterStates()[c.State],
//			Operation: &dbaasv1beta1.RunningOperation{
//				TotalSteps:    c.Operation.TotalSteps,
//				FinishedSteps: c.Operation.FinishedSteps,
//				Message:       c.Operation.Message,
//			},
//			Exposed: c.Exposed,
//		}
//
//		if c.Params.Image != "" {
//			imageAndTag := strings.Split(c.Params.Image, ":")
//			if len(imageAndTag) != 2 {
//				return nil, errors.Errorf("failed to parse PSMDB version out of %q", c.Params.Image)
//			}
//			currentDBVersion := imageAndTag[1]
//
//			nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, psmdbOperator, operatorVersion, currentDBVersion)
//			if err != nil {
//				return nil, err
//			}
//			cluster.AvailableImage = nextVersionImage
//			cluster.InstalledImage = c.Params.Image
//		}
//
//		clusters[i] = &cluster
//	}
//
//	return clusters, nil
//}
//
//func (s *DBClusterService) listPXCClusters(ctx context.Context, kubernetesClusterID string, operatorVersion string) ([]*dbaasv1beta1.PXCCluster, error) {
//	clusters, err := models.FindDBClusters(s.db.Querier, models.DBClusterFilters{
//		KubernetesClusterID: kubernetesClusterID,
//		ClusterType:         models.PXCType,
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	pxcClusters := make([]*dbaasv1beta1.PXCCluster, len(clusters))
//	for i, c := range clusters {
//		cluster := dbaasv1beta1.PXCCluster{
//			Name: c.Name,
//			Params: &dbaasv1beta1.PXCClusterParams{
//				ClusterSize: c.PXCClusterParams.ClusterSize,
//			},
//			// TODO: what do to with it?
//			//State: dbClusterStates()[c.State],
//			//Operation: &dbaasv1beta1.RunningOperation{
//			//	TotalSteps:    c.Operation.TotalSteps,
//			//	FinishedSteps: c.Operation.FinishedSteps,
//			//	Message:       c.Operation.Message,
//			//},
//			Exposed: c.Exposed,
//		}
//
//		if c.PXCClusterParams.Pxc != nil {
//			cluster.Params.Pxc = &dbaasv1beta1.PXCClusterParams_PXC{
//				DiskSize: c.PXCClusterParams.Pxc.DiskSize,
//			}
//			if c.PXCClusterParams.Pxc.ComputeResources != nil {
//				cluster.Params.Pxc.ComputeResources = &dbaasv1beta1.ComputeResources{
//					CpuM:        c.PXCClusterParams.Pxc.ComputeResources.CpuM,
//					MemoryBytes: c.PXCClusterParams.Pxc.ComputeResources.MemoryBytes,
//				}
//			}
//		}
//
//		if c.PXCClusterParams.Haproxy != nil {
//			if c.PXCClusterParams.Haproxy.ComputeResources != nil {
//				cluster.Params.Haproxy = &dbaasv1beta1.PXCClusterParams_HAProxy{
//					ComputeResources: &dbaasv1beta1.ComputeResources{
//						CpuM:        c.PXCClusterParams.Haproxy.ComputeResources.CpuM,
//						MemoryBytes: c.PXCClusterParams.Haproxy.ComputeResources.MemoryBytes,
//					},
//				}
//			}
//		} else if c.PXCClusterParams.Proxysql != nil {
//			if c.PXCClusterParams.Proxysql.ComputeResources != nil {
//				cluster.Params.Proxysql = &dbaasv1beta1.PXCClusterParams_ProxySQL{
//					DiskSize: c.PXCClusterParams.Proxysql.DiskSize,
//					ComputeResources: &dbaasv1beta1.ComputeResources{
//						CpuM:        c.PXCClusterParams.Proxysql.ComputeResources.CpuM,
//						MemoryBytes: c.PXCClusterParams.Proxysql.ComputeResources.MemoryBytes,
//					},
//				}
//			}
//		}
//
//		if c.InstalledImage != "" {
//			imageAndTag := strings.Split(c.PXCClusterParams.Pxc.Image, ":")
//			if len(imageAndTag) != 2 {
//				return nil, errors.Errorf("failed to parse Xtradb Cluster version out of %q", c.PXCClusterParams.Pxc.Image)
//			}
//			currentDBVersion := imageAndTag[1]
//
//			nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, pxcOperator, operatorVersion, currentDBVersion)
//			if err != nil {
//				return nil, err
//			}
//			cluster.AvailableImage = nextVersionImage
//			cluster.InstalledImage = c.InstalledImage
//		}
//
//		pxcClusters[i] = &cluster
//	}
//	return pxcClusters, nil
//}

// RestartDBCluster restarts DB cluster by given name and type.
func (s *DBClusterService) RestartDBCluster(ctx context.Context, req *dbaasv1beta1.RestartDBClusterRequest) (*dbaasv1beta1.RestartDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	switch req.ClusterType {
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
