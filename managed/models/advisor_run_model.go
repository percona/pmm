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
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// AdvisorRunStatus represents the state of an Advisor run.
type AdvisorRunStatus string

// Available Advisor run states.
const (
	// AdvisorRunStatusQueued means the run was requested and waits for the leader to start it.
	AdvisorRunStatusQueued AdvisorRunStatus = "queued"
	// AdvisorRunStatusRunning means the run is executing checks.
	AdvisorRunStatusRunning AdvisorRunStatus = "running"
	// AdvisorRunStatusCompleted means the run finished normally.
	AdvisorRunStatusCompleted AdvisorRunStatus = "completed"
	// AdvisorRunStatusInterrupted means PMM restarted mid-run; the insights it saved are kept.
	AdvisorRunStatusInterrupted AdvisorRunStatus = "interrupted"
	// AdvisorRunStatusAborted means the run stopped before running any check.
	AdvisorRunStatusAborted AdvisorRunStatus = "aborted"
)

// AdvisorRun represents a single execution of Advisor checks. Counts are filled
// in when the run finishes; while it runs, FindAdvisorRuns derives them from its insights.
//
//reform:advisor_runs
type AdvisorRun struct {
	ID          string           `reform:"id,pk"`
	TriggeredBy CheckTriggeredBy `reform:"triggered_by"`
	Status      AdvisorRunStatus `reform:"status"`
	// CheckNames, ServiceIDs and Intervals narrow the run; empty means all.
	CheckNames pq.StringArray `reform:"check_names"`
	ServiceIDs pq.StringArray `reform:"service_ids"`
	Intervals  pq.StringArray `reform:"intervals"`
	StartedAt  time.Time      `reform:"started_at"`
	FinishedAt *time.Time     `reform:"finished_at"`
	// PlannedChecksCount and PlannedServicesCount count the checks and services the run planned to cover.
	PlannedChecksCount   int `reform:"planned_checks_count"`
	PlannedServicesCount int `reform:"planned_services_count"`
	// ChecksCount and ServicesCount count the checks and services with an ok or failed outcome.
	ChecksCount   int `reform:"checks_count"`
	ServicesCount int `reform:"services_count"`
	// FindingsCount counts insights with a failed status, i.e. actual findings.
	FindingsCount int `reform:"findings_count"`
	// ErrorsCount counts checks that could not be executed at all.
	ErrorsCount    int    `reform:"errors_count"`
	SeverityCounts []byte `reform:"severity_counts"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *AdvisorRun) BeforeInsert() error {
	if r.StartedAt.IsZero() {
		r.StartedAt = Now()
	}
	if len(r.SeverityCounts) == 0 {
		r.SeverityCounts = nil
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *AdvisorRun) BeforeUpdate() error {
	if len(r.SeverityCounts) == 0 {
		r.SeverityCounts = nil
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *AdvisorRun) AfterFind() error {
	r.StartedAt = r.StartedAt.UTC()
	if r.FinishedAt != nil {
		finished := r.FinishedAt.UTC()
		r.FinishedAt = &finished
	}
	if len(r.SeverityCounts) == 0 {
		r.SeverityCounts = nil
	}
	return nil
}

// IsRunning reports whether the run has not recorded a completion yet.
func (r *AdvisorRun) IsRunning() bool {
	return r.FinishedAt == nil
}

// GetSeverityCounts decodes the per-severity finding counts.
func (r *AdvisorRun) GetSeverityCounts() (map[Severity]int, error) {
	if len(r.SeverityCounts) == 0 {
		return nil, nil //nolint:nilnil
	}
	m := make(map[Severity]int)
	err := json.Unmarshal(r.SeverityCounts, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to decode severity counts: %w", err)
	}
	return m, nil
}

// SetSeverityCounts encodes the per-severity finding counts.
func (r *AdvisorRun) SetSeverityCounts(m map[Severity]int) error {
	if len(m) == 0 {
		r.SeverityCounts = nil
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to encode severity counts: %w", err)
	}
	r.SeverityCounts = b
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*AdvisorRun)(nil)
	_ reform.BeforeUpdater  = (*AdvisorRun)(nil)
	_ reform.AfterFinder    = (*AdvisorRun)(nil)
)
