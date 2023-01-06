// Copyright (C) 2022 Percona LLC
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// CheckMongoDBBackupPreconditions checks compatibility of different types of scheduled backups and on-demand backups for MongoDB.
// WARNING: This function valid only when executed as part of transaction with serializable isolation level.
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
				// Needed for scheduled PITR backup update.
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
		// Snapshot backup can be enabled if there is no enabled PITR backup.
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
			return status.Errorf(codes.FailedPrecondition, "A snapshot backup for service '%s' can be enabled/done only if "+
				"there are no other scheduled backups for this service.", serviceID)
		}

		return status.Errorf(codes.FailedPrecondition, "A snapshot backup for cluster '%s' can be enabled/done only if "+
			"there is no enabled PITR backup for this cluster.", clusterName)

	case models.Incremental:
		return status.Error(codes.InvalidArgument, "Incremental backups unsupported for MongoDB")
	}

	return nil
}
