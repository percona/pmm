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
	"strconv"
	"strings"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	corev1 "k8s.io/api/core/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

type DBClusterService struct {
	db                   *reform.DB
	l                    *logrus.Entry
	grafanaClient        grafanaClient
	kubernetesClient     kubernetesClient
	versionServiceClient *VersionServiceClient

	dbaasv1beta1.UnimplementedDBClustersServer
}

// NewDBClusterService creates DB Clusters Service.
func NewDBClusterService(db *reform.DB, grafanaClient grafanaClient, kubernetesClient kubernetesClient, versionServiceClient *VersionServiceClient) dbaasv1beta1.DBClustersServer { //nolint:lll
	l := logrus.WithField("component", "dbaas_db_cluster")
	return &DBClusterService{
		db:                   db,
		l:                    l,
		kubernetesClient:     kubernetesClient,
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
	if err := s.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}
	dbClusters, err := s.kubernetesClient.ListDatabaseClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing database clusters")
	}
	psmdbOperatorVersion, err := s.kubernetesClient.GetPSMDBOperatorVersion(ctx)
	if err != nil {
		s.l.Errorf("failed determining version of psmdb operator: %v", err)
	}

	pxcOperatorVersion, err := s.kubernetesClient.GetPXCOperatorVersion(ctx)
	if err != nil {
		s.l.Errorf("failed determining version of pxc operator: %v", err)
	}
	psmdbClusters := []*dbaasv1beta1.PSMDBCluster{}
	pxcClusters := []*dbaasv1beta1.PXCCluster{}

	for _, cluster := range dbClusters.Items {
		switch cluster.Spec.Database {
		case kubernetes.DatabaseTypePXC:
			c, err := s.getPXCCluster(ctx, cluster, pxcOperatorVersion)
			if err != nil {
				s.l.Errorf("failed getting PXC cluster: %v", err)
			}
			pxcClusters = append(pxcClusters, c)
		case kubernetes.DatabaseTypePSMDB:
			c, err := s.getPSMDBCluster(ctx, cluster, psmdbOperatorVersion)
			if err != nil {
				s.l.Errorf("failed getting PSMDB cluster: %v", err)
			}
			psmdbClusters = append(psmdbClusters, c)
		default:
			s.l.Errorf("unsupported database type %s", cluster.Spec.Database)
		}
	}

	return &dbaasv1beta1.ListDBClustersResponse{
		PxcClusters:   pxcClusters,
		PsmdbClusters: psmdbClusters,
	}, nil
}

func (s DBClusterService) getClusterResource(instance dbaasv1.DBInstanceSpec) (diskSize int64, memory int64, cpu int, err error) {
	disk, ok := (&instance.DiskSize).AsInt64()
	if ok {
		diskSize = disk
	}
	mem, ok := (&instance.Memory).AsInt64()
	if ok {
		memory = mem
	}
	cpuResource := (&instance.CPU).String()
	var cpuMillis int64
	if strings.HasSuffix(cpuResource, "m") {
		cpuResource = cpuResource[:len(cpuResource)-1]
		cpuMillis, err = strconv.ParseInt(cpuResource, 10, 64)
		if err != nil {
			return
		}
	}
	cpu = int(cpuMillis)
	var floatCPU float64
	if cpuMillis == 0 {
		floatCPU, err = strconv.ParseFloat(cpuResource, 64)
		if err != nil {
			return
		}
		cpu = int(floatCPU * 1000)
	}
	return
}

