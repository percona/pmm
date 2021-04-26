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

	"github.com/golang/protobuf/ptypes"
	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// ArtifactsService represents artifacts API.
type ArtifactsService struct {
	l  *logrus.Entry
	db *reform.DB
}

// NewArtifactsService creates new artifacts API service.
func NewArtifactsService(db *reform.DB) *ArtifactsService {
	return &ArtifactsService{
		l:  logrus.WithField("component", "management/backup/artifacts"),
		db: db,
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

	artifacts, err := models.FindArtifacts(q)
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
	for _, b := range artifacts {
		serviceIDs = append(serviceIDs, b.ServiceID)
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
	createdAt, err := ptypes.TimestampProto(a.CreatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert timestamp")
	}

	l, ok := locations[a.LocationID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert artifact with id '%s': no location id '%s' in the map", a.ID, a.LocationID)
	}

	s, ok := services[a.ServiceID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert artifact with id '%s': no service id '%s' in the map", a.ID, a.ServiceID)
	}

	dm, err := convertDataModel(a.DataModel)
	if err != nil {
		return nil, errors.Wrapf(err, "artifact id '%s'", a.ID)
	}

	status, err := convertBackupStatus(a.Status)
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
		ServiceName:  s.ServiceName,
		DataModel:    *dm,
		Status:       *status,
		CreatedAt:    createdAt,
	}, nil
}

// Check interfaces.
var (
	_ backupv1beta1.ArtifactsServer = (*ArtifactsService)(nil)
)
