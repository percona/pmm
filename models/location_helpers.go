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
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go"
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

func s3ConfigFilled(c *S3LocationConfig) error {
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

	if c.BucketName == "" {
		return status.Error(codes.InvalidArgument, "S3 bucketName field is empty.")
	}

	return nil
}

func parseEndpoint(endpoint string) (*url.URL, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err)
	}

	// User could specify the endpoint without scheme, so according to RFC 3986 the host won't be parsed.
	// Try to prepend scheme and parse new url.
	if parsedURL.Host == "" {
		parsedURL, err = url.Parse("https://" + endpoint)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err)
		}
	}

	return parsedURL, nil
}

// checkS3Config checks S3 config and returns the flag that indicates if
// secure connection should be used and the parsed host.
func checkS3Config(c *S3LocationConfig) (bool, string, error) {
	if err := s3ConfigFilled(c); err != nil {
		return false, "", err
	}

	parsedURL, err := parseEndpoint(c.Endpoint)
	if err != nil {
		return false, "", err
	}

	if parsedURL.Host == "" {
		return false, "", status.Error(codes.InvalidArgument, "No host found in the Endpoint.")
	}

	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return false, "", status.Error(codes.InvalidArgument, "Path is not allowed for Endpoint.")
	}

	secure := true
	switch parsedURL.Scheme {
	case "http":
		secure = false
	case "https":
	default:
		return false, "", status.Errorf(codes.InvalidArgument, "Invalid scheme '%s'", parsedURL.Scheme)
	}

	return secure, parsedURL.Host, nil
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

// FindBackupLocationByID finds a Backup Location by its ID.
func FindBackupLocationByID(q *reform.Querier, id string) (*BackupLocation, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Location ID.")
	}

	location := &BackupLocation{ID: id}
	switch err := q.Reload(location); err {
	case nil:
		return location, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Backup location with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// FindBackupLocationsByIDs finds backup locations by IDs.
func FindBackupLocationsByIDs(q *reform.Querier, ids []string) (map[string]*BackupLocation, error) {
	if len(ids) == 0 {
		return map[string]*BackupLocation{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE id IN (%s)", p)
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	all, err := q.SelectAllFrom(BackupLocationTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	locations := make(map[string]*BackupLocation, len(all))
	for _, l := range all {
		location := l.(*BackupLocation)
		locations[location.ID] = location
	}
	return locations, nil
}

// BackupLocationConfig groups all backup locations configs.
type BackupLocationConfig struct {
	PMMClientConfig *PMMClientLocationConfig
	PMMServerConfig *PMMServerLocationConfig
	S3Config        *S3LocationConfig
}

// Validate checks if there is exactly one config with required fields and returns if config is set.
func (c BackupLocationConfig) Validate() (bool, error) {
	var err error
	configCount := 0
	if c.S3Config != nil {
		configCount++
		_, _, err = checkS3Config(c.S3Config)
	}

	if c.PMMServerConfig != nil {
		configCount++
		err = checkPMMServerLocationConfig(c.PMMServerConfig)
	}

	if c.PMMClientConfig != nil {
		configCount++
		err = checkPMMClientLocationConfig(c.PMMClientConfig)
	}

	if configCount > 1 {
		return false, status.Error(codes.InvalidArgument, "Only one config is allowed.")
	}

	return configCount == 1, err
}

// FillLocationConfig fills provided location according to backup config.
func (c BackupLocationConfig) FillLocationConfig(location *BackupLocation) {
	location.Type = ""
	location.PMMClientConfig = nil
	location.PMMServerConfig = nil
	location.S3Config = nil

	switch {
	case c.S3Config != nil:
		location.Type = S3BackupLocationType
		location.S3Config = c.S3Config

	case c.PMMServerConfig != nil:
		location.Type = PMMServerBackupLocationType
		location.PMMServerConfig = c.PMMServerConfig

	case c.PMMClientConfig != nil:
		location.Type = PMMClientBackupLocationType
		location.PMMClientConfig = c.PMMClientConfig
	}
}

// CreateBackupLocationParams are params for creating new backup location.
type CreateBackupLocationParams struct {
	Name        string
	Description string

	BackupLocationConfig
}

// CreateBackupLocation creates backup location.
func CreateBackupLocation(q *reform.Querier, params CreateBackupLocationParams) (*BackupLocation, error) {
	configSet, err := params.Validate()
	if err != nil {
		return nil, err
	}

	if !configSet {
		return nil, status.Error(codes.InvalidArgument, "Missing location config.")
	}

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

	params.FillLocationConfig(row)

	if err := q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to create backup location")
	}

	return row, nil
}

// ChangeBackupLocationParams are params for updating existing backup location.
type ChangeBackupLocationParams struct {
	Name        string
	Description string

	BackupLocationConfig
}

// ChangeBackupLocation updates existing location by specified locationID and params.
func ChangeBackupLocation(q *reform.Querier, locationID string, params ChangeBackupLocationParams) (*BackupLocation, error) {
	configSet, err := params.Validate()
	if err != nil {
		return nil, err
	}

	row, err := FindBackupLocationByID(q, locationID)
	if err != nil {
		return nil, err
	}

	if params.Name != "" && params.Name != row.Name {
		if err := checkUniqueBackupLocationName(q, params.Name); err != nil {
			return nil, err
		}
		row.Name = params.Name
	}

	if params.Description != "" {
		row.Description = params.Description
	}

	// Replace old configuration by config from params
	if configSet {
		params.FillLocationConfig(row)
	}

	if err := q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update backup location")
	}

	return row, nil
}

func testS3Config(c *S3LocationConfig) error {
	secure, host, err := checkS3Config(c)
	if err != nil {
		return err
	}

	minioClient, err := minio.New(host, c.AccessKey, c.SecretKey, secure)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "%s", err)
	}

	exists, err := minioClient.BucketExists(c.BucketName)
	if err != nil {
		if er, ok := err.(minio.ErrorResponse); ok {
			return status.Errorf(codes.InvalidArgument, "%s: %s.", er.Code, er.Message)
		}

		return status.Errorf(codes.Internal, "%s", err)
	}

	if !exists {
		return status.Errorf(codes.InvalidArgument, "Bucket doesn't exist")
	}

	return nil
}

// VerifyBackupLocationParams are params for verifying location and credentials.
type VerifyBackupLocationParams struct {
	BackupLocationConfig
}

// VerifyBackupLocationConfig checks and tests backup location config.
func VerifyBackupLocationConfig(params *VerifyBackupLocationParams) error {
	configSet, err := params.Validate()
	if err != nil {
		return err
	}

	if !configSet {
		return status.Error(codes.InvalidArgument, "Missing location config.")
	}

	switch {
	case params.S3Config != nil:
		if err := testS3Config(params.S3Config); err != nil {
			return err
		}
	case params.PMMServerConfig != nil:
		if err := checkPMMServerLocationConfig(params.PMMServerConfig); err != nil {
			return err
		}
	case params.PMMClientConfig != nil:
		if err := checkPMMClientLocationConfig(params.PMMClientConfig); err != nil {
			return err
		}
	}

	return nil
}

// RemoveBackupLocation removes BackupLocation by ID.
func RemoveBackupLocation(q *reform.Querier, id string, mode RemoveMode) error {
	if _, err := FindBackupLocationByID(q, id); err != nil {
		return err
	}

	// @TODO - force delete https://jira.percona.com/browse/PMM-7475
	if err := q.Delete(&BackupLocation{ID: id}); err != nil {
		return errors.Wrap(err, "failed to delete BackupLocation")
	}

	return nil
}
