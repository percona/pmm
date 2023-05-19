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
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

const (
	postgresqlDefaultClusterSize = 3
	postgresqlDefaultCPUM        = 1000
	postgresqlDefaultMemoryBytes = 2000000000
	postgresqlDefaultDiskSize    = 25000000000
	pgBouncerDefaultCPUM         = 500
	pgBouncerDefaultMemoryBytes  = 500000000
)

// PostgresqlClustersService implements PostgresqlClusterServer methods.
type PostgresqlClustersService struct {
	db                *reform.DB
	l                 *logrus.Entry
	grafanaClient     grafanaClient
	componentsService componentsService
	kubeStorage       *KubeStorage
	versionServiceURL string

	dbaasv1beta1.UnimplementedPostgresqlClustersServer
}

// NewPostgresqlClusterService creates Postgresql Service.
func NewPostgresqlClusterService(db *reform.DB, grafanaClient grafanaClient, componentsService componentsService,
	versionServiceURL string,
) dbaasv1beta1.PostgresqlClustersServer {
	l := logrus.WithField("component", "postgresql_cluster")
	return &PostgresqlClustersService{
		db:                db,
		l:                 l,
		grafanaClient:     grafanaClient,
		versionServiceURL: versionServiceURL,
		componentsService: componentsService,
		kubeStorage:       NewKubeStorage(db),
	}
}

// GetPostgresqlClusterCredentials returns a Postgresql cluster credentials.
func (s PostgresqlClustersService) GetPostgresqlClusterCredentials(ctx context.Context,
	req *dbaasv1beta1.GetPostgresqlClusterCredentialsRequest,
) (*dbaasv1beta1.GetPostgresqlClusterCredentialsResponse, error) {
	kubeClient, err := s.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	dbCluster, err := kubeClient.GetDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting database cluster")
	}
	secret, err := kubeClient.GetSecret(ctx, dbCluster.Spec.SecretsName)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting secret")
	}

	port, err := strconv.ParseInt(string(secret.Data["port"]), 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting port")
	}

	resp := dbaasv1beta1.GetPostgresqlClusterCredentialsResponse{
		ConnectionCredentials: &dbaasv1beta1.PostgresqlClusterConnectionCredentials{
			Username: string(secret.Data["user"]),
			Password: string(secret.Data["password"]),
			Host:     dbCluster.Status.Host,
			Port:     int32(port),
		},
	}

	return &resp, nil
}

// CreatePostgresqlCluster creates Postgresql cluster with given parameters.
//
//nolint:dupl
func (s PostgresqlClustersService) CreatePostgresqlCluster(ctx context.Context,
	req *dbaasv1beta1.CreatePostgresqlClusterRequest,
) (*dbaasv1beta1.CreatePostgresqlClusterResponse, error) {
	if req.Params == nil {
		req.Params = &dbaasv1beta1.PostgresqlClusterParams{}
	}

	kubeClient, err := s.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	if err := s.fillDefaults(req); err != nil {
		return nil, errors.Wrap(err, "cannot create postgresql cluster")
	}

	if req.Params.Instance.StorageClass == "" {
		className, err := kubeClient.GetDefaultStorageClassName(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get storage classes")
		}
		req.Params.Instance.StorageClass = className
	}
	clusterType, err := kubeClient.GetClusterType(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting cluster type")
	}
	backupLocation, err := s.getBackupLocation(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting backup location")
	}
	dbCluster, _, err := kubernetes.DatabaseClusterForPostgresql(req, clusterType, backupLocation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CR specification")
	}

	err = kubeClient.CreateDatabaseCluster(dbCluster)

	return &dbaasv1beta1.CreatePostgresqlClusterResponse{}, nil
}

