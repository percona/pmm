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
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// uninstallMarkerPath is the on-host install marker PMM Extensions writes and reads; a teardown removes
// it last, so the sealed plan carries it.
const uninstallMarkerPath = "/etc/mongod.om-bootstrap"

// TriggerUninstall seals a uninstallable cluster's per-member removal plan and records a checking
// uninstall operation the driver gates on readiness; it is idempotent per cluster and admin-only.
func (s *Service) TriggerUninstall(ctx context.Context, req *omv1.TriggerUninstallRequest) (*omv1.TriggerUninstallResponse, error) {
	clusterID := req.GetClusterId()
	if clusterID == "" {
		return nil, status.Error(codes.InvalidArgument, "cluster_id is required")
	}

	var operationID string
	err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		cluster, txErr := findUninstallableCluster(tx, clusterID)
		if txErr != nil {
			return txErr
		}
		existing, txErr := openUninstallOperation(tx, clusterID)
		if txErr != nil {
			return txErr
		}
		if existing != "" {
			operationID = existing
			return nil
		}
		txErr = refuseLockedCluster(tx, clusterID)
		if txErr != nil {
			return txErr
		}
		params, txErr := sealUninstallPlan(tx, cluster)
		if txErr != nil {
			return txErr
		}
		operationID = uuid.New().String()
		op := &models.OmOperation{
			ID:             operationID,
			Kind:           uninstallKind,
			KindVersion:    1,
			ClusterID:      &clusterID,
			Status:         models.OmOperationChecking,
			Version:        1,
			ParamsRevision: 1,
			Trigger:        models.OmTriggerUser,
			Principal:      []byte(`{"actor":"operator"}`),
			Params:         params,
			ErrorFields:    []byte("{}"),
		}
		insertErr := tx.Insert(op)
		if insertErr != nil {
			return status.Errorf(codes.Internal, "failed to record the uninstall operation: %s", insertErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &omv1.TriggerUninstallResponse{OperationId: operationID}, nil
}

// errResealNotMoved marks a reseal whose version fence matched no failed operation, mapped to the
// not-moved precondition so a stale or concurrent reseal changes nothing.
var errResealNotMoved = errors.New("reseal moved no operation")

// ConfirmUninstallLivePlan re-seals a drift-blocked uninstall plan to the members' live package sets and
// returns it to checking so readiness re-runs; it is fenced by version and applies only to drift.
func (s *Service) ConfirmUninstallLivePlan(ctx context.Context, req *omv1.ConfirmUninstallLivePlanRequest) (*omv1.ConfirmUninstallLivePlanResponse, error) {
	op, _, err := getOperation(s.db.Querier, req.GetOperationId())
	if err != nil {
		return nil, operationReadError(err)
	}
	if op.Kind != uninstallKind {
		return nil, status.Error(codes.FailedPrecondition, "operation is not an uninstall")
	}
	if op.Status != models.OmOperationFailed || op.ErrorCode == nil || *op.ErrorCode != uninstallReasonDrift {
		return nil, status.Error(codes.FailedPrecondition, "operation is not blocked on plan drift")
	}
	if s.uninstallFactReader == nil {
		return nil, status.Error(codes.Unavailable, "the operation driver is not configured")
	}
	resealed, err := s.resealLivePackages(ctx, uninstallParamsOf(op))
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(resealed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to re-seal the uninstall plan: %s", err)
	}
	err = s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		res, txErr := tx.Exec(
			"UPDATE om_operations SET params = $1, params_revision = params_revision + 1, "+
				"report_digest = NULL, error_code = NULL, finished_at = NULL, status = 'checking', version = version + 1 "+
				"WHERE id = $2 AND status = 'failed' AND version = $3",
			encoded, op.ID, req.GetVersion())
		if txErr != nil {
			return txErr
		}
		affected, txErr := res.RowsAffected()
		if txErr != nil {
			return txErr
		}
		if affected == 0 {
			return errResealNotMoved
		}
		return nil
	})
	if errors.Is(err, errResealNotMoved) {
		return nil, operationNotMoved(op.Status)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to re-seal the uninstall plan: %s", err)
	}
	refreshed, err := s.reloadOperation(op.ID)
	if err != nil {
		return nil, err
	}
	return &omv1.ConfirmUninstallLivePlanResponse{Operation: refreshed}, nil
}

