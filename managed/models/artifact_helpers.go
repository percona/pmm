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

package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// ArtifactFilters represents filters for artifacts list.
type ArtifactFilters struct {
	// Return only artifacts that provide insights for that Service.
	ServiceID string
	// Return only artifacts that belong to specified location.
	LocationID string
	// Return only artifacts that was created by specified scheduled task.
	ScheduleID string
	// Return only artifacts by specified status.
	Status BackupStatus
	// Filters by folder.
	Folder *string
}

// FindArtifacts returns artifact list sorted by creation time in DESCENDING order.
func FindArtifacts(q *reform.Querier, filters ArtifactFilters) ([]*Artifact, error) {
	var conditions []string
	var args []interface{}
	idx := 1
	if filters.ServiceID != "" {
		conditions = append(conditions, fmt.Sprintf("service_id = %s", q.Placeholder(idx)))
		args = append(args, filters.ServiceID)
		idx++
	}

	if filters.LocationID != "" {
		if _, err := FindBackupLocationByID(q, filters.LocationID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("location_id = %s", q.Placeholder(idx)))
		args = append(args, filters.LocationID)
		idx++
	}

	if filters.ScheduleID != "" {
		conditions = append(conditions, fmt.Sprintf("schedule_id = %s", q.Placeholder(idx)))
		args = append(args, filters.ScheduleID)
		idx++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = %s", q.Placeholder(idx)))
		args = append(args, filters.Status)
		idx++
	}

	if filters.Folder != nil {
		conditions = append(conditions, fmt.Sprintf("folder = %s", q.Placeholder(idx)))
		args = append(args, *filters.Folder)
		// idx++
	}

	var whereClause string
	if len(conditions) != 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	rows, err := q.SelectAllFrom(ArtifactTable, fmt.Sprintf("%s ORDER BY created_at DESC", whereClause), args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select artifacts")
	}

	artifacts := make([]*Artifact, 0, len(rows))
	for _, r := range rows {
		artifacts = append(artifacts, r.(*Artifact)) //nolint:forcetypeassert
	}

	return artifacts, nil
}

// FindArtifactsByIDs finds artifacts by IDs.
func FindArtifactsByIDs(q *reform.Querier, ids []string) (map[string]*Artifact, error) {
	if len(ids) == 0 {
		return make(map[string]*Artifact), nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE id IN (%s)", p)
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	all, err := q.SelectAllFrom(ArtifactTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	artifacts := make(map[string]*Artifact, len(all))
	for _, l := range all {
		artifact := l.(*Artifact) //nolint:forcetypeassert
		artifacts[artifact.ID] = artifact
	}
	return artifacts, nil
}

// FindArtifactByID returns artifact by given ID if found, ErrNotFound if not.
func FindArtifactByID(q *reform.Querier, id string) (*Artifact, error) {
	if id == "" {
		return nil, errors.New("provided artifact id is empty")
	}

	artifact := &Artifact{ID: id}
	err := q.Reload(artifact)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, errors.Wrapf(ErrNotFound, "artifact by id '%s'", id)
		}
		return nil, errors.WithStack(err)
	}

	return artifact, nil
}

// FindArtifactByName returns artifact by given name if found, ErrNotFound if not.
func FindArtifactByName(q *reform.Querier, name string) (*Artifact, error) {
	if name == "" {
		return nil, errors.New("provided backup artifact name is empty")
	}
	artifact := &Artifact{}
	err := q.FindOneTo(artifact, "name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, errors.Wrapf(ErrNotFound, "backup artifact with name %q not found.", name) //nolint:revive
		}
		return nil, errors.WithStack(err)
	}

	return artifact, nil
}

func checkUniqueArtifactName(q *reform.Querier, name string) error {
	if name == "" {
		panic("empty Artifact Name")
	}

	var artifact Artifact
	err := q.FindOneTo(&artifact, "name", name)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Artifact with name %q already exists.", name)
}

// CreateArtifactParams are params for creating a new artifact.
type CreateArtifactParams struct {
	Name             string
	Vendor           string
	DBVersion        string
	LocationID       string
	ServiceID        string
	DataModel        DataModel
	Mode             BackupMode
	Status           BackupStatus
	ScheduleID       string
	IsShardedCluster bool
	Folder           string
}

