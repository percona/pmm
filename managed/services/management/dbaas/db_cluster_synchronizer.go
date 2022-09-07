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
	"sync"
	"time"

	dbaascontrollerv1beta1 "github.com/percona-platform/dbaas-api/gen/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type deletingDBCluster struct {
	kubernetesClusterID string
	dbClusterName       string
	clusterType         models.DBClusterType
}

// DBClustersSynchronizer synchronizes DB Clusters between real kubernetes cluster and our DB.
type DBClustersSynchronizer struct {
	db                        *reform.DB
	l                         *logrus.Entry
	controllerClient          dbaasClient
	rw                        sync.RWMutex
	deletingClusters          map[deletingDBCluster]struct{}
	forceSyncDeletingClusters chan deletingDBCluster
}

// NewDBClustersSynchronizer creates new DB Clusters synchronizer.
func NewDBClustersSynchronizer(db *reform.DB, controllerClient dbaasClient) *DBClustersSynchronizer {
	l := logrus.WithField("component", "dbaas_db_cluster_synchronizer")
	service := &DBClustersSynchronizer{
		db:                        db,
		l:                         l,
		controllerClient:          controllerClient,
		deletingClusters:          make(map[deletingDBCluster]struct{}),
		forceSyncDeletingClusters: make(chan deletingDBCluster, 10),
	}
	return service
}

// Run runs synchronization logic.
func (s *DBClustersSynchronizer) Run(ctx context.Context) {
	s.l.Info("Starting...")
	defer s.l.Info("Done.")

	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to get settings: %+v.", err)
		return
	}
	if !settings.DBaaS.Enabled {
		return
	}
	s.l.Info("Sync DB clusters")
	s.syncAllDBClusters(ctx)
	syncTicker := time.NewTicker(10 * time.Minute)
	deleteTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-syncTicker.C:
			s.l.Info("Sync DB clusters")
			s.syncAllDBClusters(ctx)
		case <-deleteTicker.C:
			s.syncDeletingDBClusters(ctx)
		case c := <-s.forceSyncDeletingClusters:
			s.syncDeletingDBCluster(ctx, c)
		case <-ctx.Done():
			return
		}
	}
}

func (s *DBClustersSynchronizer) syncAllDBClusters(ctx context.Context) {
	clusters, err := models.FindAllKubernetesClusters(s.db.Querier)
	if err != nil {
		s.l.Errorf("couldn't import db clusters: %q", err)
		return
	}
	wg := sync.WaitGroup{}
	for _, k := range clusters {
		kubernetesCluster := k
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.SyncDBClusters(ctx, kubernetesCluster); err != nil {
				s.l.Errorf("couldn't import DB cluster for kubernetes cluster %s: %q", kubernetesCluster.KubernetesClusterName, err)
			}
		}()
	}
	wg.Wait()
	return
}

// SyncDBClusters syncs db clusters list between real kubernetes cluster and our DB.
func (s *DBClustersSynchronizer) SyncDBClusters(ctx context.Context, kubernetesCluster *models.KubernetesCluster) error {
	connection, err := s.controllerClient.CheckKubernetesClusterConnection(ctx, kubernetesCluster.KubeConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't check connection to Kubernetes cluster")
	}

	// To avoid issues if PXC operator isn't installed
	pxc := &dbaascontrollerv1beta1.ListPXCClustersResponse{
		Clusters: []*dbaascontrollerv1beta1.PXCCluster{},
	}
	if connection.Operators.PxcOperatorVersion != "" {
		pxc, err = s.controllerClient.ListPXCClusters(ctx, &dbaascontrollerv1beta1.ListPXCClustersRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		})
		if err != nil {
			return errors.Wrap(err, "couldn't get the list of PXC clusters from Kubernetes")
		}
	}

	psmdb := &dbaascontrollerv1beta1.ListPSMDBClustersResponse{
		Clusters: []*dbaascontrollerv1beta1.PSMDBCluster{},
	}
	if connection.Operators.PsmdbOperatorVersion != "" {
		psmdb, err = s.controllerClient.ListPSMDBClusters(ctx, &dbaascontrollerv1beta1.ListPSMDBClustersRequest{
			KubeAuth: &dbaascontrollerv1beta1.KubeAuth{
				Kubeconfig: kubernetesCluster.KubeConfig,
			},
		})
		if err != nil {
			return errors.Wrap(err, "couldn't get the list of PSMDB clusters from Kubernetes")
		}
	}

	if err != nil {
		return err
	}

	for _, c := range pxc.Clusters {
		cluster, err := models.CreateOrUpdateDBCluster(s.db.Querier, models.PXCType, &models.DBClusterParams{
			KubernetesClusterID: kubernetesCluster.ID,
			Name:                c.Name,
			InstalledImage:      c.Params.Pxc.Image,
		})
		if err != nil {
			return errors.Wrapf(err, "couldn't store PXC cluster to database")
		}
		if c.State == dbaascontrollerv1beta1.DBClusterState_DB_CLUSTER_STATE_DELETING {
			s.WatchDBClusterDeletion(cluster)
		}
	}

	for _, c := range psmdb.Clusters {
		_, err = models.CreateOrUpdateDBCluster(s.db.Querier, models.PSMDBType, &models.DBClusterParams{
			KubernetesClusterID: kubernetesCluster.ID,
			Name:                c.Name,
			InstalledImage:      c.Params.Image,
		})
		if err != nil {
			return errors.Wrapf(err, "couldn't store PSMDB cluster to database")
		}
	}
	clusters, err := models.FindDBClustersForKubernetesCluster(s.db.Querier, kubernetesCluster.ID)
	if err != nil {
		return errors.Wrapf(err, "couldn't get DB clusters list from database")
	}
	for _, cluster := range clusters {
		var found bool
		switch cluster.ClusterType {
		case models.PXCType:
			for _, pxcCluster := range pxc.Clusters {
				if cluster.Name == pxcCluster.Name {
					found = true
					break
				}
			}
		case models.PSMDBType:
			for _, pxcCluster := range psmdb.Clusters {
				if cluster.Name == pxcCluster.Name {
					found = true
					break
				}
			}
		}
		if !found {
			s.l.Infof("Removing db cluster %s", cluster.Name)
			err := s.RemoveDBCluster(cluster)
			if err != nil {
				return errors.Wrapf(err, "couldn't remove DB cluster from DB")
			}
		}
	}
	return nil
}

