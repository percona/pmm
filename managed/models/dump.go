package models

import (
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

type DumpStatus string

const (
	DumpStatusInProgress = DumpStatus("in_progress")
	DumpStatusSuccess    = DumpStatus("success")
	DumpStatusError      = DumpStatus("error")
)

// Validate validates Dumps status.
func (ds DumpStatus) Validate() error {
	switch ds {
	case DumpStatusInProgress:
	case DumpStatusSuccess:
	case DumpStatusError:
	default:
		return NewInvalidArgumentError("invalid dump status '%s'", ds)
	}

	return nil
}

// Pointer returns a pointer to status value.
func (ds DumpStatus) Pointer() *DumpStatus {
	return &ds
}

// Dump represents pmm dump artifact.
//
//reform:dumps
type Dump struct {
	ID        string     `reform:"id,pk"`
	Status    DumpStatus `reform:"status"`
	NodeIDs   []string   `reform:"node_ids"`
	StartTime time.Time  `reform:"start_time"`
	EndTime   time.Time  `reform:"end_time"`
	CreatedAt time.Time  `reform:"created_at"`
	UpdatedAt time.Time  `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (d *Dump) BeforeInsert() error {
	now := Now()
	d.CreatedAt = now
	d.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (d *Dump) BeforeUpdate() error {
	d.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (d *Dump) AfterFind() error {
	d.CreatedAt = d.CreatedAt.UTC()
	d.UpdatedAt = d.UpdatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Dump)(nil)
	_ reform.BeforeUpdater  = (*Dump)(nil)
	_ reform.AfterFinder    = (*Dump)(nil)
)
