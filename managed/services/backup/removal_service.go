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
	"database/sql"
	"path"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// RemovalService manage removing of backup artifacts.
type RemovalService struct {
	l  *logrus.Entry
	db *reform.DB
}

// NewRemovalService creates new backup removal service.
func NewRemovalService(db *reform.DB) *RemovalService {
	return &RemovalService{
		l:  logrus.WithField("component", "services/backup/removal"),
		db: db,
	}
}

// DeleteArtifact deletes specified artifact.
func (s *RemovalService) DeleteArtifact(ctx context.Context, artifactID string, removeFiles bool) error {
	var artifact *models.Artifact
	var err error

	if txErr := s.db.InTransactionContext(s.db.Querier.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
		artifact, err = models.FindArtifactByID(s.db.Querier, artifactID)
		if err != nil {
			return err
		}

		err = artifact.CanDelete()
		if err != nil {
			return err
		}

		err = s.beginDeletingArtifact(artifact)
		if err != nil {
			return err
		}

		return nil
	}); txErr != nil {
		return txErr
	}

	if removeFiles {
		err := s.DeleteArtifactFiles(ctx, artifact, len(artifact.StorageRecList))
		if err != nil {
			s.setFailedToDeleteBackupStatus(artifactID)
			return err
		}
	}

	return s.db.InTransaction(func(tx *reform.TX) error {
		restoreItems, err := models.FindRestoreHistoryItems(tx.Querier, models.RestoreHistoryItemFilters{
			ArtifactID: artifactID,
		})
		if err != nil {
			s.setFailedToDeleteBackupStatus(artifactID)
			return err
		}

		for _, ri := range restoreItems {
			if err := models.RemoveRestoreHistoryItem(tx.Querier, ri.ID); err != nil {
				s.setFailedToDeleteBackupStatus(artifactID)
				return err
			}
		}

		err = models.DeleteArtifact(tx.Querier, artifactID)
		if err != nil {
			s.setFailedToDeleteBackupStatus(artifactID)
		}
		return nil
	})
}

// beginDeletingArtifact checks if the artifact isn't in use at the moment and sets deleting status,
// so it will not be used to restore backup.
func (s *RemovalService) beginDeletingArtifact(artifact *models.Artifact) error {
	restoreItems, err := models.FindRestoreHistoryItems(s.db.Querier, models.RestoreHistoryItemFilters{
		ArtifactID: artifact.ID,
		Status:     models.RestoreStatusPointer(models.InProgressRestoreStatus),
	})
	if err != nil {
		return err
	}

	if len(restoreItems) != 0 {
		return status.Errorf(codes.FailedPrecondition, "Cannot delete artifact with ID %q: "+
			"artifact is used by currently running restore operation.", artifact.ID)
	}

	if _, err := models.UpdateArtifact(s.db.Querier, artifact.ID, models.UpdateArtifactParams{
		Status: models.BackupStatusPointer(models.DeletingBackupStatus),
	}); err != nil {
		return err
	}

	return nil
}

func (s *RemovalService) setFailedToDeleteBackupStatus(id string) {
	if _, updateErr := models.UpdateArtifact(s.db.Querier, id, models.UpdateArtifactParams{
		Status: models.BackupStatusPointer(models.FailedToDeleteBackupStatus),
	}); updateErr != nil {
		s.l.WithError(updateErr).Errorf("failed to set status %q for artifact %q", models.FailedToDeleteBackupStatus, id)
	}
}

// DeleteArtifactFiles deletes artifact files.
// If artifact represents a single snapshot, there is only one record representing artifact files.
// If artifact represents continuous backup (PITR), artifact may contain several records,
// and it's possible to delete only first N of them to implement retention policy.
func (s *RemovalService) DeleteArtifactFiles(ctx context.Context, artifact *models.Artifact, firstN int) error {
	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return err
	}

	storage := Location2Storage(location)
	s3Config := location.S3Config

	if storage == nil || s3Config == nil {
		return nil
	}

	// Old artifact records don't contain representation file list.
	if len(artifact.StorageRecList) == 0 {
		folderName := artifact.Name + "/"

		if err := storage.RemoveRecursive(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, folderName); err != nil {
			s.l.WithError(err).Errorf("failed to remove folder %s of artifact %s", folderName, artifact.ID)
		}

		return nil
	}

	for _, artifactRepr := range artifact.StorageRecList[:firstN] {
		for _, file := range artifactRepr.FileList {
			if file.IsDirectory {
				// Recursive listing finds all the objects with the specified prefix.
				// There could be a problem e.g. when we have artifacts `backup-daily` and `backup-daily-1`, so
				// listing by prefix `backup-daily` gives us both artifacts.
				// To avoid such a situation we need to append a slash.
				folderName := path.Join(*artifact.Folder, file.Name) + "/"

				if err := storage.RemoveRecursive(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, folderName); err != nil {
					s.l.WithError(err).Errorf("failed to remove folder %s of artifact %s", folderName, artifact.ID)
				}
			} else {
				fileName := path.Join(*artifact.Folder, file.Name)
				if err := storage.Remove(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, fileName); err != nil {
					s.l.WithError(err).Errorf("failed to remove file %s of artifact %s", file.Name, artifact.ID)
				}
			}
		}
	}

	return nil
}
