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

// CreateOmBootstrapRunConfig stores one bootstrap run's environment and cluster,
// as given at trigger time.
//
// Callers create this at most once per run, right after SEP accepts it -- there is
// nothing to update afterward, since neither field ever changes for the life of a
// run. Not encrypted, unlike OmBootstrapSecret: environment and cluster are plain
// inventory labels, the same as models.Service's own Environment/Cluster fields,
// not secrets.
func CreateOmBootstrapRunConfig(q *reform.Querier, config *OmBootstrapRunConfig) error {
	err := q.Insert(config)
	if err != nil {
		return fmt.Errorf("failed to insert OM bootstrap run config: %w", err)
	}
	return nil
}

// FindOmBootstrapRunConfigByRunID returns one run's stored environment and
// cluster, or ErrNotFound when the caller left both blank at trigger time, or
// triggered the run before this table existed -- see OmBootstrapRunConfig's own
// doc comment on why that is not an error.
func FindOmBootstrapRunConfigByRunID(q *reform.Querier, runID string) (*OmBootstrapRunConfig, error) {
	if runID == "" {
		return nil, NewInvalidArgumentError("run_id shouldn't be empty")
	}
	config := &OmBootstrapRunConfig{RunID: runID}
	err := q.Reload(config)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to select OM bootstrap run config: %w", err)
	}
	return config, nil
}

// MarkOmBootstrapRunRegistered records that PMM has finished registering every host
// of one bootstrap run with its own inventory, which is what takes the run out of
// the stepper's succeeded-run sweep -- see OmBootstrapRunConfig's own doc comment.
//
// Upserts, because the row only exists from trigger time for a run that came with
// an environment or cluster worth storing: an unlabelled run reaches this point
// with no row at all, and its two empty labels are the correct values for it.
func MarkOmBootstrapRunRegistered(q *reform.Querier, runID string) error {
	if runID == "" {
		return NewInvalidArgumentError("run_id shouldn't be empty")
	}

	registeredAt := Now()
	config, err := FindOmBootstrapRunConfigByRunID(q, runID)
	if errors.Is(err, ErrNotFound) {
		return CreateOmBootstrapRunConfig(q, &OmBootstrapRunConfig{RunID: runID, RegisteredAt: &registeredAt})
	}
	if err != nil {
		return err
	}

	config.RegisteredAt = &registeredAt
	err = q.Update(config)
	if err != nil {
		return fmt.Errorf("failed to update OM bootstrap run config: %w", err)
	}
	return nil
}

// FindRegisteredOmBootstrapRunIDs returns the id of every run PMM has finished
// registering.
//
// One query for the whole set rather than one per run: the stepper asks this on
// every tick to decide which succeeded runs still need work, and the answer for a
// server with a long bootstrap history is "none of them".
func FindRegisteredOmBootstrapRunIDs(q *reform.Querier) (map[string]struct{}, error) {
	rows, err := q.SelectAllFrom(OmBootstrapRunConfigTable, "WHERE registered_at IS NOT NULL")
	if err != nil {
		return nil, fmt.Errorf("failed to select registered OM bootstrap runs: %w", err)
	}

	registered := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		registered[row.(*OmBootstrapRunConfig).RunID] = struct{}{} //nolint:forcetypeassert
	}
	return registered, nil
}
