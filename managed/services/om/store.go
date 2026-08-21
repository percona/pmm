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

// documentJSON is how a snapshot is stored and read back.
//
// Deliberately protojson rather than encoding/json: the document is a protobuf message,
// and only protojson agrees with the wire format the API serves -- wrapper types as bare
// values or null, timestamps as RFC 3339. Round-tripping through encoding/json would
// store a differently-shaped document than the one the UI receives.
var (
	documentJSON      = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
	documentJSONParse = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// persist writes one run and its document, then prunes the history.
//
// Both rows go in one transaction: a run without its document is not a state any reader
// should have to handle. Failure is reported but not fatal -- the document is already
// published in memory, and losing the record of a collection is worth less than refusing
// to serve the collection.
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

	// InTransactionContext rather than InTransaction so a shutdown or a cancelled
	// collection aborts the write instead of finishing it. managed/AGENTS.md prescribes
	// this form, and it is what most of managed/services uses.
	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		err := models.CreateOmTopologyRun(tx.Querier, row, snapshot)
		if err != nil {
			return err
		}
		return models.PruneOmTopologyRuns(tx.Querier, runHistory)
	})
}

// restore loads the newest stored document, so a restarted pmm-managed can answer before
// it has collected anything of its own.
//
// A document read back from the database is re-staled against the clock rather than
// trusted: stale was computed when it was written, and the whole point of restoring one
// is that time has passed since.
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

// listRuns reads the run history back out of the database.
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

// getRun reads one run back out of the database.
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

// snapshotStale reports whether an observation is older than the grace period.
func snapshotStale(observedAt *timestamppb.Timestamp) bool {
	if observedAt == nil {
		return false
	}
	return time.Since(observedAt.AsTime()) > staleAfter
}

// The enum boundary. Statuses are stored as the collector's own lowercase strings and
// translated here, in one place, rather than persisted as enum numbers: a run row stays
// readable in psql, and renumbering the enum cannot silently reinterpret history.
//
// An unrecognised stored value maps to UNSPECIFIED rather than being guessed at. That is
// the honest answer for a row written by a newer version, and it is visible on the wire
// instead of masquerading as a status the caller knows.

// runStatusToProto maps a stored run status onto the wire enum.
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

// runStatusFromProto maps the wire enum back onto the stored representation.
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

// sourceStatusToProto maps a stored source status onto the wire enum.
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

// sourceStatusFromProto maps the wire enum back onto the stored representation.
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
