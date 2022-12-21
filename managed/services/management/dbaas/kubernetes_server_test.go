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
	"testing"

	"github.com/google/uuid"
	controllerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/dbaas/kubernetes"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	pmmversion "github.com/percona/pmm/version"
)

func TestKubernetesServer(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, ks dbaasv1beta1.KubernetesServer, dbaasClient *mockDbaasClient, kubernetesClient *mockKubernetesClient, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient = &mockDbaasClient{}
		kubernetesClient = &mockKubernetesClient{}
		grafanaClient := &mockGrafanaClient{}

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
			require.NoError(t, sqlDB.Close())
		}
		versionService := NewVersionServiceClient("https://check-dev.percona.com/versions/v1")
		ks = NewKubernetesServer(db, dbaasClient, kubernetesClient, versionService, grafanaClient)
		return
	}

	// This is for local testing. When running local tests, if pmmversion.PMMVersion is empty
	// these lines in kubernetes_server.go will throw an error and tests won't finish.
	//
	//     pmmVersion, err := goversion.NewVersion(pmmversion.PMMVersion)
	//     if err != nil {
	//     	return nil, status.Error(codes.Internal, err.Error())
	//     }
	//
	if pmmversion.PMMVersion == "" {
		pmmversion.PMMVersion = "2.30.0"
	}

	t.Run("Basic", func(t *testing.T) {
		ctx, ks, dc, kubernetesClient, teardown := setup(t)
		kubernetesClient.On("SetKubeconfig", mock.Anything).Return(nil)
		kubernetesClient.On("SetKubeconfig", mock.Anything).Return(nil)
		defer teardown(t)
		kubeconfig := "preferences: {}\n"

		clusters, err := ks.ListKubernetesClusters(ctx, &dbaasv1beta1.ListKubernetesClustersRequest{})
		require.NoError(t, err)
		require.Empty(t, clusters.KubernetesClusters)
		dc.On("CheckKubernetesClusterConnection", ctx, kubeconfig).Return(&controllerv1beta1.CheckKubernetesClusterConnectionResponse{
			Operators: &controllerv1beta1.Operators{
				PxcOperatorVersion:   "",
				PsmdbOperatorVersion: onePointEight,
			},
			Status: controllerv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK,
		}, nil)

		dc.On("InstallOLMOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallOLMOperatorResponse{}, nil)
		dc.On("InstallOperator", mock.Anything, mock.Anything).Return(&controllerv1beta1.InstallOperatorResponse{}, nil)
		mockGetSubscriptionResponse := &controllerv1beta1.GetSubscriptionResponse{
			Subscription: &controllerv1beta1.Subscription{
				InstallPlanName: "mocked-install-plan",
			},
		}
		dc.On("GetSubscription", mock.Anything, mock.Anything).Return(mockGetSubscriptionResponse, nil)
		dc.On("ApproveInstallPlan", mock.Anything, mock.Anything).Return(&controllerv1beta1.ApproveInstallPlanResponse{}, nil)
		dc.On("StopMonitoring", mock.Anything, mock.Anything).Return(&controllerv1beta1.StopMonitoringResponse{}, nil)

		kubernetesClusterName := "test-cluster"
		registerKubernetesClusterResponse, err := ks.RegisterKubernetesCluster(ctx, &dbaasv1beta1.RegisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
			KubeAuth:              &dbaasv1beta1.KubeAuth{Kubeconfig: kubeconfig},
		})
		require.NoError(t, err)
		assert.NotNil(t, registerKubernetesClusterResponse)

		getClusterResponse, err := ks.GetKubernetesCluster(ctx, &dbaasv1beta1.GetKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		require.NoError(t, err)
		assert.NotNil(t, getClusterResponse)
		assert.NotNil(t, getClusterResponse.KubeAuth)
		assert.Equal(t, kubeconfig, getClusterResponse.KubeAuth.Kubeconfig)

		clusters, err = ks.ListKubernetesClusters(ctx, &dbaasv1beta1.ListKubernetesClustersRequest{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(clusters.KubernetesClusters))
		expected := []*dbaasv1beta1.ListKubernetesClustersResponse_Cluster{
			{
				KubernetesClusterName: kubernetesClusterName,
				Operators: &dbaasv1beta1.Operators{
					Pxc:   &dbaasv1beta1.Operator{Status: dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_NOT_INSTALLED},
					Psmdb: &dbaasv1beta1.Operator{Version: onePointEight, Status: dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_UNSUPPORTED},
					Dbaas: &dbaasv1beta1.Operator{Version: "", Status: dbaasv1beta1.OperatorsStatus_OPERATORS_STATUS_UNSUPPORTED},
				},
				Status: dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK,
			},
		}
		assert.Equal(t, expected[0].Operators, clusters.KubernetesClusters[0].Operators)
		assert.Equal(t, expected[0].KubernetesClusterName, clusters.KubernetesClusters[0].KubernetesClusterName)
		assert.True(
			t,
			clusters.KubernetesClusters[0].Status == dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_OK ||
				clusters.KubernetesClusters[0].Status == dbaasv1beta1.KubernetesClusterStatus_KUBERNETES_CLUSTER_STATUS_PROVISIONING,
		)
		mockK8sResp := []dbaasv1.DatabaseCluster{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "first-pxc-test",
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:      "pxc",
					DatabaseImage: "percona/percona-xtradb-cluster:8.0.27-18.1",
					ClusterSize:   5,
					DBInstance: dbaasv1.DBInstanceSpec{
						CPU:      resource.MustParse("3m"),
						Memory:   resource.MustParse("256"),
						DiskSize: resource.MustParse("1073741824"),
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type: "proxysql",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2m"),
								corev1.ResourceMemory: resource.MustParse("124"),
							},
						},
					},
				},
				Status: dbaasv1.DatabaseClusterStatus{
					Ready: 15,
					Size:  15,
				},
			},
		}
		listDatabaseMock := kubernetesClient.On("ListDatabaseClusters", ctx)
		listDatabaseMock.Return(&dbaasv1.DatabaseClusterList{Items: mockK8sResp}, nil)

		_, err = ks.UnregisterKubernetesCluster(ctx, &dbaasv1beta1.UnregisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		tests.AssertGRPCError(t, status.Newf(codes.FailedPrecondition, "Kubernetes cluster %s has database clusters", kubernetesClusterName), err)

		mockK8sResp = []dbaasv1.DatabaseCluster{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "first-psmdb-test",
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:      "psmdb",
					DatabaseImage: "percona/percona-server-mongodb:4.4.5-7",
					ClusterSize:   5,
					DBInstance: dbaasv1.DBInstanceSpec{
						CPU:      resource.MustParse("3m"),
						Memory:   resource.MustParse("256"),
						DiskSize: resource.MustParse("1073741824"),
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type: "mongos",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2m"),
								corev1.ResourceMemory: resource.MustParse("124"),
							},
						},
					},
				},
				Status: dbaasv1.DatabaseClusterStatus{
					Ready: 10,
					Size:  10,
				},
			},
		}
		listDatabaseMock.Return(&dbaasv1.DatabaseClusterList{Items: mockK8sResp}, nil)
		tests.AssertGRPCError(t, status.Newf(codes.FailedPrecondition, "Kubernetes cluster %s has database clusters", kubernetesClusterName), err)
		mockK8sResp = []dbaasv1.DatabaseCluster{}

		listDatabaseMock.Return(&dbaasv1.DatabaseClusterList{Items: mockK8sResp}, nil)
		unregisterKubernetesClusterResponse, err := ks.UnregisterKubernetesCluster(ctx, &dbaasv1beta1.UnregisterKubernetesClusterRequest{
			KubernetesClusterName: kubernetesClusterName,
		})
		require.NoError(t, err)
		assert.NotNil(t, unregisterKubernetesClusterResponse)

		clusters, err = ks.ListKubernetesClusters(ctx, &dbaasv1beta1.ListKubernetesClustersRequest{})
		assert.NoError(t, err)
		assert.Empty(t, clusters.KubernetesClusters)
	})
}

