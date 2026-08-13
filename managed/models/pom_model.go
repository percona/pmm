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

package models

import (
	"database/sql/driver"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// PomRunStatus is the terminal state of one POM discovery run.
type PomRunStatus string

// The states a POM run can end in. A run whose sources all answered is a success even if
// some services were never seen: a service inventory knows and metrics have not is a fact
// about the estate, not a failure of the run.
const (
	// PomRunSuccess means every source answered.
	PomRunSuccess = PomRunStatus("success")
	// PomRunPartial means at least one source answered incompletely or errored.
	PomRunPartial = PomRunStatus("partial")
	// PomRunFailed means no source answered.
	PomRunFailed = PomRunStatus("failed")
)

// PomSourceReport says how completely one discovery source answered, and what it saw.
type PomSourceReport struct {
	Source string            `json:"source"`
	Status string            `json:"status"`
	Facts  int32             `json:"facts"`
	Detail map[string]string `json:"detail,omitempty"`
}

// PomSourceReports is the per-source receipt for one run, stored as JSONB.
//
// This is what makes a thin snapshot legible rather than merely thin: a run recorded with
// metrics=ok and probe=failed is still correct about every version and honest about
// reachability.
type PomSourceReports []PomSourceReport

// Value implements database/sql/driver.Valuer.
func (r PomSourceReports) Value() (driver.Value, error) { return jsonValue(r) }

// Scan implements database/sql.Scanner.
func (r *PomSourceReports) Scan(src any) error { return jsonScan(r, src) }

// PomRunError describes one thing that went wrong during a run.
type PomRunError struct {
	Scope       string `json:"scope"`
	ServiceName string `json:"service_name,omitempty"`
	Code        string `json:"code"`
	Message     string `json:"message"`
}

// PomRunErrors is a run's failures, stored as JSONB.
type PomRunErrors []PomRunError

// Value implements database/sql/driver.Valuer.
func (e PomRunErrors) Value() (driver.Value, error) { return jsonValue(e) }

// Scan implements database/sql.Scanner.
func (e *PomRunErrors) Scan(src any) error { return jsonScan(e, src) }

// PomRun records one execution of POM discovery.
//
// The run ID is the snapshot key, so a document always names the pass that produced it.
//
//reform:pom_runs
type PomRun struct {
	RunID      string       `reform:"run_id,pk"`
	StartedAt  time.Time    `reform:"started_at"`
	FinishedAt *time.Time   `reform:"finished_at"`
	Status     PomRunStatus `reform:"status"`

	ServicesTotal    int32 `reform:"services_total"`
	ServicesResolved int32 `reform:"services_resolved"`
	ServicesOrphaned int32 `reform:"services_orphaned"`
	ProbesOK         int32 `reform:"probes_ok"`
	ServicesStale    int32 `reform:"services_stale"`

	OriginNode string           `reform:"origin_node"`
	Sources    PomSourceReports `reform:"sources"`
	Errors     PomRunErrors     `reform:"errors"`

	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (r *PomRun) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.StartedAt = r.StartedAt.UTC()
	if r.FinishedAt != nil {
		finished := r.FinishedAt.UTC()
		r.FinishedAt = &finished
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (r *PomRun) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.StartedAt = r.StartedAt.UTC()
	if r.FinishedAt != nil {
		finished := r.FinishedAt.UTC()
		r.FinishedAt = &finished
	}
	return nil
}

// PomSnapshot holds one run's topology document.
//
// Stored as one JSONB document rather than a relational tree on purpose: the topology
// model is still moving -- replica sets, sharded clusters, routers, more metrics -- and a
// schema that has to be migrated for every shape change is a schema that discourages
// changing the shape. A reader checks schema_version instead.
//
//reform:pom_snapshots
type PomSnapshot struct {
	RunID         string     `reform:"run_id,pk"`
	GeneratedAt   time.Time  `reform:"generated_at"`
	ObservedAt    *time.Time `reform:"observed_at"`
	Stale         bool       `reform:"stale"`
	SchemaVersion int32      `reform:"schema_version"`
	Document      []byte     `reform:"document"`
	CreatedAt     time.Time  `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (s *PomSnapshot) BeforeInsert() error {
	s.CreatedAt = Now()
	s.GeneratedAt = s.GeneratedAt.UTC()
	if s.ObservedAt != nil {
		observed := s.ObservedAt.UTC()
		s.ObservedAt = &observed
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (s *PomSnapshot) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.GeneratedAt = s.GeneratedAt.UTC()
	if s.ObservedAt != nil {
		observed := s.ObservedAt.UTC()
		s.ObservedAt = &observed
	}
	return nil
}

// Check that the models satisfy reform's hooks.
var (
	_ reform.BeforeInserter = (*PomRun)(nil)
	_ reform.AfterFinder    = (*PomRun)(nil)
	_ reform.BeforeInserter = (*PomSnapshot)(nil)
	_ reform.AfterFinder    = (*PomSnapshot)(nil)
)
