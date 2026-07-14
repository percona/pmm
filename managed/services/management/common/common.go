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

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	managementbackup "github.com/percona/pmm/managed/services/management/backup"
)

// ErrClusterLocked is returned when there is an unfinished job that doesn't allow to change service cluster name.
var ErrClusterLocked = errors.New("cluster/service is locked")

// MgmtServices represents a collection of management services.
type MgmtServices struct {
	BackupsService        *managementbackup.BackupsService
	ArtifactsService      *managementbackup.ArtifactsService
	RestoreHistoryService *managementbackup.RestoreHistoryService
}

// RemoveScheduledTasks removes scheduled backup tasks and check there are no running backup/restore tasks in case user changes service cluster label.
func (s *MgmtServices) RemoveScheduledTasks(ctx context.Context, db *reform.DB, params *models.ChangeStandardLabelsParams) error {
	if params.Cluster == nil {
		return nil
	}

	service, err := models.FindServiceByID(db.Querier, params.ServiceID)
	if err != nil {
		return err
	}

	var servicesInCurrentCluster, servicesInNewCluster []*models.Service

	if service.Cluster != "" {
		servicesInCurrentCluster, err = models.FindServices(db.Querier, models.ServiceFilters{Cluster: service.Cluster})
		if err != nil {
			return err
		}
	}

	if *params.Cluster != "" {
		servicesInNewCluster, err = models.FindServices(db.Querier, models.ServiceFilters{Cluster: *params.Cluster})
		if err != nil {
			return err
		}
	}

	allServices := append(servicesInCurrentCluster, servicesInNewCluster...) //nolint:gocritic
	allServices = append(allServices, service)

	sMap := make(map[string]struct{})
	for _, service := range allServices {
		sMap[service.ServiceID] = struct{}{}
	}

	scheduledTasks, err := s.BackupsService.ListScheduledBackups(ctx, &backuppb.ListScheduledBackupsRequest{})
	if err != nil {
		return err
	}

	// Remove scheduled tasks.
	for _, task := range scheduledTasks.ScheduledBackups {
		if _, ok := sMap[task.ServiceId]; ok {
			_, err = s.BackupsService.RemoveScheduledBackup(ctx, &backuppb.RemoveScheduledBackupRequest{ScheduledBackupId: task.ScheduledBackupId})
			if err != nil {
				return err
			}
		}
	}

	// Check no backup tasks running.
	artifacts, err := s.ArtifactsService.ListArtifacts(ctx, &backuppb.ListArtifactsRequest{})
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
			return errors.Wrapf(ErrClusterLocked, "there is an unfinished backup job for service %s or other service in the same cluster", service.ServiceID)
		}
	}

	// Check no restore tasks running.
	restores, err := s.RestoreHistoryService.ListRestoreHistory(ctx, &backuppb.ListRestoreHistoryRequest{})
	if err != nil {
		return err
	}

	for _, restoreItem := range restores.Items {
		if _, ok := sMap[restoreItem.ServiceId]; ok && restoreItem.Status == backuppb.RestoreStatus_RESTORE_STATUS_IN_PROGRESS {
			return errors.Wrapf(ErrClusterLocked, "there is an unfinished restore job for service %s or other service in the same cluster", service.ServiceID)
		}
	}

	return nil
}
