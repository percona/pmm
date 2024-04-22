// Copyright (C) 2024 Percona LLC
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

package services

import (
	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// CheckMongoDBBackupPreconditions checks compatibility of different types of scheduled backups and on-demand backups for MongoDB.
//
// WARNING: This function is valid only when executed as part of transaction with serializable isolation level.
func CheckMongoDBBackupPreconditions(q *reform.Querier, mode models.BackupMode, clusterName, serviceID, scheduleID string) error {
	filter := models.ScheduledTasksFilter{
		Disabled:    pointer.ToBool(false),
		ClusterName: clusterName,
	}

	if clusterName == "" {
		// For backward compatibility. There may be existing scheduled backups for mongoDB services without specified cluster name.
		filter.ServiceID = serviceID
	}

	switch mode {
	case models.PITR:
		// PITR backup can be enabled only if there is no other scheduled backups.
		tasks, err := models.FindScheduledTasks(q, filter)
		if err != nil {
			return err
		}

		for _, task := range tasks {
			if task.ID == scheduleID {
				// When we are updating existing scheduled PITR backup we should pass this check.
				continue
			}

			if clusterName == "" {
				// For backward compatibility
				return status.Errorf(codes.FailedPrecondition, "A PITR backup for the service with ID '%s' can be enabled only if "+
					"there are no other scheduled backups for this service.", serviceID)
			}

			return status.Errorf(codes.FailedPrecondition, "A PITR backup for the cluster '%s' can be enabled only if "+
				"there are no other scheduled backups for this cluster.", clusterName)
		}
	case models.Snapshot:
		// Snapshot backup can be enabled or performed if there is no enabled PITR backup.
		filter.Mode = models.PITR
		tasks, err := models.FindScheduledTasks(q, filter)
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			return nil
		}

		if clusterName == "" {
			// For backward compatibility
			return status.Errorf(codes.FailedPrecondition, "A snapshot backup for service '%s' can be performed only if "+
				"there are no other scheduled backups for this service.", serviceID)
		}

		return status.Errorf(codes.FailedPrecondition, "A snapshot backup for cluster '%s' can be performed only if "+
			"there is no enabled PITR backup for this cluster.", clusterName)

	case models.Incremental:
		return status.Error(codes.InvalidArgument, "Incremental backups unsupported for MongoDB")
	}

	return nil
}

// CheckArtifactOverlapping checks if there are other artifacts or scheduled tasks pointing to the same location and folder.
// Placing MySQL and MongoDB artifacts in the same folder is not desirable, while placing MongoDB artifacts of different clusters
// in the same folder may cause data inconsistency.
//
// WARNING: This function is valid only when executed as part of transaction with serializable isolation level.
func CheckArtifactOverlapping(q *reform.Querier, serviceID, locationID, folder string) error {
	// TODO This doesn't work for all cases. For example, there may exist more than one storage locations pointing to the same place.

	const (
		usedByArtifactMsg      = "Same location and folder already used for artifact %s of other service: %s"
		usedByScheduledTaskMsg = "Same location and folder already used for scheduled task %s of other service: %s"
	)

	service, err := models.FindServiceByID(q, serviceID)
	if err != nil {
		return err
	}

	artifacts, err := models.FindArtifacts(q, models.ArtifactFilters{
		LocationID: locationID,
		Folder:     &folder,
	})
	if err != nil {
		return err
	}

	for _, artifact := range artifacts {
		// We skip artifacts made on services that are no longer exists in PMM. However, in future we can improve this function
		// by storing required information right in artifact model.
		if artifact.ServiceID != "" && artifact.ServiceID != serviceID {
			svc, err := models.FindServiceByID(q, artifact.ServiceID)
			if err != nil {
				return err
			}

			if service.ServiceType == models.MySQLServiceType && svc.ServiceType == models.MySQLServiceType {
				continue
			}

			if service.ServiceType == models.MongoDBServiceType && svc.ServiceType == models.MongoDBServiceType {
				if svc.Cluster != service.Cluster {
					return errors.Wrapf(ErrLocationFolderPairAlreadyUsed, usedByArtifactMsg, artifact.ID, serviceID)
				}
				continue
			}

			return errors.Wrapf(ErrLocationFolderPairAlreadyUsed, usedByArtifactMsg, artifact.ID, serviceID)
		}
	}

	tasks, err := models.FindScheduledTasks(q, models.ScheduledTasksFilter{
		LocationID: locationID,
		Folder:     &folder,
	})
	if err != nil {
		return err
	}

	var svcID string

	for _, task := range tasks {
		svcID, err = task.ServiceID()
		if err != nil {
			return err
		}

		if svcID != serviceID {
			if service.ServiceType == models.MySQLServiceType && task.Type == models.ScheduledMySQLBackupTask {
				continue
			}

			if service.ServiceType == models.MongoDBServiceType && task.Type == models.ScheduledMongoDBBackupTask {
				if task.Data.MongoDBBackupTask.ClusterName != service.Cluster {
					return errors.Wrapf(ErrLocationFolderPairAlreadyUsed, usedByScheduledTaskMsg, task.ID, serviceID)
				}
				continue
			}

			return errors.Wrapf(ErrLocationFolderPairAlreadyUsed, usedByScheduledTaskMsg, task.ID, serviceID)
		}
	}

	return nil
}
