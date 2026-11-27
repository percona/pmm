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
	"encoding/json"
	"fmt"
	"time"

	"github.com/percona/pmm/managed/models"
)

// uninstallKind removes a managed cluster PMM installed, from its hosts. The scan profile the check
// provider reads is derived from this name as 'uninstall_check'.
const uninstallKind = "uninstall"

// Uninstall-teardown fact names from PMM Extensions' uninstall_check scan; only the bools are declared
// facts, the rest are read per member from the subject facts, which the aggregate drops.
const (
	factMarkerPresent = "uninstall_check.marker_present"
	factHostReachable = "uninstall_check.host_reachable"
	factCanElevate    = "uninstall_check.can_elevate"
	factConfigSHA256  = "uninstall_check.config_sha256"
	factBinaryVersion = "uninstall_check.binary_version"
	factMarkerRunID   = "uninstall_check.marker_run_id"
	factOwnedPackages = "uninstall_check.owned_packages"
)

// Uninstall readiness reason-codes stored in om_operations.error_code when a rule blocks, for a
// client to render.
const (
	uninstallReasonMarkerAbsent    = "marker_absent"
	uninstallReasonMarkerMismatch  = "marker_mismatch"
	uninstallReasonHostUnreachable = "host_unreachable"
	uninstallReasonCannotElevate   = "cannot_elevate"
	uninstallReasonDrift           = "drift"
)

// uninstallFactMaxAge bounds how old an uninstall-readiness fact may be before it reads as missing.
const uninstallFactMaxAge = 10 * time.Minute

// uninstallDeadline bounds how long an uninstall may stay queued and running before it is timed out.
const uninstallDeadline = 30 * time.Minute

// Uninstall step phase and names: a stop then a path-safe remove run on each member host, then the member's
// service is forgotten from om_inventory so install no longer sees a MongoDB service on the emptied host.
const (
	uninstallHostPhase   = "host"
	uninstallStopStep    = "uninstall_stop"
	uninstallRemoveStep  = "uninstall_remove"
	uninstallForgetStep  = "uninstall_forget"
	uninstallConfirmVerb = "REMOVE"
)

// uninstallStepsPerMember is how many steps a teardown runs on each member: a stop, a remove, then a forget.
const uninstallStepsPerMember = 3

// uninstallConfirmShortIDLen bounds the operation-id prefix the confirm phrase shows to disambiguate
// non-unique cluster names.
const uninstallConfirmShortIDLen = 6

// uninstallMember is one member's sealed removal plan: what a teardown deletes on that host.
type uninstallMember struct {
	MemberID    string   `json:"member_id"`
	Host        string   `json:"host"`
	NodeID      string   `json:"node_id,omitempty"`
	Packages    []string `json:"packages,omitempty"`
	DataPath    string   `json:"data_path,omitempty"`
	ConfigFile  string   `json:"config_file,omitempty"`
	KeyfilePath string   `json:"keyfile_path,omitempty"`
	LogPath     string   `json:"log_path,omitempty"`
	MarkerPath  string   `json:"marker_path,omitempty"`
	ControlUnit string   `json:"control_unit,omitempty"`
}

// uninstallParams is an uninstall operation's sealed removal plan, stored in om_operations.params and
// read back by the lock set, the check subjects and the teardown steps.
type uninstallParams struct {
	ClusterID    string            `json:"cluster_id"`
	ClusterName  string            `json:"cluster_name"`
	InstallRunID string            `json:"install_run_id"`
	OS           string            `json:"os,omitempty"`
	Members      []uninstallMember `json:"members"`
}

// uninstallParamsOf reads an uninstall operation's params, returning an empty value when they are
// unset or malformed; the trigger seals them before the row is written.
func uninstallParamsOf(op *models.OmOperation) uninstallParams {
	params := uninstallParams{}
	if len(op.Params) > 0 {
		_ = json.Unmarshal(op.Params, &params)
	}
	return params
}

// uninstallConfirmPhrase is the server-validated phrase an operator types to confirm a destructive
// uninstall; the short id disambiguates non-unique cluster names.
func uninstallConfirmPhrase(op *models.OmOperation) string {
	params := uninstallParamsOf(op)
	shortID := op.ID
	if len(shortID) > uninstallConfirmShortIDLen {
		shortID = shortID[:uninstallConfirmShortIDLen]
	}
	return fmt.Sprintf("%s %s %s", uninstallConfirmVerb, params.ClusterName, shortID)
}

