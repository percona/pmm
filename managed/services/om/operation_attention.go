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
	"fmt"
	"slices"

	"github.com/lib/pq"
	"gopkg.in/reform.v1"
)

// Attention reasons this epic raises or registers; later epics raise their own through the same API.
const (
	AttentionDrift              = "drift"
	AttentionIdentityMismatch   = "identity_mismatch"
	AttentionOperationFailed    = "operation_failed"
	AttentionCredentialsInvalid = "credentials_invalid"
)

// AttentionBlock names the active attention row that blocks a kind from entering running.
type AttentionBlock struct {
	Reason    string
	SubjectID string
	Path      string
}

// checkAttentionGate returns the first active (resolved_at IS NULL) attention row that blocks the
// kind under its policy for a cluster and its locked subjects, or ok=false when nothing blocks.
func checkAttentionGate(q *reform.Querier, policy AttentionPolicy, clusterID string, lockedSubjects []string) (AttentionBlock, bool, error) {
	if policy.ExemptAll {
		return AttentionBlock{}, false, nil
	}
	subjects := append([]string{clusterID}, lockedSubjects...)
	rows, err := q.Query(
		"SELECT reason, subject_id, path FROM om_attention "+
			"WHERE cluster_id = $1 AND resolved_at IS NULL AND subject_id = ANY($2::uuid[])",
		clusterID, pq.Array(subjects))
	if err != nil {
		return AttentionBlock{}, false, fmt.Errorf("failed to read attention: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var block AttentionBlock
		err = rows.Scan(&block.Reason, &block.SubjectID, &block.Path)
		if err != nil {
			return AttentionBlock{}, false, fmt.Errorf("failed to scan attention: %w", err)
		}
		if blocksKind(policy, block, clusterID) {
			return block, true, rows.Err()
		}
	}
	return AttentionBlock{}, false, rows.Err()
}

// blocksKind decides whether one active attention row blocks a kind under its policy.
func blocksKind(policy AttentionPolicy, block AttentionBlock, clusterID string) bool {
	if policy.Exempt(block.Reason) {
		return false
	}
	if block.Reason == AttentionIdentityMismatch {
		return block.SubjectID == clusterID
	}
	if len(policy.DeclaredPaths) == 0 {
		return true
	}
	return slices.Contains(policy.DeclaredPaths, block.Path)
}
