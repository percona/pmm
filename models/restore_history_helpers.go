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
	"gopkg.in/reform.v1"
)

// FindRestoreHistoryItems returns restore history list.
func FindRestoreHistoryItems(q *reform.Querier) ([]*RestoreHistoryItem, error) {
	rows, err := q.SelectAllFrom(RestoreHistoryItemTable, "ORDER BY started_at DESC")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select restore history")
	}

	items := make([]*RestoreHistoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, r.(*RestoreHistoryItem))
	}

	return items, nil
}

func findRestoreHistoryItemByID(q *reform.Querier, id string) (*RestoreHistoryItem, error) {
	if id == "" {
		return nil, errors.New("provided id is empty")
	}

	item := &RestoreHistoryItem{ID: id}
	switch err := q.Reload(item); err {
	case nil:
		return item, nil
	case reform.ErrNoRows:
		return nil, errors.Wrapf(ErrNotFound, "restore history item by id '%s'", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// CreateRestoreHistoryItemParams are params for creating a new restore history item.
type CreateRestoreHistoryItemParams struct {
	ArtifactID string
	ServiceID  string
	Status     RestoreStatus
}

// Validate validates params used for creating a restore history item.
func (p *CreateRestoreHistoryItemParams) Validate() error {
	if p.ArtifactID == "" {
		return errors.Wrap(ErrInvalidArgument, "artifact_id shouldn't be empty")
	}
	if p.ServiceID == "" {
		return errors.Wrap(ErrInvalidArgument, "service_id shouldn't be empty")
	}

	return p.Status.Validate()
}

// CreateRestoreHistoryItem creates a restore history item entry in DB.
func CreateRestoreHistoryItem(q *reform.Querier, params CreateRestoreHistoryItemParams) (*RestoreHistoryItem, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	id := "/restore_id/" + uuid.New().String()
	_, err := findRestoreHistoryItemByID(q, id)
	switch {
	case err == nil:
		return nil, errors.Errorf("restore history item with id '%s' already exists", id)
	case errors.Is(err, ErrNotFound):
	default:
		return nil, errors.WithStack(err)
	}

	row := &RestoreHistoryItem{
		ID:         id,
		ArtifactID: params.ArtifactID,
		ServiceID:  params.ServiceID,
		Status:     params.Status,
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to insert restore history item")
	}

	return row, nil
}

// ChangeRestoreHistoryItemParams are params for changing existing restore history item.
type ChangeRestoreHistoryItemParams struct {
	Status RestoreStatus
}

// ChangeRestoreHistoryItem updates existing restore history item.
func ChangeRestoreHistoryItem(
	q *reform.Querier,
	restoreID string,
	params ChangeRestoreHistoryItemParams,
) (*RestoreHistoryItem, error) {
	row, err := findRestoreHistoryItemByID(q, restoreID)
	if err != nil {
		return nil, err
	}
	row.Status = params.Status

	if err := q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update restore history item")
	}

	return row, nil
}

// RemoveRestoreHistoryItem removes restore history item by ID.
func RemoveRestoreHistoryItem(q *reform.Querier, id string) error {
	if _, err := findRestoreHistoryItemByID(q, id); err != nil {
		return err
	}

	if err := q.Delete(&RestoreHistoryItem{ID: id}); err != nil {
		return errors.Wrapf(err, "failed to remove restore history item by id '%s'", id)
	}
	return nil
}
