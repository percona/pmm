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

package kubernetes

import (
	"testing"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
)

func TestDatabaseClusterForPXC(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		input       *dbaasv1beta1.CreatePXCClusterRequest
		clusterType ClusterType
		expected    *dbaasv1.DatabaseCluster
	}{
		{
			name: "Basic PXC cluster with ProxySQL",
			input: &dbaasv1beta1.CreatePXCClusterRequest{
				Name:           "test-pxc-whatever",
				Expose:         false,
				InternetFacing: false,
				SourceRanges:   []string{},
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						Image: "pxc_image",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 2000,
						},
						DiskSize:      2000,
						Configuration: "",
						StorageClass:  "",
					},
					Proxysql: &dbaasv1beta1.PXCClusterParams_ProxySQL{
						Image: "something",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        100,
							MemoryBytes: 100,
						},
						DiskSize: 100,
					},
				},
			},
			clusterType: ClusterTypeGeneric,
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					SecretsName:    "dbaas-test-pxc-whatever-pxc-secrets",
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:          "proxysql",
						Image:         "something",
						Size:          1,
						Configuration: "",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("100"),
								corev1.ResourceCPU:    resource.MustParse("100m"),
							},
						},
						LoadBalancerSourceRanges: nil,
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
		},
		{
			name: "Basic PXC cluster with HAProxy",
			input: &dbaasv1beta1.CreatePXCClusterRequest{
				Name:           "test-pxc-whatever",
				Expose:         false,
				InternetFacing: false,
				SourceRanges:   []string{},
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						Image: "pxc_image",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 2000,
						},
						DiskSize:      2000,
						Configuration: "",
						StorageClass:  "",
					},
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{
						Image: "something",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        100,
							MemoryBytes: 100,
						},
					},
				},
			},
			clusterType: ClusterTypeGeneric,
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					SecretsName:    "dbaas-test-pxc-whatever-pxc-secrets",
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:          "haproxy",
						Image:         "something",
						Size:          1,
						Configuration: "",
						TrafficPolicy: "Cluster",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("100"),
								corev1.ResourceCPU:    resource.MustParse("100m"),
							},
						},
						LoadBalancerSourceRanges: nil,
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
		},
		{
			name: "Basic PXC cluster with HAProxy without compute resources",
			input: &dbaasv1beta1.CreatePXCClusterRequest{
				Name:           "test-pxc-whatever",
				Expose:         false,
				InternetFacing: false,
				SourceRanges:   []string{},
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						Image: "pxc_image",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 2000,
						},
						DiskSize:      2000,
						Configuration: "",
						StorageClass:  "",
					},
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{
						Image: "something",
					},
				},
			},
			clusterType: ClusterTypeGeneric,
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					SecretsName:    "dbaas-test-pxc-whatever-pxc-secrets",
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						TrafficPolicy:            "Cluster",
						LoadBalancerSourceRanges: nil,
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
		},
		{
			name: "Basic exposed PXC cluster with HAProxy (EKS)",
			input: &dbaasv1beta1.CreatePXCClusterRequest{
				Name:           "test-pxc-whatever",
				Expose:         true,
				InternetFacing: false,
				SourceRanges:   []string{},
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						Image: "pxc_image",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 2000,
						},
						DiskSize:      2000,
						Configuration: "",
						StorageClass:  "",
					},
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{
						Image: "something",
					},
				},
			},
			clusterType: ClusterTypeEKS,
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					SecretsName:    "dbaas-test-pxc-whatever-pxc-secrets",
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:       "haproxy",
						ExposeType: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type":         "ip",
							"service.beta.kubernetes.io/aws-load-balancer-scheme":                  "internet-facing",
							"service.beta.kubernetes.io/aws-load-balancer-target-group-attributes": "preserve_client_ip.enabled=true",
						},

						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						TrafficPolicy:            "Cluster",
						LoadBalancerSourceRanges: nil,
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
		},
		{
			name: "Basic exposed PXC cluster with HAProxy and internet facing",
			input: &dbaasv1beta1.CreatePXCClusterRequest{
				Name:           "test-pxc-whatever",
				Expose:         true,
				InternetFacing: true,
				SourceRanges:   []string{},
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						Image: "pxc_image",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 2000,
						},
						DiskSize:      2000,
						Configuration: "",
						StorageClass:  "",
					},
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{
						Image: "something",
					},
				},
			},
			clusterType: ClusterTypeEKS,
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					SecretsName:    "dbaas-test-pxc-whatever-pxc-secrets",
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeLoadBalancer,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						TrafficPolicy:            "Cluster",
						LoadBalancerSourceRanges: nil,
						Annotations: map[string]string{
							"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type":         "ip",
							"service.beta.kubernetes.io/aws-load-balancer-scheme":                  "internet-facing",
							"service.beta.kubernetes.io/aws-load-balancer-target-group-attributes": "preserve_client_ip.enabled=true",
							"service.beta.kubernetes.io/aws-load-balancer-type":                    "external",
						},
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
		},
		{
			name: "Basic exposed PXC cluster with HAProxy",
			input: &dbaasv1beta1.CreatePXCClusterRequest{
				Name:           "test-pxc-whatever",
				Expose:         true,
				InternetFacing: false,
				SourceRanges:   []string{},
				Params: &dbaasv1beta1.PXCClusterParams{
					ClusterSize: 1,
					Pxc: &dbaasv1beta1.PXCClusterParams_PXC{
						Image: "pxc_image",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 2000,
						},
						DiskSize:      2000,
						Configuration: "",
						StorageClass:  "",
					},
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{
						Image: "something",
					},
				},
			},
			clusterType: ClusterTypeGeneric,
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					SecretsName:    "dbaas-test-pxc-whatever-pxc-secrets",
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeLoadBalancer,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						TrafficPolicy:            "Cluster",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
		},
	}
	for _, testCase := range testCases {
		tt := testCase
		cluster, _, err := DatabaseClusterForPXC(tt.input, tt.clusterType, &models.BackupLocation{Type: models.S3BackupLocationType})
		assert.NoError(t, err, tt.name)
		assert.Equal(t, tt.expected, cluster, tt.name)
	}
}

