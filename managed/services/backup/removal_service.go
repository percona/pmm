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

package backup

import (
	"context"
	"database/sql"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// RemovalService manages removing of backup artifacts.
type RemovalService struct {
	l              *logrus.Entry
	db             *reform.DB
	pbmPITRService pbmPITRService
}

// NewRemovalService creates new backup removal service.
func NewRemovalService(db *reform.DB, pbmPITRService pbmPITRService) *RemovalService {
	return &RemovalService{
		l:              logrus.WithField("component", "services/backup/removal"),
		db:             db,
		pbmPITRService: pbmPITRService,
	}
}

// DeleteArtifact deletes specified artifact along with files if specified.
func (s *RemovalService) DeleteArtifact(storage Storage, artifactID string, removeFiles bool) error {
	artifact, prevStatus, err := s.lockArtifact(artifactID, models.DeletingBackupStatus)
	if err != nil {
		return err
	}

	// For cases when it's not clear can files be removed or not - cannot tell new from legacy artifacts.
	if prevStatus != models.SuccessBackupStatus && len(artifact.MetadataList) == 0 {
		removeFiles = false
	}

	defer func() {
		if err != nil {
			_ = s.setArtifactStatus(artifactID, models.FailedToDeleteBackupStatus)
		}
	}()

	restoreItems, err := models.FindRestoreHistoryItems(s.db.Querier, models.RestoreHistoryItemFilters{
		ArtifactID: artifactID,
	})
	if err != nil {
		return err
	}

	for _, ri := range restoreItems {
		if err = models.RemoveRestoreHistoryItem(s.db.Querier, ri.ID); err != nil {
			return err
		}
	}

	removeFilesHelper := func() {
		var err error
		defer func() {
			if err != nil {
				_ = s.setArtifactStatus(artifactID, models.FailedToDeleteBackupStatus)
			}
		}()

		location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
		if err != nil {
			s.l.WithError(err).Error("couldn't get location")
			return
		}

		if err = s.deleteArtifactFiles(context.Background(), storage, location, artifact, len(artifact.MetadataList)); err != nil {
			s.l.WithError(err).Error("couldn't delete artifact files")
			return
		}

		if artifact.Vendor == string(models.MongoDBServiceType) && artifact.Mode == models.PITR {
			if err = s.deleteArtifactPITRChunks(context.Background(), storage, location, artifact, nil); err != nil {
				s.l.WithError(err).Error("couldn't delete artifact PITR chunks")
				return
			}
		}

		err = models.DeleteArtifact(s.db.Querier, artifactID)
		if err != nil {
			s.l.WithError(err).Error("couldn't delete artifact")
		}
	}

	if removeFiles {
		go removeFilesHelper()
		return nil
	}

	err = models.DeleteArtifact(s.db.Querier, artifactID)
	if err != nil {
		return err
	}

	return nil
}

// TrimPITRArtifact removes first N first records from PITR artifact. Removes snapshots, PITR chunks and corresponding data from database.
func (s *RemovalService) TrimPITRArtifact(storage Storage, artifactID string, firstN int) error {
	artifact, oldStatus, err := s.lockArtifact(artifactID, models.CleanupInProgressStatus)
	if err != nil {
		return err
	}

	go func() {
		var err error
		defer func() {
			if err != nil {
				s.l.Error("Couldn't trim artifact files. Restoring is not guaranteed for files outside of retention policy limit.")
				// We need to release PITR artifact in case of error, otherwise it will be blocked for restoring.
				if err = s.releaseArtifact(artifactID, oldStatus); err != nil {
					s.l.WithError(err).Errorf("couldn't unlock artifact %q", artifactID)
					return
				}
			}
		}()

		location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
		if err != nil {
			return
		}

		if err = s.deleteArtifactFiles(context.Background(), storage, location, artifact, firstN); err != nil {
			s.l.WithError(err).Error("couldn't delete artifact files")
			return
		}

		if err = artifact.MetadataRemoveFirstN(s.db.Querier, uint32(firstN)); err != nil {
			s.l.WithError(err).Error("couldn't delete artifact metadata")
			return
		}

		if err = s.deleteArtifactPITRChunks(context.Background(), storage, location, artifact, artifact.MetadataList[0].RestoreTo); err != nil {
			s.l.WithError(err).Error("couldn't delete artifact PITR chunks")
			return
		}

		if err = s.releaseArtifact(artifactID, oldStatus); err != nil {
			s.l.WithError(err).Errorf("couldn't unlock artifact %q", artifactID)
			return
		}
	}()

	return nil
}

// lockArtifact checks if the artifact isn't in use at the moment and sets deleting status,
// so it will not be used to restore backup.
func (s *RemovalService) lockArtifact(artifactID string, lockingStatus models.BackupStatus) (*models.Artifact, models.BackupStatus, error) {
	var currentStatus models.BackupStatus

	if models.IsArtifactFinalStatus(lockingStatus) {
		return nil, "", errors.Wrapf(ErrIncorrectArtifactStatus, "couldn't lock artifact, requested new status %s (present in list of final statuses) for artifact %s",
			lockingStatus, artifactID)
	}

	var (
		artifact *models.Artifact
		err      error
	)

	if errTx := s.db.InTransactionContext(s.db.Querier.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
		artifact, err = models.FindArtifactByID(tx.Querier, artifactID)
		if err != nil {
			return err
		}

		currentStatus = artifact.Status

		if !models.IsArtifactFinalStatus(artifact.Status) {
			return errors.Wrapf(ErrIncorrectArtifactStatus, "artifact with ID %q isn't in a final status", artifact.ID)
		}

		restoreItems, err := models.FindRestoreHistoryItems(tx.Querier, models.RestoreHistoryItemFilters{
			ArtifactID: artifact.ID,
			Status:     models.InProgressRestoreStatus.Pointer(),
		})
		if err != nil {
			return err
		}

		if len(restoreItems) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Cannot delete artifact with ID %q: "+
				"artifact is used by currently running restore operation.", artifact.ID)
		}

		if _, err := models.UpdateArtifact(tx.Querier, artifact.ID, models.UpdateArtifactParams{
			Status: lockingStatus.Pointer(),
		}); err != nil {
			return err
		}

		return nil
	}); errTx != nil {
		return nil, "", errTx
	}

	return artifact, currentStatus, nil
}

