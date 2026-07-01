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

// Package backupscript dispatches script-based MySQL physical backups to DB
// nodes through Nomad, renders the XtraBackup payload config, and catalogs runs.
package backupscript

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// defaultMyCnfPath is where the Nomad template stanza renders the templated
// client credentials on the node.
const defaultMyCnfPath = "secrets/my.cnf"

// defaultCompressionAlgorithm matches the payload's first supported algorithm.
const defaultCompressionAlgorithm = "zstd"

// ConfigParams are the structured knobs used to render the payload YAML.
// They intentionally map onto the CURRENT payload's REPLICA-era schema, not the
// stale SLAVE-era keys that SEP emits (which default XTRABACKUP_REPLICA_INFO to
// True and add --slave-info, the failure hit on the 8.4 node).
type ConfigParams struct {
	// Alias is the per-server identifier in SERVER_LIST (the PMM service name).
	Alias string
	// Host the payload connects to (localhost for on-node execution).
	Host string
	// Port of the MySQL server.
	Port int32
	// BackupDir is the root BACKUP_DIR on the node.
	BackupDir string
	// Compress toggles COMPRESS.
	Compress bool
	// CompressionAlgorithm maps to COMPRESSION_ALGORITHM.
	CompressionAlgorithm string
	// Copies maps to XTRABACKUP_COPIES.
	Copies int32
	// ReplicaInfo maps to XTRABACKUP_REPLICA_INFO (never the SLAVE key).
	ReplicaInfo bool
	// XtrabackupBinary maps to XTRABACKUP_BIN_CMD.
	XtrabackupBinary string
}

// allServers holds the global ALL_SERVERS defaults. Field order defines the
// rendered YAML key order.
type allServers struct {
	BackupDir             string `yaml:"BACKUP_DIR"`
	Compress              bool   `yaml:"COMPRESS"`
	CompressionAlgorithm  string `yaml:"COMPRESSION_ALGORITHM"`
	XtrabackupCopies      int32  `yaml:"XTRABACKUP_COPIES"`
	XtrabackupReplicaInfo bool   `yaml:"XTRABACKUP_REPLICA_INFO"`
	XtrabackupBinCmd      string `yaml:"XTRABACKUP_BIN_CMD"`
}

// serverEntry is one target in SERVER_LIST. We emit exactly one entry per job so
// there is never a need for a payload --server filter flag.
type serverEntry struct {
	Alias        string `yaml:"ALIAS"`
	Host         string `yaml:"HOST"`
	Port         int32  `yaml:"PORT"`
	BackupDir    string `yaml:"BACKUP_DIR"`
	DefaultsFile string `yaml:"DEFAULTS_FILE"`
}

// payloadConfig is the top-level ALL_SERVERS/SERVER_LIST document.
type payloadConfig struct {
	AllServers allServers    `yaml:"ALL_SERVERS"`
	ServerList []serverEntry `yaml:"SERVER_LIST"`
}

// RenderConfig builds the authoritative backup_config.yaml for the payload from
// structured params. It always emits the current REPLICA-era keys and exactly
// one SERVER_LIST entry.
func RenderConfig(p ConfigParams) (string, error) {
	if p.Alias == "" {
		return "", fmt.Errorf("alias is required")
	}
	if p.BackupDir == "" {
		return "", fmt.Errorf("backup_dir is required")
	}

	algo := p.CompressionAlgorithm
	if algo == "" {
		algo = defaultCompressionAlgorithm
	}
	bin := p.XtrabackupBinary
	if bin == "" {
		bin = "xtrabackup"
	}
	host := p.Host
	if host == "" {
		host = "localhost"
	}
	port := p.Port
	if port == 0 {
		port = 3306
	}
	copies := p.Copies
	if copies == 0 {
		copies = 2
	}

	cfg := payloadConfig{
		AllServers: allServers{
			BackupDir:             p.BackupDir,
			Compress:              p.Compress,
			CompressionAlgorithm:  algo,
			XtrabackupCopies:      copies,
			XtrabackupReplicaInfo: p.ReplicaInfo,
			XtrabackupBinCmd:      bin,
		},
		ServerList: []serverEntry{{
			Alias:        p.Alias,
			Host:         host,
			Port:         port,
			BackupDir:    p.BackupDir,
			DefaultsFile: defaultMyCnfPath,
		}},
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to render backup config: %w", err)
	}
	return string(out), nil
}
