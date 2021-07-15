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

// Package backup provides backup functionality.
package backup

import (
	"context"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// ArtifactsService represents artifacts API.
type ArtifactsService struct {
	l  *logrus.Entry
	db *reform.DB
	s3 awsS3
}

// NewArtifactsService creates new artifacts API service.
func NewArtifactsService(db *reform.DB, s3 awsS3) *ArtifactsService {
	return &ArtifactsService{
		l:  logrus.WithField("component", "management/backup/artifacts"),
		db: db,
		s3: s3,
	}
}

// Enabled returns if service is enabled and can be used.
func (s *ArtifactsService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.BackupManagement.Enabled
}

// ListArtifacts returns a list of all artifacts.
func (s *ArtifactsService) ListArtifacts(context.Context, *backupv1beta1.ListArtifactsRequest) (*backupv1beta1.ListArtifactsResponse, error) {
	q := s.db.Querier

	artifacts, err := models.FindArtifacts(q, nil)
	if err != nil {
		return nil, err
	}

	locationIDs := make([]string, 0, len(artifacts))
	for _, b := range artifacts {
		locationIDs = append(locationIDs, b.LocationID)
	}
	locations, err := models.FindBackupLocationsByIDs(q, locationIDs)
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, 0, len(artifacts))
	for _, a := range artifacts {
		if a.ServiceID != "" {
			serviceIDs = append(serviceIDs, a.ServiceID)
		}
	}

	services, err := models.FindServicesByIDs(q, serviceIDs)
	if err != nil {
		return nil, err
	}

	artifactsResponse := make([]*backupv1beta1.Artifact, 0, len(artifacts))
	for _, b := range artifacts {
		convertedArtifact, err := convertArtifact(b, services, locations)
		if err != nil {
			return nil, err
		}
		artifactsResponse = append(artifactsResponse, convertedArtifact)
	}
	return &backupv1beta1.ListArtifactsResponse{
		Artifacts: artifactsResponse,
	}, nil
}

func (s *ArtifactsService) canDeleteArtifact(q *reform.Querier, artifactID string) (*models.Artifact, error) {
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

// beginDeletingArtifact checks if the artifact isn't in use at the moment and sets deleting status,
// so it will not be used to restore backup.
func (s *ArtifactsService) beginDeletingArtifact(
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

// DeleteArtifact deletes specified artifact.
func (s *ArtifactsService) DeleteArtifact(
	ctx context.Context,
	req *backupv1beta1.DeleteArtifactRequest,
) (*backupv1beta1.DeleteArtifactResponse, error) {
	artifactName, s3Config, err := s.beginDeletingArtifact(req.ArtifactId)
	if err != nil {
		return nil, err
	}

	if req.RemoveFiles && s3Config != nil {
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
			artifactName+"/",
		); err != nil {
			if _, updateErr := models.UpdateArtifact(s.db.Querier, req.ArtifactId, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.FailedToDeleteBackupStatus),
			}); updateErr != nil {
				s.l.WithError(updateErr).
					Errorf("failed to set status %q for artifact %q", models.FailedToDeleteBackupStatus, req.ArtifactId)
			}

			return nil, err
		}
	}

	if err := s.db.InTransaction(func(tx *reform.TX) error {
		restoreItems, err := models.FindRestoreHistoryItems(tx.Querier, models.RestoreHistoryItemFilters{
			ArtifactID: req.ArtifactId,
		})
		if err != nil {
			return err
		}

		for _, ri := range restoreItems {
			if err := models.RemoveRestoreHistoryItem(tx.Querier, ri.ID); err != nil {
				return err
			}
		}

		return models.DeleteArtifact(tx.Querier, req.ArtifactId)
	}); err != nil {
		return nil, err
	}

	return &backupv1beta1.DeleteArtifactResponse{}, nil
}

func convertDataModel(dataModel models.DataModel) (*backupv1beta1.DataModel, error) {
	var dm backupv1beta1.DataModel
	switch dataModel {
	case models.PhysicalDataModel:
		dm = backupv1beta1.DataModel_PHYSICAL
	case models.LogicalDataModel:
		dm = backupv1beta1.DataModel_LOGICAL
	default:
		return nil, errors.Errorf("invalid data model '%s'", dataModel)
	}

	return &dm, nil
}

func convertBackupStatus(status models.BackupStatus) (*backupv1beta1.BackupStatus, error) {
	var s backupv1beta1.BackupStatus
	switch status {
	case models.PendingBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_PENDING
	case models.InProgressBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_IN_PROGRESS
	case models.PausedBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_PAUSED
	case models.SuccessBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_SUCCESS
	case models.ErrorBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_ERROR
	case models.DeletingBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_DELETING
	case models.FailedToDeleteBackupStatus:
		s = backupv1beta1.BackupStatus_BACKUP_STATUS_FAILED_TO_DELETE
	default:
		return nil, errors.Errorf("invalid status '%s'", status)
	}

	return &s, nil
}

func convertArtifact(
	a *models.Artifact,
	services map[string]*models.Service,
	locations map[string]*models.BackupLocation,
) (*backupv1beta1.Artifact, error) {
	createdAt := timestamppb.New(a.CreatedAt)
	if err := createdAt.CheckValid(); err != nil {
		return nil, errors.Wrap(err, "failed to convert timestamp")
	}

	l, ok := locations[a.LocationID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert artifact with id '%s': no location id '%s' in the map", a.ID, a.LocationID)
	}

	var serviceName string
	if s, ok := services[a.ServiceID]; ok {
		serviceName = s.ServiceName
	}

	dm, err := convertDataModel(a.DataModel)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	backupStatus, err := convertBackupStatus(a.Status)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	return &backupv1beta1.Artifact{
		ArtifactId:   a.ID,
		Name:         a.Name,
		Vendor:       a.Vendor,
		LocationId:   a.LocationID,
		LocationName: l.Name,
		ServiceId:    a.ServiceID,
		ServiceName:  serviceName,
		DataModel:    *dm,
		Status:       *backupStatus,
		CreatedAt:    createdAt,
	}, nil
}

// Check interfaces.
var (
	_ backupv1beta1.ArtifactsServer = (*ArtifactsService)(nil)
)
