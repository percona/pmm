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
	"fmt"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// Values a fresh install's accepted state records that the scan does not observe, so they are the
// install's own facts and are not drift-compared until a probe reads them.
const (
	installControlMethod  = "systemd"
	installMethod         = "packages"
	installAuthMechanism  = "SCRAM-SHA-256"
	installAuthMembership = "keyfile"
)

// reconcileAcceptedStates writes the first accepted state for every managed cluster that has none yet,
// from its members' managed-scan facts; a cluster whose facts are not in yet is left for a later tick.
func (s *ScanScheduler) reconcileAcceptedStates(ctx context.Context) {
	if s.reader == nil || s.metrics == nil {
		return
	}
	clusters, err := clustersNeedingAcceptedState(s.db.Querier)
	if err != nil {
		s.l.WithError(err).Warn("failed to list clusters needing an accepted state")
		return
	}
	for clusterID, operationID := range clusters {
		if ctx.Err() != nil {
			return
		}
		s.writeAcceptedState(ctx, clusterID, operationID)
	}
}

// writeAcceptedState builds one cluster's accepted state from the scan and appends it as version one,
// re-checking under the row lock so a concurrent tick writes it once.
func (s *ScanScheduler) writeAcceptedState(ctx context.Context, clusterID, operationID string) {
	rows, err := s.db.SelectAllFrom(models.OmClusterMemberTable, "WHERE cluster_id = $1 AND removed_at IS NULL ORDER BY id", clusterID)
	if err != nil {
		s.l.WithField("cluster_id", clusterID).WithError(err).Warn("failed to read members for the accepted state")
		return
	}
	members := make([]*models.OmClusterMember, len(rows))
	for i, row := range rows {
		members[i] = row.(*models.OmClusterMember) //nolint:forcetypeassert
	}

	state, ready := buildInstallAcceptedState(ctx, s.reader, s.metrics, members)
	if !ready {
		return
	}
	document, err := models.MarshalAcceptedState(state)
	if err != nil {
		s.l.WithField("cluster_id", clusterID).WithError(err).Warn("failed to encode the accepted state")
		return
	}

	err = s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var current *int32
		scanErr := tx.QueryRow("SELECT state_version FROM om_clusters WHERE id = $1 FOR UPDATE", clusterID).Scan(&current)
		if scanErr != nil {
			return scanErr
		}
		if current != nil {
			return nil
		}
		_, appendErr := models.AppendClusterState(tx, clusterID, models.AcceptedStateSchemaVersion, document, operationID)
		return appendErr
	})
	if err != nil {
		s.l.WithField("cluster_id", clusterID).WithError(err).Warn("failed to append the accepted state")
	}
}

// buildInstallAcceptedState assembles a cluster's accepted state from each member's credential-free
// scan facts and its service's exporter-metric facts, reporting false until every member's are in.
func buildInstallAcceptedState(
	ctx context.Context,
	reader factObservationReader,
	metrics metricsFactReader,
	members []*models.OmClusterMember,
) (*models.AcceptedState, bool) {
	serviceIDs := make([]string, 0, len(members))
	for _, member := range members {
		if member.ServiceID == nil {
			return nil, false
		}
		serviceIDs = append(serviceIDs, *member.ServiceID)
	}
	metricFacts, err := metrics.ServiceMetricsFacts(ctx, serviceIDs)
	if err != nil {
		return nil, false
	}

	state := &models.AcceptedState{
		Auth:        models.AcceptedAuth{Mechanisms: []string{installAuthMechanism}, Membership: installAuthMembership},
		ReplicaSets: map[string]models.AcceptedReplicaSet{},
		Members:     map[string]models.AcceptedMember{},
	}
	for _, member := range members {
		facts, _, err := reader.Observation(ctx, subjectKindMember, member.ID, scanProfileManaged)
		if err != nil || len(facts) == 0 {
			return nil, false
		}
		memberMetrics := metricFacts[*member.ServiceID]
		binaryVersion, _ := facts["binary_version"].(string)
		configSHA, _ := facts["database.config_sha256"].(string)
		setName, _ := facts["database.set_name"].(string)
		fcv, _ := memberMetrics[fieldFCV].(string)
		rsVersion, versionOK := memberMetrics[fieldRSConfigVersion].(float64)
		rsTerm, termOK := memberMetrics[fieldRSConfigTerm].(float64)
		if binaryVersion == "" || fcv == "" || configSHA == "" || setName == "" || !versionOK || !termOK || member.RSMemberID == nil {
			return nil, false
		}
		state.FCV = fcv
		state.ReplicaSets[setName] = models.AcceptedReplicaSet{
			ObservedFrom: models.AcceptedObservedFrom{ConfigVersion: int32(rsVersion), Term: int64(rsTerm)},
		}
		rsMemberID := *member.RSMemberID
		state.Members[member.ID] = models.AcceptedMember{
			Process:       string(models.OmMemberMongod),
			Host:          member.Host,
			ReplicaSet:    setName,
			BinaryVersion: binaryVersion,
			Control:       models.AcceptedControl{Method: installControlMethod},
			Install:       models.AcceptedInstall{Method: installMethod},
			ConfigSHA256:  configSHA,
			RSMemberID:    &rsMemberID,
		}
	}
	if len(state.Members) == 0 {
		return nil, false
	}
	return state, true
}

// clustersNeedingAcceptedState returns the managed clusters with no accepted state yet, mapped to the
// operation that installed them, which the appended state is attributed to.
func clustersNeedingAcceptedState(q *reform.Querier) (map[string]string, error) {
	rows, err := q.SelectAllFrom(models.OmClusterTable,
		"WHERE status = $1 AND removed_at IS NULL AND state_version IS NULL", string(models.OmClusterManaged))
	if err != nil {
		return nil, fmt.Errorf("failed to select clusters needing an accepted state: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		cluster := row.(*models.OmCluster) //nolint:forcetypeassert
		out[cluster.ID] = cluster.CreatedBy
	}
	return out, nil
}