// releaseArtifact releases artifact lock by setting one of the final artifact statuses.
func (s *RemovalService) releaseArtifact(artifactID string, setStatus models.BackupStatus) error {
	if !models.IsArtifactFinalStatus(setStatus) {
		return errors.Wrapf(ErrIncorrectArtifactStatus, "couldn't release artifact, requested new status %s (not present in list of final statuses) for artifact %s",
			setStatus, artifactID)
	}

	if err := s.setArtifactStatus(artifactID, setStatus); err != nil {
		return err
	}
	return nil
}

// setArtifactStatus sets provided artifact status. Write error logs if status cannot be set.
func (s *RemovalService) setArtifactStatus(artifactID string, status models.BackupStatus) error {
	if _, err := models.UpdateArtifact(s.db.Querier, artifactID, models.UpdateArtifactParams{
		Status: status.Pointer(),
	}); err != nil {
		s.l.WithError(err).Errorf("failed to set status %q for artifact %q", status, artifactID)
		return err
	}
	return nil
}

// deleteArtifactFiles deletes artifact files.
// If artifact represents a single snapshot, there is only one record representing artifact files.
// If artifact represents continuous backup (PITR), artifact may contain several records,
// and it's possible to delete only first N of them to implement retention policy.
func (s *RemovalService) deleteArtifactFiles(ctx context.Context, storage Storage, location *models.BackupLocation, artifact *models.Artifact, firstN int) error {
	s3Config := location.S3Config
	if storage == nil || s3Config == nil {
		s.l.Debug("Storage not specified.")
		return nil
	}

	// Old artifact records don't contain representation file list.
	if len(artifact.MetadataList) == 0 {
		folderName := artifact.Name + "/"

		s.l.Debugf("Deleting folder %s.", folderName)
		if err := storage.RemoveRecursive(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, folderName); err != nil {
			return errors.Wrapf(err, "failed to remove folder %s of artifact %s", folderName, artifact.ID)
		}

		return nil
	}

	for _, metadata := range artifact.MetadataList[:firstN] {
		for _, file := range metadata.FileList {
			if file.IsDirectory {
				// Recursive listing finds all the objects with the specified prefix.
				// There could be a problem e.g. when we have artifacts `backup-daily` and `backup-daily-1`, so
				// listing by prefix `backup-daily` gives us both artifacts.
				// To avoid such a situation we need to append a slash.
				folderName := path.Join(artifact.Folder, file.Name) + "/"
				s.l.Debugf("Deleting folder %s.", folderName)
				if err := storage.RemoveRecursive(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, folderName); err != nil {
					return errors.Wrapf(err, "failed to remove folder %s of artifact %s", folderName, artifact.ID)
				}
			} else {
				fileName := path.Join(artifact.Folder, file.Name)
				s.l.Debugf("Deleting file %s.", fileName)
				if err := storage.Remove(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, fileName); err != nil {
					return errors.Wrapf(err, "failed to remove file %s of artifact %s", file.Name, artifact.ID)
				}
			}
		}
	}

	return nil
}

// deleteArtifactPITRChunks deletes artifact PITR chunks. If "until" provided, deletes only chunks created before that time. Deletes all artifact chunks otherwise.
func (s *RemovalService) deleteArtifactPITRChunks(
	ctx context.Context,
	storage Storage,
	location *models.BackupLocation,
	artifact *models.Artifact,
	until *time.Time,
) error {
	s3Config := location.S3Config
	if storage == nil || s3Config == nil {
		s.l.Debug("Storage not specified.")
		return nil
	}

	chunks, err := s.pbmPITRService.GetPITRFiles(ctx, storage, location, artifact, until)
	if err != nil {
		return errors.Wrap(err, "failed to get pitr chunks")
	}

	if len(chunks) == 0 {
		s.l.Debug("No chunks to delete.")
		return nil
	}

	for _, chunk := range chunks {
		s.l.Debugf("Deleting %s.", chunk.FName)

		if err := storage.Remove(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, chunk.FName); err != nil {
			return errors.Wrapf(err, "failed to remove pitr chunk '%s' (%v) from storage", chunk.FName, chunk)
		}
	}

	return nil
}
