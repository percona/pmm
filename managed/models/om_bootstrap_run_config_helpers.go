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
