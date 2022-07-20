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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
)

// RestoreHistoryService represents restore history API.
type RestoreHistoryService struct {
	l  *logrus.Entry
	db *reform.DB

	backupv1beta1.UnimplementedRestoreHistoryServer
}

// NewRestoreHistoryService creates new restore history API service.
func NewRestoreHistoryService(db *reform.DB) *RestoreHistoryService {
	return &RestoreHistoryService{
		l:  logrus.WithField("component", "management/backup/restore_history"),
		db: db,
	}
}

// Enabled returns if service is enabled and can be used.
func (s *RestoreHistoryService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.BackupManagement.Enabled
}

// ListRestoreHistory returns a list of restore history.
func (s *RestoreHistoryService) ListRestoreHistory(
	context.Context,
	*backupv1beta1.ListRestoreHistoryRequest,
) (*backupv1beta1.ListRestoreHistoryResponse, error) {
	var items []*models.RestoreHistoryItem
	var services map[string]*models.Service
	var artifacts map[string]*models.Artifact
	var locations map[string]*models.BackupLocation

	err := s.db.InTransaction(func(tx *reform.TX) error {
		q := tx.Querier

		var err error
		items, err = models.FindRestoreHistoryItems(q, models.RestoreHistoryItemFilters{})
		if err != nil {
			return err
		}

		artifactIDs := make([]string, 0, len(items))
		serviceIDs := make([]string, 0, len(items))
		for _, i := range items {
			artifactIDs = append(artifactIDs, i.ArtifactID)
			serviceIDs = append(serviceIDs, i.ServiceID)
		}
		artifacts, err = models.FindArtifactsByIDs(q, artifactIDs)
		if err != nil {
			return err
		}

		locationIDs := make([]string, 0, len(artifacts))
		for _, a := range artifacts {
			locationIDs = append(locationIDs, a.LocationID)
		}
		locations, err = models.FindBackupLocationsByIDs(q, locationIDs)
		if err != nil {
			return err
		}

		services, err = models.FindServicesByIDs(q, serviceIDs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	artifactsResponse := make([]*backupv1beta1.RestoreHistoryItem, 0, len(artifacts))
	for _, i := range items {
		convertedArtifact, err := convertRestoreHistoryItem(i, services, artifacts, locations)
		if err != nil {
			return nil, err
		}

		artifactsResponse = append(artifactsResponse, convertedArtifact)
	}
	return &backupv1beta1.ListRestoreHistoryResponse{
		Items: artifactsResponse,
	}, nil
}

func convertRestoreStatus(status models.RestoreStatus) (*backupv1beta1.RestoreStatus, error) {
	var s backupv1beta1.RestoreStatus
	switch status {
	case models.InProgressRestoreStatus:
		s = backupv1beta1.RestoreStatus_RESTORE_STATUS_IN_PROGRESS
	case models.SuccessRestoreStatus:
		s = backupv1beta1.RestoreStatus_RESTORE_STATUS_SUCCESS
	case models.ErrorRestoreStatus:
		s = backupv1beta1.RestoreStatus_RESTORE_STATUS_ERROR
	default:
		return nil, errors.Errorf("invalid status '%s'", status)
	}

	return &s, nil
}

//nolint:funlen
func convertRestoreHistoryItem(
	i *models.RestoreHistoryItem,
	services map[string]*models.Service,
	artifacts map[string]*models.Artifact,
	locations map[string]*models.BackupLocation,
) (*backupv1beta1.RestoreHistoryItem, error) {
	startedAt := timestamppb.New(i.StartedAt)
	if err := startedAt.CheckValid(); err != nil {
		return nil, errors.Wrap(err, "failed to convert startedAt timestamp")
	}

	var finishedAt *timestamppb.Timestamp
	if i.FinishedAt != nil {
		finishedAt = timestamppb.New(*i.FinishedAt)
		if err := finishedAt.CheckValid(); err != nil {
			return nil, errors.Wrap(err, "failed to convert finishedAt timestamp")
		}
	}

	artifact, ok := artifacts[i.ArtifactID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert restore history item with id '%s': no artifact id '%s' in the map", i.ID, i.ArtifactID)
	}

	l, ok := locations[artifact.LocationID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert restore history item with id '%s': no location id '%s' in the map",
			i.ID, artifact.LocationID)
	}

	s, ok := services[i.ServiceID]
	if !ok {
		return nil, errors.Errorf(
			"failed to convert restore history item with id '%s': no service id '%s' in the map", i.ID, i.ServiceID)
	}

	dm, err := convertDataModel(artifact.DataModel)
	if err != nil {
		return nil, errors.Wrapf(err, "restore history item id '%s'", i.ID)
	}

	status, err := convertRestoreStatus(i.Status)
	if err != nil {
		return nil, errors.Wrapf(err, "restore history item id '%s'", i.ID)
	}

	return &backupv1beta1.RestoreHistoryItem{
		RestoreId:    i.ID,
		ArtifactId:   i.ArtifactID,
		Name:         artifact.Name,
		Vendor:       artifact.Vendor,
		LocationId:   artifact.LocationID,
		LocationName: l.Name,
		ServiceId:    i.ServiceID,
		ServiceName:  s.ServiceName,
		DataModel:    dm,
		Status:       *status,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
	}, nil
}

// Check interfaces.
var (
	_ backupv1beta1.RestoreHistoryServer = (*RestoreHistoryService)(nil)
)
