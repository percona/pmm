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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// RetentionService handles retention for artifacts.
type RetentionService struct {
	db         *reform.DB
	l          *logrus.Entry
	removalSVC removalService
}

// NewRetentionService creates new retention service for artifacts.
func NewRetentionService(db *reform.DB, removalSVC removalService) *RetentionService {
	return &RetentionService{
		l:          logrus.WithField("component", "management/backup/retention"),
		db:         db,
		removalSVC: removalSVC,
	}
}

// EnforceRetention enforce retention on provided scheduled backup task
// it removes any old successful artifacts below retention threshold.
func (s *RetentionService) EnforceRetention(scheduleID string) error {
	task, err := models.FindScheduledTaskByID(s.db.Querier, scheduleID)
	if err != nil {
		return err
	}

	retention, err := task.Retention()
	if err != nil {
		return err
	}

	if retention == 0 {
		return nil
	}

	mode, err := task.Mode()
	if err != nil {
		return err
	}

	locationID, err := task.LocationID()
	if err != nil {
		return err
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, locationID)
	if err != nil {
		return err
	}

	storage := GetStorageForLocation(location)

	switch mode {
	case models.Snapshot:
		err = s.retentionSnapshot(storage, scheduleID, retention)
	case models.PITR:
		err = s.retentionPITR(storage, scheduleID, retention)
	default:
		s.l.Warnf("Retention policy is not implemented for backup mode %s", mode)
		return nil
	}

	return err
}

func (s *RetentionService) retentionSnapshot(storage Storage, scheduleID string, retention uint32) error {
	artifacts, err := models.FindArtifacts(s.db.Querier, models.ArtifactFilters{
		ScheduleID: scheduleID,
		Status:     models.SuccessBackupStatus,
	})
	if err != nil {
		return err
	}

	if int(retention) >= len(artifacts) {
		return nil
	}

	for _, artifact := range artifacts[retention:] {
		if err := s.removalSVC.DeleteArtifact(storage, artifact.ID, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *RetentionService) retentionPITR(storage Storage, scheduleID string, retention uint32) error {
	artifacts, err := models.FindArtifacts(s.db.Querier, models.ArtifactFilters{
		ScheduleID: scheduleID,
		Status:     models.SuccessBackupStatus,
	})
	if err != nil {
		return err
	}

	if len(artifacts) == 0 {
		return nil
	}

	if len(artifacts) > 1 {
		return errors.Errorf("Can be only one artifact entity for PITR in the database but found %d", len(artifacts))
	}

	artifact := artifacts[0]
	trimBy := len(artifact.MetadataList) - int(retention)
	if trimBy <= 0 {
		return nil
	}

	return s.removalSVC.TrimPITRArtifact(storage, artifact.ID, trimBy)
}
