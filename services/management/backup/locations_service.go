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

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// LocationsService represents backup locations API.
type LocationsService struct {
	db *reform.DB
}

// NewLocationsService creates new backup locations API service.
func NewLocationsService(db *reform.DB) *LocationsService {
	return &LocationsService{
		db: db,
	}
}

// ListLocations returns list of all available backup locations.
func (s *LocationsService) ListLocations(ctx context.Context, req *backupv1beta1.ListLocationsRequest) (*backupv1beta1.ListLocationsResponse, error) {
	locations, err := models.FindBackupLocations(s.db.Querier)
	if err != nil {
		return nil, err
	}
	res := make([]*backupv1beta1.Location, len(locations))
	for i, location := range locations {
		loc, err := convertLocation(location)
		if err != nil {
			return nil, err
		}
		res[i] = loc
	}
	return &backupv1beta1.ListLocationsResponse{
		Locations: res,
	}, nil
}

// AddLocation adds new backup location.
func (s *LocationsService) AddLocation(ctx context.Context, req *backupv1beta1.AddLocationRequest) (*backupv1beta1.AddLocationResponse, error) {
	params := models.CreateBackupLocationParams{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.S3Config != nil {
		params.S3Config = &models.S3LocationConfig{
			Endpoint:   req.S3Config.Endpoint,
			AccessKey:  req.S3Config.AccessKey,
			SecretKey:  req.S3Config.SecretKey,
			BucketName: req.S3Config.BucketName,
		}
	}
	if req.PmmServerConfig != nil {
		params.PMMServerConfig = &models.PMMServerLocationConfig{
			Path: req.PmmServerConfig.Path,
		}
	}

	if req.PmmClientConfig != nil {
		params.PMMClientConfig = &models.PMMClientLocationConfig{
			Path: req.PmmClientConfig.Path,
		}
	}

	loc, err := models.CreateBackupLocation(s.db.Querier, params)
	if err != nil {
		return nil, err
	}

	return &backupv1beta1.AddLocationResponse{
		LocationId: loc.ID,
	}, nil
}

// ChangeLocation changes existing backup location.
func (s *LocationsService) ChangeLocation(ctx context.Context, req *backupv1beta1.ChangeLocationRequest) (*backupv1beta1.ChangeLocationResponse, error) {
	params := models.ChangeBackupLocationParams{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.S3Config != nil {
		params.S3Config = &models.S3LocationConfig{
			Endpoint:   req.S3Config.Endpoint,
			AccessKey:  req.S3Config.AccessKey,
			SecretKey:  req.S3Config.SecretKey,
			BucketName: req.S3Config.BucketName,
		}
	}

	if req.PmmServerConfig != nil {
		params.PMMServerConfig = &models.PMMServerLocationConfig{
			Path: req.PmmServerConfig.Path,
		}
	}

	if req.PmmClientConfig != nil {
		params.PMMClientConfig = &models.PMMClientLocationConfig{
			Path: req.PmmClientConfig.Path,
		}
	}

	_, err := models.ChangeBackupLocation(s.db.Querier, req.LocationId, params)
	if err != nil {
		return nil, err
	}

	return &backupv1beta1.ChangeLocationResponse{}, nil
}

func convertLocation(location *models.BackupLocation) (*backupv1beta1.Location, error) {
	loc := &backupv1beta1.Location{
		LocationId:  location.ID,
		Name:        location.Name,
		Description: location.Description,
	}
	switch location.Type {
	case models.PMMClientBackupLocationType:
		config := location.PMMClientConfig
		loc.Config = &backupv1beta1.Location_PmmClientConfig{
			PmmClientConfig: &backupv1beta1.PMMClientLocationConfig{
				Path: config.Path,
			},
		}
	case models.PMMServerBackupLocationType:
		config := location.PMMServerConfig
		loc.Config = &backupv1beta1.Location_PmmServerConfig{
			PmmServerConfig: &backupv1beta1.PMMServerLocationConfig{
				Path: config.Path,
			},
		}
	case models.S3BackupLocationType:
		config := location.S3Config
		loc.Config = &backupv1beta1.Location_S3Config{
			S3Config: &backupv1beta1.S3LocationConfig{
				Endpoint:   config.Endpoint,
				AccessKey:  config.AccessKey,
				SecretKey:  config.SecretKey,
				BucketName: config.BucketName,
			},
		}
	default:
		return nil, errors.Errorf("unknown backup location type %s", location.Type)
	}
	return loc, nil
}

// RemoveLocation removes backup location.
func (s *LocationsService) RemoveLocation(ctx context.Context, req *backupv1beta1.RemoveLocationRequest) (*backupv1beta1.RemoveLocationResponse, error) {
	mode := models.RemoveRestrict
	if req.Force {
		mode = models.RemoveCascade
	}
	if err := models.RemoveBackupLocation(s.db.Querier, req.LocationId, mode); err != nil {
		return nil, err
	}
	return &backupv1beta1.RemoveLocationResponse{}, nil
}

// Check interfaces.
var (
	_ backupv1beta1.LocationsServer = (*LocationsService)(nil)
)
