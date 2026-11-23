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

// OmOwnedItemAction is how PMM came to own an item.
type OmOwnedItemAction string

// Owned-item actions.
const (
	OmOwnedCreated    = OmOwnedItemAction("created")
	OmOwnedAdopted    = OmOwnedItemAction("adopted")
	OmOwnedChanged    = OmOwnedItemAction("changed")
	OmOwnedHandedOver = OmOwnedItemAction("handed_over")
)

// OmOwnedItemState is an owned item's lifecycle state.
type OmOwnedItemState string

// Owned-item states.
const (
	OmOwnedPlanned = OmOwnedItemState("planned")
	OmOwnedPresent = OmOwnedItemState("present")
	OmOwnedRemoved = OmOwnedItemState("removed")
)

// OmOwnedItemOnRelease is what happens to an owned item when PMM stops managing a cluster.
type OmOwnedItemOnRelease string

// Owned-item release policies.
const (
	OmOnReleaseRemove = OmOwnedItemOnRelease("remove")
	OmOnReleaseRevert = OmOwnedItemOnRelease("revert")
	OmOnReleaseKeep   = OmOwnedItemOnRelease("keep")
	OmOnReleaseAsk    = OmOwnedItemOnRelease("ask")
)

// OmOwnedItem is a user, role, file, change or marker PMM owns on a cluster.
//
//reform:om_owned_items
type OmOwnedItem struct {
	ID                 string               `reform:"id,pk"`
	ClusterID          string               `reform:"cluster_id"`
	ReplicaSetRef      *string              `reform:"replica_set_ref"`
	MemberID           *string              `reform:"member_id"`
	Kind               string               `reform:"kind"`
	NaturalKey         string               `reform:"natural_key"`
	Action             OmOwnedItemAction    `reform:"action"`
	Purpose            string               `reform:"purpose"`
	State              OmOwnedItemState     `reform:"state"`
	Spec               []byte               `reform:"spec"`
	Before             []byte               `reform:"before"`
	BeforeSecretID     *string              `reform:"before_secret_id"`
	OnRelease          OmOwnedItemOnRelease `reform:"on_release"`
	SecretID           *string              `reform:"secret_id"`
	CreatedByOperation string               `reform:"created_by_operation"`
	RemovedByOperation *string              `reform:"removed_by_operation"`
	CreatedAt          time.Time            `reform:"created_at"`
	UpdatedAt          time.Time            `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (i *OmOwnedItem) BeforeInsert() error {
	now := Now()
	if i.CreatedAt.IsZero() {
		i.CreatedAt = now
	}
	if i.UpdatedAt.IsZero() {
		i.UpdatedAt = now
	}
	if i.Spec == nil {
		i.Spec = []byte("{}")
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater.
func (i *OmOwnedItem) BeforeUpdate() error {
	i.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder.
func (i *OmOwnedItem) AfterFind() error {
	i.CreatedAt = i.CreatedAt.UTC()
	i.UpdatedAt = i.UpdatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*OmOwnedItem)(nil)
	_ reform.BeforeUpdater  = (*OmOwnedItem)(nil)
	_ reform.AfterFinder    = (*OmOwnedItem)(nil)
)
