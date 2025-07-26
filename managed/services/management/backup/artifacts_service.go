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

	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
)

// ArtifactsService represents artifacts API.
type ArtifactsService struct {
	l              *logrus.Entry
	db             *reform.DB
	removalSVC     removalService
	pbmPITRService pbmPITRService

	backuppb.UnimplementedArtifactsServer
}

// NewArtifactsService creates new artifacts API service.
func NewArtifactsService(db *reform.DB, removalSVC removalService, pbmPITRService pbmPITRService) *ArtifactsService {
	return &ArtifactsService{
		l:              logrus.WithField("component", "management/backup/artifacts"),
		db:             db,
		removalSVC:     removalSVC,
		pbmPITRService: pbmPITRService,
	}
}

// Enabled returns if service is enabled and can be used.
func (s *ArtifactsService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return !settings.BackupManagement.Disabled
}

// ListArtifacts returns a list of all artifacts.
func (s *ArtifactsService) ListArtifacts(context.Context, *backuppb.ListArtifactsRequest) (*backuppb.ListArtifactsResponse, error) {
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

	artifactsResponse := make([]*backuppb.Artifact, 0, len(artifacts))
	for _, b := range artifacts {
		convertedArtifact, err := convertArtifact(b, services, locations)
		if err != nil {
			return nil, err
		}
		artifactsResponse = append(artifactsResponse, convertedArtifact)
	}
	return &backuppb.ListArtifactsResponse{
		Artifacts: artifactsResponse,
	}, nil
}

// DeleteArtifact deletes specified artifact and its files.
func (s *ArtifactsService) DeleteArtifact(
	ctx context.Context, //nolint:revive
	req *backuppb.DeleteArtifactRequest,
) (*backuppb.DeleteArtifactResponse, error) {
	artifact, err := models.FindArtifactByID(s.db.Querier, req.ArtifactId)
	if err != nil {
		return nil, err
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return nil, err
	}

	storage := backup.GetStorageForLocation(location)

	if err := s.removalSVC.DeleteArtifact(storage, req.ArtifactId, req.RemoveFiles); err != nil {
		return nil, err
	}
	return &backuppb.DeleteArtifactResponse{}, nil
}

// ListPitrTimeranges lists available PITR timelines/time-ranges (for MongoDB).
func (s *ArtifactsService) ListPitrTimeranges(
	ctx context.Context,
	req *backuppb.ListPitrTimerangesRequest,
) (*backuppb.ListPitrTimerangesResponse, error) {
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
		return nil, status.Errorf(codes.FailedPrecondition, "Artifact is not a PITR artifact.")
	}

	if artifact.IsShardedCluster {
		return nil, status.Errorf(codes.FailedPrecondition, "Getting PITR timeranges is not supported for sharded cluster artifacts.")
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return nil, err
	}

	storage := backup.GetStorageForLocation(location)

	timelines, err := s.pbmPITRService.ListPITRTimeranges(ctx, storage, location, artifact)
	if err != nil {
		return nil, err
	}
	result := make([]*backuppb.PitrTimerange, 0, len(timelines))
	for _, tl := range timelines {
		result = append(result, &backuppb.PitrTimerange{
			StartTimestamp: timestamppb.New(time.Unix(int64(tl.Start), 0)),
			EndTimestamp:   timestamppb.New(time.Unix(int64(tl.End), 0)),
		})
	}
	return &backuppb.ListPitrTimerangesResponse{
		Timeranges: result,
	}, nil
}

func convertDataModel(model models.DataModel) (backuppb.DataModel, error) {
	switch model {
	case models.PhysicalDataModel:
		return backuppb.DataModel_PHYSICAL, nil
	case models.LogicalDataModel:
		return backuppb.DataModel_LOGICAL, nil
	default:
		return 0, errors.Errorf("unknown data model: %s", model)
	}
}

func convertBackupStatus(status models.BackupStatus) (backuppb.BackupStatus, error) {
	switch status {
	case models.PendingBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_PENDING, nil
	case models.InProgressBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_IN_PROGRESS, nil
	case models.PausedBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_PAUSED, nil
	case models.SuccessBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_SUCCESS, nil
	case models.ErrorBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_ERROR, nil
	case models.DeletingBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_DELETING, nil
	case models.FailedToDeleteBackupStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_FAILED_TO_DELETE, nil
	case models.CleanupInProgressStatus:
		return backuppb.BackupStatus_BACKUP_STATUS_CLEANUP_IN_PROGRESS, nil
	default:
		return 0, errors.Errorf("invalid status '%s'", status)
	}
}

func convertBackupCompression(compression models.BackupCompression) (backuppb.BackupCompression, error) {
	switch compression {
	case models.QuickLZ:
		return backuppb.BackupCompression_QUICKLZ, nil
	case models.ZSTD:
		return backuppb.BackupCompression_ZSTD, nil
	case models.LZ4:
		return backuppb.BackupCompression_LZ4, nil
	default:
		return 0, nil
	}
}

func convertArtifact(
	a *models.Artifact,
	services map[string]*models.Service,
	locationModels map[string]*models.BackupLocation,
) (*backuppb.Artifact, error) {
	createdAt := timestamppb.New(a.CreatedAt)
	if err := createdAt.CheckValid(); err != nil {
		return nil, errors.Wrap(err, "failed to convert timestamp")
	}

	l, ok := locationModels[a.LocationID]
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

	compression, err := convertBackupCompression(a.Compression)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	return &backuppb.Artifact{
		ArtifactId:       a.ID,
		Name:             a.Name,
		Vendor:           a.Vendor,
		LocationId:       a.LocationID,
		LocationName:     l.Name,
		ServiceId:        a.ServiceID,
		ServiceName:      serviceName,
		DataModel:        dataModel,
		Mode:             backupMode,
		Status:           backupStatus,
		CreatedAt:        createdAt,
		IsShardedCluster: a.IsShardedCluster,
		Folder:           a.Folder,
		Compression:      compression,
		MetadataList:     artifactMetadataListToProto(a),
	}, nil
}

// artifactMetadataListToProto returns artifact metadata list in protobuf format.
func artifactMetadataListToProto(artifact *models.Artifact) []*backuppb.Metadata {
	res := make([]*backuppb.Metadata, len(artifact.MetadataList))
	for i, metadata := range artifact.MetadataList {
		res[i] = &backuppb.Metadata{}
		res[i].FileList = make([]*backuppb.File, len(metadata.FileList))

		for j, file := range metadata.FileList {
			res[i].FileList[j] = &backuppb.File{
				Name:        file.Name,
				IsDirectory: file.IsDirectory,
			}
		}

		if metadata.RestoreTo != nil {
			res[i].RestoreTo = timestamppb.New(*metadata.RestoreTo)
		}

		if metadata.BackupToolData != nil {
			if metadata.BackupToolData.PbmMetadata != nil {
				res[i].BackupToolMetadata = &backuppb.Metadata_PbmMetadata{
					PbmMetadata: &backuppb.PbmMetadata{Name: metadata.BackupToolData.PbmMetadata.Name},
				}
			}
		}
	}
	return res
}

// Check interfaces.
var (
	_ backuppb.ArtifactsServer = (*ArtifactsService)(nil)
)
