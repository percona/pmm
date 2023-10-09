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
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// FindActionResultByID finds ActionResult by ID.
func FindActionResultByID(q *reform.Querier, id string) (*ActionResult, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty ActionResult ID.")
	}

	res := &ActionResult{ID: id}
	err := q.Reload(res)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "ActionResult with ID %q not found.", id)
		}
		return nil, errors.WithStack(err)
	}

	return res, nil
}

// CreateActionResult stores an action result in action results storage.
func CreateActionResult(q *reform.Querier, pmmAgentID string) (*ActionResult, error) {
	result := &ActionResult{ID: "/action_id/" + uuid.New().String(), PMMAgentID: pmmAgentID}
	if err := q.Insert(result); err != nil {
		return nil, errors.WithStack(err)
	}
	return result, nil
}

// ChangeActionResult updates an action result in action results storage.
func ChangeActionResult(q *reform.Querier, actionID, pmmAgentID, aError, output string, done bool) error {
	result := &ActionResult{
		ID:         actionID,
		PMMAgentID: pmmAgentID,
		Done:       done,
		Error:      aError,
		Output:     output,
	}
	if err := q.Update(result); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// CleanupOldActionResults deletes action results older than a specified date.
func CleanupOldActionResults(q *reform.Querier, olderThan time.Time) error {
	_, err := q.DeleteFrom(ActionResultTable, " WHERE updated_at <= $1", olderThan)
	return err
}
