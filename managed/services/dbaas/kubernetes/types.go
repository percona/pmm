package kubernetes

import (
	dbaasv1 "github.com/gen1us2k/dbaas-operator/api/v1"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
)

func DatabaseClusterForPXC(cluster *dbaasv1beta1.CreatePXCClusterRequest) *dbaasv1.DatabaseCluster {
	return nil
}
func DatabaseClusterForPSMDB(cluster *dbaasv1beta1.CreatePSMDBClusterRequest) *dbaasv1.DatabaseCluster {
	return nil
}
func ToCreatePSMDBRequest(cluster *dbaasv1beta1.UpdatePSMDBClusterRequest) *dbaasv1beta1.CreatePSMDBClusterRequest {
	return nil
}
func ToCreatePXCRequest(cluster *dbaasv1beta1.UpdatePXCClusterRequest) *dbaasv1beta1.CreatePXCClusterRequest {
	return nil
}
