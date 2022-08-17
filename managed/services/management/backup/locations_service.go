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

	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
)

// LocationsService represents backup locations API.
type LocationsService struct {
	db *reform.DB
	s3 awsS3
	l  *logrus.Entry

	backupv1beta1.UnimplementedLocationsServer
}

// NewLocationsService creates new backup locations API service.
func NewLocationsService(db *reform.DB, s3 awsS3) *LocationsService {
	return &LocationsService{
		l:  logrus.WithField("component", "management/backup/locations"),
		db: db,
		s3: s3,
	}
}

// Enabled returns if service is enabled and can be used.
func (s *LocationsService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.BackupManagement.Enabled
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

	if err := params.Validate(models.BackupLocationValidationParams{
		RequireConfig:    true,
		WithBucketRegion: false,
	}); err != nil {
		return nil, err
	}

	if params.S3Config != nil {
		bucketLocation, err := s.getBucketLocation(ctx, params.S3Config)
		if err != nil {
			return nil, err
		}

		params.S3Config.BucketRegion = bucketLocation
	}

	var locationModel *models.BackupLocation
	err := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		locationModel, err = models.CreateBackupLocation(tx.Querier, params)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &backupv1beta1.AddLocationResponse{
		LocationId: locationModel.ID,
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
	if err := params.Validate(models.BackupLocationValidationParams{
		RequireConfig:    false,
		WithBucketRegion: false,
	}); err != nil {
		return nil, err
	}

	if params.S3Config != nil {
		bucketLocation, err := s.getBucketLocation(ctx, params.S3Config)
		if err != nil {
			return nil, err
		}

		params.S3Config.BucketRegion = bucketLocation
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		_, err := models.ChangeBackupLocation(tx.Querier, req.LocationId, params)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &backupv1beta1.ChangeLocationResponse{}, nil
}

// TestLocationConfig tests backup location and credentials.
func (s *LocationsService) TestLocationConfig(
	ctx context.Context,
	req *backupv1beta1.TestLocationConfigRequest,
) (*backupv1beta1.TestLocationConfigResponse, error) {
	var locationConfig models.BackupLocationConfig

	if req.S3Config != nil {
		locationConfig.S3Config = &models.S3LocationConfig{
			Endpoint:   req.S3Config.Endpoint,
			AccessKey:  req.S3Config.AccessKey,
			SecretKey:  req.S3Config.SecretKey,
			BucketName: req.S3Config.BucketName,
		}
	}

	if req.PmmServerConfig != nil {
		locationConfig.PMMServerConfig = &models.PMMServerLocationConfig{
			Path: req.PmmServerConfig.Path,
		}
	}

	if req.PmmClientConfig != nil {
		locationConfig.PMMClientConfig = &models.PMMClientLocationConfig{
			Path: req.PmmClientConfig.Path,
		}
	}

	if err := locationConfig.Validate(models.BackupLocationValidationParams{
		RequireConfig:    true,
		WithBucketRegion: false,
	}); err != nil {
		return nil, err
	}

	if req.S3Config != nil {
		if err := s.checkBucket(ctx, locationConfig.S3Config); err != nil {
			return nil, err
		}
	}

	return &backupv1beta1.TestLocationConfigResponse{}, nil
}

// RemoveLocation removes backup location.
func (s *LocationsService) RemoveLocation(ctx context.Context, req *backupv1beta1.RemoveLocationRequest) (*backupv1beta1.RemoveLocationResponse, error) {
	mode := models.RemoveRestrict
	if req.Force {
		mode = models.RemoveCascade
	}

	err := s.db.InTransaction(func(tx *reform.TX) error {
		return models.RemoveBackupLocation(tx.Querier, req.LocationId, mode)
	})
	if err != nil {
		return nil, err
	}

	return &backupv1beta1.RemoveLocationResponse{}, nil
}

func convertLocation(locationModel *models.BackupLocation) (*backupv1beta1.Location, error) {
	loc := &backupv1beta1.Location{
		LocationId:  locationModel.ID,
		Name:        locationModel.Name,
		Description: locationModel.Description,
	}
	switch locationModel.Type {
	case models.PMMClientBackupLocationType:
		config := locationModel.PMMClientConfig
		loc.Config = &backupv1beta1.Location_PmmClientConfig{
			PmmClientConfig: &backupv1beta1.PMMClientLocationConfig{
				Path: config.Path,
			},
		}
	case models.PMMServerBackupLocationType:
		config := locationModel.PMMServerConfig
		loc.Config = &backupv1beta1.Location_PmmServerConfig{
			PmmServerConfig: &backupv1beta1.PMMServerLocationConfig{
				Path: config.Path,
			},
		}
	case models.S3BackupLocationType:
		config := locationModel.S3Config
		loc.Config = &backupv1beta1.Location_S3Config{
			S3Config: &backupv1beta1.S3LocationConfig{
				Endpoint:   config.Endpoint,
				AccessKey:  config.AccessKey,
				SecretKey:  config.SecretKey,
				BucketName: config.BucketName,
			},
		}
	default:
		return nil, errors.Errorf("unknown backup location type %s", locationModel.Type)
	}
	return loc, nil
}

func (s *LocationsService) getBucketLocation(ctx context.Context, c *models.S3LocationConfig) (string, error) {
	bucketLocation, err := s.s3.GetBucketLocation(ctx, c.Endpoint, c.AccessKey, c.SecretKey, c.BucketName)
	if err != nil {
		if minioErr, ok := err.(minio.ErrorResponse); ok {
			return "", status.Errorf(codes.InvalidArgument, "%s: %s.", minioErr.Code, minioErr.Message)
		}
		return "", status.Errorf(codes.Internal, "%s", err)
	}

	return bucketLocation, nil
}

func (s *LocationsService) checkBucket(ctx context.Context, c *models.S3LocationConfig) error {
	exists, err := s.s3.BucketExists(ctx, c.Endpoint, c.AccessKey, c.SecretKey, c.BucketName)
	if err != nil {
		if minioErr, ok := err.(minio.ErrorResponse); ok {
			return status.Errorf(codes.InvalidArgument, "%s: %s.", minioErr.Code, minioErr.Message)
		}

		return status.Error(codes.Internal, err.Error())
	}

	if !exists {
		return status.Errorf(codes.InvalidArgument, "Bucket doesn't exist")
	}

	return nil
}

// Check interfaces.
var (
	_ backupv1beta1.LocationsServer = (*LocationsService)(nil)
)
