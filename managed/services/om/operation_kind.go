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

package om

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// OperationHook runs inside a driver transaction and must be idempotent.
type OperationHook func(tx *reform.TX, op *models.OmOperation) error

// AttentionPolicy declares which active attention reasons block a kind from entering running. A
// feature kind registers its own; the gate defaults an unregistered kind to blocked-by-any.
type AttentionPolicy struct {
	ExemptAll     bool
	ExemptReasons map[string]bool
	DeclaredPaths []string
}

// Exempt reports whether a kind ignores an attention reason entirely.
func (p AttentionPolicy) Exempt(reason string) bool {
	return p.ExemptAll || p.ExemptReasons[reason]
}

// OperationKind is a feature kind's declaration: the knobs the driver reads instead of carrying
// kind-specific code. Later epics register kinds; this ticket ships only a test kind.
type OperationKind struct {
	Kind                    string
	ConfirmsFor             func(models.OmOperationTrigger) bool
	ConfirmPhrase           func(op *models.OmOperation) string
	CancellableWhileRunning bool
	Resumable               bool
	NonInterruptibleSteps   map[string]bool
	LockSet                 func(op *models.OmOperation) []LockResource
	Steps                   func(op *models.OmOperation) []StepPlan
	Deadline                time.Duration
	Attention               AttentionPolicy
	AuthMethods             []string
	OnEnterRunning          OperationHook
	OnSucceeded             OperationHook
	OnTerminal              OperationHook
}

// Confirms reports whether an operation of this kind confirms before running for a trigger.
func (k OperationKind) Confirms(trigger models.OmOperationTrigger) bool {
	if k.ConfirmsFor == nil {
		return false
	}
	return k.ConfirmsFor(trigger)
}

// confirmPhrase is the server-validated phrase this kind seals on an operation at confirmation
// time, empty when the kind requires none.
func (k OperationKind) confirmPhrase(op *models.OmOperation) string {
	if k.ConfirmPhrase == nil {
		return ""
	}
	return k.ConfirmPhrase(op)
}

// OperationRegistry is the set of known operation kinds; a feature registers its kind here.
type OperationRegistry struct {
	kinds map[string]OperationKind
}

// NewOperationRegistry returns an empty registry.
func NewOperationRegistry() *OperationRegistry {
	return &OperationRegistry{kinds: make(map[string]OperationKind)}
}

// Register adds a kind, rejecting a blank or duplicate name.
func (r *OperationRegistry) Register(kind OperationKind) error {
	if kind.Kind == "" {
		return errors.New("operation registry: kind must not be empty")
	}
	_, exists := r.kinds[kind.Kind]
	if exists {
		return fmt.Errorf("operation registry: kind '%s' is already registered", kind.Kind)
	}
	r.kinds[kind.Kind] = kind
	return nil
}

// Kind returns a registered kind.
func (r *OperationRegistry) Kind(name string) (OperationKind, bool) {
	kind, ok := r.kinds[name]
	return kind, ok
}
