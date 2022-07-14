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
	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
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
	controllerClient  dbaasClient
	grafanaClient     grafanaClient
	componentsService componentsService
	versionServiceURL string

	dbaasv1beta1.UnimplementedPSMDBClustersServer
}

// NewPSMDBClusterService creates PSMDB Service.
func NewPSMDBClusterService(db *reform.DB, dbaasClient dbaasClient, grafanaClient grafanaClient,
	componentsService componentsService, versionServiceURL string,
) dbaasv1beta1.PSMDBClustersServer {
	l := logrus.WithField("component", "psmdb_cluster")
	return &PSMDBClusterService{
		db:                db,
		l:                 l,
		controllerClient:  dbaasClient,
		grafanaClient:     grafanaClient,
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

	if err := s.fillDefaults(ctx, kubernetesCluster, req, psmdbComponents); err != nil {
		return nil, errors.Wrap(err, "cannot create PSMDB cluster")
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
			BackupImage: backupImage,
			ClusterSize: req.Params.ClusterSize,
			Replicaset: &dbaascontrollerv1beta1.PSMDBClusterParams_ReplicaSet{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Replicaset.ComputeResources.CpuM,
					MemoryBytes: req.Params.Replicaset.ComputeResources.MemoryBytes,
				},
				DiskSize: req.Params.Replicaset.DiskSize,
			},
			VersionServiceUrl: s.versionServiceURL,
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

func (s PSMDBClusterService) fillDefaults(ctx context.Context, kubernetesCluster *models.KubernetesCluster,
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