func (s DBClusterService) getPXCCluster(ctx context.Context, cluster dbaasv1.DatabaseCluster, operatorVersion string) (*dbaasv1beta1.PXCCluster, error) {
	_, internetFacing := cluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"]
	diskSize, memory, cpu, err := s.getClusterResource(cluster.Spec.DBInstance)
	if err != nil {
		return nil, err
	}
	c := &dbaasv1beta1.PXCCluster{
		Name: cluster.Name,
		Params: &dbaasv1beta1.PXCClusterParams{
			ClusterSize: cluster.Spec.ClusterSize,
			Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
				DiskSize: diskSize,
				ComputeResources: &dbaasv1beta1.ComputeResources{
					CpuM:        int32(cpu),
					MemoryBytes: memory,
				},
				Configuration: cluster.Spec.DatabaseConfig,
			},
		},
		State:          dbClusterStates()[cluster.Status.State],
		Exposed:        cluster.Spec.LoadBalancer.ExposeType == corev1.ServiceTypeNodePort || cluster.Spec.LoadBalancer.ExposeType == corev1.ServiceTypeLoadBalancer,
		InternetFacing: internetFacing,
		Operation: &dbaasv1beta1.RunningOperation{
			TotalSteps:    cluster.Status.Size,
			FinishedSteps: cluster.Status.Ready,
			Message:       cluster.Status.Message,
		},
	}
	if cluster.Spec.DBInstance.StorageClassName != nil {
		c.Params.Pxc.StorageClass = *cluster.Spec.DBInstance.StorageClassName
	}
	if cluster.Spec.LoadBalancer.Type == "proxysql" {
		compute, err := s.getComputeResources(cluster.Spec.LoadBalancer.Resources.Requests)
		if err != nil {
			s.l.Errorf("could not parse resources for proxysql %v", err)
		}
		c.Params.Proxysql = &dbaasv1beta1.PXCClusterParams_ProxySQL{
			ComputeResources: compute,
			Image:            cluster.Spec.LoadBalancer.Image,
		}
	}
	if cluster.Spec.LoadBalancer.Type == "haproxy" {
		compute, err := s.getComputeResources(cluster.Spec.LoadBalancer.Resources.Requests)
		if err != nil {
			s.l.Errorf("could not parse resources for proxysql %v", err)
		}
		c.Params.Haproxy = &dbaasv1beta1.PXCClusterParams_HAProxy{
			ComputeResources: compute,
			Image:            cluster.Spec.LoadBalancer.Image,
		}
	}
	imageAndTag := strings.Split(cluster.Spec.DatabaseImage, ":")
	if len(imageAndTag) != 2 {
		return nil, errors.Errorf("failed to parse Xtradb Cluster version out of %q", cluster.Spec.DatabaseImage)
	}
	currentDBVersion := imageAndTag[1]

	nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, pxcOperator, operatorVersion, currentDBVersion)
	if err != nil {
		return nil, err
	}
	c.AvailableImage = nextVersionImage
	c.InstalledImage = cluster.Spec.DatabaseImage
	return c, nil
}

func (s DBClusterService) getComputeResources(resources corev1.ResourceList) (*dbaasv1beta1.ComputeResources, error) {
	compute := &dbaasv1beta1.ComputeResources{}
	cpuLimit, ok := resources[corev1.ResourceCPU]
	if ok {
		cpu := (&cpuLimit).String()
		if strings.HasSuffix(cpu, "m") {
			cpu = cpu[:len(cpu)-1]
			millis, err := strconv.ParseUint(cpu, 10, 64)
			if err != nil {
				return compute, err
			}
			compute.CpuM = int32(millis)
		}
		if compute.CpuM == 0 {
			floatCPU, err := strconv.ParseFloat(cpu, 64)
			if err != nil {
				return compute, err
			}
			compute.CpuM = int32(floatCPU * 1000)
		}
	}
	memLimit, ok := resources[corev1.ResourceMemory]
	if ok {
		mem, ok := (&memLimit).AsInt64()
		if ok {
			compute.MemoryBytes = mem
		}
	}
	return compute, nil
}

