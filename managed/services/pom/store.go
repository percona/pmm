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

package pom

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	pomv1 "github.com/percona/pmm/api/pom/v1"
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
func (s *Service) persist(response *pomv1.GetTopologyResponse, run *pomv1.Run, originNode string) error {
	document, err := documentJSON.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal the topology document: %w", err)
	}

	row := &models.PomRun{
		RunID:            run.RunId,
		StartedAt:        run.StartedAt.AsTime(),
		Status:           models.PomRunStatus(run.Status),
		ServicesTotal:    run.Counts.ServicesTotal,
		ServicesResolved: run.Counts.ServicesResolved,
		ServicesOrphaned: run.Counts.ServicesOrphaned,
		ProbesOK:         run.Counts.ProbesOk,
		ServicesStale:    run.Counts.ServicesStale,
		OriginNode:       originNode,
		Sources:          make(models.PomSourceReports, 0, len(run.Sources)),
		Errors:           make(models.PomRunErrors, 0, len(run.Errors)),
	}
	if run.FinishedAt != nil {
		finished := run.FinishedAt.AsTime()
		row.FinishedAt = &finished
	}
	for _, source := range run.Sources {
		row.Sources = append(row.Sources, models.PomSourceReport{
			Source: source.Source, Status: source.Status,
			Facts: source.Facts, Detail: source.Detail,
		})
	}
	for _, e := range run.Errors {
		row.Errors = append(row.Errors, models.PomRunError{
			Scope: e.Scope, ServiceName: e.ServiceName.GetValue(),
			Code: e.Code, Message: e.Message,
		})
	}

	snapshot := &models.PomSnapshot{
		GeneratedAt:   response.Snapshot.GeneratedAt.AsTime(),
		Stale:         response.Snapshot.Stale,
		SchemaVersion: response.Snapshot.SchemaVersion,
		Document:      document,
	}
	if response.Snapshot.ObservedAt != nil {
		observed := response.Snapshot.ObservedAt.AsTime()
		snapshot.ObservedAt = &observed
	}

	return s.db.InTransaction(func(tx *reform.TX) error {
		err := models.CreatePomRun(tx.Querier, row, snapshot)
		if err != nil {
			return err
		}
		return models.PrunePomRuns(tx.Querier, runHistory)
	})
}

// restore loads the newest stored document, so a restarted pmm-managed can answer before
// it has collected anything of its own.
//
// A document read back from the database is re-staled against the clock rather than
// trusted: stale was computed when it was written, and the whole point of restoring one
// is that time has passed since.
func (s *Service) restore() (*pomv1.GetTopologyResponse, time.Time, error) {
	var snapshot *models.PomSnapshot
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		snapshot, err = models.FindLatestPomSnapshot(tx.Querier)
		return err
	})
	if errTX != nil {
		if errors.Is(errTX, models.ErrNotFound) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, errTX
	}

	response := &pomv1.GetTopologyResponse{}
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
func (s *Service) listRuns(limit int) ([]*pomv1.Run, error) {
	var rows []*models.PomRun
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		rows, err = models.FindPomRuns(tx.Querier, limit)
		return err
	})
	if errTX != nil {
		return nil, errTX
	}
	runs := make([]*pomv1.Run, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runFromModel(row))
	}
	return runs, nil
}

// getRun reads one run back out of the database.
func (s *Service) getRun(runID string) (*pomv1.Run, error) {
	var row *models.PomRun
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		row, err = models.FindPomRunByID(tx.Querier, runID)
		return err
	})
	if errTX != nil {
		return nil, errTX
	}
	return runFromModel(row), nil
}

func runFromModel(row *models.PomRun) *pomv1.Run {
	run := &pomv1.Run{
		RunId:     row.RunID,
		Status:    string(row.Status),
		StartedAt: timestamppb.New(row.StartedAt),
		Counts: &pomv1.RunCounts{
			ServicesTotal:    row.ServicesTotal,
			ServicesResolved: row.ServicesResolved,
			ServicesOrphaned: row.ServicesOrphaned,
			ProbesOk:         row.ProbesOK,
			ServicesStale:    row.ServicesStale,
		},
		Sources: make([]*pomv1.SourceReport, 0, len(row.Sources)),
		Errors:  make([]*pomv1.RunError, 0, len(row.Errors)),
	}
	if row.FinishedAt != nil {
		run.FinishedAt = timestamppb.New(*row.FinishedAt)
	}
	for _, source := range row.Sources {
		run.Sources = append(run.Sources, &pomv1.SourceReport{
			Source: source.Source, Status: source.Status,
			Facts: source.Facts, Detail: source.Detail,
		})
	}
	for _, e := range row.Errors {
		run.Errors = append(run.Errors, &pomv1.RunError{
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
