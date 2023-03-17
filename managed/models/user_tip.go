package models

import (
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// UserTip represents tip for user.
//
//reform:user_tip
type UserTip struct {
	ID          int32 `reform:"id,pk"`
	UserID      int32 `reform:"user_id"`
	UserTipID   int32 `reform:"user_tip_id"`
	IsCompleted bool  `reform:"is_completed"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//
//nolint:unparam
func (t *UserTip) BeforeInsert() error {
	now := Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//
//nolint:unparam
func (t *UserTip) BeforeUpdate() error {
	t.UpdatedAt = Now()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*UserTip)(nil)
	_ reform.BeforeUpdater  = (*UserTip)(nil)
)
