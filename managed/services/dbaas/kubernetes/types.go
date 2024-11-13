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
	"fmt"
	"strconv"
	"strings"

	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/percona/pmm/managed/models"
)

const (
	dbaasAPI            = "dbaas.percona.com/v1"
	dbaasKind           = "DatabaseCluster"
	pxcSecretNameTmpl   = "dbaas-%s-pxc-secrets"   //nolint:gosec
	psmdbSecretNameTmpl = "dbaas-%s-psmdb-secrets" //nolint:gosec
	// DatabaseTypePXC is a pxc database.
	DatabaseTypePXC dbaasv1.EngineType = "pxc"
	// DatabaseTypePSMDB is a psmdb database.
	DatabaseTypePSMDB dbaasv1.EngineType = "psmdb"
	externalNLB       string             = "external"

	dbTemplateKindAnnotationKey = "dbaas.percona.com/dbtemplate-kind"
	dbTemplateNameAnnotationKey = "dbaas.percona.com/dbtemplate-name"
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

// DatabaseClusterForPXC fills dbaasv1.DatabaseCluster struct with data provided for specified cluster type.
func DatabaseClusterForPXC(cluster *dbaasv1beta1.CreatePXCClusterRequest, clusterType ClusterType, backupLocation *models.BackupLocation) (*dbaasv1.DatabaseCluster, *dbaasv1.DatabaseClusterRestore, error) { //nolint:lll
	if (cluster.Params.Proxysql != nil) == (cluster.Params.Haproxy != nil) {
		return nil, nil, errors.New("pxc cluster must have one and only one proxy type defined")
	}
	if backupLocation != nil && backupLocation.Type != models.S3BackupLocationType {
		return nil, nil, errors.New("only s3 compatible storages are supported for backup/restore")
	}
	diskSize := resource.NewQuantity(cluster.Params.Pxc.DiskSize, resource.DecimalSI)
	cpu, err := resource.ParseQuantity(fmt.Sprintf("%dm", cluster.Params.Pxc.ComputeResources.CpuM))
	if err != nil {
		return nil, nil, err
	}
	clusterMemory := resource.NewQuantity(cluster.Params.Pxc.ComputeResources.MemoryBytes, resource.DecimalSI)
	dbCluster := &dbaasv1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: dbaasAPI,
			Kind:       dbaasKind,
		},
		Spec: dbaasv1.DatabaseSpec{
			Database:       DatabaseTypePXC,
			DatabaseImage:  cluster.Params.Pxc.Image,
			DatabaseConfig: cluster.Params.Pxc.Configuration,
			ClusterSize:    cluster.Params.ClusterSize,
			SecretsName:    fmt.Sprintf(pxcSecretNameTmpl, cluster.Name),
			DBInstance: dbaasv1.DBInstanceSpec{
				DiskSize: *diskSize,
				CPU:      cpu,
				Memory:   *clusterMemory,
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: &dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       &dbaasv1.BackupSpec{},
		},
	}
	if cluster.Params.Pxc.StorageClass != "" {
		dbCluster.Spec.DBInstance.StorageClassName = &cluster.Params.Pxc.StorageClass
	}
	if cluster.Params.Haproxy != nil {
		resources, err := convertComputeResource(cluster.Params.Haproxy.ComputeResources)
		if err != nil {
			return nil, nil, err
		}
		dbCluster.Spec.LoadBalancer.Image = cluster.Params.Haproxy.Image
		dbCluster.Spec.LoadBalancer.Size = cluster.Params.ClusterSize
		dbCluster.Spec.LoadBalancer.Resources = resources
		dbCluster.Spec.LoadBalancer.Type = "haproxy"
		dbCluster.Spec.LoadBalancer.TrafficPolicy = "Cluster"
	}
	if cluster.Params.Proxysql != nil {
		resources, err := convertComputeResource(cluster.Params.Proxysql.ComputeResources)
		if err != nil {
			return nil, nil, err
		}
		dbCluster.Spec.LoadBalancer.Size = cluster.Params.ClusterSize
		dbCluster.Spec.LoadBalancer.Image = cluster.Params.Proxysql.Image
		dbCluster.Spec.LoadBalancer.Resources = resources
		dbCluster.Spec.LoadBalancer.Type = "proxysql"
	}
	if cluster.Params.Backup != nil {
		storageName := strings.ToLower(backupLocation.Name)
		dbCluster.Spec.Backup.Enabled = true
		dbCluster.Spec.Backup.Storages = map[string]*dbaasv1.BackupStorageSpec{
			storageName: {
				Type: dbaasv1.BackupStorageType(backupLocation.Type),
				StorageProvider: &dbaasv1.BackupStorageProviderSpec{
					Bucket:            backupLocation.S3Config.BucketName,
					Region:            backupLocation.S3Config.BucketRegion,
					EndpointURL:       backupLocation.S3Config.Endpoint,
					CredentialsSecret: fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName),
				},
			},
		}
		dbCluster.Spec.Backup.ServiceAccountName = cluster.Params.Backup.ServiceAccount
		dbCluster.Spec.Backup.Schedule = []dbaasv1.BackupSchedule{
			{
				Name:        "schedule",
				Enabled:     true,
				Schedule:    cluster.Params.Backup.CronExpression,
				Keep:        int(cluster.Params.Backup.KeepCopies),
				StorageName: storageName,
			},
		}
	}
	if cluster.Expose {
		exposeType, ok := exposeTypeMap[clusterType]
		if !ok {
			return dbCluster, nil, fmt.Errorf("failed to recognize expose type for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.ExposeType = exposeType
		annotations, ok := exposeAnnotationsMap[clusterType]
		if !ok {
			return dbCluster, nil, fmt.Errorf("failed to recognize expose annotations for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.Annotations = annotations
		if cluster.InternetFacing && clusterType == ClusterTypeEKS {
			dbCluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = externalNLB
		}
	}
	var sourceRanges []string
	for _, sourceRange := range cluster.SourceRanges {
		if sourceRange != "" {
			sourceRanges = append(sourceRanges, sourceRange)
		}
	}
	if len(sourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = sourceRanges
	}

	if cluster.Template != nil && cluster.Template.Name != "" && cluster.Template.Kind != "" {
		if dbCluster.ObjectMeta.Annotations == nil {
			dbCluster.ObjectMeta.Annotations = make(map[string]string)
		}
		dbCluster.ObjectMeta.Annotations[dbTemplateNameAnnotationKey] = cluster.Template.Name
		dbCluster.ObjectMeta.Annotations[dbTemplateKindAnnotationKey] = cluster.Template.Kind
	}

	if cluster.Params.Restore != nil && cluster.Params.Restore.Destination != "" {
		if cluster.Params.Restore.SecretsName != "" {
			dbCluster.Spec.SecretsName = cluster.Params.Restore.SecretsName
		}
		dbCluster.Spec.Backup.Enabled = true
		storageName := strings.ToLower(backupLocation.Name)
		dbCluster.Spec.Backup.Storages = map[string]*dbaasv1.BackupStorageSpec{
			storageName: {
				Type: dbaasv1.BackupStorageType(backupLocation.Type),
				StorageProvider: &dbaasv1.BackupStorageProviderSpec{
					Bucket:            backupLocation.S3Config.BucketName,
					Region:            backupLocation.S3Config.BucketRegion,
					EndpointURL:       backupLocation.S3Config.Endpoint,
					CredentialsSecret: fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName),
				},
			},
		}

		dbRestore := &dbaasv1.DatabaseClusterRestore{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DatabaseClusterRestore",
				APIVersion: "dbaas.percona.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-restore", dbCluster.Name),
			},
			Spec: dbaasv1.DatabaseClusterRestoreSpec{
				DatabaseCluster: dbCluster.Name,
				DatabaseType:    "pxc",
				BackupSource: &dbaasv1.BackupSource{
					Destination: cluster.Params.Restore.Destination,
					StorageType: dbaasv1.BackupStorageS3,
					S3: &dbaasv1.BackupStorageProviderSpec{
						Bucket:            backupLocation.S3Config.BucketName,
						Region:            backupLocation.S3Config.BucketRegion,
						EndpointURL:       backupLocation.S3Config.Endpoint,
						CredentialsSecret: fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName),
					},
					StorageName: storageName,
				},
			},
		}
		return dbCluster, dbRestore, nil
	}
	return dbCluster, nil, nil
}

