package models

import (
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// SystemTip represents tip for user which can be completed once
// time per system.
//
//reform:system_tip
type SystemTip struct {
	ID          int32 `reform:"id,pk"`
	IsCompleted bool  `reform:"is_completed"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//
//nolint:unparam
func (t *SystemTip) BeforeInsert() error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//
//nolint:unparam
func (t *SystemTip) BeforeUpdate() error {
	t.UpdatedAt = Now()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*SystemTip)(nil)
	_ reform.BeforeUpdater  = (*SystemTip)(nil)
)
