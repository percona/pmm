package services

import (
	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// CheckMongoDBBackupPreconditions checks compatibility of different types of scheduled backups and on-demand backups for MongoDB.
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
			if task.ID != scheduleID {
				return status.Errorf(codes.FailedPrecondition, "A PITR backup for cluster '%s' can be enabled only if there no other scheduled backups for this cluster.", clusterName)
			}
		}
	case models.Snapshot:
		// Snapshot backup can be enabled it there is no enabled PITR backup.
		filter.Mode = models.PITR
		tasks, err := models.FindScheduledTasks(q, filter)
		if err != nil {
			return err
		}

		if len(tasks) != 0 {
			return status.Errorf(codes.FailedPrecondition, "A snapshot backup for cluster '%s' can be done only if there is no enabled PITR backup for this cluster.", clusterName)
		}
	case models.Incremental:
		return status.Error(codes.InvalidArgument, "Incremental backups unsupported for MongoDB")
	}

	return nil
}
