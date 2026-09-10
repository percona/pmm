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

// OmTopologyRunStatus is the terminal state of one OM collection run.
type OmTopologyRunStatus string

// The states an OM run can end in. A run whose sources all answered is a success even if
// some services were never seen: a service inventory knows and metrics have not is a fact
// about the estate, not a failure of the run.
const (
	// OmTopologyRunSuccess means every source answered.
	OmTopologyRunSuccess = OmTopologyRunStatus("success")
	// OmTopologyRunPartial means at least one source answered incompletely or errored.
	OmTopologyRunPartial = OmTopologyRunStatus("partial")
	// OmTopologyRunFailed means no source answered.
	OmTopologyRunFailed = OmTopologyRunStatus("failed")
)

// OmTopologySourceReport says how completely one collection source answered, and what it saw.
type OmTopologySourceReport struct {
	Source string            `json:"source"`
	Status string            `json:"status"`
	Facts  int32             `json:"facts"`
	Detail map[string]string `json:"detail,omitempty"`
}

// OmTopologySourceReports is the per-source receipt for one run, stored as JSONB.
//
// This is what makes a thin snapshot legible rather than merely thin: a run recorded with
// metrics=ok and probe=failed is still correct about every version and honest about
// reachability.
type OmTopologySourceReports []OmTopologySourceReport

// Value implements database/sql/driver.Valuer.
func (r OmTopologySourceReports) Value() (driver.Value, error) { return jsonValue(r) }

// Scan implements database/sql.Scanner.
func (r *OmTopologySourceReports) Scan(src any) error { return jsonScan(r, src) }

// OmTopologyRunError describes one thing that went wrong during a run.
type OmTopologyRunError struct {
	Scope       string `json:"scope"`
	ServiceName string `json:"service_name,omitempty"`
	Code        string `json:"code"`
	Message     string `json:"message"`
}

// OmTopologyRunErrors is a run's failures, stored as JSONB.
type OmTopologyRunErrors []OmTopologyRunError

// Value implements database/sql/driver.Valuer.
func (e OmTopologyRunErrors) Value() (driver.Value, error) { return jsonValue(e) }

// Scan implements database/sql.Scanner.
func (e *OmTopologyRunErrors) Scan(src any) error { return jsonScan(e, src) }

// OmTopologyRun records one execution of OM's topology collection.
//
// The run ID is the snapshot key, so a document always names the pass that produced it.
//
//reform:om_topology_runs
type OmTopologyRun struct {
	RunID      string              `reform:"run_id,pk"`
	StartedAt  time.Time           `reform:"started_at"`
	FinishedAt *time.Time          `reform:"finished_at"`
	Status     OmTopologyRunStatus `reform:"status"`

	ServicesTotal    int32 `reform:"services_total"`
	ServicesResolved int32 `reform:"services_resolved"`
	ServicesOrphaned int32 `reform:"services_orphaned"`
	ProbesOK         int32 `reform:"probes_ok"`
	ServicesStale    int32 `reform:"services_stale"`

	OriginNode string                  `reform:"origin_node"`
	Sources    OmTopologySourceReports `reform:"sources"`
	Errors     OmTopologyRunErrors     `reform:"errors"`

	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (r *OmTopologyRun) BeforeInsert() error {
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
func (r *OmTopologyRun) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.StartedAt = r.StartedAt.UTC()
	if r.FinishedAt != nil {
		finished := r.FinishedAt.UTC()
		r.FinishedAt = &finished
	}
	return nil
}

// OmTopologySnapshot holds one run's topology document.
//
// Stored as one JSONB document rather than a relational tree on purpose: the topology
// model is still moving -- replica sets, sharded clusters, routers, more metrics -- and a
// schema that has to be migrated for every shape change is a schema that discourages
// changing the shape. A reader checks schema_version instead.
//
//reform:om_topology_snapshots
type OmTopologySnapshot struct {
	RunID         string     `reform:"run_id,pk"`
	GeneratedAt   time.Time  `reform:"generated_at"`
	ObservedAt    *time.Time `reform:"observed_at"`
	Stale         bool       `reform:"stale"`
	SchemaVersion int32      `reform:"schema_version"`
	Document      []byte     `reform:"document"`
	CreatedAt     time.Time  `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (s *OmTopologySnapshot) BeforeInsert() error {
	s.CreatedAt = Now()
	s.GeneratedAt = s.GeneratedAt.UTC()
	if s.ObservedAt != nil {
		observed := s.ObservedAt.UTC()
		s.ObservedAt = &observed
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (s *OmTopologySnapshot) AfterFind() error {
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
	_ reform.BeforeInserter = (*OmTopologyRun)(nil)
	_ reform.AfterFinder    = (*OmTopologyRun)(nil)
	_ reform.BeforeInserter = (*OmTopologySnapshot)(nil)
	_ reform.AfterFinder    = (*OmTopologySnapshot)(nil)
)