// resealLivePackages reads each member's live owned-package set and replaces the plan's with it, leaving
// a member with no owned-package fact as the plan sealed it.
func (s *Service) resealLivePackages(ctx context.Context, params uninstallParams) (uninstallParams, error) {
	for i, member := range params.Members {
		facts, _, err := s.uninstallFactReader.Observation(ctx, subjectKindMember, member.MemberID, uninstallKind+"_check")
		if err != nil {
			return uninstallParams{}, status.Errorf(codes.Internal, "failed to read the live facts of member %s: %s", member.MemberID, err)
		}
		live, ok := stringListFact(facts[factOwnedPackages])
		if ok {
			params.Members[i].Packages = live
		}
	}
	return params, nil
}

// findUninstallableCluster loads a cluster and refuses one uninstall does not apply to: it removes only a
// PMM-installed replica set that is still a live managed cluster, never an imported or sharded one.
func findUninstallableCluster(tx *reform.TX, clusterID string) (*models.OmCluster, error) {
	row, err := tx.FindByPrimaryKeyFrom(models.OmClusterTable, clusterID)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "managed cluster %s was not found", clusterID)
		}
		return nil, status.Errorf(codes.Internal, "failed to load the cluster: %s", err)
	}
	cluster := row.(*models.OmCluster) //nolint:forcetypeassert
	if cluster.Origin != models.OmClusterInstalled {
		return nil, status.Errorf(codes.FailedPrecondition, "cluster %s was not installed by PMM, so it cannot be uninstalled", clusterID)
	}
	if cluster.Kind != models.OmClusterReplicaSet {
		return nil, status.Errorf(codes.FailedPrecondition, "only a replica-set cluster can be uninstalled here, not a %s cluster", cluster.Kind)
	}
	if cluster.Status != models.OmClusterManaged && cluster.Status != models.OmClusterMonitored {
		return nil, status.Errorf(codes.FailedPrecondition, "cluster %s is %s, not a live managed cluster to uninstall", clusterID, cluster.Status)
	}
	return cluster, nil
}

// openUninstallOperation returns the id of a non-terminal uninstall operation already open for the
// cluster so a repeated trigger returns it, or an empty string when none is open.
func openUninstallOperation(tx *reform.TX, clusterID string) (string, error) {
	var id string
	err := tx.QueryRow(
		"SELECT id FROM om_operations "+
			"WHERE kind = $1 AND cluster_id = $2 "+
			"AND status NOT IN ('succeeded', 'cancelled', 'rolled_back', 'abandoned', 'external') "+
			"ORDER BY created_at DESC LIMIT 1",
		uninstallKind, clusterID).Scan(&id)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return "", nil
		}
		return "", status.Errorf(codes.Internal, "failed to check for an open uninstall: %s", err)
	}
	return id, nil
}

// refuseLockedCluster refuses when another operation already holds the cluster lock, naming the holder so
// the caller can see what to wait for.
func refuseLockedCluster(tx *reform.TX, clusterID string) error {
	var holder string
	err := tx.QueryRow(
		"SELECT operation_id FROM om_locks WHERE resource_kind = 'cluster' AND resource_id = $1 LIMIT 1",
		clusterID).Scan(&holder)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return status.Errorf(codes.Internal, "failed to check the cluster lock: %s", err)
	}
	return status.Errorf(codes.FailedPrecondition, "cluster %s is busy with operation %s", clusterID, holder)
}

