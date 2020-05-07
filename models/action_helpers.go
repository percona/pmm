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
	switch err := q.Reload(res); err {
	case nil:
		return res, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "ActionResult with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// FindPmmAgentIDToRunAction finds pmm-agent-id to run action.
func FindPmmAgentIDToRunAction(pmmAgentID string, agents []*Agent) (string, error) {
	// no explicit ID is given, and there is only one
	if pmmAgentID == "" && len(agents) == 1 {
		return agents[0].AgentID, nil
	}

	// no explicit ID is given, and there are zero or several
	if pmmAgentID == "" {
		return "", status.Errorf(codes.InvalidArgument, "Couldn't find pmm-agent-id to run action")
	}

	// check that explicit agent id is correct
	for _, a := range agents {
		if a.AgentID == pmmAgentID {
			return a.AgentID, nil
		}
	}
	return "", status.Errorf(codes.FailedPrecondition, "Couldn't find pmm-agent-id to run action")
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

// CleanupOldResults deletes action results older than a specified date.
func CleanupOldResults(q *reform.Querier, olderThan time.Time) error {
	_, err := q.DeleteFrom(ActionResultTable, " WHERE updated_at <= $1", olderThan)
	return err
}
