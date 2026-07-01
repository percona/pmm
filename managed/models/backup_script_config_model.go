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

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// BackupScriptConfig is the persistent, versioned configuration for a
// script-based MySQL physical backup executed on a DB node through Nomad.
//
//reform:backup_script_configs
type BackupScriptConfig struct {
	ID                   string `reform:"id,pk"`
	Name                 string `reform:"name"`
	ServiceID            string `reform:"service_id"`
	NodeName             string `reform:"node_name"`
	BackupDir            string `reform:"backup_dir"`
	Compress             bool   `reform:"compress"`
	CompressionAlgorithm string `reform:"compression_algorithm"`
	Copies               int32  `reform:"copies"`
	// ReplicaInfo maps to the current payload key XTRABACKUP_REPLICA_INFO
	// (never the stale SLAVE key that caused the --slave-info failure).
	ReplicaInfo      bool   `reform:"replica_info"`
	XtrabackupBinary string `reform:"xtrabackup_binary"`
	RenderedYAML     string `reform:"rendered_yaml"`
	ConfigVersion    int32  `reform:"config_version"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (c *BackupScriptConfig) BeforeInsert() error {
	now := Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (c *BackupScriptConfig) BeforeUpdate() error {
	c.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (c *BackupScriptConfig) AfterFind() error {
	c.CreatedAt = c.CreatedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*BackupScriptConfig)(nil)
	_ reform.BeforeUpdater  = (*BackupScriptConfig)(nil)
	_ reform.AfterFinder    = (*BackupScriptConfig)(nil)
)
