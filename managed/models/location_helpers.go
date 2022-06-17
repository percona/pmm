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

// ParseEndpoint parse endpoint and prepend https if no scheme is provided.
func ParseEndpoint(endpoint string) (*url.URL, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	// User could specify the endpoint without scheme, so according to RFC 3986 the host won't be parsed.
	// Try to prepend scheme and parse new url.
	if parsedURL.Host == "" {
		return url.Parse("https://" + endpoint)
	}

	return parsedURL, nil
}

// checkS3Config checks S3 config.
func checkS3Config(c *S3LocationConfig, withBucketLocation bool) error {
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

	if withBucketLocation && c.BucketRegion == "" {
		return status.Error(codes.InvalidArgument, "S3 bucketRegion field is empty")
	}

	parsedURL, err := ParseEndpoint(c.Endpoint)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if parsedURL.Host == "" {
		return status.Error(codes.InvalidArgument, "No host found in the Endpoint.")
	}

	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return status.Error(codes.InvalidArgument, "Path is not allowed for Endpoint.")
	}

	switch parsedURL.Scheme {
	case "http", "https", "":
		// valid values
	default:
		return status.Errorf(codes.InvalidArgument, "Invalid scheme '%s'", parsedURL.Scheme)
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
		return make(map[string]*BackupLocation), nil
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

// BackupLocationValidationParams contains typed params for backup location validate.
type BackupLocationValidationParams struct {
	RequireConfig    bool
	WithBucketRegion bool
}

// Validate checks if there is exactly one config with required fields and returns if config is set.
func (c BackupLocationConfig) Validate(params BackupLocationValidationParams) error {
	var err error
	configCount := 0
	if c.S3Config != nil {
		configCount++
		err = checkS3Config(c.S3Config, params.WithBucketRegion)
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
		return status.Error(codes.InvalidArgument, "Only one config is allowed.")
	}

	if params.RequireConfig && configCount == 0 {
		return status.Error(codes.InvalidArgument, "Missing location config.")
	}

	return err
}

// FillLocationConfig fills provided location according to backup config.
func (c BackupLocationConfig) FillLocationConfig(location *BackupLocation) {
	switch {
	case c.S3Config != nil:
		location.Type = S3BackupLocationType
		location.S3Config = c.S3Config
		location.PMMClientConfig = nil
		location.PMMServerConfig = nil
	case c.PMMServerConfig != nil:
		location.Type = PMMServerBackupLocationType
		location.PMMServerConfig = c.PMMServerConfig
		location.PMMClientConfig = nil
		location.S3Config = nil
	case c.PMMClientConfig != nil:
		location.Type = PMMClientBackupLocationType
		location.PMMClientConfig = c.PMMClientConfig
		location.PMMServerConfig = nil
		location.S3Config = nil
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
	if err := params.Validate(BackupLocationValidationParams{
		RequireConfig:    true,
		WithBucketRegion: true,
	}); err != nil {
		return nil, err
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
	if err := params.Validate(BackupLocationValidationParams{
		RequireConfig:    false,
		WithBucketRegion: true,
	}); err != nil {
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
	params.FillLocationConfig(row)

	if err := q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update backup location")
	}

	return row, nil
}

// RemoveBackupLocation removes BackupLocation by ID.
func RemoveBackupLocation(q *reform.Querier, id string, mode RemoveMode) error {
	if _, err := FindBackupLocationByID(q, id); err != nil {
		return err
	}

	artifacts, err := FindArtifacts(q, ArtifactFilters{LocationID: id})
	if err != nil {
		return err
	}

	var restoreItems []*RestoreHistoryItem
	for _, a := range artifacts {
		items, err := FindRestoreHistoryItems(q, RestoreHistoryItemFilters{ArtifactID: a.ID})
		if err != nil {
			return err
		}

		restoreItems = append(restoreItems, items...)
	}

	tasks, err := FindScheduledTasks(q, ScheduledTasksFilter{
		LocationID: id,
	})
	if err != nil {
		return err
	}

	if mode == RemoveRestrict {
		if len(artifacts) != 0 {
			return status.Errorf(codes.FailedPrecondition, "backup location with ID %q has artifacts.", id)
		}

		if len(restoreItems) != 0 {
			return status.Errorf(codes.FailedPrecondition, "backup location with ID %q has restore history items.", id)
		}

		if len(tasks) != 0 {
			return status.Errorf(codes.FailedPrecondition, "backup location with ID %q has scheduled tasks.", id)
		}
	}

	for _, i := range restoreItems {
		if err := RemoveRestoreHistoryItem(q, i.ID); err != nil {
			return err
		}
	}

	for _, a := range artifacts {
		if err := DeleteArtifact(q, a.ID); err != nil {
			return err
		}
	}

	for _, t := range tasks {
		if err := RemoveScheduledTask(q, t.ID); err != nil {
			return err
		}
	}

	if err := q.Delete(&BackupLocation{ID: id}); err != nil {
		return errors.Wrap(err, "failed to delete BackupLocation")
	}

	return nil
}
