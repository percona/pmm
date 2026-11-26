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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func passingCheckSet() CheckSet {
	return CheckSet{
		Facts: []CheckFact{{Name: "ready"}},
		Rules: []BlockingRule{func(CheckContext) RuleResult { return RuleResult{} }},
	}
}

func readyFacts() map[string]any { return map[string]any{"ready": true} }

func driveKind(confirms bool) OperationKind {
	return OperationKind{
		Kind:        "test",
		ConfirmsFor: func(models.OmOperationTrigger) bool { return confirms },
		LockSet: func(op *models.OmOperation) []LockResource {
			return []LockResource{{Kind: "cluster", ID: *op.ClusterID, Mode: models.OmLockExclusive}}
		},
	}
}

func TestDriveCheckingBlockingRuleFails(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	blocking := CheckSet{
		Facts: []CheckFact{{Name: "ready"}},
		Rules: []BlockingRule{func(CheckContext) RuleResult { return RuleResult{Blocked: true, Reason: "not_ready"} }},
	}
	outcome, err := driveChecking(t.Context(), db, op, driveKind(false), blocking, map[string]any{"ready": false}, nil, &fakeScanRequester{})
	require.NoError(t, err)
	assert.Equal(t, CheckFailed, outcome)

	var status string
	var errorCode *string
	require.NoError(t, db.QueryRow("SELECT status, error_code FROM om_operations WHERE id = $1", op.ID).Scan(&status, &errorCode))
	assert.Equal(t, string(models.OmOperationFailed), status)
	require.NotNil(t, errorCode)
	assert.Equal(t, "not_ready", *errorCode)
}

func TestDriveCheckingAwaitsConfirmation(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	outcome, err := driveChecking(t.Context(), db, op, driveKind(true), passingCheckSet(), readyFacts(), nil, &fakeScanRequester{})
	require.NoError(t, err)
	assert.Equal(t, AwaitingConfirmation, outcome)

	var status string
	var digest *string
	require.NoError(t, db.QueryRow("SELECT status, report_digest FROM om_operations WHERE id = $1", op.ID).Scan(&status, &digest))
	assert.Equal(t, string(models.OmOperationAwaitingConfirmation), status)
	assert.NotNil(t, digest)
}

func TestDriveCheckingRunsWithoutConfirmation(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	outcome, err := driveChecking(t.Context(), db, op, driveKind(false), passingCheckSet(), readyFacts(), nil, &fakeScanRequester{})
	require.NoError(t, err)
	assert.Equal(t, StartedRunning, outcome)

	var status string
	require.NoError(t, db.QueryRow("SELECT status FROM om_operations WHERE id = $1", op.ID).Scan(&status))
	assert.Equal(t, string(models.OmOperationRunning), status)
	assert.Equal(t, 1, heldLocks(t, db, op.ID))
}

func TestDriveCheckingRequestsRescanWhenStale(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)
	op.ReportDigest = new("stale-digest")

	fake := &fakeScanRequester{}
	outcome, err := driveChecking(t.Context(), db, op, driveKind(false), passingCheckSet(), readyFacts(), nil, fake)
	require.NoError(t, err)
	assert.Equal(t, CheckRescanRequested, outcome)
	assert.Len(t, fake.requests, 1)
}

func TestDriveCheckingRequestsRescanWhenOverAge(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	op := checkingOp(t, db.Querier, clusterID)

	checkSet := CheckSet{
		Facts: []CheckFact{{Name: "ready", MaxAge: time.Minute}},
		Rules: []BlockingRule{func(CheckContext) RuleResult { return RuleResult{} }},
	}
	digest, err := computeReportDigest(checkSet, readyFacts())
	require.NoError(t, err)
	op.ReportDigest = &digest

	fake := &fakeScanRequester{}
	outcome, err := driveChecking(t.Context(), db, op, driveKind(false), checkSet, readyFacts(), map[string]time.Duration{"ready": time.Hour}, fake)
	require.NoError(t, err)
	assert.Equal(t, CheckRescanRequested, outcome)
	assert.Len(t, fake.requests, 1)
}
