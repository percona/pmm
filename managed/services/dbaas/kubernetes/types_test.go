package kubernetes

import (
	"testing"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					Haproxy: &dbaasv1beta1.PXCClusterParams_HAProxy{},
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
					Database:       databasePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "\n[mysqld]\nwsrep_provider_options=\"gcache.size=600M\"\nwsrep_trx_fragment_unit='bytes'\nwsrep_trx_fragment_size=3670016\n",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: "2000",
						CPU:      "200m",
						Memory:   "2000",
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: dbaasv1.PMMSpec{},
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
						LoadBalancerSourceRanges: []string{},
					},
					Backup: dbaasv1.BackupSpec{},
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
					Database:       databasePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "\n[mysqld]\nwsrep_provider_options=\"gcache.size=600M\"\nwsrep_trx_fragment_unit='bytes'\nwsrep_trx_fragment_size=3670016\n",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: "2000",
						CPU:      "200m",
						Memory:   "2000",
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:          "haproxy",
						Image:         "something",
						Size:          1,
						Configuration: "",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("100"),
								corev1.ResourceCPU:    resource.MustParse("100m"),
							},
						},
						LoadBalancerSourceRanges: []string{},
					},
					Backup: dbaasv1.BackupSpec{},
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
					Database:       databasePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "\n[mysqld]\nwsrep_provider_options=\"gcache.size=600M\"\nwsrep_trx_fragment_unit='bytes'\nwsrep_trx_fragment_size=3670016\n",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: "2000",
						CPU:      "200m",
						Memory:   "2000",
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: []string{},
					},
					Backup: dbaasv1.BackupSpec{},
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
					Database:       databasePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "\n[mysqld]\nwsrep_provider_options=\"gcache.size=600M\"\nwsrep_trx_fragment_unit='bytes'\nwsrep_trx_fragment_size=3670016\n",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: "2000",
						CPU:      "200m",
						Memory:   "2000",
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: dbaasv1.PMMSpec{},
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
						LoadBalancerSourceRanges: []string{},
					},
					Backup: dbaasv1.BackupSpec{},
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
					Database:       databasePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "\n[mysqld]\nwsrep_provider_options=\"gcache.size=600M\"\nwsrep_trx_fragment_unit='bytes'\nwsrep_trx_fragment_size=3670016\n",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: "2000",
						CPU:      "200m",
						Memory:   "2000",
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeLoadBalancer,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: []string{},
						Annotations: map[string]string{
							"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type":         "ip",
							"service.beta.kubernetes.io/aws-load-balancer-scheme":                  "internet-facing",
							"service.beta.kubernetes.io/aws-load-balancer-target-group-attributes": "preserve_client_ip.enabled=true",
							"service.beta.kubernetes.io/aws-load-balancer-type":                    "external",
						},
					},
					Backup: dbaasv1.BackupSpec{},
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
					Database:       databasePXC,
					DatabaseImage:  "pxc_image",
					DatabaseConfig: "\n[mysqld]\nwsrep_provider_options=\"gcache.size=600M\"\nwsrep_trx_fragment_unit='bytes'\nwsrep_trx_fragment_size=3670016\n",
					ClusterSize:    1,
					DBInstance: dbaasv1.DBInstanceSpec{
						DiskSize: "2000",
						CPU:      "200m",
						Memory:   "2000",
					},
					Monitoring: dbaasv1.MonitoringSpec{
						PMM: dbaasv1.PMMSpec{},
					},
					LoadBalancer: dbaasv1.LoadBalancerSpec{
						Type:                     "haproxy",
						ExposeType:               corev1.ServiceTypeLoadBalancer,
						Image:                    "something",
						Size:                     1,
						Configuration:            "",
						LoadBalancerSourceRanges: []string{},
						Annotations:              map[string]string{},
					},
					Backup: dbaasv1.BackupSpec{},
				},
			},
		},
	}
	for _, testCase := range testCases {
		tt := testCase
		cluster, err := DatabaseClusterForPXC(tt.input, tt.clusterType)
		assert.NoError(t, err, tt.name)
		assert.Equal(t, tt.expected, cluster, tt.name)
	}
}
