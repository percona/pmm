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

// Package common contains common and cross-service logics.
package common

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/reform.v1"

	backuppb "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
	managementbackup "github.com/percona/pmm/managed/services/management/backup"
)

// ErrClusterLocked is returned when there is an unfinished job that doesn't allow to change service cluster name.
var ErrClusterLocked = errors.New("cluster/service is locked")

// MgmtServices represents a collection of management services.
type MgmtServices struct {
	BackupService  *managementbackup.BackupService
	RestoreService *managementbackup.RestoreService
}

// NewMgmtServices creates a new MgmtServices instance.
func NewMgmtServices(bs *managementbackup.BackupService, rs *managementbackup.RestoreService) *MgmtServices {
	return &MgmtServices{
		BackupService:  bs,
		RestoreService: rs,
	}
}

// RemoveScheduledTasks removes the scheduled backups of the Services that a cluster change affects. It first checks
// that none of them has a backup or restore running, so a change that is rejected removes nothing.
func (s *MgmtServices) RemoveScheduledTasks(ctx context.Context, db *reform.DB, params *models.ChangeStandardLabelsParams) error {
	affected, err := affectedServices(db, params)
	if err != nil || affected == nil {
		return err
	}

	err = s.checkNoRunningJobs(ctx, params.ServiceID, affected)
	if err != nil {
		return err
	}

	scheduledTasks, err := s.BackupService.ListScheduledBackups(ctx, &backuppb.ListScheduledBackupsRequest{})
	if err != nil {
		return err
	}

	for _, task := range scheduledTasks.ScheduledBackups {
		if _, ok := affected[task.ServiceId]; ok {
			_, err = s.BackupService.RemoveScheduledBackup(ctx, &backuppb.RemoveScheduledBackupRequest{ScheduledBackupId: task.ScheduledBackupId})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CheckClusterUnlocked returns ErrClusterLocked if a Service that a cluster change affects has a backup or restore
// running, as RemoveScheduledTasks does, without removing anything.
func (s *MgmtServices) CheckClusterUnlocked(ctx context.Context, db *reform.DB, params *models.ChangeStandardLabelsParams) error {
	affected, err := affectedServices(db, params)
	if err != nil || affected == nil {
		return err
	}

	return s.checkNoRunningJobs(ctx, params.ServiceID, affected)
}

// affectedServices returns the IDs of the Services that a cluster change affects: the Service itself and the
// Services in its current and in its new cluster. It returns nil if the change sets no cluster.
func affectedServices(db *reform.DB, params *models.ChangeStandardLabelsParams) (map[string]struct{}, error) {
	if params.Cluster == nil || *params.Cluster == "" {
		return nil, nil //nolint:nilnil
	}

	service, err := models.FindServiceByID(db.Querier, params.ServiceID)
	if err != nil {
		return nil, err
	}

	var servicesInCurrentCluster, servicesInNewCluster []*models.Service

	if service.Cluster != "" {
		servicesInCurrentCluster, err = models.FindServices(db.Querier, models.ServiceFilters{Cluster: service.Cluster})
		if err != nil {
			return nil, err
		}
	}

	servicesInNewCluster, err = models.FindServices(db.Querier, models.ServiceFilters{Cluster: *params.Cluster})
	if err != nil {
		return nil, err
	}

	allServices := make([]*models.Service, 0, len(servicesInCurrentCluster)+len(servicesInNewCluster)+1)
	allServices = append(allServices, servicesInCurrentCluster...)
	allServices = append(allServices, servicesInNewCluster...)
	allServices = append(allServices, service)

	sMap := make(map[string]struct{}, len(allServices))
	for _, service := range allServices {
		sMap[service.ServiceID] = struct{}{}
	}

	return sMap, nil
}

// checkNoRunningJobs returns ErrClusterLocked if any of the affected Services has a backup or restore running.
func (s *MgmtServices) checkNoRunningJobs(ctx context.Context, serviceID string, sMap map[string]struct{}) error {
	// Check no backup tasks running.
	artifacts, err := s.BackupService.ListArtifacts(ctx, &backuppb.ListArtifactsRequest{})
	if err != nil {
		return err
	}

	statusNotFinal := func(status backuppb.BackupStatus) bool {
		switch status {
		case
			backuppb.BackupStatus_BACKUP_STATUS_IN_PROGRESS,
			backuppb.BackupStatus_BACKUP_STATUS_PENDING,
			backuppb.BackupStatus_BACKUP_STATUS_PAUSED:
			return true
		default:
			return false
		}
	}

	for _, artifact := range artifacts.Artifacts {
		if _, ok := sMap[artifact.ServiceId]; ok && statusNotFinal(artifact.Status) {
			return fmt.Errorf("there is an unfinished backup job for service %s or other service in the same cluster: %w", serviceID, ErrClusterLocked)
		}
	}

	// Check no restore tasks running.
	restores, err := s.RestoreService.ListRestores(ctx, &backuppb.ListRestoresRequest{})
	if err != nil {
		return err
	}

	for _, restoreItem := range restores.Items {
		if _, ok := sMap[restoreItem.ServiceId]; ok && restoreItem.Status == backuppb.RestoreStatus_RESTORE_STATUS_IN_PROGRESS {
			return fmt.Errorf("there is an unfinished restore job for service %s or other service in the same cluster: %w", serviceID, ErrClusterLocked)
		}
	}

	return nil
}
