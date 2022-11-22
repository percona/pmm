package kubernetes

import (
	"fmt"
	"strconv"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dbaasAPI                         = "dbaas.percona.com/v1"
	dbaasKind                        = "DatabaseCluster"
	databasePXC   dbaasv1.EngineType = "pxc"
	databasePSMDB dbaasv1.EngineType = "psmdb"
)

func convertComputeResource(res *dbaasv1beta1.ComputeResources) (corev1.ResourceRequirements, error) {
	req := corev1.ResourceRequirements{}
	if res == nil {
		return req, nil
	}
	cpu, err := resource.ParseQuantity(fmt.Sprintf("%dm", res.CpuM))
	if err != nil {
		return req, err
	}
	memory, err := resource.ParseQuantity(strconv.FormatInt(res.MemoryBytes, 10))
	if err != nil {
		return req, err
	}
	req.Limits = corev1.ResourceList{}
	req.Limits[corev1.ResourceCPU] = cpu
	req.Limits[corev1.ResourceMemory] = memory
	return req, nil
}

func DatabaseClusterForPXC(cluster *dbaasv1beta1.CreatePXCClusterRequest) (*dbaasv1.DatabaseCluster, error) {
	dbCluster := &dbaasv1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: dbaasAPI,
			Kind:       dbaasKind,
		},
		Spec: dbaasv1.DatabaseSpec{
			Database:       databasePXC,
			DatabaseImage:  cluster.Params.Pxc.Image,
			DatabaseConfig: cluster.Params.Pxc.Configuration,
			ClusterSize:    cluster.Params.ClusterSize,
			DBInstance: dbaasv1.DBInstanceSpec{
				// FIXME: Implement a better solution
				StorageClassName: &cluster.Params.Pxc.StorageClass,
				DiskSize:         strconv.FormatInt(cluster.Params.Pxc.DiskSize, 10),
				CPU:              fmt.Sprintf("%dm", cluster.Params.Pxc.ComputeResources.CpuM),
				Memory:           strconv.FormatInt(cluster.Params.Pxc.ComputeResources.MemoryBytes, 10),
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       dbaasv1.BackupSpec{},
		},
	}
	//TODO: Fill HAProxy image
	if cluster.Params.Haproxy != nil {
		resources, err := convertComputeResource(cluster.Params.Haproxy.ComputeResources)
		if err != nil {
			return nil, err
		}
		dbCluster.Spec.LoadBalancer.Image = cluster.Params.Haproxy.Image
		dbCluster.Spec.LoadBalancer.Size = cluster.Params.ClusterSize
		dbCluster.Spec.LoadBalancer.Resources = resources
		dbCluster.Spec.LoadBalancer.Type = "haproxy"
	}
	if cluster.Params.Proxysql != nil {
		resources, err := convertComputeResource(cluster.Params.Proxysql.ComputeResources)
		if err != nil {
			return nil, err
		}
		dbCluster.Spec.LoadBalancer.Size = cluster.Params.ClusterSize
		dbCluster.Spec.LoadBalancer.Image = cluster.Params.Proxysql.Image
		dbCluster.Spec.LoadBalancer.Resources = resources
		dbCluster.Spec.LoadBalancer.Type = "proxysql"
	}
	if cluster.Expose {
		dbCluster.Spec.LoadBalancer.ExposeType = corev1.ServiceTypeClusterIP
	}
	dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = cluster.SourceRanges
	return dbCluster, nil
}
func DatabaseClusterForPSMDB(cluster *dbaasv1beta1.CreatePSMDBClusterRequest) *dbaasv1.DatabaseCluster {
	return nil
}
func ToCreatePSMDBRequest(cluster *dbaasv1beta1.UpdatePSMDBClusterRequest) *dbaasv1beta1.CreatePSMDBClusterRequest {
	return nil
}
func ToCreatePXCRequest(cluster *dbaasv1beta1.UpdatePXCClusterRequest) *dbaasv1beta1.CreatePXCClusterRequest {
	return &dbaasv1beta1.CreatePXCClusterRequest{}
}
