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

//go:generate ../../bin/reform

// Component stores info about DBaaS Component.
type Component struct {
	DisabledVersions []string
	DefaultVersion   string
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c Component) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *Component) Scan(src interface{}) error { return jsonScan(c, src) }

// KubernetesCluster represents a Kubernetes cluster as stored in database.
//
//reform:kubernetes_clusters
type KubernetesCluster struct {
	ID                    string     `reform:"id,pk"`
	KubernetesClusterName string     `reform:"kubernetes_cluster_name"`
	KubeConfig            string     `reform:"kube_config"`
	IsReady               bool       `reform:"ready"`
	PXC                   *Component `reform:"pxc"`
	ProxySQL              *Component `reform:"proxysql"`
	HAProxy               *Component `reform:"haproxy"`
	Mongod                *Component `reform:"mongod"`
	CreatedAt             time.Time  `reform:"created_at"`
	UpdatedAt             time.Time  `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *KubernetesCluster) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *KubernetesCluster) BeforeUpdate() error {
	s.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *KubernetesCluster) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*KubernetesCluster)(nil)
	_ reform.BeforeUpdater  = (*KubernetesCluster)(nil)
	_ reform.AfterFinder    = (*KubernetesCluster)(nil)
)
