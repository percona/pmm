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
	"time"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

// CreateBackupScriptConfigParams are params for creating a new backup script config.
type CreateBackupScriptConfigParams struct {
	Name                 string
	ServiceID            string
	NodeName             string
	BackupDir            string
	Compress             bool
	CompressionAlgorithm string
	Copies               int32
	ReplicaInfo          bool
	XtrabackupBinary     string
	RenderedYAML         string
}

// CreateBackupScriptConfig persists a new versioned backup config.
func CreateBackupScriptConfig(q *reform.Querier, params CreateBackupScriptConfigParams) (*BackupScriptConfig, error) {
	id := uuid.New().String()

	row := &BackupScriptConfig{
		ID:                   id,
		Name:                 params.Name,
		ServiceID:            params.ServiceID,
		NodeName:             params.NodeName,
		BackupDir:            params.BackupDir,
		Compress:             params.Compress,
		CompressionAlgorithm: params.CompressionAlgorithm,
		Copies:               params.Copies,
		ReplicaInfo:          params.ReplicaInfo,
		XtrabackupBinary:     params.XtrabackupBinary,
		RenderedYAML:         params.RenderedYAML,
		ConfigVersion:        1,
	}
	if err := q.Insert(row); err != nil {
		return nil, fmt.Errorf("failed to create backup script config: %w", err)
	}
	return row, nil
}

// FindBackupScriptConfigByID returns a backup script config by its ID.
func FindBackupScriptConfigByID(q *reform.Querier, id string) (*BackupScriptConfig, error) {
	if id == "" {
		return nil, errors.New("provided backup script config ID is empty")
	}
	row := &BackupScriptConfig{ID: id}
	switch err := q.Reload(row); {
	case err == nil:
		return row, nil
	case errors.Is(err, reform.ErrNoRows):
		return nil, fmt.Errorf("%w: backup script config with ID %q not found", ErrNotFound, id)
	default:
		return nil, err
	}
}

// FindBackupScriptConfigs returns all stored backup script configs.
func FindBackupScriptConfigs(q *reform.Querier) ([]*BackupScriptConfig, error) {
	rows, err := q.SelectAllFrom(BackupScriptConfigTable, "ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to list backup script configs: %w", err)
	}
	configs := make([]*BackupScriptConfig, len(rows))
	for i, r := range rows {
		configs[i] = r.(*BackupScriptConfig) //nolint:forcetypeassert
	}
	return configs, nil
}

// RemoveBackupScriptConfig deletes a backup script config by ID.
func RemoveBackupScriptConfig(q *reform.Querier, id string) error {
	if _, err := FindBackupScriptConfigByID(q, id); err != nil {
		return err
	}
	if err := q.Delete(&BackupScriptConfig{ID: id}); err != nil {
		return fmt.Errorf("failed to delete backup script config: %w", err)
	}
	return nil
}

// CreateBackupScriptRunParams are params for cataloging a new dispatched run.
type CreateBackupScriptRunParams struct {
	RunID      string
	ConfigID   string
	ServiceID  string
	NodeName   string
	NomadJobID string
	StartedAt  time.Time
}

// CreateBackupScriptRun catalogs a newly dispatched backup run in PENDING state.
func CreateBackupScriptRun(q *reform.Querier, params CreateBackupScriptRunParams) (*BackupScriptRun, error) {
	row := &BackupScriptRun{
		ID:         params.RunID,
		ConfigID:   params.ConfigID,
		ServiceID:  params.ServiceID,
		NodeName:   params.NodeName,
		NomadJobID: params.NomadJobID,
		Status:     ScriptBackupPending,
		StartedAt:  params.StartedAt,
	}
	if err := q.Insert(row); err != nil {
		return nil, fmt.Errorf("failed to create backup script run: %w", err)
	}
	return row, nil
}

// FindBackupScriptRunByID returns a backup script run by its ID.
func FindBackupScriptRunByID(q *reform.Querier, id string) (*BackupScriptRun, error) {
	if id == "" {
		return nil, errors.New("provided backup script run ID is empty")
	}
	row := &BackupScriptRun{ID: id}
	switch err := q.Reload(row); {
	case err == nil:
		return row, nil
	case errors.Is(err, reform.ErrNoRows):
		return nil, fmt.Errorf("%w: backup script run with ID %q not found", ErrNotFound, id)
	default:
		return nil, err
	}
}

// FindBackupScriptRuns returns the catalog of dispatched backup runs.
func FindBackupScriptRuns(q *reform.Querier) ([]*BackupScriptRun, error) {
	rows, err := q.SelectAllFrom(BackupScriptRunTable, "ORDER BY started_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to list backup script runs: %w", err)
	}
	runs := make([]*BackupScriptRun, len(rows))
	for i, r := range rows {
		runs[i] = r.(*BackupScriptRun) //nolint:forcetypeassert
	}
	return runs, nil
}

// UpdateBackupScriptRun persists status/result changes for a run.
func UpdateBackupScriptRun(q *reform.Querier, run *BackupScriptRun) error {
	if err := q.Update(run); err != nil {
		return fmt.Errorf("failed to update backup script run: %w", err)
	}
	return nil
}
