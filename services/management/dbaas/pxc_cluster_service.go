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

// Package dbaas contains all logic related to dbaas services.
package dbaas

import (
	"context"
	"fmt"
	"math/rand"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// PXCClustersService implements PXCClusterServer methods.
type PXCClustersService struct {
	db                   *reform.DB
	l                    *logrus.Entry
	controllerClient     dbaasClient
	grafanaClient        grafanaClient
	versionServiceClient versionService

	dbaasv1beta1.UnimplementedPXCClustersServer
}

// NewPXCClusterService creates PXC Service.
func NewPXCClusterService(db *reform.DB, controllerClient dbaasClient, grafanaClient grafanaClient, versionServiceClient versionService) dbaasv1beta1.PXCClustersServer {
	l := logrus.WithField("component", "pxc_cluster")
	return &PXCClustersService{
		db:                   db,
		l:                    l,
		controllerClient:     controllerClient,
		grafanaClient:        grafanaClient,
		versionServiceClient: versionServiceClient,
	}
}

// GetPXCClusterCredentials returns a PXC cluster credentials.
func (s PXCClustersService) GetPXCClusterCredentials(ctx context.Context, req *dbaasv1beta1.GetPXCClusterCredentialsRequest) (*dbaasv1beta1.GetPXCClusterCredentialsResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := &dbaascontrollerv1beta1.GetPXCClusterCredentialsRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
	}

	cluster, err := s.controllerClient.GetPXCClusterCredentials(ctx, in)
	if err != nil {
		return nil, err
	}

	_ = kubernetesCluster
	resp := dbaasv1beta1.GetPXCClusterCredentialsResponse{
		ConnectionCredentials: &dbaasv1beta1.PXCClusterConnectionCredentials{
			Username: cluster.Credentials.Username,
			Password: cluster.Credentials.Password,
			Host:     cluster.Credentials.Host,
			Port:     cluster.Credentials.Port,
		},
	}

	return &resp, nil
}

// CreatePXCCluster creates PXC cluster with given parameters.
//nolint:dupl
func (s PXCClustersService) CreatePXCCluster(ctx context.Context, req *dbaasv1beta1.CreatePXCClusterRequest) (*dbaasv1beta1.CreatePXCClusterResponse, error) {
	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	// Check if one and only one of proxies is set.
	if (req.Params.Proxysql != nil) == (req.Params.Haproxy != nil) {
		return nil, errors.New("pxc cluster must have one and only one proxy type defined")
	}

	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	var pmmParams *dbaascontrollerv1beta1.PMMParams
	var apiKeyID int64
	if settings.PMMPublicAddress != "" {
		var apiKey string
		apiKeyName := fmt.Sprintf("pxc-%s-%s-%d", req.KubernetesClusterName, req.Name, rand.Int63())
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

	in := dbaascontrollerv1beta1.CreatePXCClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
		Pmm:  pmmParams,
		Params: &dbaascontrollerv1beta1.PXCClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Pxc: &dbaascontrollerv1beta1.PXCClusterParams_PXC{
				Image:            req.Params.Pxc.Image,
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{},
				DiskSize:         req.Params.Pxc.DiskSize,
			},
			VersionServiceUrl: s.versionServiceClient.GetVersionServiceURL(),
		},
		Expose: req.Expose,
	}
	if req.Params.Proxysql != nil {
		in.Params.Proxysql = &dbaascontrollerv1beta1.PXCClusterParams_ProxySQL{
			Image:            req.Params.Proxysql.Image,
			ComputeResources: &dbaascontrollerv1beta1.ComputeResources{},
			DiskSize:         req.Params.Proxysql.DiskSize,
		}
		if req.Params.Proxysql.ComputeResources != nil {
			in.Params.Proxysql.ComputeResources = &dbaascontrollerv1beta1.ComputeResources{
				CpuM:        req.Params.Proxysql.ComputeResources.CpuM,
				MemoryBytes: req.Params.Proxysql.ComputeResources.MemoryBytes,
			}
		}
	} else {
		in.Params.Haproxy = &dbaascontrollerv1beta1.PXCClusterParams_HAProxy{
			Image:            req.Params.Haproxy.Image,
			ComputeResources: &dbaascontrollerv1beta1.ComputeResources{},
		}
		if req.Params.Haproxy.ComputeResources != nil {
			in.Params.Haproxy.ComputeResources = &dbaascontrollerv1beta1.ComputeResources{
				CpuM:        req.Params.Haproxy.ComputeResources.CpuM,
				MemoryBytes: req.Params.Haproxy.ComputeResources.MemoryBytes,
			}
		}
	}

	if req.Params.Pxc.ComputeResources != nil {
		in.Params.Pxc.ComputeResources = &dbaascontrollerv1beta1.ComputeResources{
			CpuM:        req.Params.Pxc.ComputeResources.CpuM,
			MemoryBytes: req.Params.Pxc.ComputeResources.MemoryBytes,
		}
	}

	_, err = s.controllerClient.CreatePXCCluster(ctx, &in)
	if err != nil {
		if apiKeyID != 0 {
			e := s.grafanaClient.DeleteAPIKeyByID(ctx, apiKeyID)
			if e != nil {
				s.l.Warnf("couldn't delete created API Key %v: %s", apiKeyID, e)
			}
		}
		return nil, err
	}

	return &dbaasv1beta1.CreatePXCClusterResponse{}, nil
}

