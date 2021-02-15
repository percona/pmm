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

package models

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkUniqueBackupLocationID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Location ID")
	}

	location := &BackupLocation{ID: id}
	switch err := q.Reload(location); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Location with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func checkUniqueBackupLocationName(q *reform.Querier, name string) error {
	if name == "" {
		panic("empty Location Name")
	}

	var location BackupLocation
	switch err := q.FindOneTo(&location, "name", name); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Location with name %q already exists.", name)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

func checkPMMServerLocationConfig(c *PMMServerLocationConfig) error {
	if c == nil {
		return status.Error(codes.InvalidArgument, "PMM server location config is empty.")
	}
	if c.Path == "" {
		return status.Error(codes.InvalidArgument, "PMM server config path field is empty.")
	}
	return nil
}

func checkPMMClientLocationConfig(c *PMMClientLocationConfig) error {
	if c == nil {
		return status.Error(codes.InvalidArgument, "PMM client location config is empty.")
	}
	if c.Path == "" {
		return status.Error(codes.InvalidArgument, "PMM client config path field is empty.")
	}
	return nil
}

func checkS3Config(c *S3LocationConfig) error {
	if c == nil {
		return status.Error(codes.InvalidArgument, "S3 location config is empty.")
	}

	if c.Endpoint == "" {
		return status.Error(codes.InvalidArgument, "S3 endpoint field is empty.")
	}

	if c.AccessKey == "" {
		return status.Error(codes.InvalidArgument, "S3 accessKey field is empty.")
	}

	if c.SecretKey == "" {
		return status.Error(codes.InvalidArgument, "S3 secretKey field is empty.")
	}

	return nil
}

// FindBackupLocations returns saved backup locations configuration.
func FindBackupLocations(q *reform.Querier) ([]*BackupLocation, error) {
	rows, err := q.SelectAllFrom(BackupLocationTable, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select backup locations")
	}

	locations := make([]*BackupLocation, len(rows))
	for i, s := range rows {
		locations[i] = s.(*BackupLocation)
	}

	return locations, nil
}

// CreateBackupLocationParams are params for creating new backup location.
type CreateBackupLocationParams struct {
	Name        string
	Description string

	PMMClientConfig *PMMClientLocationConfig
	PMMServerConfig *PMMServerLocationConfig
	S3Config        *S3LocationConfig
}

// CreateBackupLocation persists backup location.
func CreateBackupLocation(q *reform.Querier, params CreateBackupLocationParams) (*BackupLocation, error) {
	id := "/location_id/" + uuid.New().String()

	if err := checkUniqueBackupLocationID(q, id); err != nil {
		return nil, err
	}

	if err := checkUniqueBackupLocationName(q, params.Name); err != nil {
		return nil, err
	}

	row := &BackupLocation{
		ID:          id,
		Name:        params.Name,
		Description: params.Description,
	}
	configCount := 0
	if params.S3Config != nil {
		configCount++
	}
	if params.PMMServerConfig != nil {
		configCount++
	}
	if params.PMMClientConfig != nil {
		configCount++
	}

	if configCount > 1 {
		return nil, status.Error(codes.InvalidArgument, "Only one config is allowed.")
	}

	switch {
	case params.S3Config != nil:
		if err := checkS3Config(params.S3Config); err != nil {
			return nil, err
		}
		row.Type = S3BackupLocationType
		row.S3Config = params.S3Config

	case params.PMMServerConfig != nil:
		if err := checkPMMServerLocationConfig(params.PMMServerConfig); err != nil {
			return nil, err
		}
		row.Type = PMMServerBackupLocationType
		row.PMMServerConfig = params.PMMServerConfig

	case params.PMMClientConfig != nil:
		if err := checkPMMClientLocationConfig(params.PMMClientConfig); err != nil {
			return nil, err
		}
		row.Type = PMMClientBackupLocationType
		row.PMMClientConfig = params.PMMClientConfig

	default:
		return nil, status.Error(codes.InvalidArgument, "Missing location type.")
	}

	if err := q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to create backup location")
	}

	return row, nil

}
