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

// OmOperationStatus is an operation's lifecycle status.
type OmOperationStatus string

// Operation statuses; external is a non-driver lock hold a PMM backup writer uses, never the driver.
const (
	OmOperationChecking             = OmOperationStatus("checking")
	OmOperationAwaitingConfirmation = OmOperationStatus("awaiting_confirmation")
	OmOperationQueued               = OmOperationStatus("queued")
	OmOperationRunning              = OmOperationStatus("running")
	OmOperationCancelling           = OmOperationStatus("cancelling")
	OmOperationSucceeded            = OmOperationStatus("succeeded")
	OmOperationFailed               = OmOperationStatus("failed")
	OmOperationCancelled            = OmOperationStatus("cancelled")
	OmOperationRolledBack           = OmOperationStatus("rolled_back")
	OmOperationAbandoned            = OmOperationStatus("abandoned")
	OmOperationExternal             = OmOperationStatus("external")
)

// OmOperationTrigger says what started an operation.
type OmOperationTrigger string

// Operation triggers.
const (
	OmTriggerUser     = OmOperationTrigger("user")
	OmTriggerSchedule = OmOperationTrigger("schedule")
	OmTriggerPolicy   = OmOperationTrigger("policy")
)

// OmOperation is the history and handle on one managed-cluster operation.
//
//reform:om_operations
type OmOperation struct {
	ID                 string             `reform:"id,pk"`
	Kind               string             `reform:"kind"`
	KindVersion        int32              `reform:"kind_version"`
	ClusterID          *string            `reform:"cluster_id"`
	ParentOperationID  *string            `reform:"parent_operation_id"`
	Status             OmOperationStatus  `reform:"status"`
	Version            int32              `reform:"version"`
	ParamsRevision     int32              `reform:"params_revision"`
	Trigger            OmOperationTrigger `reform:"trigger"`
	Principal          []byte             `reform:"principal"`
	Params             []byte             `reform:"params"`
	Report             []byte             `reform:"report"`
	ReportDigest       *string            `reform:"report_digest"`
	ErrorCode          *string            `reform:"error_code"`
	ErrorFields        []byte             `reform:"error_fields"`
	StateVersionBefore *int32             `reform:"state_version_before"`
	StateVersionAfter  *int32             `reform:"state_version_after"`
	NotBefore          *time.Time         `reform:"not_before"`
	DeadlineAt         *time.Time         `reform:"deadline_at"`
	ConfirmedBy        *string            `reform:"confirmed_by"`
	CancelledBy        *string            `reform:"cancelled_by"`
	CreatedAt          time.Time          `reform:"created_at"`
	StartedAt          *time.Time         `reform:"started_at"`
	FinishedAt         *time.Time         `reform:"finished_at"`
}

// BeforeInsert implements reform.BeforeInserter. It sets the defaults the migration drops
// from reform-managed tables (version, params_revision, the NOT NULL JSONB objects).
func (o *OmOperation) BeforeInsert() error {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = Now()
	}
	if o.Version == 0 {
		o.Version = 1
	}
	if o.ParamsRevision == 0 {
		o.ParamsRevision = 1
	}
	if o.Params == nil {
		o.Params = []byte("{}")
	}
	if o.ErrorFields == nil {
		o.ErrorFields = []byte("{}")
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (o *OmOperation) AfterFind() error {
	o.CreatedAt = o.CreatedAt.UTC()
	for _, t := range []**time.Time{&o.NotBefore, &o.DeadlineAt, &o.StartedAt, &o.FinishedAt} {
		if *t != nil {
			u := (*t).UTC()
			*t = &u
		}
	}
	return nil
}

// OmStepTargetKind is what a dispatched step targets.
type OmStepTargetKind string

// Step target kinds. A run-level step targets its operation; member and replica_set are
// scoped to the operation's cluster; node is cross-cluster.
const (
	OmStepTargetMember     = OmStepTargetKind("member")
	OmStepTargetReplicaSet = OmStepTargetKind("replica_set")
	OmStepTargetNode       = OmStepTargetKind("node")
	OmStepTargetRun        = OmStepTargetKind("run")
)

// OmStepStatus is the only status of a dispatch, written before anything is sent.
type OmStepStatus string

// Step statuses.
const (
	OmStepPlanned     = OmStepStatus("planned")
	OmStepDispatching = OmStepStatus("dispatching")
	OmStepRunning     = OmStepStatus("running")
	OmStepSucceeded   = OmStepStatus("succeeded")
	OmStepFailed      = OmStepStatus("failed")
	OmStepUnknown     = OmStepStatus("unknown")
)

// OmOperationStep is one dispatch of an operation; Extensions keeps a matching receipt with no status.
//
//reform:om_operation_steps
type OmOperationStep struct {
	ID             string           `reform:"id,pk"`
	OperationID    string           `reform:"operation_id"`
	TargetKind     OmStepTargetKind `reform:"target_kind"`
	TargetID       string           `reform:"target_id"`
	Phase          string           `reform:"phase"`
	Step           string           `reform:"step"`
	Attempt        int32            `reform:"attempt"`
	IdempotencyKey string           `reform:"idempotency_key"`
	Epoch          int64            `reform:"epoch"`
	Status         OmStepStatus     `reform:"status"`
	Result         []byte           `reform:"result"`
	TaskHistoryID  *string          `reform:"task_history_id"`
	DispatchedAt   *time.Time       `reform:"dispatched_at"`
	FinishedAt     *time.Time       `reform:"finished_at"`
}

// AfterFind implements reform.AfterFinder.
func (s *OmOperationStep) AfterFind() error {
	if s.DispatchedAt != nil {
		t := s.DispatchedAt.UTC()
		s.DispatchedAt = &t
	}
	if s.FinishedAt != nil {
		t := s.FinishedAt.UTC()
		s.FinishedAt = &t
	}
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*OmOperation)(nil)
	_ reform.AfterFinder    = (*OmOperation)(nil)
	_ reform.AfterFinder    = (*OmOperationStep)(nil)
)
