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
	"database/sql/driver"
	"time"
)

//go:generate $PMM_RELEASE_PATH/reform

// ComputeResources represents container computer resources requests or limits.
type ComputeResources struct {
	CpuM        int32 `json:"cpu_m,omitempty"`
	MemoryBytes int64 `json:"memory_bytes,omitempty"`
}

// ComponentParams represents DB cluster component params.
type ComponentParams struct {
	Image            string            `json:"image,omitempty"`
	ComputeResources *ComputeResources `json:"compute_resources,omitempty"`
	DiskSize         int64             `json:"disk_size,omitempty"`
}

type PXCClusterParams struct {
	ClusterSize int32            `json:"cluster_size,omitempty"`
	Pxc         *ComponentParams `json:"pxc,omitempty"`
	Proxysql    *ComponentParams `json:"proxysql,omitempty"`
	Haproxy     *ComponentParams `json:"haproxy,omitempty"`
}

type PSMDBClusterParams struct {
	ClusterSize int32            `json:"cluster_size,omitempty"`
	Psmdb       *ComponentParams `json:"psmdb,omitempty"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c PXCClusterParams) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *PXCClusterParams) Scan(src interface{}) error { return jsonScan(c, src) }

type DBClusterType string

// Agent types (in the same order as in agents.proto).
const (
	PSMDBType DBClusterType = "psmdb"
	PXCType   DBClusterType = "pxc"
)

// DBCluster represents a Database cluster as stored in database.
//reform:db_clusters
type DBCluster struct {
	ID                  string        `reform:"id,pk"`
	ClusterType         DBClusterType `reform:"cluster_type"`
	KubernetesClusterID string        `reform:"kubernetes_cluster_id"`
	Name                string        `reform:"name"`
	InstalledImage      string        `reform:"installed_image"`
	CreatedAt           time.Time     `reform:"created_at"`
	UpdatedAt           time.Time     `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *DBCluster) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *DBCluster) BeforeUpdate() error {
	r.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *DBCluster) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()

	return nil
}
