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
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

const (
	psmdbDefaultClusterSize = 3
	psmdbDefaultCPUM        = 1000
	psmdbDefaultMemoryBytes = 2000000000
	psmdbDefaultDiskSize    = 25000000000
)

// PSMDBClusterService implements PSMDBClusterServer methods.
type PSMDBClusterService struct {
	db                *reform.DB
	l                 *logrus.Entry
	grafanaClient     grafanaClient
	kubernetesClient  kubernetesClient
	componentsService componentsService
	versionServiceURL string

	dbaasv1beta1.UnimplementedPSMDBClustersServer
}

// NewPSMDBClusterService creates PSMDB Service.
func NewPSMDBClusterService(db *reform.DB, grafanaClient grafanaClient, kubernetesClient kubernetesClient,
	componentsService componentsService, versionServiceURL string,
) dbaasv1beta1.PSMDBClustersServer {
	l := logrus.WithField("component", "psmdb_cluster")
	return &PSMDBClusterService{
		db:                db,
		l:                 l,
		grafanaClient:     grafanaClient,
		kubernetesClient:  kubernetesClient,
		componentsService: componentsService,
		versionServiceURL: versionServiceURL,
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

// GetPSMDBClusterCredentials returns a PSMDB cluster credentials by cluster name.
func (s PSMDBClusterService) GetPSMDBClusterCredentials(ctx context.Context, req *dbaasv1beta1.GetPSMDBClusterCredentialsRequest) (*dbaasv1beta1.GetPSMDBClusterCredentialsResponse, error) { //nolint:lll
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	if err := s.kubernetesClient.SetKubeconfig(ctx, kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}
	dbCluster, err := s.kubernetesClient.GetDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting database cluster")
	}
	secret, err := s.kubernetesClient.GetSecret(ctx, fmt.Sprintf(psmdbSecretNameTmpl, req.Name))
	if err != nil {
		return nil, errors.Wrap(err, "failed getting secret")
	}

	resp := dbaasv1beta1.GetPSMDBClusterCredentialsResponse{
		ConnectionCredentials: &dbaasv1beta1.GetPSMDBClusterCredentialsResponse_PSMDBCredentials{
			Username:   string(secret.Data["MONGODB_USER_ADMIN_USER"]),
			Password:   string(secret.Data["MONGODB_USER_ADMIN_PASSWORD"]),
			Host:       dbCluster.Status.Host,
			Port:       27017,
			Replicaset: "rs0",
		},
	}

	return &resp, nil
}

// CreatePSMDBCluster creates PSMDB cluster with given parameters.
//
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
	if err := s.kubernetesClient.SetKubeconfig(ctx, kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}

	psmdbComponents, err := s.componentsService.GetPSMDBComponents(ctx, &dbaasv1beta1.GetPSMDBComponentsRequest{
		KubernetesClusterName: kubernetesCluster.KubernetesClusterName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get the list of PSMDB components")
	}
	if psmdbComponents == nil || len(psmdbComponents.Versions) < 1 {
		return nil, errors.New("version service returned an empty list for the PSMDB components")
	}

	var backupImage string
	backupComponent, err := DefaultComponent(psmdbComponents.Versions[0].Matrix.Backup)
	if err != nil {
		s.l.Warnf("Cannot get the backup component: %s", err)
	} else {
		backupImage = backupComponent.ImagePath
	}
	_ = backupImage

	if err := s.fillDefaults(ctx, kubernetesCluster, req, psmdbComponents); err != nil {
		return nil, errors.Wrap(err, "cannot create PSMDB cluster")
	}
	if req.Params.Replicaset.StorageClass == "" {
		className, err := s.kubernetesClient.GetDefaultStorageClassName(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get storage classes")
		}
		req.Params.Replicaset.StorageClass = className
	}
	clusterType, err := s.kubernetesClient.GetClusterType(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting cluster type")
	}
	dbCluster, err := kubernetes.DatabaseClusterForPSMDB(req, clusterType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CR specification")
	}
	dbCluster.Spec.SecretsName = fmt.Sprintf(psmdbSecretNameTmpl, req.Name)

	var apiKeyID int64
	if settings.PMMPublicAddress != "" {
		var apiKey string
		apiKeyName := fmt.Sprintf("psmdb-%s-%s-%d", req.KubernetesClusterName, req.Name, rand.Int63())
		apiKeyID, apiKey, err = s.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
		if err != nil {
			return nil, err
		}
		dbCluster.Spec.Monitoring.PMM.PublicAddress = settings.PMMPublicAddress
		dbCluster.Spec.Monitoring.PMM.Login = "api_key"
		dbCluster.Spec.Monitoring.PMM.Password = apiKey
	}
	// TODO: Setup backups

	err = s.kubernetesClient.CreateDatabaseCluster(ctx, dbCluster)
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

func (s PSMDBClusterService) fillDefaults(_ context.Context, _ *models.KubernetesCluster,
	req *dbaasv1beta1.CreatePSMDBClusterRequest, psmdbComponents *dbaasv1beta1.GetPSMDBComponentsResponse,
) error {
	if req.Name != "" {
		r := regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")
		if !r.MatchString(req.Name) {
			return errInvalidClusterName
		}
	}
	if req.Params == nil {
		req.Params = &dbaasv1beta1.PSMDBClusterParams{}
	}

	if req.Params.ClusterSize < 1 {
		req.Params.ClusterSize = psmdbDefaultClusterSize
	}

	if req.Params.Replicaset == nil {
		req.Params.Replicaset = &dbaasv1beta1.PSMDBClusterParams_ReplicaSet{}
	}

	if req.Params.Replicaset.DiskSize == 0 {
		req.Params.Replicaset.DiskSize = psmdbDefaultDiskSize
	}

	if req.Params.Replicaset.ComputeResources == nil {
		req.Params.Replicaset.ComputeResources = &dbaasv1beta1.ComputeResources{
			CpuM:        psmdbDefaultCPUM,
			MemoryBytes: psmdbDefaultMemoryBytes,
		}
	}
	if req.Params.Replicaset.ComputeResources.CpuM == 0 {
		req.Params.Replicaset.ComputeResources.CpuM = psmdbDefaultCPUM
	}
	if req.Params.Replicaset.ComputeResources.MemoryBytes == 0 {
		req.Params.Replicaset.ComputeResources.MemoryBytes = psmdbDefaultMemoryBytes
	}

	psmdbComponent, err := DefaultComponent(psmdbComponents.Versions[0].Matrix.Mongod)
	if err != nil {
		return errors.Wrap(err, "cannot get the recommended MongoDB image name")
	}

	if req.Params.Image == "" {
		req.Params.Image = psmdbComponent.ImagePath
	}

	if req.Name == "" {
		// Image is a string like this: percona/percona-server-mongodb:4.2.12-13
		// We need only the version part to build the cluster name.
		parts := strings.Split(req.Params.Image, ":")

		// This is to generate an unique name.
		uuids := strings.ReplaceAll(uuid.New().String(), "-", "")
		uuids = uuids[len(uuids)-5:]

		req.Name = fmt.Sprintf("psmdb-%s-%s", strings.ReplaceAll(parts[len(parts)-1], ".", "-"), uuids)
		if len(req.Name) > 22 { // Kubernetes limitation
			req.Name = req.Name[:21]
		}
	}
	//}

	return nil
}

// UpdatePSMDBCluster updates PSMDB cluster.
func (s PSMDBClusterService) UpdatePSMDBCluster(ctx context.Context, req *dbaasv1beta1.UpdatePSMDBClusterRequest) (*dbaasv1beta1.UpdatePSMDBClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}
	if err := s.kubernetesClient.SetKubeconfig(ctx, kubernetesCluster.KubeConfig); err != nil {
		return nil, errors.Wrap(err, "failed creating kubernetes client")
	}
	dbCluster, err := s.kubernetesClient.GetDatabaseCluster(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	err = kubernetes.UpdatePatchForPSMDB(dbCluster, req)
	if err != nil {
		return nil, err
	}

	err = s.kubernetesClient.PatchDatabaseCluster(ctx, dbCluster)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UpdatePSMDBClusterResponse{}, nil
}

// GetPSMDBClusterResources returns expected resources to be consumed by the cluster.
func (s PSMDBClusterService) GetPSMDBClusterResources(ctx context.Context, req *dbaasv1beta1.GetPSMDBClusterResourcesRequest) (*dbaasv1beta1.GetPSMDBClusterResourcesResponse, error) { //nolint:lll
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
