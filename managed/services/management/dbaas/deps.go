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

// Package dbaas contains all APIs related to DBaaS.
package dbaas

import (
	"context"

	goversion "github.com/hashicorp/go-version"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"google.golang.org/grpc"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
)

//go:generate $PMM_RELEASE_PATH/mockery -name=dbaasClient -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=versionService -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=grafanaClient -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=componentsService -case=snake -inpkg -testonly
//go:generate $PMM_RELEASE_PATH/mockery -name=dbClusterSynchronizer -case=snake -inpkg -testonly

type dbaasClient interface {
	// Connect connects the client to dbaas-controller API.
	Connect(ctx context.Context) error
	// Disconnect disconnects the client from dbaas-controller API.
	Disconnect() error
	// CheckKubernetesClusterConnection checks connection to Kubernetes cluster and returns statuses of the cluster and operators.
	CheckKubernetesClusterConnection(ctx context.Context, kubeConfig string) (*controllerv1beta1.CheckKubernetesClusterConnectionResponse, error)
	// ListPXCClusters returns a list of PXC clusters.
	ListPXCClusters(ctx context.Context, in *controllerv1beta1.ListPXCClustersRequest, opts ...grpc.CallOption) (*controllerv1beta1.ListPXCClustersResponse, error)
	// CreatePXCCluster creates a new PXC cluster.
	CreatePXCCluster(ctx context.Context, in *controllerv1beta1.CreatePXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.CreatePXCClusterResponse, error)
	// UpdatePXCCluster updates existing PXC cluster.
	UpdatePXCCluster(ctx context.Context, in *controllerv1beta1.UpdatePXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.UpdatePXCClusterResponse, error)
	// DeletePXCCluster deletes PXC cluster.
	DeletePXCCluster(ctx context.Context, in *controllerv1beta1.DeletePXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.DeletePXCClusterResponse, error)
	// RestartPXCCluster restarts PXC cluster.
	RestartPXCCluster(ctx context.Context, in *controllerv1beta1.RestartPXCClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.RestartPXCClusterResponse, error)
	// GetPXCCluster gets a PXC cluster.
	GetPXCCluster(ctx context.Context, kubeconfig, name string) (*controllerv1beta1.GetPXCClusterResponse, error)
	// GetPXCClusterCredentials returns an PXC cluster credentials.
	GetPXCClusterCredentials(ctx context.Context, in *controllerv1beta1.GetPXCClusterCredentialsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetPXCClusterCredentialsResponse, error)
	// ListPSMDBClusters returns a list of PSMDB clusters.
	ListPSMDBClusters(ctx context.Context, in *controllerv1beta1.ListPSMDBClustersRequest, opts ...grpc.CallOption) (*controllerv1beta1.ListPSMDBClustersResponse, error)
	// CreatePSMDBCluster creates a new PSMDB cluster.
	CreatePSMDBCluster(ctx context.Context, in *controllerv1beta1.CreatePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.CreatePSMDBClusterResponse, error)
	// UpdatePSMDBCluster updates existing PSMDB cluster.
	UpdatePSMDBCluster(ctx context.Context, in *controllerv1beta1.UpdatePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.UpdatePSMDBClusterResponse, error)
	// DeletePSMDBCluster deletes PSMDB cluster.
	DeletePSMDBCluster(ctx context.Context, in *controllerv1beta1.DeletePSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.DeletePSMDBClusterResponse, error)
	// RestartPSMDBCluster restarts PSMDB cluster.
	RestartPSMDBCluster(ctx context.Context, in *controllerv1beta1.RestartPSMDBClusterRequest, opts ...grpc.CallOption) (*controllerv1beta1.RestartPSMDBClusterResponse, error)
	// GetPSMDBCluster gets a PSMDB cluster.
	GetPSMDBCluster(ctx context.Context, kubeconfig, name string) (*controllerv1beta1.GetPSMDBClusterResponse, error)
	// GetPSMDBClusterCredentials gets a PSMDB cluster credentials.
	GetPSMDBClusterCredentials(ctx context.Context, in *controllerv1beta1.GetPSMDBClusterCredentialsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetPSMDBClusterCredentialsResponse, error)
	// GetLogs gets logs out of cluster containers and events out of pods.
	GetLogs(ctx context.Context, in *controllerv1beta1.GetLogsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetLogsResponse, error)
	// GetResources returns all and available resources of a Kubernetes cluster.
	GetResources(ctx context.Context, in *controllerv1beta1.GetResourcesRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetResourcesResponse, error)
	// InstallPXCOperator installs kubernetes pxc operator.
	InstallPXCOperator(ctx context.Context, in *controllerv1beta1.InstallPXCOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPXCOperatorResponse, error)
	// InstallPSMDBOperator installs kubernetes psmdb operator.
	InstallPSMDBOperator(ctx context.Context, in *controllerv1beta1.InstallPSMDBOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPSMDBOperatorResponse, error)
	// StartMonitoring sets up victoria metrics operator to monitor kubernetes cluster.
	StartMonitoring(ctx context.Context, in *controllerv1beta1.StartMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StartMonitoringResponse, error)
	// StopMonitoring removes victoria metrics operator from the cluster.
	StopMonitoring(ctx context.Context, in *controllerv1beta1.StopMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StopMonitoringResponse, error)
}

