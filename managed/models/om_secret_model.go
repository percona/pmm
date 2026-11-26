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

// OmSecretGeneration is a secret's rotation generation.
type OmSecretGeneration string

// Secret generations. The current one is replaced only after a step proves a pending value.
const (
	OmSecretCurrent = OmSecretGeneration("current")
	OmSecretPending = OmSecretGeneration("pending")
	OmSecretRetired = OmSecretGeneration("retired")
)

// OmSecret is a credential only PMM holds; write it through CreateOmSecret so payload is encrypted.
//
//reform:om_secrets
type OmSecret struct {
	ID            string             `reform:"id,pk"`
	Purpose       string             `reform:"purpose"`
	Discriminator *string            `reform:"discriminator"`
	ClusterID     *string            `reform:"cluster_id"`
	ReplicaSetRef *string            `reform:"replica_set_ref"`
	OperationID   *string            `reform:"operation_id"`
	OwnerKind     *string            `reform:"owner_kind"`
	Generation    OmSecretGeneration `reform:"generation"`
	Username      *string            `reform:"username"`
	Payload       string             `reform:"payload"`
	CreatedAt     time.Time          `reform:"created_at"`
	RotatedAt     *time.Time         `reform:"rotated_at"`
	ExpiresAt     *time.Time         `reform:"expires_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (s *OmSecret) BeforeInsert() error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = Now()
	}
	if s.Generation == "" {
		s.Generation = OmSecretCurrent
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (s *OmSecret) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	if s.RotatedAt != nil {
		t := s.RotatedAt.UTC()
		s.RotatedAt = &t
	}
	if s.ExpiresAt != nil {
		t := s.ExpiresAt.UTC()
		s.ExpiresAt = &t
	}
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*OmSecret)(nil)
	_ reform.AfterFinder    = (*OmSecret)(nil)
)
