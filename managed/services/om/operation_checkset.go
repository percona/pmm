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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// CheckFact is one fact a check set depends on, with the maximum age at which it is still fresh. A
// secret-typed fact never enters the report digest.
type CheckFact struct {
	Name   string
	MaxAge time.Duration
	Secret bool
}

// CheckContext is what a blocking rule reads: the non-secret fact values and the operation's cluster.
type CheckContext struct {
	ClusterID string
	Facts     map[string]any
}

// RuleResult is a blocking rule's verdict: Blocked names what it blocks and why.
type RuleResult struct {
	Blocked bool
	Reason  string
	Path    string
}

// BlockingRule evaluates one precondition against the facts and accepted state.
type BlockingRule func(CheckContext) RuleResult

// CheckSet is a kind's declared facts and blocking rules; the first real set is install readiness (E0-S1).
type CheckSet struct {
	Facts []CheckFact
	Rules []BlockingRule
}

// ScanRequester issues a scan through E0-T4's named interface; a fake records the request in tests.
type ScanRequester interface {
	RequestScan(operationID, profile, idempotencyKey string) error
}

// computeReportDigest is the staleness token pmm-managed computes over the declared non-secret fact
// values, in a canonical form; a secret-typed fact present in the facts is rejected so it cannot leak.
func computeReportDigest(set CheckSet, facts map[string]any) (string, error) {
	included := make(map[string]any, len(set.Facts))
	for _, fact := range set.Facts {
		value, present := facts[fact.Name]
		if fact.Secret {
			if present {
				return "", fmt.Errorf("secret-typed fact %s cannot enter the report digest", fact.Name)
			}
			continue
		}
		included[fact.Name] = value
	}
	encoded, err := json.Marshal(included)
	if err != nil {
		return "", fmt.Errorf("failed to encode report facts: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// runBlockingRules returns the first rule that blocks, or an empty result when all pass.
func runBlockingRules(set CheckSet, ctx CheckContext) RuleResult {
	for _, rule := range set.Rules {
		result := rule(ctx)
		if result.Blocked {
			return result
		}
	}
	return RuleResult{}
}

// reportStale reports whether the stored digest no longer matches the live facts, or any declared
// fact is older than its maximum age, so the driver must request a fresh scan.
func reportStale(set CheckSet, storedDigest string, facts map[string]any, ages map[string]time.Duration) (bool, error) {
	live, err := computeReportDigest(set, facts)
	if err != nil {
		return false, err
	}
	if live != storedDigest {
		return true, nil
	}
	for _, fact := range set.Facts {
		age, ok := ages[fact.Name]
		if ok && age > fact.MaxAge {
			return true, nil
		}
	}
	return false, nil
}