type versionService interface {
	// Matrix calls version service with given params and returns components matrix.
	Matrix(ctx context.Context, params componentsParams) (*VersionServiceResponse, error)
	// GetNextDatabaseImage returns image of the dabase version that is a direct successor of currently installed version.
	GetNextDatabaseImage(ctx context.Context, operatorType, operatorVersion, installedDBVersion string) (string, error)
	// GetVersionServiceURL version service used by version service client.
	GetVersionServiceURL() string
	// IsDatabaseVersionSupportedByOperator returns false and err when request to version service fails. Otherwise returns boolen telling
	// if given database version is supported by given operator version, error is nil in that case.
	IsDatabaseVersionSupportedByOperator(ctx context.Context, operatorType, operatorVersion, databaseVersion string) (bool, error)
	// SupportedOperatorVersionsList returns list of operators versions supported by certain PMM version.
	SupportedOperatorVersionsList(ctx context.Context, pmmVersion string) (map[string][]string, error)
	// LatestOperatorVersion returns latest operators versions available based on given params.
	LatestOperatorVersion(ctx context.Context, pmmVersion string) (latestPXCOperatorVersion, latestPSMDBOperatorVersion *goversion.Version, err error)
	// NextOperatorVersion returns operator versions that is a direct successor of currently installed one.
	// Compatibility with PMM is not taken into account.
	NextOperatorVersion(ctx context.Context, operatorType, installedVersion string) (nextOperatorVersion *goversion.Version, err error)
}

// grafanaClient is a subset of methods of grafana.Client used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type grafanaClient interface {
	CreateAdminAPIKey(ctx context.Context, name string) (int64, string, error)
	DeleteAPIKeysWithPrefix(ctx context.Context, name string) error
	DeleteAPIKeyByID(ctx context.Context, id int64) error
}

type componentsService interface {
	GetPSMDBComponents(context.Context, *dbaasv1beta1.GetPSMDBComponentsRequest) (*dbaasv1beta1.GetPSMDBComponentsResponse, error)
	GetPXCComponents(context.Context, *dbaasv1beta1.GetPXCComponentsRequest) (*dbaasv1beta1.GetPXCComponentsResponse, error)
	ChangePSMDBComponents(context.Context, *dbaasv1beta1.ChangePSMDBComponentsRequest) (*dbaasv1beta1.ChangePSMDBComponentsResponse, error)
	ChangePXCComponents(context.Context, *dbaasv1beta1.ChangePXCComponentsRequest) (*dbaasv1beta1.ChangePXCComponentsResponse, error)
	CheckForOperatorUpdate(context.Context, *dbaasv1beta1.CheckForOperatorUpdateRequest) (*dbaasv1beta1.CheckForOperatorUpdateResponse, error)
	InstallOperator(context.Context, *dbaasv1beta1.InstallOperatorRequest) (*dbaasv1beta1.InstallOperatorResponse, error)
}

type dbClusterSynchronizer interface {
	WatchDBClusterDeletion(cluster *models.DBCluster)
	SyncDBClusters(ctx context.Context, kubernetesCluster *models.KubernetesCluster) error
	RemoveDBCluster(cluster *models.DBCluster) error
}