// DatabaseClusterForPSMDB fills dbaasv1.DatabaseCluster struct with data provided for specified cluster type.
func DatabaseClusterForPSMDB(cluster *dbaasv1beta1.CreatePSMDBClusterRequest, clusterType ClusterType, backupLocation *models.BackupLocation, backupImage string) (*dbaasv1.DatabaseCluster, *dbaasv1.DatabaseClusterRestore, error) { //nolint:lll
	if backupLocation != nil && backupLocation.Type != models.S3BackupLocationType {
		return nil, nil, errors.New("only s3 compatible storages are supported for backup/restore")
	}
	diskSize := resource.NewQuantity(cluster.Params.Replicaset.DiskSize, resource.DecimalSI)
	cpu, err := resource.ParseQuantity(fmt.Sprintf("%dm", cluster.Params.Replicaset.ComputeResources.CpuM))
	if err != nil {
		return nil, nil, err
	}
	clusterMemory := resource.NewQuantity(cluster.Params.Replicaset.ComputeResources.MemoryBytes, resource.DecimalSI)
	dbCluster := &dbaasv1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: dbaasAPI,
			Kind:       dbaasKind,
		},
		Spec: dbaasv1.DatabaseSpec{
			Database:       DatabaseTypePSMDB,
			DatabaseImage:  cluster.Params.Image,
			DatabaseConfig: cluster.Params.Replicaset.Configuration,
			ClusterSize:    cluster.Params.ClusterSize,
			SecretsName:    fmt.Sprintf(psmdbSecretNameTmpl, cluster.Name),
			DBInstance: dbaasv1.DBInstanceSpec{
				DiskSize: *diskSize,
				CPU:      cpu,
				Memory:   *clusterMemory,
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: &dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       &dbaasv1.BackupSpec{},
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
			return dbCluster, nil, fmt.Errorf("failed to recognize expose type for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.ExposeType = exposeType
		annotations, ok := exposeAnnotationsMap[clusterType]
		if !ok {
			return dbCluster, nil, fmt.Errorf("failed to recognize expose annotations for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.Annotations = annotations
		if cluster.InternetFacing && clusterType == ClusterTypeEKS {
			dbCluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = externalNLB
		}
	}
	if cluster.Params.Backup != nil {
		dbCluster.Spec.Backup.Enabled = true
		if backupImage != "" {
			dbCluster.Spec.Backup.Image = backupImage
		}
		storageName := strings.ToLower(backupLocation.Name)
		dbCluster.Spec.Backup.Storages = map[string]*dbaasv1.BackupStorageSpec{
			storageName: {
				Type: dbaasv1.BackupStorageType(backupLocation.Type),
				StorageProvider: &dbaasv1.BackupStorageProviderSpec{
					Bucket:            backupLocation.S3Config.BucketName,
					Region:            backupLocation.S3Config.BucketRegion,
					EndpointURL:       backupLocation.S3Config.Endpoint,
					CredentialsSecret: fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName),
				},
			},
		}
		dbCluster.Spec.Backup.ServiceAccountName = cluster.Params.Backup.ServiceAccount
		dbCluster.Spec.Backup.Schedule = []dbaasv1.BackupSchedule{
			{
				Name:        "schedule",
				Enabled:     true,
				Schedule:    cluster.Params.Backup.CronExpression,
				Keep:        int(cluster.Params.Backup.KeepCopies),
				StorageName: storageName,
			},
		}
	}
	var sourceRanges []string
	for _, sourceRange := range cluster.SourceRanges {
		if sourceRange != "" {
			sourceRanges = append(sourceRanges, sourceRange)
		}
	}
	if len(sourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = sourceRanges
	}

	if cluster.Template != nil && cluster.Template.Name != "" && cluster.Template.Kind != "" {
		if dbCluster.ObjectMeta.Annotations == nil {
			dbCluster.ObjectMeta.Annotations = make(map[string]string)
		}
		dbCluster.ObjectMeta.Annotations[dbTemplateNameAnnotationKey] = cluster.Template.Name
		dbCluster.ObjectMeta.Annotations[dbTemplateKindAnnotationKey] = cluster.Template.Kind
	}

	if cluster.Params.Restore != nil && cluster.Params.Restore.Destination != "" {
		if cluster.Params.Restore.SecretsName != "" {
			dbCluster.Spec.SecretsName = cluster.Params.Restore.SecretsName
		}
		dbCluster.Spec.Backup.Enabled = true
		storageName := strings.ToLower(backupLocation.Name)
		dbCluster.Spec.Backup.Storages = map[string]*dbaasv1.BackupStorageSpec{
			storageName: {
				Type: dbaasv1.BackupStorageType(backupLocation.Type),
				StorageProvider: &dbaasv1.BackupStorageProviderSpec{
					Bucket:            backupLocation.S3Config.BucketName,
					Region:            backupLocation.S3Config.BucketRegion,
					EndpointURL:       backupLocation.S3Config.Endpoint,
					CredentialsSecret: fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName),
				},
			},
		}

		dbRestore := &dbaasv1.DatabaseClusterRestore{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DatabaseClusterRestore",
				APIVersion: "dbaas.percona.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-restore", dbCluster.Name),
			},
			Spec: dbaasv1.DatabaseClusterRestoreSpec{
				DatabaseCluster: dbCluster.Name,
				DatabaseType:    "psmdb",
				BackupSource: &dbaasv1.BackupSource{
					Destination: cluster.Params.Restore.Destination,
					StorageType: dbaasv1.BackupStorageS3,
					S3: &dbaasv1.BackupStorageProviderSpec{
						Bucket:            backupLocation.S3Config.BucketName,
						Region:            backupLocation.S3Config.BucketRegion,
						EndpointURL:       backupLocation.S3Config.Endpoint,
						CredentialsSecret: fmt.Sprintf("%s-backup", dbCluster.Spec.SecretsName),
					},
					StorageName: storageName,
				},
			},
		}
		return dbCluster, dbRestore, nil
	}
	return dbCluster, nil, nil
}

