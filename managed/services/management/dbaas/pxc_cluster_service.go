// Copyright (C) 2023 Percona LLC
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
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

const (
	pxcDefaultClusterSize   = 3
	pxcDefaultCPUM          = 1000
	pxcDefaultMemoryBytes   = 2000000000
	pxcDefaultDiskSize      = 25000000000
	proxyDefaultCPUM        = 500
	proxyDefaultMemoryBytes = 500000000
	haProxyTemplate         = "percona/percona-xtradb-cluster-operator:%s-haproxy"
	proxySQLTemplate        = "percona/percona-xtradb-cluster-operator:%s-proxysql"
)

var errInvalidClusterName = errors.New("invalid cluster name. It must start with a letter and have only letters, numbers and -")

// PXCClustersService implements PXCClusterServer methods.
type PXCClustersService struct {
	db                *reform.DB
	l                 *logrus.Entry
	grafanaClient     grafanaClient
	componentsService componentsService
	kubeStorage       *KubeStorage
	versionServiceURL string

	dbaasv1beta1.UnimplementedPXCClustersServer
}

// NewPXCClusterService creates PXC Service.
func NewPXCClusterService(db *reform.DB, grafanaClient grafanaClient, componentsService componentsService, //nolint:ireturn
	versionServiceURL string,
) dbaasv1beta1.PXCClustersServer {
	l := logrus.WithField("component", "pxc_cluster")
	return &PXCClustersService{
		db:                db,
		l:                 l,
		grafanaClient:     grafanaClient,
		versionServiceURL: versionServiceURL,
		componentsService: componentsService,
		kubeStorage:       NewKubeStorage(db),
	}
}

// GetPXCClusterCredentials returns a PXC cluster credentials.
func (s PXCClustersService) GetPXCClusterCredentials(ctx context.Context, req *dbaasv1beta1.GetPXCClusterCredentialsRequest) (*dbaasv1beta1.GetPXCClusterCredentialsResponse, error) { //nolint:lll
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

	resp := dbaasv1beta1.GetPXCClusterCredentialsResponse{
		ConnectionCredentials: &dbaasv1beta1.PXCClusterConnectionCredentials{
			Username: "root",
			Password: string(secret.Data["root"]),
			Host:     dbCluster.Status.Host,
			Port:     3306,
		},
	}

	return &resp, nil
}

// CreatePXCCluster creates PXC cluster with given parameters.
//
//nolint:dupl
func (s PXCClustersService) CreatePXCCluster(ctx context.Context, req *dbaasv1beta1.CreatePXCClusterRequest) (*dbaasv1beta1.CreatePXCClusterResponse, error) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, errInvalidClusterName
	}

	if req.Params == nil {
		req.Params = &dbaasv1beta1.PXCClusterParams{}
	}
	// Check if one and only one of proxies is set.
	if req.Params.Proxysql != nil && req.Params.Haproxy != nil {
		return nil, errors.New("pxc cluster must have one and only one proxy type defined")
	}

	kubeClient, err := s.kubeStorage.GetOrSetClient(req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	if err := s.fillDefaults(ctx, req.KubernetesClusterName, req, kubeClient); err != nil {
		return nil, errors.Wrap(err, "cannot create pxc cluster")
	}

	if req.Params.Pxc.StorageClass == "" {
		className, err := kubeClient.GetDefaultStorageClassName(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get storage classes")
		}
		req.Params.Pxc.StorageClass = className
	}
	clusterType, err := kubeClient.GetClusterType(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting cluster type")
	}
	backupLocation, err := s.getBackupLocation(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting backup location")
	}
	dbCluster, dbRestore, err := kubernetes.DatabaseClusterForPXC(req, clusterType, backupLocation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CR specification")
	}

	secrets, err := generatePasswords(map[string][]byte{
		"root":         {},
		"xtrabackup":   {},
		"monitor":      {},
		"clustercheck": {},
		"proxyadmin":   {},
		"operator":     {},
		"replication":  {},
	})
	if err != nil {
		return nil, err
	}
	var apiKeyID int64
	if settings.PMMPublicAddress != "" {
		var apiKey string
		apiKeyName := fmt.Sprintf("pxc-%s-%s-%d", req.KubernetesClusterName, req.Name, rand.Int63()) //nolint:gosec
		apiKeyID, apiKey, err = s.grafanaClient.CreateAdminAPIKey(ctx, apiKeyName)
		if err != nil {
			return nil, err
		}
		dbCluster.Spec.Monitoring.PMM.PublicAddress = settings.PMMPublicAddress
		dbCluster.Spec.Monitoring.PMM.Login = "api_key"
		dbCluster.Spec.Monitoring.PMM.Image = getPMMClientImage() //nolint:contextcheck

		secrets["pmmserver"] = []byte(apiKey)
	}
	if req.Params.Restore == nil || (req.Params.Restore != nil && req.Params.Restore.SecretsName == "") {
		err = kubeClient.CreatePMMSecret(dbCluster.Spec.SecretsName, secrets)
		if err != nil {
			return nil, err
		}
	}
	err = kubeClient.CreateDatabaseCluster(dbCluster)
	if err != nil {
		if apiKeyID != 0 {
			e := s.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID)
			if e != nil {
				s.l.Warnf("couldn't delete created API Key %v: %s", apiKeyID, e)
			}
		}
	}
	if req.Params.Backup != nil || req.Params.Restore != nil && backupLocation != nil {
		secretsName := fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName)
		secrets := kubernetes.SecretForBackup(backupLocation)
		if err := kubeClient.CreatePMMSecret(secretsName, secrets); err != nil {
			return nil, errors.Wrap(err, "failed to create a secret")
		}
	}
	if dbRestore != nil {
		if err := kubeClient.CreateRestore(dbRestore); err != nil {
			return nil, err
		}
	}
	return &dbaasv1beta1.CreatePXCClusterResponse{}, nil
}

