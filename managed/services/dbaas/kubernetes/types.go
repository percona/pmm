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
)

var errSimultaneous = errors.New("field suspend and resume cannot be true simultaneously")

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
	// TODO: Fill HAProxy image
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
				// FIXME: Implement a better solution
				StorageClassName: &cluster.Params.Replicaset.StorageClass,
				DiskSize:         strconv.FormatInt(cluster.Params.Replicaset.DiskSize, 10),
				CPU:              fmt.Sprintf("%dm", cluster.Params.Replicaset.ComputeResources.CpuM),
				Memory:           strconv.FormatInt(cluster.Params.Replicaset.ComputeResources.MemoryBytes, 10),
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       dbaasv1.BackupSpec{},
		},
	}
	dbCluster.Spec.LoadBalancer.Size = cluster.Params.ClusterSize
	dbCluster.Spec.LoadBalancer.Type = "mongos"
	if cluster.Expose {
		dbCluster.Spec.LoadBalancer.ExposeType = corev1.ServiceTypeClusterIP
	}
	dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = cluster.SourceRanges
	return dbCluster
}

func UpdatePatchForPSMDB(cluster *dbaasv1beta1.UpdatePSMDBClusterRequest) (*dbaasv1.DatabaseCluster, error) {
	if cluster.Params.Suspend && cluster.Params.Resume {
		return nil, errSimultaneous
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
			DBInstance: dbaasv1.DBInstanceSpec{
				// FIXME: Implement a better solution
				CPU:    fmt.Sprintf("%dm", cluster.Params.Replicaset.ComputeResources.CpuM),
				Memory: strconv.FormatInt(cluster.Params.Replicaset.ComputeResources.MemoryBytes, 10),
			},
			ClusterSize: cluster.Params.ClusterSize,
		},
	}
	if cluster.Params.Replicaset.StorageClass != "" {
		dbCluster.Spec.DBInstance.StorageClassName = &cluster.Params.Replicaset.StorageClass
	}
	if cluster.Params.Suspend {
		dbCluster.Spec.Pause = true
	}
	if cluster.Params.Resume {
		dbCluster.Spec.Pause = false
	}
	return dbCluster, nil
}

func UpdatePatchForPXC(cluster *dbaasv1beta1.UpdatePXCClusterRequest) (*dbaasv1.DatabaseCluster, error) {
	if cluster.Params.Suspend && cluster.Params.Resume {
		return nil, errSimultaneous
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
			DBInstance: dbaasv1.DBInstanceSpec{
				// FIXME: Implement a better solution
				CPU:    fmt.Sprintf("%dm", cluster.Params.Pxc.ComputeResources.CpuM),
				Memory: strconv.FormatInt(cluster.Params.Pxc.ComputeResources.MemoryBytes, 10),
			},
			ClusterSize: cluster.Params.ClusterSize,
		},
	}
	if cluster.Params.Pxc.StorageClass != "" {
		dbCluster.Spec.DBInstance.StorageClassName = &cluster.Params.Pxc.StorageClass
	}
	if cluster.Params.Suspend {
		dbCluster.Spec.Pause = true
	}
	if cluster.Params.Resume {
		dbCluster.Spec.Pause = false
	}
	if cluster.Params.Haproxy != nil {
		resources, err := convertComputeResource(cluster.Params.Haproxy.ComputeResources)
		if err != nil {
			return nil, err
		}
		dbCluster.Spec.LoadBalancer.Resources = resources

	}
	if cluster.Params.Proxysql != nil {
		resources, err := convertComputeResource(cluster.Params.Proxysql.ComputeResources)
		if err != nil {
			return nil, err
		}
		dbCluster.Spec.LoadBalancer.Resources = resources
	}
	return dbCluster, nil
}