func TestGetResources(t *testing.T) {
	const (
		clusterName = "test-cluster"
		kubeConfig  = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
  name: local
contexts:
- context:
    cluster: local
    user: local
  name: local
current-context: local`
	)
	setup := func(t *testing.T) (ks dbaasv1beta1.KubernetesServer, kubernetesClient *mockKubernetesClient, teardown func(t *testing.T)) {
		t.Helper()

		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient := &mockDbaasClient{}
		kubernetesClient = &mockKubernetesClient{}
		grafanaClient := &mockGrafanaClient{}

		kubernetesCluster, err := models.CreateKubernetesCluster(db.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: clusterName,
			KubeConfig:            kubeConfig,
		})
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
			assert.NoError(t, db.Delete(kubernetesCluster))
			require.NoError(t, sqlDB.Close())
		}
		versionService := NewVersionServiceClient("https://check-dev.percona.com/versions/v1")
		ks = NewKubernetesServer(db, dbaasClient, kubernetesClient, versionService, grafanaClient)
		return
	}
	t.Run("GetResources", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)
		kubernetesClient.On("GetClusterType", mock.Anything).Return(kubernetes.ClusterTypeMinikube, nil)
		kubernetesClient.On("GetAllClusterResources", mock.Anything, kubernetes.ClusterTypeMinikube, mock.Anything).Return(uint64(100), uint64(200), uint64(300), nil)
		kubernetesClient.On("GetConsumedCPUAndMemory", mock.Anything, "").Return(uint64(50), uint64(100), nil)
		kubernetesClient.On("GetConsumedDiskBytes", mock.Anything, kubernetes.ClusterTypeMinikube, mock.Anything).Return(uint64(150), nil)

		resp, err := ks.GetResources(context.Background(), &dbaasv1beta1.GetResourcesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.Nil(t, err)
		assert.Equal(t, &dbaasv1beta1.GetResourcesResponse{
			All: &dbaasv1beta1.Resources{
				CpuM:        100,
				MemoryBytes: 200,
				DiskSize:    300,
			},
			Available: &dbaasv1beta1.Resources{
				CpuM:        50,
				MemoryBytes: 100,
				DiskSize:    150,
			},
		}, resp)
	})

	t.Run("GetResources invalid cluster name", func(t *testing.T) {
		ks, _, teardown := setup(t)
		defer teardown(t)

		_, err := ks.GetResources(context.Background(), &dbaasv1beta1.GetResourcesRequest{
			KubernetesClusterName: "invalid-cluster",
		})
		assert.NotNil(t, err)
	})

	t.Run("GetResources GetClusterType error", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)

		kubernetesClient.On("GetClusterType", mock.Anything).Return(kubernetes.ClusterTypeUnknown, errors.New("error"))

		_, err := ks.GetResources(context.Background(), &dbaasv1beta1.GetResourcesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.NotNil(t, err)
	})

	t.Run("GetResources GetAllClusterResources error", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)
		kubernetesClient.On("GetClusterType", mock.Anything).Return(kubernetes.ClusterTypeMinikube, nil)

		kubernetesClient.On("GetAllClusterResources", mock.Anything, kubernetes.ClusterTypeMinikube, mock.Anything).Return(uint64(0), uint64(0), uint64(0), errors.New("error"))

		_, err := ks.GetResources(context.Background(), &dbaasv1beta1.GetResourcesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.NotNil(t, err)
	})

	t.Run("GetResources GetConsumedCPUAndMemory error", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)
		kubernetesClient.On("GetClusterType", mock.Anything).Return(kubernetes.ClusterTypeMinikube, nil)
		kubernetesClient.On("GetAllClusterResources", mock.Anything, kubernetes.ClusterTypeMinikube, mock.Anything).Return(uint64(100), uint64(200), uint64(300), nil)

		kubernetesClient.On("GetConsumedCPUAndMemory", mock.Anything, "").Return(uint64(0), uint64(0), errors.New("error"))

		_, err := ks.GetResources(context.Background(), &dbaasv1beta1.GetResourcesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.NotNil(t, err)
	})

	t.Run("GetResources GetConsumedDiskBytes error", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)
		kubernetesClient.On("GetClusterType", mock.Anything).Return(kubernetes.ClusterTypeMinikube, nil)
		kubernetesClient.On("GetAllClusterResources", mock.Anything, kubernetes.ClusterTypeMinikube, mock.Anything).Return(uint64(100), uint64(200), uint64(300), nil)
		kubernetesClient.On("GetConsumedCPUAndMemory", mock.Anything, "").Return(uint64(50), uint64(100), nil)

		kubernetesClient.On("GetConsumedDiskBytes", mock.Anything, kubernetes.ClusterTypeMinikube, mock.Anything).Return(uint64(0), errors.New("error"))

		_, err := ks.GetResources(context.Background(), &dbaasv1beta1.GetResourcesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.NotNil(t, err)
	})
}

func TestListStorageClasses(t *testing.T) {
	const (
		clusterName = "test-cluster"
		kubeConfig  = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
  name: local
contexts:
- context:
    cluster: local
    user: local
  name: local
current-context: local`
	)
	setup := func(t *testing.T) (ks dbaasv1beta1.KubernetesServer, kubernetesClient *mockKubernetesClient, teardown func(t *testing.T)) {
		t.Helper()

		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		dbaasClient := &mockDbaasClient{}
		kubernetesClient = &mockKubernetesClient{}
		grafanaClient := &mockGrafanaClient{}

		kubernetesCluster, err := models.CreateKubernetesCluster(db.Querier, &models.CreateKubernetesClusterParams{
			KubernetesClusterName: clusterName,
			KubeConfig:            kubeConfig,
		})
		require.NoError(t, err)

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)
			dbaasClient.AssertExpectations(t)
			assert.NoError(t, db.Delete(kubernetesCluster))
			require.NoError(t, sqlDB.Close())
		}
		versionService := NewVersionServiceClient("https://check-dev.percona.com/versions/v1")
		ks = NewKubernetesServer(db, dbaasClient, kubernetesClient, versionService, grafanaClient)
		return
	}
	t.Run("ListStorageClasses", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)
		kubernetesClient.On("GetStorageClasses", mock.Anything).Return(&storagev1.StorageClassList{
			Items: []storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "local-storage",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "standard",
					},
				},
			},
		}, nil)
		resp, err := ks.ListStorageClasses(context.Background(), &dbaasv1beta1.ListStorageClassesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.Nil(t, err)
		assert.Equal(t, &dbaasv1beta1.ListStorageClassesResponse{
			StorageClasses: []string{
				"local-storage",
				"standard",
			},
		}, resp)
	})

	t.Run("ListStorageClasses invalid cluster name", func(t *testing.T) {
		ks, _, teardown := setup(t)
		defer teardown(t)

		_, err := ks.ListStorageClasses(context.Background(), &dbaasv1beta1.ListStorageClassesRequest{
			KubernetesClusterName: "invalid-cluster",
		})
		assert.NotNil(t, err)
	})

	t.Run("ListStorageClasses GetStorageClasses error", func(t *testing.T) {
		ks, kubernetesClient, teardown := setup(t)
		defer teardown(t)

		kubernetesClient.On("SetKubeconfig", mock.Anything, mock.Anything).Return(nil)

		kubernetesClient.On("GetStorageClasses", mock.Anything).Return(nil, errors.New("error"))

		_, err := ks.ListStorageClasses(context.Background(), &dbaasv1beta1.ListStorageClassesRequest{
			KubernetesClusterName: "test-cluster",
		})
		assert.NotNil(t, err)
	})
}