//nolint:cyclop
func (s PXCClustersService) fillDefaults(ctx context.Context, kubernetesClusterName string,
	req *dbaasv1beta1.CreatePXCClusterRequest, kubeClient kubernetesClient,
) error {
	if req.Name != "" {
		r := regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")
		if !r.MatchString(req.Name) {
			return errInvalidClusterName
		}
	}
	if req.Params == nil {
		req.Params = &dbaasv1beta1.PXCClusterParams{}
	}

	if req.Params.ClusterSize < 1 {
		req.Params.ClusterSize = pxcDefaultClusterSize
	}

	if req.Params.Pxc == nil {
		req.Params.Pxc = &dbaasv1beta1.PXCClusterParams_PXC{}
	}

	if req.Params.Pxc.DiskSize == 0 {
		req.Params.Pxc.DiskSize = pxcDefaultDiskSize
	}

	if req.Params.Pxc.ComputeResources == nil {
		req.Params.Pxc.ComputeResources = &dbaasv1beta1.ComputeResources{
			CpuM:        pxcDefaultCPUM,
			MemoryBytes: pxcDefaultMemoryBytes,
		}
	}
	if req.Params.Pxc.ComputeResources.CpuM == 0 {
		req.Params.Pxc.ComputeResources.CpuM = pxcDefaultCPUM
	}
	if req.Params.Pxc.ComputeResources.MemoryBytes == 0 {
		req.Params.Pxc.ComputeResources.MemoryBytes = pxcDefaultMemoryBytes
	}

	// If none of them was specified, use HAProxy by default.
	if req.Params.Proxysql == nil && req.Params.Haproxy == nil {
		req.Params.Haproxy = &dbaasv1beta1.PXCClusterParams_HAProxy{
			ComputeResources: &dbaasv1beta1.ComputeResources{
				CpuM:        proxyDefaultCPUM,
				MemoryBytes: proxyDefaultMemoryBytes,
			},
		}
	}

	if req.Params.Haproxy != nil {
		if req.Params.Haproxy.ComputeResources == nil {
			req.Params.Haproxy.ComputeResources = &dbaasv1beta1.ComputeResources{
				CpuM:        proxyDefaultCPUM,
				MemoryBytes: proxyDefaultMemoryBytes,
			}
		}
		if req.Params.Haproxy.ComputeResources.CpuM == 0 {
			req.Params.Haproxy.ComputeResources.CpuM = proxyDefaultCPUM
		}
		if req.Params.Haproxy.ComputeResources.MemoryBytes == 0 {
			req.Params.Haproxy.ComputeResources.MemoryBytes = proxyDefaultMemoryBytes
		}
		if req.Params.Haproxy.Image == "" {
			// PXC operator requires to specify HAproxy image
			// It uses default operator distribution based on version
			// following the template operatorimage:version-haproxy
			version, err := kubeClient.GetPXCOperatorVersion(ctx)
			if err != nil {
				return err
			}
			req.Params.Haproxy.Image = fmt.Sprintf(haProxyTemplate, version)
		}
	}

	if req.Params.Proxysql != nil {
		if req.Params.Proxysql.ComputeResources == nil {
			req.Params.Proxysql.ComputeResources = &dbaasv1beta1.ComputeResources{
				CpuM:        proxyDefaultCPUM,
				MemoryBytes: proxyDefaultMemoryBytes,
			}
		}
		if req.Params.Proxysql.ComputeResources.CpuM == 0 {
			req.Params.Proxysql.ComputeResources.CpuM = proxyDefaultCPUM
		}
		if req.Params.Proxysql.ComputeResources.MemoryBytes == 0 {
			req.Params.Proxysql.ComputeResources.MemoryBytes = proxyDefaultMemoryBytes
		}
		if req.Params.Proxysql.Image == "" {
			// PXC operator requires to specify ProxySQL image
			// It uses default operator distribution based on version
			// following the template operatorimage:version-proxysql
			version, err := kubeClient.GetPXCOperatorVersion(ctx)
			if err != nil {
				return err
			}
			req.Params.Proxysql.Image = fmt.Sprintf(proxySQLTemplate, version)
		}
	}

	// Only call the version service if it is really needed.
	if req.Name == "" || req.Params.Pxc.Image == "" {
		pxcComponents, err := s.componentsService.GetPXCComponents(ctx, &dbaasv1beta1.GetPXCComponentsRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		if err != nil {
			return errors.New("cannot get the list of PXC components")
		}

		component, err := DefaultComponent(pxcComponents.Versions[0].Matrix.Pxc)
		if err != nil {
			return errors.Wrap(err, "cannot get the recommended PXC image name")
		}

		if req.Params.Pxc.Image == "" {
			req.Params.Pxc.Image = component.ImagePath
		}

		if req.Name == "" {
			// Image is a string like this: percona/percona-server-mongodb:4.2.12-13
			// We need only the version part to build the cluster name.
			parts := strings.Split(req.Params.Pxc.Image, ":")
			req.Name = fmt.Sprintf("pxc-%s-%04d", strings.ReplaceAll(parts[len(parts)-1], ".", "-"), rand.Int63n(9999)) //nolint:gosec
			if len(req.Name) > 22 {                                                                                     // Kubernetes limitation
				req.Name = req.Name[:21]
			}
		}
	}

	return nil
}

// UpdatePXCCluster updates PXC cluster.
//
//nolint:dupl
func (s PXCClustersService) UpdatePXCCluster(ctx context.Context, req *dbaasv1beta1.UpdatePXCClusterRequest) (*dbaasv1beta1.UpdatePXCClusterResponse, error) {
	if (req.Params.Proxysql != nil) && (req.Params.Haproxy != nil) {
		return nil, errors.New("can't update both proxies, only one is in use")
	}
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
	err = kubernetes.UpdatePatchForPXC(dbCluster, req, clusterType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CR specification")
	}

	err = kubeClient.PatchDatabaseCluster(dbCluster)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UpdatePXCClusterResponse{}, nil
}

// GetPXCClusterResources returns expected resources to be consumed by the cluster.
func (s PXCClustersService) GetPXCClusterResources(_ context.Context, req *dbaasv1beta1.GetPXCClusterResourcesRequest) (*dbaasv1beta1.GetPXCClusterResourcesResponse, error) { //nolint:lll
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	clusterSize := uint64(req.Params.ClusterSize)
	var proxyComputeResources *dbaasv1beta1.ComputeResources
	var disk uint64
	if req.Params.Proxysql != nil {
		disk = uint64(req.Params.Proxysql.DiskSize) * clusterSize
		proxyComputeResources = req.Params.Proxysql.ComputeResources
	} else {
		proxyComputeResources = req.Params.Haproxy.ComputeResources
	}
	memory := uint64(req.Params.Pxc.ComputeResources.MemoryBytes+proxyComputeResources.MemoryBytes) * clusterSize
	cpu := uint64(req.Params.Pxc.ComputeResources.CpuM+proxyComputeResources.CpuM) * clusterSize
	disk += uint64(req.Params.Pxc.DiskSize) * clusterSize

	if settings.PMMPublicAddress != "" {
		memory += 1000000000 * clusterSize
		cpu += 1000 * clusterSize
	}

	return &dbaasv1beta1.GetPXCClusterResourcesResponse{
		Expected: &dbaasv1beta1.Resources{
			CpuM:        cpu,
			MemoryBytes: memory,
			DiskSize:    disk,
		},
	}, nil
}

func (s PXCClustersService) getBackupLocation(req *dbaasv1beta1.CreatePXCClusterRequest) (*models.BackupLocation, error) {
	if req.Params != nil && req.Params.Backup != nil && req.Params.Backup.LocationId != "" {
		return models.FindBackupLocationByID(s.db.Querier, req.Params.Backup.LocationId)
	}
	if req.Params != nil && req.Params.Restore != nil && req.Params.Restore.LocationId != "" {
		return models.FindBackupLocationByID(s.db.Querier, req.Params.Restore.LocationId)
	}
	return nil, nil //nolint:nilnil
}
