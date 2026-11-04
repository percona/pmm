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
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// documentJSON uses protojson so the stored document matches the wire format the API serves.
var (
	documentJSON      = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
	documentJSONParse = protojson.UnmarshalOptions{DiscardUnknown: true}
)

func (s *Service) persist(ctx context.Context, response *omv1.GetTopologyResponse, run *omv1.TopologyRun, originNode string) error {
	document, err := documentJSON.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal the topology document: %w", err)
	}

	row := &models.OmTopologyRun{
		RunID:            run.RunId,
		StartedAt:        run.StartTime.AsTime(),
		Status:           runStatusFromProto(run.Status),
		ServicesTotal:    run.Counts.TotalServices,
		ServicesResolved: run.Counts.ResolvedServices,
		ServicesOrphaned: run.Counts.OrphanedServices,
		ProbesOK:         run.Counts.SuccessfulProbes,
		ServicesStale:    run.Counts.StaleServices,
		OriginNode:       originNode,
		Sources:          make(models.OmTopologySourceReports, 0, len(run.Sources)),
		Errors:           make(models.OmTopologyRunErrors, 0, len(run.Errors)),
	}
	if run.EndTime != nil {
		finished := run.EndTime.AsTime()
		row.FinishedAt = &finished
	}
	for _, source := range run.Sources {
		row.Sources = append(row.Sources, models.OmTopologySourceReport{
			Source: source.Source, Status: string(sourceStatusFromProto(source.Status)),
			Facts: source.Facts, Detail: source.Detail,
		})
	}
	for _, e := range run.Errors {
		row.Errors = append(row.Errors, models.OmTopologyRunError{
			Scope: e.Scope, ServiceName: e.GetServiceName(),
			Code: e.Code, Message: e.Message,
		})
	}

	snapshot := &models.OmTopologySnapshot{
		GeneratedAt:   response.Snapshot.GeneratedAt.AsTime(),
		Stale:         response.Snapshot.Stale,
		SchemaVersion: response.Snapshot.SchemaVersion,
		Document:      document,
	}
	if response.Snapshot.ObservedAt != nil {
		observed := response.Snapshot.ObservedAt.AsTime()
		snapshot.ObservedAt = &observed
	}

	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		err := models.CreateOmTopologyRun(tx.Querier, row, snapshot)
		if err != nil {
			return err
		}
		return models.PruneOmTopologyRuns(tx.Querier, runHistory)
	})
}

// restore lets a restarted pmm-managed answer before its first collection. Stale is recomputed
// because it was computed when the document was written.
func (s *Service) restore(ctx context.Context) (*omv1.GetTopologyResponse, time.Time, error) {
	var snapshot *models.OmTopologySnapshot
	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		snapshot, err = models.FindLatestOmTopologySnapshot(tx.Querier)
		return err
	})
	if errTX != nil {
		if errors.Is(errTX, models.ErrNotFound) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, errTX
	}

	response := &omv1.GetTopologyResponse{}
	err := documentJSONParse.Unmarshal(snapshot.Document, response)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse the stored topology document: %w", err)
	}
	if response.Snapshot != nil {
		response.Snapshot.Stale = snapshotStale(response.Snapshot.ObservedAt)
	}
	return response, snapshot.GeneratedAt, nil
}

func (s *Service) listRuns(ctx context.Context, limit int) ([]*omv1.TopologyRun, error) {
	var rows []*models.OmTopologyRun
	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		rows, err = models.FindOmTopologyRuns(tx.Querier, limit)
		return err
	})
	if errTX != nil {
		return nil, errTX
	}
	runs := make([]*omv1.TopologyRun, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runFromModel(row))
	}
	return runs, nil
}

func (s *Service) getRun(ctx context.Context, runID string) (*omv1.TopologyRun, error) {
	var row *models.OmTopologyRun
	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		row, err = models.FindOmTopologyRunByID(tx.Querier, runID)
		return err
	})
	if errTX != nil {
		return nil, errTX
	}
	return runFromModel(row), nil
}

func runFromModel(row *models.OmTopologyRun) *omv1.TopologyRun {
	run := &omv1.TopologyRun{
		RunId:     row.RunID,
		Status:    runStatusToProto(row.Status),
		StartTime: timestamppb.New(row.StartedAt),
		Counts: &omv1.TopologyRunCounts{
			TotalServices:    row.ServicesTotal,
			ResolvedServices: row.ServicesResolved,
			OrphanedServices: row.ServicesOrphaned,
			SuccessfulProbes: row.ProbesOK,
			StaleServices:    row.ServicesStale,
		},
		Sources: make([]*omv1.SourceReport, 0, len(row.Sources)),
		Errors:  make([]*omv1.TopologyRunError, 0, len(row.Errors)),
	}
	if row.FinishedAt != nil {
		run.EndTime = timestamppb.New(*row.FinishedAt)
	}
	for _, source := range row.Sources {
		run.Sources = append(run.Sources, &omv1.SourceReport{
			Source: source.Source, Status: sourceStatusToProto(SourceStatus(source.Status)),
			Facts: source.Facts, Detail: source.Detail,
		})
	}
	for _, e := range row.Errors {
		run.Errors = append(run.Errors, &omv1.TopologyRunError{
			Scope: e.Scope, ServiceName: optional(e.ServiceName),
			Code: e.Code, Message: e.Message,
		})
	}
	return run
}

func snapshotStale(observedAt *timestamppb.Timestamp) bool {
	if observedAt == nil {
		return false
	}
	return time.Since(observedAt.AsTime()) > staleAfter
}

func runStatusToProto(status models.OmTopologyRunStatus) omv1.RunStatus {
	switch string(status) {
	case runStatusSuccess:
		return omv1.RunStatus_RUN_STATUS_SUCCESS
	case runStatusPartial:
		return omv1.RunStatus_RUN_STATUS_PARTIAL
	case runStatusFailed:
		return omv1.RunStatus_RUN_STATUS_FAILED
	default:
		return omv1.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

func runStatusFromProto(status omv1.RunStatus) models.OmTopologyRunStatus {
	switch status {
	case omv1.RunStatus_RUN_STATUS_SUCCESS:
		return models.OmTopologyRunStatus(runStatusSuccess)
	case omv1.RunStatus_RUN_STATUS_PARTIAL:
		return models.OmTopologyRunStatus(runStatusPartial)
	case omv1.RunStatus_RUN_STATUS_FAILED:
		return models.OmTopologyRunStatus(runStatusFailed)
	default:
		return ""
	}
}

func sourceStatusToProto(status SourceStatus) omv1.SourceStatus {
	switch status {
	case SourceOK:
		return omv1.SourceStatus_SOURCE_STATUS_OK
	case SourcePartial:
		return omv1.SourceStatus_SOURCE_STATUS_PARTIAL
	case SourceFailed:
		return omv1.SourceStatus_SOURCE_STATUS_FAILED
	case SourceDisabled:
		return omv1.SourceStatus_SOURCE_STATUS_DISABLED
	default:
		return omv1.SourceStatus_SOURCE_STATUS_UNSPECIFIED
	}
}

func sourceStatusFromProto(status omv1.SourceStatus) SourceStatus {
	switch status {
	case omv1.SourceStatus_SOURCE_STATUS_OK:
		return SourceOK
	case omv1.SourceStatus_SOURCE_STATUS_PARTIAL:
		return SourcePartial
	case omv1.SourceStatus_SOURCE_STATUS_FAILED:
		return SourceFailed
	case omv1.SourceStatus_SOURCE_STATUS_DISABLED:
		return SourceDisabled
	default:
		return ""
	}
}
