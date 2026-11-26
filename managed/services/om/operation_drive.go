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
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// driveChecking runs a checking operation's check set and routes it: a blocking rule fails it, a
// confirming kind awaits confirmation, else the start gate runs it; a stale report requests a rescan.
func driveChecking(
	ctx context.Context, db *reform.DB, op *models.OmOperation, kind OperationKind, checkSet CheckSet,
	facts map[string]any, ages map[string]time.Duration, scanRequester ScanRequester,
) (StartOutcome, error) {
	if op.ClusterID == nil && !kind.ClusterLess {
		return "", errors.New("driver needs a cluster-scoped operation")
	}
	digest, err := computeReportDigest(checkSet, facts)
	if err != nil {
		return "", err
	}
	if staleOrMissing(checkSet, facts, ages) {
		return requestRescan(op, scanRequester)
	}
	if op.ReportDigest != nil {
		stale, err := reportStale(checkSet, *op.ReportDigest, facts, ages)
		if err != nil {
			return "", err
		}
		if stale {
			return requestRescan(op, scanRequester)
		}
	}
	block := runBlockingRules(checkSet, CheckContext{ClusterID: clusterScope(op), Facts: facts})
	if block.Blocked {
		return failChecking(ctx, db, op, block.Reason)
	}
	if kind.Confirms(op.Trigger) {
		return awaitConfirmation(ctx, db, op, digest)
	}
	recheck := func(*reform.TX) (bool, error) { return false, nil }
	return startGate(ctx, db, op, models.OmOperationChecking, op.Version, nil, kind, lockSubjects(kind.LockSet(op)), recheck)
}

// staleOrMissing reports whether a fact the check set declares is absent or over its max age, so the
// driver requests a fresh scan and stays checking instead of judging readiness on stale or missing input.
func staleOrMissing(checkSet CheckSet, facts map[string]any, ages map[string]time.Duration) bool {
	for _, declared := range checkSet.Facts {
		_, present := facts[declared.Name]
		if !present {
			return true
		}
		if declared.MaxAge > 0 && ages[declared.Name] > declared.MaxAge {
			return true
		}
	}
	return false
}

// clusterScope returns an operation's cluster id, or the empty string for a cluster-less install.
func clusterScope(op *models.OmOperation) string {
	if op.ClusterID == nil {
		return ""
	}
	return *op.ClusterID
}

// requestRescan issues a scan request for a stale or over-age report and leaves the operation
// checking; the fake records it in tests and E0-T4 fulfils it.
func requestRescan(op *models.OmOperation, scanRequester ScanRequester) (StartOutcome, error) {
	key := op.ID + ":" + strconv.Itoa(int(op.Version))
	err := scanRequester.RequestScan(op.ID, op.Kind+"_check", key)
	if err != nil {
		return "", fmt.Errorf("failed to request rescan: %w", err)
	}
	return CheckRescanRequested, nil
}

// failChecking moves a checking operation to failed with a stable error code naming the block.
func failChecking(ctx context.Context, db *reform.DB, op *models.OmOperation, reason string) (StartOutcome, error) {
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		_, err := tx.Exec(
			"UPDATE om_operations SET status = 'failed', error_code = $1, version = version + 1, finished_at = now() "+
				"WHERE id = $2 AND status = 'checking' AND version = $3",
			reason, op.ID, op.Version)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("failed to fail checking operation: %w", err)
	}
	return CheckFailed, nil
}

// awaitConfirmation seals the reviewed report digest and moves a checking operation to
// awaiting_confirmation; the plan revision sealed at report time is what the confirm CAS fences.
func awaitConfirmation(ctx context.Context, db *reform.DB, op *models.OmOperation, digest string) (StartOutcome, error) {
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		_, err := tx.Exec(
			"UPDATE om_operations SET status = 'awaiting_confirmation', report_digest = $1, version = version + 1 "+
				"WHERE id = $2 AND status = 'checking' AND version = $3",
			digest, op.ID, op.Version)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("failed to await confirmation: %w", err)
	}
	return AwaitingConfirmation, nil
}

// lockSubjects returns the cluster/member/replica-set resource ids a lock set names, for the
// attention gate; node locks are cross-cluster and are not attention subjects.
func lockSubjects(resources []LockResource) []string {
	subjects := make([]string, 0, len(resources))
	for _, r := range resources {
		if r.Kind == "cluster" || r.Kind == "member" || r.Kind == "replica_set" {
			subjects = append(subjects, r.ID)
		}
	}
	return subjects
}