func (s DBClusterService) getPSMDBCluster(ctx context.Context, cluster dbaasv1.DatabaseCluster, operatorVersion string) (*dbaasv1beta1.PSMDBCluster, error) {
	diskSize, memory, cpu, err := s.getClusterResource(cluster.Spec.DBInstance)
	if err != nil {
		return nil, err
	}
	_, internetFacing := cluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"]
	c := &dbaasv1beta1.PSMDBCluster{
		Name: cluster.Name,
		Params: &dbaasv1beta1.PSMDBClusterParams{
			ClusterSize: cluster.Spec.ClusterSize,
			Replicaset: &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{
				DiskSize: diskSize,
				ComputeResources: &dbaasv1beta1.ComputeResources{
					CpuM:        int32(cpu),
					MemoryBytes: memory,
				},
				Configuration: cluster.Spec.DatabaseConfig,
			},
		},
		State:          dbClusterStates()[cluster.Status.State],
		Exposed:        cluster.Spec.LoadBalancer.ExposeType == corev1.ServiceTypeNodePort || cluster.Spec.LoadBalancer.ExposeType == corev1.ServiceTypeLoadBalancer,
		InternetFacing: internetFacing,
		Operation: &dbaasv1beta1.RunningOperation{
			TotalSteps:    cluster.Status.Size,
			FinishedSteps: cluster.Status.Ready,
			// TODO: Add messages
			Message: "",
		},
	}
	if cluster.Spec.DBInstance.StorageClassName != nil {
		c.Params.Replicaset.StorageClass = *cluster.Spec.DBInstance.StorageClassName
	}
	imageAndTag := strings.Split(cluster.Spec.DatabaseImage, ":")
	if len(imageAndTag) != 2 {
		return nil, errors.Errorf("failed to parse PSMDB version out of %q", cluster.Spec.DatabaseImage)
	}
	currentDBVersion := imageAndTag[1]

	nextVersionImage, err := s.versionServiceClient.GetNextDatabaseImage(ctx, psmdbOperator, operatorVersion, currentDBVersion)
	if err != nil {
		return nil, err
	}
	c.AvailableImage = nextVersionImage
	c.InstalledImage = cluster.Spec.DatabaseImage
	return c, nil
}

func (s DBClusterService) GetDBCluster(ctx context.Context, req *dbaasv1beta1.GetDBClusterRequest) (*dbaasv1beta1.GetDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	if err := s.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}
	dbCluster, err := s.kubernetesClient.GetDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting the database cluster")
	}
	psmdbOperatorVersion, err := s.kubernetesClient.GetPSMDBOperatorVersion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting psmdb operator version")
	}

	pxcOperatorVersion, err := s.kubernetesClient.GetPXCOperatorVersion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting pxc operator version")
	}
	resp := &dbaasv1beta1.GetDBClusterResponse{}
	if dbCluster.Spec.Database == kubernetes.DatabaseTypePXC && pxcOperatorVersion != "" {
		c, err := s.getPXCCluster(ctx, *dbCluster, pxcOperatorVersion)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting PXC cluster")
		}
		resp.PxcCluster = c
	}
	if dbCluster.Spec.Database == kubernetes.DatabaseTypePSMDB && psmdbOperatorVersion != "" {
		c, err := s.getPSMDBCluster(ctx, *dbCluster, psmdbOperatorVersion)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting PSMDB cluster")
		}
		resp.PsmdbCluster = c
	}
	return resp, nil
}

// RestartDBCluster restarts DB cluster by given name and type.
func (s DBClusterService) RestartDBCluster(ctx context.Context, req *dbaasv1beta1.RestartDBClusterRequest) (*dbaasv1beta1.RestartDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	if err := s.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}
	err = s.kubernetesClient.RestartDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.RestartDBClusterResponse{}, nil
}

// DeleteDBCluster deletes DB cluster by given name and type.
func (s DBClusterService) DeleteDBCluster(ctx context.Context, req *dbaasv1beta1.DeleteDBClusterRequest) (*dbaasv1beta1.DeleteDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	if err := s.kubernetesClient.SetKubeconfig(kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}
	err = s.kubernetesClient.DeleteDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	var clusterType string
	switch req.ClusterType {
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PXC:
		clusterType = string(kubernetes.DatabaseTypePXC)
	case dbaasv1beta1.DBClusterType_DB_CLUSTER_TYPE_PSMDB:
		clusterType = string(kubernetes.DatabaseTypePSMDB)
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

func dbClusterStates() map[dbaasv1.AppState]dbaasv1beta1.DBClusterState {
	return map[dbaasv1.AppState]dbaasv1beta1.DBClusterState{
		dbaasv1.AppStateUnknown:  dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_INVALID,
		dbaasv1.AppStateInit:     dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_CHANGING,
		dbaasv1.AppStateReady:    dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_READY,
		dbaasv1.AppStateError:    dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_FAILED,
		dbaasv1.AppStateStopping: dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_CHANGING,
		dbaasv1.AppStatePaused:   dbaasv1beta1.DBClusterState_DB_CLUSTER_STATE_PAUSED,
	}
}
