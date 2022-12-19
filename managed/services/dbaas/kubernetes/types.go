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

package kubernetes

import (
	"errors"
	"fmt"
	"strconv"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
)

const (
	dbaasAPI                         = "dbaas.percona.com/v1"
	dbaasKind                        = "DatabaseCluster"
	databasePXC   dbaasv1.EngineType = "pxc"
	databasePSMDB dbaasv1.EngineType = "psmdb"

	memorySmallSize  = int64(2) * 1000 * 1000 * 1000
	memoryMediumSize = int64(8) * 1000 * 1000 * 1000
	memoryLargeSize  = int64(32) * 1000 * 1000 * 1000
)

var errSimultaneous = errors.New("field suspend and resume cannot be true simultaneously")

var exposeTypeMap = map[ClusterType]corev1.ServiceType{
	ClusterTypeMinikube: corev1.ServiceTypeNodePort,
	ClusterTypeEKS:      corev1.ServiceTypeLoadBalancer,
	ClusterTypeGeneric:  corev1.ServiceTypeLoadBalancer,
}

var exposeAnnotationsMap = map[ClusterType]map[string]string{
	ClusterTypeMinikube: make(map[string]string),
	ClusterTypeEKS: {
		"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type":         "ip",
		"service.beta.kubernetes.io/aws-load-balancer-scheme":                  "internet-facing",
		"service.beta.kubernetes.io/aws-load-balancer-target-group-attributes": "preserve_client_ip.enabled=true",
	},
	ClusterTypeGeneric: make(map[string]string),
}