// UpdatePXCCluster updates PXC cluster.
//nolint:dupl
func (s PXCClustersService) UpdatePXCCluster(ctx context.Context, req *dbaasv1beta1.UpdatePXCClusterRequest) (*dbaasv1beta1.UpdatePXCClusterResponse, error) {
	kubernetesCluster, err := models.FindKubernetesClusterByName(s.db.Querier, req.KubernetesClusterName)
	if err != nil {
		return nil, err
	}

	in := dbaascontrollerv1beta1.UpdatePXCClusterRequest{
		KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
			Kubeconfig: kubernetesCluster.KubeConfig,
		},
		Name: req.Name,
	}

	if req.Params != nil {
		if req.Params.Suspend && req.Params.Resume {
			return nil, status.Error(codes.InvalidArgument, "resume and suspend cannot be set together")
		}

		// Check if only one or none of proxies is set.
		if (req.Params.Proxysql != nil) && (req.Params.Haproxy != nil) {
			return nil, errors.New("can't update both proxies, only one is in use")
		}

		in.Params = &dbaascontrollerv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
			ClusterSize: req.Params.ClusterSize,
			Suspend:     req.Params.Suspend,
			Resume:      req.Params.Resume,
		}

		if req.Params.Pxc != nil && req.Params.Pxc.ComputeResources != nil {
			in.Params.Pxc = &dbaascontrollerv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_PXC{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Pxc.ComputeResources.CpuM,
					MemoryBytes: req.Params.Pxc.ComputeResources.MemoryBytes,
				},
			}
			in.Params.Pxc.Image = req.Params.Pxc.Image
		}

		if req.Params.Proxysql != nil && req.Params.Proxysql.ComputeResources != nil {
			in.Params.Proxysql = &dbaascontrollerv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_ProxySQL{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Proxysql.ComputeResources.CpuM,
					MemoryBytes: req.Params.Proxysql.ComputeResources.MemoryBytes,
				},
			}
		}

		if req.Params.Haproxy != nil && req.Params.Haproxy.ComputeResources != nil {
			in.Params.Haproxy = &dbaascontrollerv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_HAProxy{
				ComputeResources: &dbaascontrollerv1beta1.ComputeResources{
					CpuM:        req.Params.Haproxy.ComputeResources.CpuM,
					MemoryBytes: req.Params.Haproxy.ComputeResources.MemoryBytes,
				},
			}
		}
	}

	_, err = s.controllerClient.UpdatePXCCluster(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &dbaasv1beta1.UpdatePXCClusterResponse{}, nil
}

// GetPXCClusterResources returns expected resources to be consumed by the cluster.
func (s PXCClustersService) GetPXCClusterResources(ctx context.Context, req *dbaasv1beta1.GetPXCClusterResourcesRequest) (*dbaasv1beta1.GetPXCClusterResourcesResponse, error) {
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