// UpdatePatchForPSMDB returns a patch to update a database cluster.
func UpdatePatchForPSMDB(dbCluster *dbaasv1.DatabaseCluster, updateRequest *dbaasv1beta1.UpdatePSMDBClusterRequest, clusterType ClusterType) error {
	if updateRequest.Params.Suspend && updateRequest.Params.Resume {
		return errSimultaneous
	}
	dbCluster.TypeMeta = metav1.TypeMeta{
		APIVersion: dbaasAPI,
		Kind:       dbaasKind,
	}
	if updateRequest.Template != nil && updateRequest.Template.Name != "" && updateRequest.Template.Kind != "" {
		if dbCluster.ObjectMeta.Annotations == nil {
			dbCluster.ObjectMeta.Annotations = make(map[string]string)
		}
		dbCluster.ObjectMeta.Annotations[dbTemplateNameAnnotationKey] = updateRequest.Template.Name
		dbCluster.ObjectMeta.Annotations[dbTemplateKindAnnotationKey] = updateRequest.Template.Kind
	} else {
		delete(dbCluster.ObjectMeta.Annotations, dbTemplateNameAnnotationKey)
		delete(dbCluster.ObjectMeta.Annotations, dbTemplateKindAnnotationKey)
	}
	if updateRequest.Params.ClusterSize > 0 {
		dbCluster.Spec.ClusterSize = updateRequest.Params.ClusterSize
	}
	if updateRequest.Params.Image != "" {
		dbCluster.Spec.DatabaseImage = updateRequest.Params.Image
	}
	//nolint:nestif
	if updateRequest.Params.Replicaset != nil {
		if updateRequest.Params.Replicaset.ComputeResources != nil {
			if updateRequest.Params.Replicaset.ComputeResources.CpuM > 0 {
				cpu, err := resource.ParseQuantity(fmt.Sprintf("%dm", updateRequest.Params.Replicaset.ComputeResources.CpuM))
				if err != nil {
					return err
				}
				dbCluster.Spec.DBInstance.CPU = cpu
			}
			if updateRequest.Params.Replicaset.ComputeResources.MemoryBytes > 0 {
				clusterMemory := resource.NewQuantity(updateRequest.Params.Replicaset.ComputeResources.MemoryBytes, resource.DecimalSI)
				dbCluster.Spec.DBInstance.Memory = *clusterMemory
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
	if !updateRequest.Expose {
		dbCluster.Spec.LoadBalancer.ExposeType = corev1.ServiceTypeClusterIP
	}
	if updateRequest.Expose {
		exposeType, ok := exposeTypeMap[clusterType]
		if !ok {
			return fmt.Errorf("failed to recognize expose type for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.ExposeType = exposeType
		annotations, ok := exposeAnnotationsMap[clusterType]
		if !ok {
			return fmt.Errorf("failed to recognize expose annotations for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.Annotations = annotations
		if updateRequest.InternetFacing && clusterType == ClusterTypeEKS {
			dbCluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = externalNLB
		}
	}
	var sourceRanges []string
	for _, sourceRange := range updateRequest.SourceRanges {
		if sourceRange != "" {
			sourceRanges = append(sourceRanges, sourceRange)
		}
	}
	if len(sourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = sourceRanges
	}
	if len(sourceRanges) == 0 && len(dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = sourceRanges
	}
	return nil
}

// UpdatePatchForPXC returns a patch to update a database cluster.
func UpdatePatchForPXC(dbCluster *dbaasv1.DatabaseCluster, updateRequest *dbaasv1beta1.UpdatePXCClusterRequest, clusterType ClusterType) error { //nolint:cyclop
	if updateRequest.Params.Suspend && updateRequest.Params.Resume {
		return errSimultaneous
	}
	dbCluster.TypeMeta = metav1.TypeMeta{
		APIVersion: dbaasAPI,
		Kind:       dbaasKind,
	}
	if updateRequest.Template != nil && updateRequest.Template.Name != "" && updateRequest.Template.Kind != "" {
		if dbCluster.ObjectMeta.Annotations == nil {
			dbCluster.ObjectMeta.Annotations = make(map[string]string)
		}
		dbCluster.ObjectMeta.Annotations[dbTemplateNameAnnotationKey] = updateRequest.Template.Name
		dbCluster.ObjectMeta.Annotations[dbTemplateKindAnnotationKey] = updateRequest.Template.Kind
	} else {
		delete(dbCluster.ObjectMeta.Annotations, dbTemplateNameAnnotationKey)
		delete(dbCluster.ObjectMeta.Annotations, dbTemplateKindAnnotationKey)
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
			cpu, err := resource.ParseQuantity(fmt.Sprintf("%dm", updateRequest.Params.Pxc.ComputeResources.CpuM))
			if err != nil {
				return err
			}
			dbCluster.Spec.DBInstance.CPU = cpu
		}
		if updateRequest.Params.Pxc.ComputeResources.MemoryBytes > 0 {
			clusterMemory := resource.NewQuantity(updateRequest.Params.Pxc.ComputeResources.MemoryBytes, resource.DecimalSI)
			dbCluster.Spec.DBInstance.Memory = *clusterMemory
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
	if !updateRequest.Expose {
		dbCluster.Spec.LoadBalancer.ExposeType = corev1.ServiceTypeClusterIP
	}
	if updateRequest.Expose {
		exposeType, ok := exposeTypeMap[clusterType]
		if !ok {
			return fmt.Errorf("failed to recognize expose type for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.ExposeType = exposeType
		annotations, ok := exposeAnnotationsMap[clusterType]
		if !ok {
			return fmt.Errorf("failed to recognize expose annotations for %s cluster type", clusterType)
		}
		dbCluster.Spec.LoadBalancer.Annotations = annotations
		if updateRequest.InternetFacing && clusterType == ClusterTypeEKS {
			dbCluster.Spec.LoadBalancer.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = externalNLB
		}
	}
	var sourceRanges []string
	for _, sourceRange := range updateRequest.SourceRanges {
		if sourceRange != "" {
			sourceRanges = append(sourceRanges, sourceRange)
		}
	}
	if len(sourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = sourceRanges
	}
	if len(sourceRanges) == 0 && len(dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges) != 0 {
		dbCluster.Spec.LoadBalancer.LoadBalancerSourceRanges = sourceRanges
	}
	return nil
}

// SecretForBackup returns a AWS secrets.
func SecretForBackup(backupLocation *models.BackupLocation) map[string][]byte {
	return map[string][]byte{
		"AWS_ACCESS_KEY_ID":     []byte(backupLocation.S3Config.AccessKey),
		"AWS_SECRET_ACCESS_KEY": []byte(backupLocation.S3Config.SecretKey),
	}
}
