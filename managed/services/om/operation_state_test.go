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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestOperationStateMachine(t *testing.T) {
	t.Parallel()

	legal := []struct{ from, to models.OmOperationStatus }{
		{models.OmOperationChecking, models.OmOperationRunning},
		{models.OmOperationChecking, models.OmOperationAwaitingConfirmation},
		{models.OmOperationChecking, models.OmOperationChecking},
		{models.OmOperationAwaitingConfirmation, models.OmOperationRunning},
		{models.OmOperationAwaitingConfirmation, models.OmOperationQueued},
		{models.OmOperationQueued, models.OmOperationRunning},
		{models.OmOperationRunning, models.OmOperationSucceeded},
		{models.OmOperationRunning, models.OmOperationCancelling},
		{models.OmOperationCancelling, models.OmOperationRolledBack},
		{models.OmOperationFailed, models.OmOperationRunning},
		{models.OmOperationFailed, models.OmOperationAbandoned},
		{models.OmOperationExternal, models.OmOperationCancelled},
	}
	for _, tc := range legal {
		assert.True(t, CanTransition(tc.from, tc.to), "%s -> %s should be legal", tc.from, tc.to)
	}

	illegal := []struct{ from, to models.OmOperationStatus }{
		{models.OmOperationChecking, models.OmOperationSucceeded},
		{models.OmOperationRunning, models.OmOperationChecking},
		{models.OmOperationSucceeded, models.OmOperationRunning},
		{models.OmOperationExternal, models.OmOperationRunning},
		{models.OmOperationAwaitingConfirmation, models.OmOperationSucceeded},
		{models.OmOperationAbandoned, models.OmOperationRunning},
	}
	for _, tc := range illegal {
		assert.False(t, CanTransition(tc.from, tc.to), "%s -> %s should be illegal", tc.from, tc.to)
	}
}

func TestOperationInsertStatuses(t *testing.T) {
	t.Parallel()

	assert.True(t, CanInsertStatus(models.OmOperationChecking))
	assert.True(t, CanInsertStatus(models.OmOperationExternal))
	assert.False(t, CanInsertStatus(models.OmOperationRunning))
	assert.False(t, CanInsertStatus(models.OmOperationSucceeded))
}

func TestOperationTerminalAndDriverClaims(t *testing.T) {
	t.Parallel()

	for _, s := range []models.OmOperationStatus{
		models.OmOperationSucceeded, models.OmOperationCancelled,
		models.OmOperationRolledBack, models.OmOperationAbandoned,
	} {
		assert.True(t, IsTerminalStatus(s), "%s is terminal", s)
		assert.False(t, DriverClaims(s), "driver does not claim %s", s)
	}
	assert.False(t, IsTerminalStatus(models.OmOperationFailed))
	assert.False(t, DriverClaims(models.OmOperationFailed))
	assert.False(t, DriverClaims(models.OmOperationExternal))

	for _, s := range []models.OmOperationStatus{
		models.OmOperationChecking, models.OmOperationQueued,
		models.OmOperationAwaitingConfirmation, models.OmOperationRunning, models.OmOperationCancelling,
	} {
		assert.True(t, DriverClaims(s), "driver claims %s", s)
	}
}

func TestReleasesLocksMatrix(t *testing.T) {
	t.Parallel()

	assert.True(t, ReleasesLocks(models.OmOperationSucceeded, true))
	assert.True(t, ReleasesLocks(models.OmOperationCancelled, false))
	assert.True(t, ReleasesLocks(models.OmOperationAbandoned, true))
	assert.True(t, ReleasesLocks(models.OmOperationFailed, false))
	assert.False(t, ReleasesLocks(models.OmOperationFailed, true))
	assert.False(t, ReleasesLocks(models.OmOperationRunning, false))
}
