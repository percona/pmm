// Copyright (C) 2026 Percona LLC
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

// CreatePomRun stores one run and the topology document it produced.
//
// Both in one call because they are one fact: a run with no document is not a state any
// reader should have to handle, and the caller writes them inside a transaction so no
// reader ever sees one without the other.
func CreatePomRun(q *reform.Querier, run *PomRun, snapshot *PomSnapshot) error {
	err := q.Insert(run)
	if err != nil {
		return fmt.Errorf("failed to insert POM run: %w", err)
	}
	if snapshot != nil {
		snapshot.RunID = run.RunID
		err = q.Insert(snapshot)
		if err != nil {
			return fmt.Errorf("failed to insert POM snapshot: %w", err)
		}
	}
	return nil
}

// FindPomRuns returns the most recent runs, newest first.
func FindPomRuns(q *reform.Querier, limit int) ([]*PomRun, error) {
	if limit <= 0 {
		return []*PomRun{}, nil
	}
	structs, err := q.SelectAllFrom(PomRunTable, "ORDER BY started_at DESC, run_id DESC LIMIT "+q.Placeholder(1), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to select POM runs: %w", err)
	}
	runs := make([]*PomRun, len(structs))
	for i, s := range structs {
		runs[i] = s.(*PomRun) //nolint:forcetypeassert
	}
	return runs, nil
}

// FindPomRunByID returns one run, or ErrNotFound when there is no such run.
func FindPomRunByID(q *reform.Querier, runID string) (*PomRun, error) {
	if runID == "" {
		return nil, NewInvalidArgumentError("run_id shouldn't be empty")
	}
	run := &PomRun{RunID: runID}
	err := q.Reload(run)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to select POM run: %w", err)
	}
	return run, nil
}

// FindLatestPomSnapshot returns the newest stored topology document, or ErrNotFound when
// discovery has never run.
//
// This is what lets a restarted pmm-managed serve the estate before it has collected
// anything of its own.
func FindLatestPomSnapshot(q *reform.Querier) (*PomSnapshot, error) {
	structs, err := q.SelectAllFrom(PomSnapshotTable, "ORDER BY generated_at DESC LIMIT 1")
	if err != nil {
		return nil, fmt.Errorf("failed to select the latest POM snapshot: %w", err)
	}
	if len(structs) == 0 {
		return nil, ErrNotFound
	}
	return structs[0].(*PomSnapshot), nil //nolint:forcetypeassert
}

// PrunePomRuns deletes all but the newest keep runs, cascading to their snapshots.
//
// Retention has to be bounded here rather than by an operator: discovery runs on a timer
// and on every read past the cache, so the table grows on its own. Pruning on write keeps
// it at a fixed size without a second scheduled job to forget about.
func PrunePomRuns(q *reform.Querier, keep int) error {
	if keep <= 0 {
		return NewInvalidArgumentError("keep should be positive")
	}
	_, err := q.Exec(`
		DELETE FROM pom_runs
		WHERE run_id NOT IN (
			SELECT run_id FROM pom_runs ORDER BY started_at DESC, run_id DESC LIMIT `+q.Placeholder(1)+`
		)`, keep)
	if err != nil {
		return fmt.Errorf("failed to prune POM runs: %w", err)
	}
	return nil
}
