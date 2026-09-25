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
	"time"

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// CheckResultStatus represents the outcome of an Advisor check run against a service.
type CheckResultStatus string

// Available Advisor check result statuses.
const (
	// CheckResultOK means the check ran and found no issue.
	CheckResultOK CheckResultStatus = "ok"
	// CheckResultFailed means the check ran and detected an issue.
	CheckResultFailed CheckResultStatus = "failed"
	// CheckResultError means the check could not be executed.
	CheckResultError CheckResultStatus = "error"
)

// CheckTriggeredBy represents the actor that initiated an Advisor check run.
type CheckTriggeredBy string

// Available Advisor check run initiators.
const (
	// CheckTriggeredByUser means the run was started by a user via the API or UI.
	CheckTriggeredByUser CheckTriggeredBy = "user"
	// CheckTriggeredByScheduler means the run was started by the built-in scheduler.
	CheckTriggeredByScheduler CheckTriggeredBy = "scheduler"
)

// Insight represents a single Advisor check run against a target persisted to history.
//
//reform:advisor_insights
type Insight struct {
	ID             string            `reform:"id,pk"`
	CheckName      string            `reform:"check_name"`
	Category       string            `reform:"category"`
	Interval       Interval          `reform:"interval"`
	ServiceID      string            `reform:"service_id"`
	ServiceName    string            `reform:"service_name"`
	ServiceType    ServiceType       `reform:"service_type"`
	NodeID         string            `reform:"node_id"`
	NodeName       string            `reform:"node_name"`
	Environment    string            `reform:"environment"`
	Cluster        string            `reform:"cluster"`
	ReplicationSet string            `reform:"replication_set"`
	Region         string            `reform:"region"`
	AZ             string            `reform:"az"`
	Status         CheckResultStatus `reform:"status"`
	Summary        string            `reform:"summary"`
	Description    string            `reform:"description"`
	Outcome        string            `reform:"outcome"`
	ReadMoreURL    string            `reform:"read_more_url"`
	Severity       Severity          `reform:"severity"`
	Labels         []byte            `reform:"labels"`
	CheckedAt      time.Time         `reform:"checked_at"`
	IsRead         bool              `reform:"is_read"`
	RunID          string            `reform:"run_id"`
	TriggeredBy    CheckTriggeredBy  `reform:"triggered_by"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *Insight) BeforeInsert() error {
	if r.CheckedAt.IsZero() {
		r.CheckedAt = Now()
	}
	if len(r.Labels) == 0 {
		r.Labels = nil
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *Insight) BeforeUpdate() error {
	if len(r.Labels) == 0 {
		r.Labels = nil
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *Insight) AfterFind() error {
	r.CheckedAt = r.CheckedAt.UTC()
	if len(r.Labels) == 0 {
		r.Labels = nil
	}
	return nil
}

// GetLabels decodes result labels.
func (r *Insight) GetLabels() (map[string]string, error) {
	return getLabels(r.Labels)
}

// SetLabels encodes result labels.
func (r *Insight) SetLabels(m map[string]string) error {
	return setLabels(m, &r.Labels)
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Insight)(nil)
	_ reform.BeforeUpdater  = (*Insight)(nil)
	_ reform.AfterFinder    = (*Insight)(nil)
)
