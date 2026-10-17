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

// CreateOmBootstrapRunConfig stores one bootstrap run's environment and cluster, as given at trigger time.
func CreateOmBootstrapRunConfig(q *reform.Querier, config *OmBootstrapRunConfig) error {
	err := q.Insert(config)
	if err != nil {
		return fmt.Errorf("failed to insert OM bootstrap run config: %w", err)
	}
	return nil
}

// FindOmBootstrapRunConfigByRunID returns one run's stored config, or ErrNotFound if the run has none.
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

// MarkOmBootstrapRunRegistered records that PMM has registered every host of one bootstrap run.
// It upserts because a run triggered without environment or cluster has no row yet.
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

// FindRegisteredOmBootstrapRunIDs returns the ID of every run PMM has finished registering.
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