// UninstallKind returns the uninstall kind: it locks the cluster and its members, exempts attention
// (removal supersedes it), requires a server-validated confirm phrase, and resumes a failed run.
func UninstallKind() OperationKind {
	return OperationKind{
		Kind:          uninstallKind,
		Resumable:     true,
		Deadline:      uninstallDeadline,
		Attention:     AttentionPolicy{ExemptAll: true},
		AuthMethods:   []string{"admin"},
		ConfirmsFor:   func(models.OmOperationTrigger) bool { return true },
		ConfirmPhrase: uninstallConfirmPhrase,
		LockSet: func(op *models.OmOperation) []LockResource {
			params := uninstallParamsOf(op)
			resources := make([]LockResource, 0, len(params.Members)+1)
			resources = append(resources, LockResource{Kind: "cluster", ID: params.ClusterID, Mode: models.OmLockExclusive})
			for _, member := range params.Members {
				resources = append(resources, LockResource{Kind: "member", ID: member.MemberID, Mode: models.OmLockExclusive})
			}
			return resources
		},
		Steps: func(op *models.OmOperation) []StepPlan {
			params := uninstallParamsOf(op)
			plans := make([]StepPlan, 0, len(params.Members)*uninstallStepsPerMember)
			for _, member := range params.Members {
				plans = append(plans,
					StepPlan{Phase: uninstallHostPhase, Step: uninstallStopStep, TargetKind: models.OmStepTargetMember, TargetID: member.MemberID},
					StepPlan{Phase: uninstallHostPhase, Step: uninstallRemoveStep, TargetKind: models.OmStepTargetMember, TargetID: member.MemberID},
					StepPlan{Phase: uninstallHostPhase, Step: uninstallForgetStep, TargetKind: models.OmStepTargetMember, TargetID: member.MemberID},
				)
			}
			return plans
		},
		OnEnterRunning: uninstallOnEnterRunning,
		OnSucceeded:    uninstallOnSucceeded,
		OnTerminal:     uninstallOnTerminal,
		CheckSet:       uninstallReadinessCheckSet,
		Subjects: func(op *models.OmOperation) []CheckSubject {
			params := uninstallParamsOf(op)
			subjects := make([]CheckSubject, 0, len(params.Members))
			for _, member := range params.Members {
				subjects = append(subjects, CheckSubject{Kind: subjectKindMember, Key: member.MemberID})
			}
			return subjects
		},
	}
}

// uninstallReadinessCheckSet declares uninstall readiness's bool facts and its rules; strings are not
// declared, since the aggregate drops them and a declared string fact would rescan the operation forever.
func uninstallReadinessCheckSet(op *models.OmOperation) CheckSet {
	params := uninstallParamsOf(op)
	return CheckSet{
		Facts: []CheckFact{
			{Name: factMarkerPresent, MaxAge: uninstallFactMaxAge},
			{Name: factHostReachable, MaxAge: uninstallFactMaxAge},
			{Name: factCanElevate, MaxAge: uninstallFactMaxAge},
		},
		Rules: []BlockingRule{
			boolFactRule(factMarkerPresent, uninstallReasonMarkerAbsent),
			uninstallMarkerMatchRule(params),
			boolFactRule(factHostReachable, uninstallReasonHostUnreachable),
			boolFactRule(factCanElevate, uninstallReasonCannotElevate),
			uninstallPackageDriftRule(params),
		},
	}
}

// uninstallMarkerMatchRule refuses when a member's host holds an install marker whose run id is not the
// one the plan sealed, so a teardown never removes a different PMM install; an absent marker is ignored.
func uninstallMarkerMatchRule(params uninstallParams) BlockingRule {
	return func(ctx CheckContext) RuleResult {
		for _, member := range params.Members {
			runID, ok := ctx.SubjectFacts[member.MemberID][factMarkerRunID].(string)
			if !ok || runID == "" {
				continue
			}
			if runID != params.InstallRunID {
				return RuleResult{Blocked: true, Reason: uninstallReasonMarkerMismatch, Path: member.MemberID}
			}
		}
		return RuleResult{}
	}
}

// uninstallPackageDriftRule refuses when a member's live owned-package set differs from the plan's, so a
// teardown never removes packages the plan did not record; the operator adopts the live set to proceed.
func uninstallPackageDriftRule(params uninstallParams) BlockingRule {
	return func(ctx CheckContext) RuleResult {
		for _, member := range params.Members {
			live, ok := stringListFact(ctx.SubjectFacts[member.MemberID][factOwnedPackages])
			if !ok {
				continue
			}
			if !sameStringSet(member.Packages, live) {
				return RuleResult{Blocked: true, Reason: uninstallReasonDrift, Path: member.MemberID}
			}
		}
		return RuleResult{}
	}
}

// stringListFact reads a subject fact as a list of strings, reporting not-ok when the fact is absent or
// is not a list of strings, so a missing or malformed fact never reads as an empty set.
func stringListFact(value any) ([]string, bool) {
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	list := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		list = append(list, s)
	}
	return list, true
}

// sameStringSet reports whether two string lists hold the same values, order and duplicates aside.
func sameStringSet(a, b []string) bool {
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}
