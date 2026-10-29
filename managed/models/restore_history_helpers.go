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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

// RestoreHistoryItemFilters represents filters for restore history items.
type RestoreHistoryItemFilters struct {
	// Return only items that belongs to specified service id.
	ServiceID string
	// Return only items that has specified location id.
	ArtifactID string
	// Return only items with specified status.
	Status *RestoreStatus
}

// FindRestoreHistoryItems returns restore history list.
func FindRestoreHistoryItems(q *reform.Querier, filters RestoreHistoryItemFilters) ([]*RestoreHistoryItem, error) {
	var conditions []string
	var args []any

	idx := 1
	if filters.ServiceID != "" {
		_, err := FindServiceByID(q, filters.ServiceID)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, "service_id = "+q.Placeholder(idx))
		args = append(args, filters.ServiceID)
		idx++
	}

	if filters.ArtifactID != "" {
		_, err := FindArtifactByID(q, filters.ArtifactID)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, "artifact_id = "+q.Placeholder(idx))
		args = append(args, filters.ArtifactID)
		idx++
	}

	if filters.Status != nil {
		conditions = append(conditions, "status = "+q.Placeholder(idx))
		args = append(args, *filters.Status)
	}

	var whereClause string
	if len(conditions) != 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	rows, err := q.SelectAllFrom(RestoreHistoryItemTable, whereClause+" ORDER BY started_at DESC", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select restore history: %w", err)
	}

	items := make([]*RestoreHistoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, r.(*RestoreHistoryItem)) //nolint:forcetypeassert
	}

	return items, nil
}

// FindRestoreHistoryItemByID finds restore history item. Returns ErrNotFound if requested item not found.
func FindRestoreHistoryItemByID(q *reform.Querier, id string) (*RestoreHistoryItem, error) {
	if id == "" {
		return nil, errors.New("provided id is empty")
	}

	item := &RestoreHistoryItem{ID: id}
	err := q.Reload(item)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, fmt.Errorf("restore history item by id '%s': %w", id, ErrNotFound)
		}
		return nil, err
	}

	return item, nil
}

// CreateRestoreHistoryItemParams are params for creating a new restore history item.
type CreateRestoreHistoryItemParams struct {
	ArtifactID    string
	ServiceID     string
	PITRTimestamp *time.Time
	Status        RestoreStatus
}

// Validate validates params used for creating a restore history item.
func (p *CreateRestoreHistoryItemParams) Validate() error {
	if p.ArtifactID == "" {
		return NewInvalidArgumentError("artifact_id shouldn't be empty")
	}
	if p.ServiceID == "" {
		return NewInvalidArgumentError("service_id shouldn't be empty")
	}

	return p.Status.Validate()
}

// CreateRestoreHistoryItem creates a restore history item entry in DB.
func CreateRestoreHistoryItem(q *reform.Querier, params CreateRestoreHistoryItemParams) (*RestoreHistoryItem, error) {
	err := params.Validate()
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()
	_, err = FindRestoreHistoryItemByID(q, id)
	switch {
	case err == nil:
		return nil, fmt.Errorf("restore history item with id '%s' already exists", id)
	case errors.Is(err, ErrNotFound):
	default:
		return nil, err
	}

	row := &RestoreHistoryItem{
		ID:            id,
		ArtifactID:    params.ArtifactID,
		ServiceID:     params.ServiceID,
		PITRTimestamp: params.PITRTimestamp,
		Status:        params.Status,
	}
	err = q.Insert(row)
	if err != nil {
		return nil, fmt.Errorf("failed to insert restore history item: %w", err)
	}

	return row, nil
}

// ChangeRestoreHistoryItemParams are params for changing existing restore history item.
type ChangeRestoreHistoryItemParams struct {
	Status     RestoreStatus
	FinishedAt *time.Time
}

// ChangeRestoreHistoryItem updates existing restore history item.
func ChangeRestoreHistoryItem(
	q *reform.Querier,
	restoreID string,
	params ChangeRestoreHistoryItemParams,
) (*RestoreHistoryItem, error) {
	row, err := FindRestoreHistoryItemByID(q, restoreID)
	if err != nil {
		return nil, err
	}
	row.Status = params.Status

	if params.FinishedAt != nil {
		row.FinishedAt = params.FinishedAt
	}

	err = q.Update(row)
	if err != nil {
		return nil, fmt.Errorf("failed to update restore history item: %w", err)
	}

	return row, nil
}

// RemoveRestoreHistoryItem removes restore history item by ID.
func RemoveRestoreHistoryItem(q *reform.Querier, id string) error {
	_, err := FindRestoreHistoryItemByID(q, id)
	if err != nil {
		return err
	}

	err = q.Delete(&RestoreHistoryItem{ID: id})
	if err != nil {
		return fmt.Errorf("failed to remove restore history item by id '%s': %w", id, err)
	}
	return nil
}