func TestGetFlagValue(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		args          []string
		flagName      string
		expectedValue string
	}{
		{
			args:          []string{"token", "--foo", "bar"},
			flagName:      "--foo",
			expectedValue: "bar",
		},
		{
			args:          []string{"token", "--foo", "bar"},
			flagName:      "--bar",
			expectedValue: "",
		},
		{
			args:          []string{"token", "--foo"},
			flagName:      "--foo",
			expectedValue: "",
		},
	}
	for _, tt := range testCases {
		value := getFlagValue(tt.args, tt.flagName)
		assert.Equal(t, tt.expectedValue, value)
	}
}

const awsIAMAuthenticatorKubeconfig = `kind: Config
apiVersion: v1
current-context: arn:aws:eks:zone-2:123465545:cluster/cluster
clusters:
    - cluster:
        certificate-authority-data: base64data
        name: arn:aws:eks:zone-2:123465545:cluster/cluster
        server: https://DDDDD.bla.zone-2.eks.amazonaws.com
contexts:
    - context:
        cluster: arn:aws:eks:zone-2:123465545:cluster/cluster
        name: arn:aws:eks:zone-2:123465545:cluster/cluster
        user: arn:aws:eks:zone-2:123465545:cluster/cluster
preferences: {}
users:
    - name: arn:aws:eks:zone-2:123465545:cluster/cluster
      user:
        exec:
            apiVersion: client.authentication.k8s.io/v1alpha1
            args:
                - token
                - -i
                - test-cluster1
                - --region
                - zone-2
            command: aws-iam-authenticator
            env:
                - name: AWS_STS_REGIONAL_ENDPOINTS
                  value: regional
            provideClusterInfo: false
`

