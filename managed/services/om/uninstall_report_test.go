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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

func readyUninstallFacts() map[string]any {
	return map[string]any{
		factMarkerPresent: true,
		factMarkerRunID:   "run-7",
		factHostReachable: true,
		factCanElevate:    true,
		factOwnedPackages: []any{"percona-server-mongodb"},
	}
}

func TestUninstallMemberReadiness(t *testing.T) {
	t.Parallel()
	member := uninstallMember{MemberID: "m", Packages: []string{"percona-server-mongodb"}}
	with := func(mutate func(map[string]any)) map[string]any {
		facts := readyUninstallFacts()
		mutate(facts)
		return facts
	}
	for _, tc := range []struct {
		name  string
		facts map[string]any
		want  string
	}{
		{"ready", readyUninstallFacts(), uninstallReadinessReady},
		{"no facts", map[string]any{}, uninstallReadinessChecking},
		{"marker absent", with(func(f map[string]any) { f[factMarkerPresent] = false }), uninstallReasonMarkerAbsent},
		{"marker mismatch", with(func(f map[string]any) { f[factMarkerRunID] = "run-other" }), uninstallReasonMarkerMismatch},
		{"host unreachable", with(func(f map[string]any) { f[factHostReachable] = false }), uninstallReasonHostUnreachable},
		{"cannot elevate", with(func(f map[string]any) { f[factCanElevate] = false }), uninstallReasonCannotElevate},
		{"drift", with(func(f map[string]any) { f[factOwnedPackages] = []any{"percona-server-mongodb", "extra"} }), uninstallReasonDrift},
		{"reachable missing reads as checking", with(func(f map[string]any) { delete(f, factHostReachable) }), uninstallReadinessChecking},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, uninstallMemberReadiness(member, tc.facts, "run-7"))
		})
	}
}

func insertUninstallOp(t *testing.T, db *reform.DB, clusterID string, st models.OmOperationStatus, errorCode, digest *string, members []uninstallMember) string {
	t.Helper()
	raw, err := json.Marshal(uninstallParams{ClusterID: clusterID, ClusterName: "rs0", InstallRunID: "run-7", Members: members})
	require.NoError(t, err)
	op := &models.OmOperation{
		ID: uuid.New().String(), Kind: uninstallKind, KindVersion: 1, ClusterID: &clusterID,
		Status: st, Version: 1, ParamsRevision: 1, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
		Params: raw, ReportDigest: digest, ErrorCode: errorCode,
	}
	if errorCode != nil {
		op.ErrorFields = []byte("{}")
	}
	require.NoError(t, db.Insert(op))
	return op.ID
}

func TestGetUninstallReportRejectsNonUninstall(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	svc.uninstallFactReader = &fakeFactReader{}
	clusterID := lockCluster(t, db.Querier)

	installID := uuid.New().String()
	require.NoError(t, db.Insert(&models.OmOperation{
		ID: installID, Kind: installKind, KindVersion: 1, ClusterID: &clusterID,
		Status: models.OmOperationSucceeded, Trigger: models.OmTriggerUser, Principal: []byte(`{}`),
	}))

	_, err := svc.GetUninstallReport(t.Context(), &omv1.GetUninstallReportRequest{OperationId: installID})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = svc.GetUninstallReport(t.Context(), &omv1.GetUninstallReportRequest{OperationId: uuid.New().String()})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetUninstallReportReady(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	member := uninstallMember{
		MemberID: "member-a", Host: "node00", NodeID: "node-id-00",
		Packages: []string{"percona-server-mongodb"}, DataPath: "/var/lib/mongodb",
		ConfigFile: "/etc/mongod.conf", KeyfilePath: "/etc/mongod.keyfile", LogPath: "/var/log/mongod.log",
	}
	svc.uninstallFactReader = &fakeFactReader{
		facts: map[string]map[string]any{"member-a": readyUninstallFacts()},
		at:    map[string]time.Time{"member-a": time.Now()},
	}
	clusterID := lockCluster(t, db.Querier)
	digest := "digest-1"
	opID := insertUninstallOp(t, db, clusterID, models.OmOperationAwaitingConfirmation, nil, &digest, []uninstallMember{member})

	resp, err := svc.GetUninstallReport(t.Context(), &omv1.GetUninstallReportRequest{OperationId: opID})
	require.NoError(t, err)
	report := resp.Report
	assert.True(t, report.Ready)
	assert.False(t, report.Drifted)
	assert.Equal(t, clusterID, report.ClusterId)
	require.NotNil(t, report.ClusterName)
	assert.Equal(t, "rs0", *report.ClusterName)
	assert.Equal(t, "REMOVE rs0 "+opID[:uninstallConfirmShortIDLen], report.ConfirmPhrase)
	require.NotNil(t, report.ReportDigest)
	assert.Equal(t, digest, *report.ReportDigest)
	require.Len(t, report.Members, 1)
	got := report.Members[0]
	assert.Equal(t, "member-a", got.MemberId)
	assert.Equal(t, "node00", got.Host)
	assert.Equal(t, uninstallReadinessReady, got.Readiness)
	require.NotNil(t, got.DataPath)
	assert.Equal(t, "/var/lib/mongodb", *got.DataPath)
	assert.Equal(t, []string{"percona-server-mongodb"}, got.Packages)
}

func TestGetUninstallReportDrifted(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	member := uninstallMember{MemberID: "member-a", Host: "node00", Packages: []string{"percona-server-mongodb"}}
	svc.uninstallFactReader = &fakeFactReader{
		facts: map[string]map[string]any{"member-a": {
			factMarkerPresent: true,
			factMarkerRunID:   "run-7",
			factHostReachable: true,
			factCanElevate:    true,
			factOwnedPackages: []any{"percona-server-mongodb", "percona-server-mongodb-tools"},
		}},
		at: map[string]time.Time{"member-a": time.Now()},
	}
	clusterID := lockCluster(t, db.Querier)
	drift := uninstallReasonDrift
	opID := insertUninstallOp(t, db, clusterID, models.OmOperationFailed, &drift, nil, []uninstallMember{member})

	resp, err := svc.GetUninstallReport(t.Context(), &omv1.GetUninstallReportRequest{OperationId: opID})
	require.NoError(t, err)
	assert.False(t, resp.Report.Ready)
	assert.True(t, resp.Report.Drifted)
	require.Len(t, resp.Report.Members, 1)
	assert.Equal(t, uninstallReasonDrift, resp.Report.Members[0].Readiness)
	assert.Nil(t, resp.Report.ReportDigest)
}

func TestGetUninstallReportStaleFactReadsAsChecking(t *testing.T) {
	db := lockTestDB(t)
	svc := uninstallService(t, db)
	member := uninstallMember{MemberID: "member-a", Host: "node00", Packages: []string{"percona-server-mongodb"}}
	svc.uninstallFactReader = &fakeFactReader{
		facts: map[string]map[string]any{"member-a": readyUninstallFacts()},
		at:    map[string]time.Time{"member-a": time.Now().Add(-2 * uninstallFactMaxAge)},
	}
	clusterID := lockCluster(t, db.Querier)
	opID := insertUninstallOp(t, db, clusterID, models.OmOperationChecking, nil, nil, []uninstallMember{member})

	resp, err := svc.GetUninstallReport(t.Context(), &omv1.GetUninstallReportRequest{OperationId: opID})
	require.NoError(t, err)
	assert.False(t, resp.Report.Ready)
	require.Len(t, resp.Report.Members, 1)
	assert.Equal(t, uninstallReadinessChecking, resp.Report.Members[0].Readiness)
}
