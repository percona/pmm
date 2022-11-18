package kubernetes

import (
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dbaasAPI                         = "dbaas.percona.com/v1"
	dbaasKind                        = "DatabaseCluster"
	databasePXC   dbaasv1.EngineType = "pxc"
	databasePSMDB dbaasv1.EngineType = "psmdb"
)

func DatabaseClusterForPXC(cluster *dbaasv1beta1.CreatePXCClusterRequest) *dbaasv1.DatabaseCluster {
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
			SecretsName:    "",
			DBInstance: dbaasv1.DBInstanceSpec{
				StorageClassName: cluster.Params.Pxc.StorageClass,
			},
			Monitoring: dbaasv1.MonitoringSpec{
				PMM: dbaasv1.PMMSpec{},
			},
			LoadBalancer: dbaasv1.LoadBalancerSpec{},
			Backup:       dbaasv1.BackupSpec{},
		},
	}
	return dbCluster
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