// sealUninstallPlan reads the cluster's live members, its install marker run id and its accepted-state
// manifest, and marshals the per-member removal plan the lock set, check subjects and teardown steps read.
func sealUninstallPlan(tx *reform.TX, cluster *models.OmCluster) ([]byte, error) {
	runID, err := installMarkerRunID(tx, cluster.ID)
	if err != nil {
		return nil, err
	}
	install, err := clusterInstallParams(tx, cluster)
	if err != nil {
		return nil, err
	}
	manifest, err := acceptedMembers(tx, cluster)
	if err != nil {
		return nil, err
	}
	memberRows, err := tx.SelectAllFrom(models.OmClusterMemberTable, "WHERE cluster_id = $1 AND removed_at IS NULL", cluster.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list the cluster members: %s", err)
	}
	if len(memberRows) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "cluster %s has no members to uninstall", cluster.ID)
	}

	members := make([]uninstallMember, 0, len(memberRows))
	for _, row := range memberRows {
		member := row.(*models.OmClusterMember) //nolint:forcetypeassert
		plan := uninstallMember{MemberID: member.ID, Host: member.Host, MarkerPath: uninstallMarkerPath}
		if member.NodeID != nil {
			plan.NodeID = *member.NodeID
		}
		plan.Packages = install.Packages
		plan.DataPath = install.DataPath
		plan.LogPath = install.LogPath
		plan.ControlUnit = install.ControlUnit
		accepted, ok := manifest[member.ID]
		if ok {
			plan.Packages = accepted.Install.Packages
			plan.DataPath = accepted.DataPath
			plan.ConfigFile = accepted.Config.File
			plan.KeyfilePath = accepted.KeyfilePath
			plan.LogPath = accepted.LogPath
			plan.ControlUnit = accepted.Control.Unit
		}
		members = append(members, plan)
	}

	params, err := json.Marshal(uninstallParams{
		ClusterID:    cluster.ID,
		ClusterName:  cluster.Name,
		InstallRunID: runID,
		OS:           install.OS,
		Members:      members,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to seal the uninstall plan: %s", err)
	}
	return params, nil
}

// clusterInstallParams reads the install operation's sealed params, so the teardown takes the OS, data and
// log paths, packages and control unit the install laid down even before an accepted state is recorded.
func clusterInstallParams(tx *reform.TX, cluster *models.OmCluster) (installParams, error) {
	if cluster.CreatedBy == "" {
		return installParams{}, nil
	}
	row, err := tx.FindByPrimaryKeyFrom(models.OmOperationTable, cluster.CreatedBy)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return installParams{}, nil
		}
		return installParams{}, status.Errorf(codes.Internal, "failed to read the install operation: %s", err)
	}
	op := row.(*models.OmOperation) //nolint:forcetypeassert
	return installParamsOf(op), nil
}

// installMarkerRunID reads the run id from the cluster's install marker owned item; a cluster with no
// present marker is refused, since uninstall removes only what PMM can prove it installed.
func installMarkerRunID(tx *reform.TX, clusterID string) (string, error) {
	var spec []byte
	err := tx.QueryRow(
		"SELECT spec FROM om_owned_items "+
			"WHERE cluster_id = $1 AND kind = 'marker' AND natural_key = 'install' AND state = 'present'",
		clusterID).Scan(&spec)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return "", status.Errorf(codes.FailedPrecondition, "cluster %s has no install marker, so it cannot be uninstalled", clusterID)
		}
		return "", status.Errorf(codes.Internal, "failed to read the install marker: %s", err)
	}
	var decoded struct {
		RunID string `json:"run_id"`
	}
	err = json.Unmarshal(spec, &decoded)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to decode the install marker: %s", err)
	}
	return decoded.RunID, nil
}

// acceptedMembers returns the cluster's current accepted-state members keyed by member id, or an empty
// map when no accepted state has been written yet.
func acceptedMembers(tx *reform.TX, cluster *models.OmCluster) (map[string]models.AcceptedMember, error) {
	if cluster.StateVersion == nil {
		return map[string]models.AcceptedMember{}, nil
	}
	var document []byte
	err := tx.QueryRow(
		"SELECT document FROM om_cluster_states WHERE cluster_id = $1 AND version = $2",
		cluster.ID, *cluster.StateVersion).Scan(&document)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return map[string]models.AcceptedMember{}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to read the accepted state: %s", err)
	}
	state, err := models.UnmarshalAcceptedState(document)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode the accepted state: %s", err)
	}
	return state.Members, nil
}