func (s PostgresqlClustersService) fillDefaults(req *dbaasv1beta1.CreatePostgresqlClusterRequest) error {
	if req.Name != "" {
		r := regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")
		if !r.MatchString(req.Name) {
			return errInvalidClusterName
		}
	}
	if req.Params == nil {
		req.Params = &dbaasv1beta1.PostgresqlClusterParams{}
	}

	if req.Params.ClusterSize < 1 {
		req.Params.ClusterSize = postgresqlDefaultClusterSize
	}

	if req.Params.Instance == nil {
		req.Params.Instance = &dbaasv1beta1.PostgresqlClusterParams_Instance{}
	}

	if req.Params.Instance.DiskSize == 0 {
		req.Params.Instance.DiskSize = postgresqlDefaultDiskSize
	}

	if req.Params.Instance.ComputeResources == nil {
		req.Params.Instance.ComputeResources = &dbaasv1beta1.ComputeResources{
			CpuM:        postgresqlDefaultCPUM,
			MemoryBytes: postgresqlDefaultMemoryBytes,
		}
	}
	if req.Params.Instance.ComputeResources.CpuM == 0 {
		req.Params.Instance.ComputeResources.CpuM = postgresqlDefaultCPUM
	}
	if req.Params.Instance.ComputeResources.MemoryBytes == 0 {
		req.Params.Instance.ComputeResources.MemoryBytes = postgresqlDefaultMemoryBytes
	}

	if req.Params.Pgbouncer == nil {
		req.Params.Pgbouncer = &dbaasv1beta1.PostgresqlClusterParams_PGBouncer{
			ComputeResources: &dbaasv1beta1.ComputeResources{
				CpuM:        pgBouncerDefaultCPUM,
				MemoryBytes: pgBouncerDefaultMemoryBytes,
			},
		}
	}

	if req.Params.Pgbouncer != nil {
		if req.Params.Pgbouncer.ComputeResources == nil {
			req.Params.Pgbouncer.ComputeResources = &dbaasv1beta1.ComputeResources{
				CpuM:        pgBouncerDefaultCPUM,
				MemoryBytes: pgBouncerDefaultMemoryBytes,
			}
		}
		if req.Params.Pgbouncer.ComputeResources.CpuM == 0 {
			req.Params.Pgbouncer.ComputeResources.CpuM = pgBouncerDefaultCPUM
		}
		if req.Params.Pgbouncer.ComputeResources.MemoryBytes == 0 {
			req.Params.Pgbouncer.ComputeResources.MemoryBytes = pgBouncerDefaultMemoryBytes
		}
		// FIXME using hardcoded values for PGBouncer until version service
		// supports it https://jira.percona.com/browse/K8SPG-315
		if req.Params.Pgbouncer.Image == "" {
			req.Params.Pgbouncer.Image = "perconalab/percona-postgresql-operator:main-ppg14-pgbouncer"
		}
	}

	return nil
}

// UpdatePostgresqlCluster updates Postgresql cluster.
//
//nolint:dupl
func (s PostgresqlClustersService) UpdatePostgresqlCluster(ctx context.Context,
	req *dbaasv1beta1.UpdatePostgresqlClusterRequest,
) (*dbaasv1beta1.UpdatePostgresqlClusterResponse, error) {
	kubeClient, err := s.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	dbCluster, err := kubeClient.GetDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	clusterType, err := kubeClient.GetClusterType(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting cluster type")
	}
	err = kubernetes.UpdatePatchForPostgresql(dbCluster, req, clusterType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CR specification")
	}

	err = kubeClient.PatchDatabaseCluster(dbCluster)

	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UpdatePostgresqlClusterResponse{}, nil
}

// GetPostgresqlClusterResources returns expected resources to be consumed by the cluster.
func (s PostgresqlClustersService) GetPostgresqlClusterResources(_ context.Context,
	req *dbaasv1beta1.GetPostgresqlClusterResourcesRequest,
) (*dbaasv1beta1.GetPostgresqlClusterResourcesResponse, error) {
	clusterSize := uint64(req.Params.ClusterSize)
	memory := uint64(req.Params.Instance.ComputeResources.MemoryBytes+req.Params.Pgbouncer.ComputeResources.MemoryBytes) * clusterSize
	cpu := uint64(req.Params.Instance.ComputeResources.CpuM+req.Params.Pgbouncer.ComputeResources.CpuM) * clusterSize
	disk := uint64(req.Params.Instance.DiskSize+req.Params.Pgbouncer.DiskSize) * clusterSize

	return &dbaasv1beta1.GetPostgresqlClusterResourcesResponse{
		Expected: &dbaasv1beta1.Resources{
			CpuM:        cpu,
			MemoryBytes: memory,
			DiskSize:    disk,
		},
	}, nil
}

func (s PostgresqlClustersService) getBackupLocation(req *dbaasv1beta1.CreatePostgresqlClusterRequest) (*models.BackupLocation, error) {
	if req.Params != nil && req.Params.Backup != nil && req.Params.Backup.LocationId != "" {
		return models.FindBackupLocationByID(s.db.Querier, req.Params.Backup.LocationId)
	}
	if req.Params != nil && req.Params.Restore != nil && req.Params.Restore.LocationId != "" {
		return models.FindBackupLocationByID(s.db.Querier, req.Params.Restore.LocationId)
	}
	return nil, nil //nolint:nilnil
}