const awsIAMAuthenticatorKubeconfigTransformed = `kind: Config
apiVersion: v1
current-context: arn:aws:eks:zone-2:123465545:cluster/cluster
clusters:
    - cluster:
        certificate-authority-data: base64data
        name: arn:aws:eks:zone-2:123465545:cluster/cluster
        server: https://DDDDD.bla.zone-2.eks.amazonaws.com
contexts:
    - context:
        cluster: arn:aws:eks:zone-2:123465545:cluster/cluster
        name: arn:aws:eks:zone-2:123465545:cluster/cluster
        user: arn:aws:eks:zone-2:123465545:cluster/cluster
preferences: {}
users:
    - name: arn:aws:eks:zone-2:123465545:cluster/cluster
      user:
        exec:
            apiVersion: client.authentication.k8s.io/v1alpha1
            args:
                - token
                - -i
                - test-cluster1
                - --region
                - zone-2
            command: aws-iam-authenticator
            env:
                - name: AWS_STS_REGIONAL_ENDPOINTS
                  value: regional
                - name: AWS_ACCESS_KEY_ID
                  value: keyID
                - name: AWS_SECRET_ACCESS_KEY
                  value: key
            provideClusterInfo: false
`

const awsKubeconfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: base64data
    name: arn:aws:eks:zone-2:123465545:cluster/cluster
    server: https://DDDDD.bla.zone-2.eks.amazonaws.com
