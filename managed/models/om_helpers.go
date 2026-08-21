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

	"gopkg.in/reform.v1"
)

// CreateOmTopologyRun stores one run and the topology document it produced.
//
// Both in one call because they are one fact: a run with no document is not a state any
// reader should have to handle, and the caller writes them inside a transaction so no
// reader ever sees one without the other.
func CreateOmTopologyRun(q *reform.Querier, run *OmTopologyRun, snapshot *OmTopologySnapshot) error {
	err := q.Insert(run)
	if err != nil {
		return fmt.Errorf("failed to insert OM run: %w", err)
	}
	if snapshot != nil {
		snapshot.RunID = run.RunID
		err = q.Insert(snapshot)
		if err != nil {
			return fmt.Errorf("failed to insert OM snapshot: %w", err)
		}
	}
	return nil
}

// FindOmTopologyRuns returns the most recent runs, newest first.
func FindOmTopologyRuns(q *reform.Querier, limit int) ([]*OmTopologyRun, error) {
	if limit <= 0 {
		return []*OmTopologyRun{}, nil
	}
	structs, err := q.SelectAllFrom(OmTopologyRunTable, "ORDER BY started_at DESC, run_id DESC LIMIT "+q.Placeholder(1), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to select OM runs: %w", err)
	}
	runs := make([]*OmTopologyRun, len(structs))
	for i, s := range structs {
		runs[i] = s.(*OmTopologyRun) //nolint:forcetypeassert
	}
	return runs, nil
}

// FindOmTopologyRunByID returns one run, or ErrNotFound when there is no such run.
func FindOmTopologyRunByID(q *reform.Querier, runID string) (*OmTopologyRun, error) {
	if runID == "" {
		return nil, NewInvalidArgumentError("run_id shouldn't be empty")
	}
	run := &OmTopologyRun{RunID: runID}
	err := q.Reload(run)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to select OM run: %w", err)
	}
	return run, nil
}

// FindLatestOmTopologySnapshot returns the newest stored topology document, or ErrNotFound when
// no collection has ever run.
//
// This is what lets a restarted pmm-managed serve the estate before it has collected
// anything of its own.
func FindLatestOmTopologySnapshot(q *reform.Querier) (*OmTopologySnapshot, error) {
	structs, err := q.SelectAllFrom(OmTopologySnapshotTable, "ORDER BY generated_at DESC LIMIT 1")
	if err != nil {
		return nil, fmt.Errorf("failed to select the latest OM snapshot: %w", err)
	}
	if len(structs) == 0 {
		return nil, ErrNotFound
	}
	return structs[0].(*OmTopologySnapshot), nil //nolint:forcetypeassert
}

// PruneOmTopologyRuns deletes all but the newest keep runs, cascading to their snapshots.
//
// Retention has to be bounded here rather than by an operator: collection runs on a timer
// and on every read past the cache, so the table grows on its own. Pruning on write keeps
// it at a fixed size without a second scheduled job to forget about.
func PruneOmTopologyRuns(q *reform.Querier, keep int) error {
	if keep <= 0 {
		return NewInvalidArgumentError("keep should be positive")
	}
	_, err := q.Exec(`
		DELETE FROM om_topology_runs
		WHERE run_id NOT IN (
			SELECT run_id FROM om_topology_runs ORDER BY started_at DESC, run_id DESC LIMIT `+q.Placeholder(1)+`
		)`, keep)
	if err != nil {
		return fmt.Errorf("failed to prune OM runs: %w", err)
	}
	return nil
}