// Validate validates params used for creating an artifact entry.
func (p *CreateArtifactParams) Validate() error {
	if p.Name == "" {
		return NewInvalidArgumentError("name shouldn't be empty")
	}
	if p.Vendor == "" {
		return NewInvalidArgumentError("vendor shouldn't be empty")
	}
	if p.LocationID == "" {
		return NewInvalidArgumentError("location_id shouldn't be empty")
	}
	if p.ServiceID == "" {
		return NewInvalidArgumentError("service_id shouldn't be empty")
	}

	if err := p.Mode.Validate(); err != nil {
		return err
	}

	if err := p.DataModel.Validate(); err != nil {
		return err
	}

	return p.Status.Validate()
}

// CreateArtifact creates artifact entry in DB.
func CreateArtifact(q *reform.Querier, params CreateArtifactParams) (*Artifact, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	id := "/artifact_id/" + uuid.New().String()
	_, err := FindArtifactByID(q, id)
	switch {
	case err == nil:
		return nil, errors.Errorf("artifact with id '%s' already exists", id)
	case errors.Is(err, ErrNotFound):
	default:
		return nil, errors.WithStack(err)
	}

	if err := checkUniqueArtifactName(q, params.Name); err != nil {
		return nil, err
	}

	row := &Artifact{
		ID:               id,
		Name:             params.Name,
		Vendor:           params.Vendor,
		DBVersion:        params.DBVersion,
		LocationID:       params.LocationID,
		ServiceID:        params.ServiceID,
		DataModel:        params.DataModel,
		Mode:             params.Mode,
		Status:           params.Status,
		Type:             OnDemandArtifactType,
		ScheduleID:       params.ScheduleID,
		IsShardedCluster: params.IsShardedCluster,
		Folder:           params.Folder,
	}

	if params.ScheduleID != "" {
		row.Type = ScheduledArtifactType
	}

	if err := q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to insert artifact")
	}

	return row, nil
}

// UpdateArtifactParams are params for changing existing artifact.
type UpdateArtifactParams struct {
	ServiceID        *string
	Status           *BackupStatus
	ScheduleID       *string
	IsShardedCluster bool
	Metadata         *Metadata
	Folder           *string
}

// UpdateArtifact updates existing artifact.
func UpdateArtifact(q *reform.Querier, artifactID string, params UpdateArtifactParams) (*Artifact, error) {
	row, err := FindArtifactByID(q, artifactID)
	if err != nil {
		return nil, err
	}
	if params.ServiceID != nil {
		row.ServiceID = *params.ServiceID
	}
	if params.Status != nil {
		row.Status = *params.Status
	}
	if params.ScheduleID != nil {
		row.ScheduleID = *params.ScheduleID
	}

	if params.IsShardedCluster && !row.IsShardedCluster {
		row.IsShardedCluster = true
	}

	if params.Metadata != nil {
		// We're appending to existing list to cover PITR mode cases.
		row.MetadataList = append(row.MetadataList, *params.Metadata)
	}

	if params.Folder != nil {
		row.Folder = *params.Folder
	}

	if err := q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update backup artifact")
	}

	return row, nil
}

// DeleteArtifact removes artifact by ID.
func DeleteArtifact(q *reform.Querier, id string) error {
	if _, err := FindArtifactByID(q, id); err != nil {
		return err
	}

	if err := q.Delete(&Artifact{ID: id}); err != nil {
		return errors.Wrapf(err, "failed to delete artifact by id '%s'", id)
	}
	return nil
}

// MetadataRemoveFirstN removes first N records from artifact metadata list.
func (s *Artifact) MetadataRemoveFirstN(q *reform.Querier, n uint32) error {
	if n > uint32(len(s.MetadataList)) {
		n = uint32(len(s.MetadataList))
	}
	s.MetadataList = s.MetadataList[n:]
	if err := q.Update(s); err != nil {
		return errors.Wrap(err, "failed to remove artifact metadata records")
	}
	return nil
}

// IsArtifactFinalStatus checks if artifact status is one of the final ones.
func IsArtifactFinalStatus(backupStatus BackupStatus) bool {
	switch backupStatus {
	case SuccessBackupStatus,
		ErrorBackupStatus,
		FailedToDeleteBackupStatus:
		return true
	default:
		return false
	}
}