contexts:
- context:
    cluster: arn:aws:eks:zone-2:123465545:cluster/cluster
    name: arn:aws:eks:zone-2:123465545:cluster/cluster
    user: arn:aws:eks:zone-2:123465545:cluster/cluster
current-context: "arn:aws:eks:zone-2:123465545:cluster/cluster"
kind: Config
preferences: {}
users:
- name: arn:aws:eks:zone-2:123465545:cluster/cluster
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - eks
      - get-token
      - --cluster-name
      - test-cluster1
      - --region
      - zone-2
      command: aws
      env:
      - name: AWS_STS_REGIONAL_ENDPOINTS
        value: regional
      provideClusterInfo: false
`

const awsKubeconfigWithKeys = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: base64data
    name: arn:aws:eks:zone-2:123465545:cluster/cluster
    server: https://DDDDD.bla.zone-2.eks.amazonaws.com
contexts:
- context:
    cluster: arn:aws:eks:zone-2:123465545:cluster/cluster
    name: arn:aws:eks:zone-2:123465545:cluster/cluster
    user: arn:aws:eks:zone-2:123465545:cluster/cluster
current-context: "arn:aws:eks:zone-2:123465545:cluster/cluster"
kind: Config
preferences: {}
users:
- name: arn:aws:eks:zone-2:123465545:cluster/cluster
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - eks
      - get-token
      - --cluster-name
      - test-cluster1
      - --region
      - zone-2
      command: aws
      env:
      - name: AWS_STS_REGIONAL_ENDPOINTS
        value: regional
      - name: AWS_ACCESS_KEY_ID
        value: keyID
      - name: AWS_SECRET_ACCESS_KEY
        value: key
      provideClusterInfo: false
`

func TestUseIAMAuthenticator(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		kubeconfig        string
		expectedError     error
		expectedTransform string
		keyID             string
		key               string
	}{
		{
			name:              "transform aws to aws-iam-authenticator with keys",
			kubeconfig:        awsKubeconfig,
			expectedTransform: awsIAMAuthenticatorKubeconfigTransformed,
			expectedError:     nil,
			keyID:             "keyID",
			key:               "key",
		},
		{
			name:              "transform aws with keys to aws-iam-authenticator",
			kubeconfig:        awsKubeconfigWithKeys,
			expectedTransform: awsIAMAuthenticatorKubeconfigTransformed,
			expectedError:     nil,
		},
		{
			name:              "transform aws to aws-iam-authenticator without keys",
			kubeconfig:        awsKubeconfig,
			expectedTransform: awsIAMAuthenticatorKubeconfig,
			expectedError:     nil,
		},
		{
			name:              "add environment variables to aws-iam-authenticator",
			kubeconfig:        awsIAMAuthenticatorKubeconfig,
			expectedTransform: awsIAMAuthenticatorKubeconfigTransformed,
			expectedError:     nil,
			keyID:             "keyID",
			key:               "key",
		},
		{
			name:              "return error if kubeconfig is empty",
			kubeconfig:        "     ",
			expectedTransform: "",
			expectedError:     errKubeconfigIsEmpty,
		},
		{
			name:              "don't transform aws-iam-authenticator with keys",
			kubeconfig:        awsIAMAuthenticatorKubeconfigTransformed,
			expectedTransform: awsIAMAuthenticatorKubeconfigTransformed,
			expectedError:     nil,
		},
	}
	for i, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			value, err := replaceAWSAuthIfPresent(tt.kubeconfig, tt.keyID, tt.key)
			assert.ErrorIsf(t, err, tt.expectedError, "Errors don't match in the test case number %d.", i)
			assert.Equalf(t, tt.expectedTransform, value, "Given and expected kubeconfigs don't match in the test case number %d.", i)
		})
	}
}
