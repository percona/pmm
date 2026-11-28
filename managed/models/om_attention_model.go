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

// OmAttentionSubjectKind is what an attention row is about.
type OmAttentionSubjectKind string

// Attention subject kinds.
const (
	OmAttentionCluster    = OmAttentionSubjectKind("cluster")
	OmAttentionReplicaSet = OmAttentionSubjectKind("replica_set")
	OmAttentionMember     = OmAttentionSubjectKind("member")
)

// OmAttention is a reason a cluster needs attention; it is written only by SQL functions.
//
//reform:om_attention
type OmAttention struct {
	ClusterID   string                 `reform:"cluster_id"`
	Reason      string                 `reform:"reason"`
	SubjectKind OmAttentionSubjectKind `reform:"subject_kind"`
	SubjectID   string                 `reform:"subject_id"`
	Path        string                 `reform:"path"`
	Detail      []byte                 `reform:"detail"`
	Evidence    []byte                 `reform:"evidence"`
	ObservedAt  time.Time              `reform:"observed_at"`
	ResolvedAt  *time.Time             `reform:"resolved_at"`
	Since       time.Time              `reform:"since"`
}

// AfterFind implements reform.AfterFinder.
func (a *OmAttention) AfterFind() error {
	a.ObservedAt = a.ObservedAt.UTC()
	a.Since = a.Since.UTC()
	if a.ResolvedAt != nil {
		t := a.ResolvedAt.UTC()
		a.ResolvedAt = &t
	}
	return nil
}

// check interfaces.
var (
	_ reform.AfterFinder = (*OmAttention)(nil)
)
