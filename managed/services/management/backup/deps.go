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

package backup

import (
	"context"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/scheduler"
)

//go:generate ../../../../bin/mockery -name=awsS3 -case=snake -inpkg -testonly
//go:generate ../../../../bin/mockery -name=backupService -case=snake -inpkg -testonly
//go:generate ../../../../bin/mockery -name=scheduleService -case=snake -inpkg -testonly
//go:generate ../../../../bin/mockery -name=removalService -case=snake -inpkg -testonly
//go:generate ../../../../bin/mockery -name=pitrStorageService -case=snake -inpkg -testonly

type awsS3 interface {
	GetBucketLocation(ctx context.Context, host string, accessKey, secretKey, name string) (string, error)
	BucketExists(ctx context.Context, host string, accessKey, secretKey, name string) (bool, error)
	RemoveRecursive(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix string) error
}

type backupService interface {
	PerformBackup(ctx context.Context, params backup.PerformBackupParams) (string, error)
	RestoreBackup(ctx context.Context, serviceID, artifactID string) (string, error)
	SwitchMongoPITR(ctx context.Context, serviceID string, enabled bool) error
	FindArtifactCompatibleServices(ctx context.Context, artifactID string) ([]*models.Service, error)
}

// schedulerService is a subset of method of scheduler.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type scheduleService interface {
	Run(ctx context.Context)
	Add(task scheduler.Task, params scheduler.AddParams) (*models.ScheduledTask, error)
	Remove(id string) error
	Update(id string, params models.ChangeScheduledTaskParams) error
}

type removalService interface {
	DeleteArtifact(ctx context.Context, artifactID string, removeFiles bool) error
}

// pitrStorageService provides methods that help us inspect PITR artifacts
type pitrStorageService interface {
	// ListPITRTimeranges list the available PITR timeranges for the given artifact in the provided location
	ListPITRTimeranges(ctx context.Context, artifactName string, location *models.BackupLocation) ([]backup.Timeline, error)
}