func TestUpdatePatchForPXC(t *testing.T) {
	t.Parallel()
	storageClass := "gp2"
	testCases := []struct {
		name          string
		updateRequest *dbaasv1beta1.UpdatePXCClusterRequest
		cluster       *dbaasv1.DatabaseCluster
		expected      *dbaasv1.DatabaseCluster
	}{
		{
			name: "Empty update does not update anything",
			cluster: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			updateRequest: &dbaasv1beta1.UpdatePXCClusterRequest{
				Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{},
			},
		},
		{
			name: "Pause cluster",
			cluster: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					Pause:          true,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			updateRequest: &dbaasv1beta1.UpdatePXCClusterRequest{
				Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
					Suspend: true,
				},
			},
		},
		{
			name: "Resume cluster",
			cluster: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					Pause:          true,
					DatabaseConfig: "",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			updateRequest: &dbaasv1beta1.UpdatePXCClusterRequest{
				Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
					Resume: true,
				},
			},
		},
		{
			name: "Update Cluster",
			cluster: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: resource.MustParse("2000"),
						CPU:      resource.MustParse("200m"),
						Memory:   resource.MustParse("2000"),
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			expected: &dbaasv1.DatabaseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pxc-whatever",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: dbaasAPI,
					Kind:       dbaasKind,
				},
				Spec: dbaasv1.DatabaseSpec{
					Database:       DatabaseTypePXC,
					DatabaseImage:  "updatedImage",
					DatabaseConfig: "",
					ClusterSize:    3,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize:         resource.MustParse("2000"),
						CPU:              resource.MustParse("300m"),
						Memory:           resource.MustParse("3000"),
						StorageClassName: &storageClass,
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: &dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeClusterIP,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: nil,
						Annotations:              make(map[string]string),
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("200"),
								corev1.ResourceCPU:    resource.MustParse("200m"),
							},
						},
					},
					Backup: &dbaasv1.BackupSpec{},
				},
			},
			updateRequest: &dbaasv1beta1.UpdatePXCClusterRequest{
				Params: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams{
					ClusterSize: 3,
					Pxc: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_PXC{
						Image: "updatedImage",
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        300,
							MemoryBytes: 3000,
						},
						StorageClass: "gp2",
					},
					Haproxy: &dbaasv1beta1.UpdatePXCClusterRequest_UpdatePXCClusterParams_HAProxy{
						ComputeResources: &dbaasv1beta1.ComputeResources{
							CpuM:        200,
							MemoryBytes: 200,
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		tt := testCase
		err := UpdatePatchForPXC(tt.cluster, tt.updateRequest, ClusterTypeGeneric)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, tt.cluster)
	}
}
