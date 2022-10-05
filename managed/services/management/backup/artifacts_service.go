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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
)

// ArtifactsService represents artifacts API.
type ArtifactsService struct {
	l              *logrus.Entry
	db             *reform.DB
	removalSVC     removalService
	pitrStorageSVC pitrStorageService

	backupv1beta1.UnimplementedArtifactsServer
}

// NewArtifactsService creates new artifacts API service.
func NewArtifactsService(db *reform.DB, removalSVC removalService, storage pitrStorageService) *ArtifactsService {
	return &ArtifactsService{
		l:              logrus.WithField("component", "management/backup/artifacts"),
		db:             db,
		removalSVC:     removalSVC,
		pitrStorageSVC: storage,
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

	artifacts, err := models.FindArtifacts(q, models.ArtifactFilters{})
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

// DeleteArtifact deletes specified artifact.
func (s *ArtifactsService) DeleteArtifact(
	ctx context.Context,
	req *backupv1beta1.DeleteArtifactRequest,
) (*backupv1beta1.DeleteArtifactResponse, error) {
	if err := s.removalSVC.DeleteArtifact(ctx, req.ArtifactId, req.RemoveFiles); err != nil {
		return nil, err
	}

	return &backupv1beta1.DeleteArtifactResponse{}, nil
}

// ListPitrTimeranges lists available PITR timelines/time-ranges (for MongoDB)
func (s *ArtifactsService) ListPitrTimeranges(
	ctx context.Context,
	req *backupv1beta1.ListPitrTimerangesRequest,
) (*backupv1beta1.ListPitrTimerangesResponse, error) {
	var artifact *models.Artifact
	var err error

	artifact, err = models.FindArtifactByID(s.db.Querier, req.ArtifactId)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "Artifact with ID %q not found.", req.ArtifactId)
		}
		return nil, err
	}

	if artifact.Mode != models.PITR {
		return nil, status.Errorf(codes.FailedPrecondition, "Artifact is not a PITR artifact")
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return nil, err
	}

	timelines, err := s.pitrStorageSVC.ListPITRTimeranges(ctx, artifact.Name, *location)
	if err != nil {
		return nil, err
	}
	result := make([]*backupv1beta1.PitrTimerange, 0, len(timelines))
	for _, tl := range timelines {
		result = append(result, &backupv1beta1.PitrTimerange{
			StartTimestamp: timestamppb.New(time.Unix(int64(tl.Start), 0)),
			EndTimestamp:   timestamppb.New(time.Unix(int64(tl.End), 0)),
		})
	}
	return &backupv1beta1.ListPitrTimerangesResponse{
		Timeranges: result,
	}, nil
}

func convertDataModel(model models.DataModel) (backupv1beta1.DataModel, error) {
	switch model {
	case models.PhysicalDataModel:
		return backupv1beta1.DataModel_PHYSICAL, nil
	case models.LogicalDataModel:
		return backupv1beta1.DataModel_LOGICAL, nil
	default:
		return 0, errors.Errorf("unknown data model: %s", model)
	}
}

func convertBackupStatus(status models.BackupStatus) (backupv1beta1.BackupStatus, error) {
	switch status {
	case models.PendingBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_PENDING, nil
	case models.InProgressBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_IN_PROGRESS, nil
	case models.PausedBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_PAUSED, nil
	case models.SuccessBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_SUCCESS, nil
	case models.ErrorBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_ERROR, nil
	case models.DeletingBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_DELETING, nil
	case models.FailedToDeleteBackupStatus:
		return backupv1beta1.BackupStatus_BACKUP_STATUS_FAILED_TO_DELETE, nil
	default:
		return 0, errors.Errorf("invalid status '%s'", status)
	}
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

	dataModel, err := convertDataModel(a.DataModel)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	backupStatus, err := convertBackupStatus(a.Status)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	backupMode, err := convertModelToBackupMode(a.Mode)
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
		DataModel:    dataModel,
		Mode:         backupMode,
		Status:       backupStatus,
		CreatedAt:    createdAt,
	}, nil
}

// Check interfaces.
var (
	_ backupv1beta1.ArtifactsServer = (*ArtifactsService)(nil)
)
