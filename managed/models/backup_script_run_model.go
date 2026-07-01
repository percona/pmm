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
	"database/sql/driver"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// ScriptBackupStatus reflects the lifecycle of a script-based backup run.
type ScriptBackupStatus string

// Script backup run statuses.
const (
	ScriptBackupPending ScriptBackupStatus = "pending"
	ScriptBackupRunning ScriptBackupStatus = "running"
	ScriptBackupSuccess ScriptBackupStatus = "success"
	ScriptBackupError   ScriptBackupStatus = "error"
)

// ScriptRunManifest is the machine-readable result the payload writes on
// completion and pmm-managed ingests to catalog the backup.
type ScriptRunManifest struct {
	RunID                string `json:"run_id"`
	ServiceID            string `json:"service_id"`
	NodeName             string `json:"node_name"`
	Status               string `json:"status"`
	StartedAt            string `json:"started_at"`
	FinishedAt           string `json:"finished_at"`
	BackupDir            string `json:"backup_dir"`
	SizeBytes            int64  `json:"size_bytes"`
	Compressed           bool   `json:"compressed"`
	CompressionAlgorithm string `json:"compression_algorithm"`
	PXBVersion           string `json:"pxb_version"`
	ServerVersion        string `json:"server_version"`
	CopiesKept           int32  `json:"copies_kept"`
	ConfigVersion        int32  `json:"config_version"`
	AllocID              string `json:"alloc_id"`
}

// Value implements database/sql/driver.Valuer interface.
func (m ScriptRunManifest) Value() (driver.Value, error) { return jsonValue(m) }

// Scan implements database/sql.Scanner interface.
func (m *ScriptRunManifest) Scan(src any) error { return jsonScan(m, src) }

// BackupScriptRun is a single dispatch of a config to a node via Nomad.
//
//reform:backup_script_runs
type BackupScriptRun struct {
	ID         string             `reform:"id,pk"`
	ConfigID   string             `reform:"config_id"`
	ServiceID  string             `reform:"service_id"`
	NodeName   string             `reform:"node_name"`
	NomadJobID string             `reform:"nomad_job_id"`
	Status     ScriptBackupStatus `reform:"status"`
	BackupDir  string             `reform:"backup_dir"`
	SizeBytes  int64              `reform:"size_bytes"`
	Error      string             `reform:"error"`
	Manifest   *ScriptRunManifest `reform:"manifest"`
	StartedAt  time.Time          `reform:"started_at"`
	FinishedAt *time.Time         `reform:"finished_at"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *BackupScriptRun) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *BackupScriptRun) BeforeUpdate() error {
	r.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *BackupScriptRun) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()
	r.StartedAt = r.StartedAt.UTC()
	if r.FinishedAt != nil {
		finished := r.FinishedAt.UTC()
		r.FinishedAt = &finished
	}
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*BackupScriptRun)(nil)
	_ reform.BeforeUpdater  = (*BackupScriptRun)(nil)
	_ reform.AfterFinder    = (*BackupScriptRun)(nil)
)