const (
	pxcDefaultConfigurationTemplate = `[mysqld]
wsrep_provider_options="gcache.size=%s"
wsrep_trx_fragment_unit='bytes'
wsrep_trx_fragment_size=3670016
`
	psmdbDefaultConfigurationTemplate = `
      operationProfiling:
        mode: slowOp
`
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

func DatabaseClusterForPXC(cluster *dbaasv1beta1.CreatePXCClusterRequest, clusterType ClusterType) (*dbaasv1.DatabaseCluster, error) {
	memory := cluster.Params.Pxc.ComputeResources.MemoryBytes
	gCacheSize := "600M"
	if cluster.Params.Pxc.Configuration == "" {
		if memory > memorySmallSize && memory <= memoryMediumSize {
			gCacheSize = "2.4G"
		}
		if memory > memoryMediumSize && memory <= memoryLargeSize {
			gCacheSize = "9.6G"
		}
		if memory > memoryLargeSize {
			gCacheSize = "9.6G"
		}
		cluster.Params.Pxc.Configuration = fmt.Sprintf(pxcDefaultConfigurationTemplate, gCacheSize)
	}
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
				DiskSize: strconv.FormatInt(cluster.Params.Pxc.DiskSize, 10),
				CPU:      fmt.Sprintf("%dm", cluster.Params.Pxc.ComputeResources.CpuM),
				Memory:   strconv.FormatInt(cluster.Params.Pxc.ComputeResources.MemoryBytes, 10),
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: &dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       dbaasv1.BackupSpec{},
		},
	}
	if cluster.Params.Pxc.StorageClass != "" {
		dbCluster.Spec.DBInstance.StorageClassName = &cluster.Params.Pxc.StorageClass
	}
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
		exposeType, ok := exposeTypeMap[clusterType]
		if !ok {
			return dbCluster, fmt.Errorf("failed to recognize expose type for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.ExposeType = exposeType
		annotations, ok := exposeAnnotationsMap[clusterType]
		if !ok {
			return dbCluster, fmt.Errorf("failed to recognize expose annotations for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.Annotations = annotations
		if cluster.InternetFacing && clusterType == ClusterTypeEKS {
			dbCluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = "external"
		}

	}
	if len(cluster.SourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = cluster.SourceRanges
	}
	return dbCluster, nil
}

func DatabaseClusterForPSMDB(cluster *dbaasv1beta1.CreatePSMDBClusterRequest, clusterType ClusterType) (*dbaasv1.DatabaseCluster, error) {
	if cluster.Params.Replicaset.Configuration == "" {
		cluster.Params.Replicaset.Configuration = psmdbDefaultConfigurationTemplate
	}
	dbCluster := &dbaasv1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: dbaasAPI,
			Kind:       dbaasKind,
		},
		Spec: dbaasv1.DatabaseSpec{
			Database:       databasePSMDB,
			DatabaseImage:  cluster.Params.Image,
			DatabaseConfig: cluster.Params.Replicaset.Configuration,
			ClusterSize:    cluster.Params.ClusterSize,
			DBInstance: dbaasv1.DBInstanceSpec{
				DiskSize: strconv.FormatInt(cluster.Params.Replicaset.DiskSize, 10),
				CPU:      fmt.Sprintf("%dm", cluster.Params.Replicaset.ComputeResources.CpuM),
				Memory:   strconv.FormatInt(cluster.Params.Replicaset.ComputeResources.MemoryBytes, 10),
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: &dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       dbaasv1.BackupSpec{},
		},
	}
	if cluster.Params.Replicaset.StorageClass != "" {
		dbCluster.Spec.DBInstance.StorageClassName = &cluster.Params.Replicaset.StorageClass
	}
	dbCluster.Spec.LoadBalancer.Size = cluster.Params.ClusterSize
	dbCluster.Spec.LoadBalancer.Type = "mongos"
	if cluster.Expose {
		exposeType, ok := exposeTypeMap[clusterType]
		if !ok {
			return dbCluster, fmt.Errorf("failed to recognize expose type for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.ExposeType = exposeType
		annotations, ok := exposeAnnotationsMap[clusterType]
		if !ok {
			return dbCluster, fmt.Errorf("failed to recognize expose annotations for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.Annotations = annotations
		if cluster.InternetFacing && clusterType == ClusterTypeEKS {
			dbCluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = "external"
		}
	}
	if len(cluster.SourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = cluster.SourceRanges
	}
	return dbCluster, nil
}

func UpdatePatchForPSMDB(dbCluster *dbaasv1.DatabaseCluster, updateRequest *dbaasv1beta1.UpdatePSMDBClusterRequest) error {
	if updateRequest.Params.Suspend && updateRequest.Params.Resume {
		return errSimultaneous
	}
	dbCluster.TypeMeta = metav1.TypeMeta{
		APIVersion: dbaasAPI,
		Kind:       dbaasKind,
	}
	if updateRequest.Params.ClusterSize > 0 {
		dbCluster.Spec.ClusterSize = updateRequest.Params.ClusterSize
	}
	if updateRequest.Params.Image != "" {
		dbCluster.Spec.DatabaseImage = updateRequest.Params.Image
	}
	if updateRequest.Params.Replicaset != nil {
		if updateRequest.Params.Replicaset.ComputeResources != nil {
			if updateRequest.Params.Replicaset.ComputeResources.CpuM > 0 {
				dbCluster.Spec.DBInstance.CPU = fmt.Sprintf("%dm", updateRequest.Params.Replicaset.ComputeResources.CpuM)
			}
			if updateRequest.Params.Replicaset.ComputeResources.MemoryBytes > 0 {
				dbCluster.Spec.DBInstance.Memory = strconv.FormatInt(updateRequest.Params.Replicaset.ComputeResources.MemoryBytes, 10)
			}
		}
		if updateRequest.Params.Replicaset.Configuration != "" {
			dbCluster.Spec.DatabaseConfig = updateRequest.Params.Replicaset.Configuration
		}

		if updateRequest.Params.Replicaset.StorageClass != "" {
			dbCluster.Spec.DBInstance.StorageClassName = &updateRequest.Params.Replicaset.StorageClass
		}
	}
	if updateRequest.Params.Suspend {
		dbCluster.Spec.Pause = true
	}
	if updateRequest.Params.Resume {
		dbCluster.Spec.Pause = false
	}
	return nil
}

func UpdatePatchForPXC(dbCluster *dbaasv1.DatabaseCluster, updateRequest *dbaasv1beta1.UpdatePXCClusterRequest) error {
	if updateRequest.Params.Suspend && updateRequest.Params.Resume {
		return errSimultaneous
	}
	dbCluster.TypeMeta = metav1.TypeMeta{
		APIVersion: dbaasAPI,
		Kind:       dbaasKind,
	}
	if updateRequest.Params.ClusterSize > 0 {
		dbCluster.Spec.ClusterSize = updateRequest.Params.ClusterSize
	}
	if updateRequest.Params.Pxc != nil {
		if updateRequest.Params.Pxc.Image != "" {
			dbCluster.Spec.DatabaseImage = updateRequest.Params.Pxc.Image
		}
		if updateRequest.Params.Pxc.Configuration != "" {
			dbCluster.Spec.DatabaseConfig = updateRequest.Params.Pxc.Configuration
		}
		if updateRequest.Params.Pxc.StorageClass != "" {
			dbCluster.Spec.DBInstance.StorageClassName = &updateRequest.Params.Pxc.StorageClass
		}
	}

	if updateRequest.Params.Pxc != nil && updateRequest.Params.Pxc.ComputeResources != nil {
		if updateRequest.Params.Pxc.ComputeResources.CpuM > 0 {
			dbCluster.Spec.DBInstance.CPU = fmt.Sprintf("%dm", updateRequest.Params.Pxc.ComputeResources.CpuM)
		}
		if updateRequest.Params.Pxc.ComputeResources.MemoryBytes > 0 {
			dbCluster.Spec.DBInstance.Memory = strconv.FormatInt(updateRequest.Params.Pxc.ComputeResources.MemoryBytes, 10)
		}
	}
	if updateRequest.Params.Haproxy != nil && updateRequest.Params.Haproxy.ComputeResources != nil {
		resources, err := convertComputeResource(updateRequest.Params.Haproxy.ComputeResources)
		if err != nil {
			return err
		}
		dbCluster.Spec.LoadBalancer.Resources = resources

	}
	if updateRequest.Params.Proxysql != nil && updateRequest.Params.Proxysql.ComputeResources != nil {
		resources, err := convertComputeResource(updateRequest.Params.Proxysql.ComputeResources)
		if err != nil {
			return err
		}
		dbCluster.Spec.LoadBalancer.Resources = resources
	}
	if updateRequest.Params.Suspend {
		dbCluster.Spec.Pause = true
	}
	if updateRequest.Params.Resume {
		dbCluster.Spec.Pause = false
	}
	return nil
}
