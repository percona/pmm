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

import "github.com/percona/pmm/managed/models"

// insertStatuses are the only statuses a row may be created with: handlers insert checking; the
// external-writer lock API inserts external directly.
var insertStatuses = map[models.OmOperationStatus]bool{
	models.OmOperationChecking: true,
	models.OmOperationExternal: true,
}

// legalTransitions is the operation state machine: for each status, the statuses it may move to.
// A self-edge (checking->checking, queued->queued) is a re-check or re-queue.
var legalTransitions = map[models.OmOperationStatus]map[models.OmOperationStatus]bool{
	models.OmOperationChecking: set(
		models.OmOperationAwaitingConfirmation, models.OmOperationQueued, models.OmOperationRunning,
		models.OmOperationChecking, models.OmOperationFailed, models.OmOperationCancelled,
	),
	models.OmOperationAwaitingConfirmation: set(
		models.OmOperationRunning, models.OmOperationChecking, models.OmOperationQueued, models.OmOperationCancelled,
	),
	models.OmOperationQueued: set(
		models.OmOperationRunning, models.OmOperationQueued, models.OmOperationChecking,
		models.OmOperationCancelled, models.OmOperationFailed,
	),
	models.OmOperationRunning: set(
		models.OmOperationCancelling, models.OmOperationSucceeded, models.OmOperationFailed,
		models.OmOperationCancelled, models.OmOperationRolledBack,
	),
	models.OmOperationCancelling: set(
		models.OmOperationCancelled, models.OmOperationRolledBack, models.OmOperationFailed,
	),
	models.OmOperationFailed: set(
		models.OmOperationCancelled, models.OmOperationRunning, models.OmOperationAbandoned,
	),
	models.OmOperationExternal: set(models.OmOperationCancelled),
}

// terminalStatuses are the fully terminal statuses the driver never claims; failed is excluded
// because it waits for Resume or Abandon.
var terminalStatuses = set(
	models.OmOperationSucceeded, models.OmOperationCancelled,
	models.OmOperationRolledBack, models.OmOperationAbandoned,
)

// driverStatuses are the statuses the single fenced driver claims; it never claims a terminal, a
// failed (until Resume), or an external hold.
var driverStatuses = set(
	models.OmOperationChecking, models.OmOperationQueued, models.OmOperationAwaitingConfirmation,
	models.OmOperationRunning, models.OmOperationCancelling,
)

func set[T comparable](values ...T) map[T]bool {
	out := make(map[T]bool, len(values))
	for _, v := range values {
		out[v] = true
	}
	return out
}

// CanInsertStatus reports whether a row may be created with the given status.
func CanInsertStatus(status models.OmOperationStatus) bool {
	return insertStatuses[status]
}

// CanTransition reports whether the state machine allows a move from one status to another.
func CanTransition(from, to models.OmOperationStatus) bool {
	return legalTransitions[from][to]
}

// IsTerminalStatus reports whether a status is fully terminal, so the driver leaves it alone.
func IsTerminalStatus(status models.OmOperationStatus) bool {
	return terminalStatuses[status]
}

// DriverClaims reports whether the fenced driver acts on a row in this status.
func DriverClaims(status models.OmOperationStatus) bool {
	return driverStatuses[status]
}
