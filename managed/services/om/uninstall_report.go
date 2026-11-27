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
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// Report-only readiness verdicts, beside the uninstall reason-codes: "ready" is a member a teardown
// may remove, "checking" one no fresh scan has reported yet.
const (
	uninstallReadinessReady    = "ready"
	uninstallReadinessChecking = "checking"
)

// GetUninstallReport returns an uninstall operation's readiness report: what the teardown will remove and
// whether each member is ready, the confirm phrase and the reviewed digest, redacted of params and output.
func (s *Service) GetUninstallReport(ctx context.Context, req *omv1.GetUninstallReportRequest) (*omv1.GetUninstallReportResponse, error) {
	op, _, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	if op.Kind != uninstallKind {
		return nil, status.Errorf(codes.FailedPrecondition, "operation '%s' is not an uninstall", op.ID)
	}
	params := uninstallParamsOf(op)
	report := &omv1.UninstallReport{
		OperationId:   op.ID,
		ClusterId:     params.ClusterID,
		ConfirmPhrase: uninstallConfirmPhrase(op),
		ReportDigest:  op.ReportDigest,
		Ready:         op.Status == models.OmOperationAwaitingConfirmation,
		Drifted:       op.ErrorCode != nil && *op.ErrorCode == uninstallReasonDrift,
	}
	if params.ClusterName != "" {
		report.ClusterName = &params.ClusterName
	}
	maxAges := uninstallFactMaxAges(op)
	now := time.Now()
	report.Members = make([]*omv1.UninstallReportMember, 0, len(params.Members))
	for _, member := range params.Members {
		facts, collectedAt, obsErr := s.uninstallFactReader.Observation(ctx, subjectKindMember, member.MemberID, uninstallKind+"_check")
		if obsErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to read member %s readiness: %s", member.MemberID, obsErr)
		}
		fresh := freshFacts(facts, collectedAt, maxAges, now)
		report.Members = append(report.Members, uninstallReportMember(member, uninstallMemberReadiness(member, fresh, params.InstallRunID)))
	}
	return &omv1.GetUninstallReportResponse{Report: report}, nil
}

// uninstallFactMaxAges is the uninstall check set's declared fact ages, so the report drops a stale fact
// the same way the readiness gate does and the member reads as checking rather than carrying a stale verdict.
func uninstallFactMaxAges(op *models.OmOperation) map[string]time.Duration {
	ages := map[string]time.Duration{}
	for _, fact := range uninstallReadinessCheckSet(op).Facts {
		if fact.MaxAge > 0 {
			ages[fact.Name] = fact.MaxAge
		}
	}
	return ages
}

// uninstallMemberReadiness is one member's readiness for the report, mirroring the readiness rules' order
// so the display never disagrees with the gate: a missing bool reads as checking, not as a passing fact.
func uninstallMemberReadiness(member uninstallMember, facts map[string]any, installRunID string) string {
	present, ok := facts[factMarkerPresent].(bool)
	if !ok {
		return uninstallReadinessChecking
	}
	if !present {
		return uninstallReasonMarkerAbsent
	}
	if runID, ok := facts[factMarkerRunID].(string); ok && runID != "" && runID != installRunID {
		return uninstallReasonMarkerMismatch
	}
	reachable, ok := facts[factHostReachable].(bool)
	if !ok {
		return uninstallReadinessChecking
	}
	if !reachable {
		return uninstallReasonHostUnreachable
	}
	canElevate, ok := facts[factCanElevate].(bool)
	if !ok {
		return uninstallReadinessChecking
	}
	if !canElevate {
		return uninstallReasonCannotElevate
	}
	if live, ok := stringListFact(facts[factOwnedPackages]); ok && !sameStringSet(member.Packages, live) {
		return uninstallReasonDrift
	}
	return uninstallReadinessReady
}

// uninstallReportMember maps a sealed member plan and its readiness to the report's wire form, spelling an
// empty optional path as unset rather than "".
func uninstallReportMember(member uninstallMember, readiness string) *omv1.UninstallReportMember {
	entry := &omv1.UninstallReportMember{
		MemberId:  member.MemberID,
		Host:      member.Host,
		Packages:  member.Packages,
		Readiness: readiness,
	}
	entry.NodeId = optional(member.NodeID)
	entry.DataPath = optional(member.DataPath)
	entry.ConfigFile = optional(member.ConfigFile)
	entry.KeyfilePath = optional(member.KeyfilePath)
	entry.LogPath = optional(member.LogPath)
	return entry
}