func (s *DBClustersSynchronizer) WatchDBClusterDeletion(cluster *models.DBCluster) {
	s.rw.Lock()
	defer s.rw.Unlock()
	c := deletingDBCluster{
		kubernetesClusterID: cluster.KubernetesClusterID,
		dbClusterName:       cluster.Name,
		clusterType:         cluster.ClusterType,
	}
	s.deletingClusters[c] = struct{}{}
	s.forceSyncDeletingClusters <- c
}

func (s *DBClustersSynchronizer) syncDeletingDBClusters(ctx context.Context) {
	s.rw.RLock()
	cp := make(map[deletingDBCluster]struct{}, len(s.deletingClusters))
	for c := range s.deletingClusters {
		cp[c] = struct{}{}
	}
	s.rw.RUnlock()

	var wg sync.WaitGroup
	for cluster := range cp {
		c := cluster
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.syncDeletingDBCluster(ctx, c)
		}()
	}
	wg.Wait()
}

func (s *DBClustersSynchronizer) checkDBClusterExists(ctx context.Context, kubeconfig string, dbCluster *models.DBCluster) (bool, error) {
	var err error
	switch dbCluster.ClusterType {
	case models.PXCType:
		_, err = s.controllerClient.GetPXCCluster(ctx, kubeconfig, dbCluster.Name)
	case models.PSMDBType:
		_, err = s.controllerClient.GetPSMDBCluster(ctx, kubeconfig, dbCluster.Name)
	}
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *DBClustersSynchronizer) syncDeletingDBCluster(ctx context.Context, c deletingDBCluster) {
	k, err := models.FindKubernetesClusterByID(s.db.Querier, c.kubernetesClusterID)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			s.removeFromDeletingClusters(c)
		}
		s.l.Warn(errors.Wrap(err, "can't get kubernetes cluster"))
		return
	}
	dbCluster, err := models.FindDBCluster(s.db.Querier, c.kubernetesClusterID, c.dbClusterName, c.clusterType)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			s.removeFromDeletingClusters(c)
		}
		s.l.Warn(errors.Wrap(err, "can't get DB cluster"))
		return
	}
	exist, err := s.checkDBClusterExists(ctx, k.KubeConfig, dbCluster)
	if err != nil {
		s.l.Warn(err)
		return
	}
	// Remove non-existing clusters.
	if !exist {
		err = s.RemoveDBCluster(dbCluster)
		if err != nil {
			s.l.Warnln(err)
		}
	}
}

func (s *DBClustersSynchronizer) RemoveDBCluster(cluster *models.DBCluster) error {
	s.removeFromDeletingClusters(deletingDBCluster{
		kubernetesClusterID: cluster.KubernetesClusterID,
		dbClusterName:       cluster.Name,
		clusterType:         cluster.ClusterType,
	})
	_, err := models.RemoveDBCluster(s.db.Querier, cluster.ID)
	if err != nil {
		return errors.Wrap(err, "can't remove DB cluster")
	}
	return nil
}

func (s *DBClustersSynchronizer) removeFromDeletingClusters(c deletingDBCluster) {
	s.rw.Lock()
	defer s.rw.Unlock()
	delete(s.deletingClusters, c)
}
