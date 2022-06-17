// pmm-managed
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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// RemovalService manage removing of backup artifacts.
type RemovalService struct {
	l  *logrus.Entry
	db *reform.DB
	s3 s3
}

// NewRemovalService creates new backup removal service.
func NewRemovalService(db *reform.DB, s3 s3) *RemovalService {
	return &RemovalService{
		l:  logrus.WithField("component", "services/backup/removal"),
		db: db,
		s3: s3,
	}
}

// DeleteArtifact deletes specified artifact.
func (s *RemovalService) DeleteArtifact(ctx context.Context, artifactID string, removeFiles bool) error {
	artifactName, s3Config, err := s.beginDeletingArtifact(artifactID)
	if err != nil {
		return err
	}

	if s3Config != nil && removeFiles {
		if err := s.s3.RemoveRecursive(
			ctx,
			s3Config.Endpoint,
			s3Config.AccessKey,
			s3Config.SecretKey,
			s3Config.BucketName,
			// Recursive listing finds all the objects with the specified prefix.
			// There could be a problem e.g. when we have artifacts `backup-daily` and `backup-daily-1`, so
			// listing by prefix `backup-daily` gives us both artifacts.
			// To avoid such a situation we need to append a slash.
			artifactName+"/"); err != nil {
			if _, updateErr := models.UpdateArtifact(s.db.Querier, artifactID, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.FailedToDeleteBackupStatus),
			}); updateErr != nil {
				s.l.WithError(updateErr).
					Errorf("failed to set status %q for artifact %q", models.FailedToDeleteBackupStatus, artifactID)
			}

			return err
		}
	}

	return s.db.InTransaction(func(tx *reform.TX) error {
		restoreItems, err := models.FindRestoreHistoryItems(tx.Querier, models.RestoreHistoryItemFilters{
			ArtifactID: artifactID,
		})
		if err != nil {
			return err
		}

		for _, ri := range restoreItems {
			if err := models.RemoveRestoreHistoryItem(tx.Querier, ri.ID); err != nil {
				return err
			}
		}

		return models.DeleteArtifact(tx.Querier, artifactID)
	})
}

// beginDeletingArtifact checks if the artifact isn't in use at the moment and sets deleting status,
// so it will not be used to restore backup.
func (s *RemovalService) beginDeletingArtifact(
	artifactID string,
) (string, *models.S3LocationConfig, error) {
	var s3Config *models.S3LocationConfig
	var artifactName string
	if err := s.db.InTransaction(func(tx *reform.TX) error {
		artifact, err := s.canDeleteArtifact(tx.Querier, artifactID)
		if err != nil {
			return err
		}

		artifactName = artifact.Name

		inProgressStatus := models.InProgressRestoreStatus
		restoreItems, err := models.FindRestoreHistoryItems(tx.Querier, models.RestoreHistoryItemFilters{
			ArtifactID: artifactID,
			Status:     &inProgressStatus,
		})
		if err != nil {
			return err
		}

		if len(restoreItems) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Cannot delete artifact with ID %q: "+
				"artifact is used by currently running restore operation.", artifactID)
		}

		location, err := models.FindBackupLocationByID(tx.Querier, artifact.LocationID)
		if err != nil {
			return err
		}

		s3Config = location.S3Config

		if _, err := models.UpdateArtifact(tx.Querier, artifactID, models.UpdateArtifactParams{
			Status: models.BackupStatusPointer(models.DeletingBackupStatus),
		}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", nil, err
	}

	return artifactName, s3Config, nil
}

func (s *RemovalService) canDeleteArtifact(q *reform.Querier, artifactID string) (*models.Artifact, error) {
	artifact, err := models.FindArtifactByID(q, artifactID)
	switch {
	case err == nil:
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Errorf(codes.NotFound, "Artifact with ID %q not found.", artifactID)
	default:
		return nil, err
	}

	switch artifact.Status {
	case models.SuccessBackupStatus,
		models.ErrorBackupStatus,
		models.FailedToDeleteBackupStatus:
	case models.DeletingBackupStatus,
		models.InProgressBackupStatus,
		models.PausedBackupStatus,
		models.PendingBackupStatus:
		return nil, status.Errorf(codes.FailedPrecondition, "Artifact with ID %q isn't in the final state.", artifactID)
	default:
		return nil, status.Errorf(codes.Internal, "Unhandled status %q", artifact.Status)
	}

	return artifact, nil
}
