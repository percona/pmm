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

// Package dbaas contains all APIs related to DBaaS.
//
//nolint:lll
package dbaas

import (
	"context"

	goversion "github.com/hashicorp/go-version"
	olmalpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/version"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
)

type dbaasClient interface {
	// Connect connects the client to dbaas-controller API.
	Connect(ctx context.Context) error
	// Disconnect disconnects the client from dbaas-controller API.
	Disconnect() error
	// GetLogs gets logs out of cluster containers and events out of pods.
	GetLogs(ctx context.Context, in *controllerv1beta1.GetLogsRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetLogsResponse, error)
	// GetResources returns all and available resources of a Kubernetes cluster.
	GetResources(ctx context.Context, in *controllerv1beta1.GetResourcesRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetResourcesResponse, error)
	InstallPXCOperator(ctx context.Context, in *controllerv1beta1.InstallPXCOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPXCOperatorResponse, error)
	// InstallPSMDBOperator installs kubernetes psmdb operator.
	InstallPSMDBOperator(ctx context.Context, in *controllerv1beta1.InstallPSMDBOperatorRequest, opts ...grpc.CallOption) (*controllerv1beta1.InstallPSMDBOperatorResponse, error)
	// StartMonitoring sets up victoria metrics operator to monitor kubernetes cluster.
	StartMonitoring(ctx context.Context, in *controllerv1beta1.StartMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StartMonitoringResponse, error)
	// StopMonitoring removes victoria metrics operator from the cluster.
	StopMonitoring(ctx context.Context, in *controllerv1beta1.StopMonitoringRequest, opts ...grpc.CallOption) (*controllerv1beta1.StopMonitoringResponse, error)
	// GetKubeConfig gets inluster config and converts it to kubeConfig
	GetKubeConfig(ctx context.Context, in *controllerv1beta1.GetKubeconfigRequest, opts ...grpc.CallOption) (*controllerv1beta1.GetKubeconfigResponse, error)
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

type kubernetesClient interface { //nolint:interfacebloat
	SetKubeconfig(string) error
	ListDatabaseClusters(context.Context) (*dbaasv1.DatabaseClusterList, error)
	GetDatabaseCluster(context.Context, string) (*dbaasv1.DatabaseCluster, error)
	RestartDatabaseCluster(context.Context, string) error
	PatchDatabaseCluster(*dbaasv1.DatabaseCluster) error
	CreateDatabaseCluster(*dbaasv1.DatabaseCluster) error
	DeleteDatabaseCluster(context.Context, string) error
	GetDefaultStorageClassName(context.Context) (string, error)
	GetPXCOperatorVersion(context.Context) (string, error)
	GetPSMDBOperatorVersion(context.Context) (string, error)
	GetSecret(context.Context, string) (*corev1.Secret, error)
	GetClusterType(context.Context) (kubernetes.ClusterType, error)
	CreatePMMSecret(string, map[string][]byte) error
	GetAllClusterResources(context.Context, kubernetes.ClusterType, *corev1.PersistentVolumeList) (uint64, uint64, uint64, error)
	GetConsumedCPUAndMemory(context.Context, string) (uint64, uint64, error)
	GetConsumedDiskBytes(context.Context, kubernetes.ClusterType, *corev1.PersistentVolumeList) (uint64, error)
	GetPersistentVolumes(ctx context.Context) (*corev1.PersistentVolumeList, error)
	GetStorageClasses(ctx context.Context) (*storagev1.StorageClassList, error)
	CreateRestore(*dbaasv1.DatabaseClusterRestore) error
	ListSecrets(context.Context) (*corev1.SecretList, error)
	// InstallOLMOperator installs the OLM in the Kubernetes cluster.
	InstallOLMOperator(ctx context.Context) error
	// InstallOperator installs an operator via OLM.
	InstallOperator(ctx context.Context, req kubernetes.InstallOperatorRequest) error
	// ListSubscriptions all the subscriptions in the namespace.
	ListSubscriptions(ctx context.Context, namespace string) (*olmalpha1.SubscriptionList, error)
	// UpgradeOperator upgrades an operator to the next available version.
	UpgradeOperator(ctx context.Context, namespace, name string) error
	// GetServerVersion returns server version
	GetServerVersion() (*version.Info, error)
	ListTemplates(ctx context.Context, engine, namespace string) ([]*dbaasv1beta1.Template, error)
}

type kubeStorageManager interface {
	GetOrSetClient(name string) (kubernetesClient, error)
	DeleteClient(name string) error
}
