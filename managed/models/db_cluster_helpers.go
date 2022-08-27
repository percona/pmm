package models

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

type DBClusterFilters struct {
	Name                string
	KubernetesClusterID string
	ClusterType         DBClusterType
}

func FindDBClustersForKubernetesCluster(q *reform.Querier, kubernetesClusterID string) ([]*DBCluster, error) {
	structs, err := q.SelectAllFrom(DBClusterTable, "WHERE kubernetes_cluster_id = $1 ORDER BY created_at DESC", kubernetesClusterID)
	if err != nil {
		return nil, err
	}
	dbClusters := make([]*DBCluster, len(structs))
	for i, s := range structs {
		dbClusters[i] = s.(*DBCluster)
	}
	return dbClusters, nil
}

// DBClusterParams params for add/update db cluster.
type DBClusterParams struct {
	KubernetesClusterID string
	Name                string
	InstalledImage      string
}

// CreateOrUpdateDBCluster creates DB Cluster with given type.
func CreateOrUpdateDBCluster(q *reform.Querier, dbClusterType DBClusterType, params *DBClusterParams) (*DBCluster, error) {
	_, err := FindKubernetesClusterByID(q, params.KubernetesClusterID)
	if err != nil {
		return nil, err
	}

	row := &DBCluster{
		ClusterType:         dbClusterType,
		KubernetesClusterID: params.KubernetesClusterID,
		Name:                params.Name,
		InstalledImage:      params.InstalledImage,
	}

	dbCluster, err := FindDBCluster(q, params.KubernetesClusterID, params.Name, dbClusterType)
	if err == nil {
		row.ID = dbCluster.ID
		if err := q.Save(row); err != nil {
			return nil, err
		}
	} else if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		id := "/dbcluster_id/" + uuid.New().String()
		if err := checkUniqueDBClusterID(q, id); err != nil {
			return nil, err
		}

		row.ID = id

		if err := q.Insert(row); err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// FindDBClusterByID finds DB cluster by ID.
func FindDBClusterByID(q *reform.Querier, id string) (*DBCluster, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty DB Cluster ID.")
	}

	dbCluster := &DBCluster{ID: id}
	switch err := q.Reload(dbCluster); err {
	case nil:
		return dbCluster, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "DB Cluster with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// FindDBCluster finds DB cluster by Kubernetes cluster ID, DB name and DB type.
func FindDBCluster(q *reform.Querier, kubernetesClusterID string, dbClusterName string, clusterType DBClusterType) (*DBCluster, error) {
	if kubernetesClusterID == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty K8S Cluster ID.")
	}
	if dbClusterName == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty DB Cluster Name.")
	}

	tail := "WHERE kubernetes_cluster_id = $1 AND name = $2 and cluster_type = $3 ORDER BY created_at DESC"
	dbCluster, err := q.SelectOneFrom(DBClusterTable, tail, kubernetesClusterID, dbClusterName, clusterType)
	switch err {
	case nil:
		return dbCluster.(*DBCluster), nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "DB Cluster with name %q not found in kubernetes cluster with ID %q.", dbClusterName, kubernetesClusterID)
	default:
		return nil, errors.WithStack(err)
	}
}

// RemoveDBCluster removes DB cluster by ID.
func RemoveDBCluster(q *reform.Querier, id string) (*DBCluster, error) {
	c, err := FindDBClusterByID(q, id)
	if err != nil {
		return nil, err
	}

	err = q.Delete(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete DB Cluster")
	}
	return c, nil
}

func checkUniqueDBClusterID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty DB Cluster ID")
	}

	agent := &DBCluster{ID: id}
	switch err := q.Reload(agent); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "DB Cluster with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}
