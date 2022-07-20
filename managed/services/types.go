// Copyright (C) 2017 Percona LLC
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

package services

import (
	"github.com/percona-platform/saas/pkg/check"

	"github.com/percona/pmm/managed/models"
)

// Target contains required info about STT check target.
type Target struct {
	AgentID       string
	ServiceID     string
	ServiceName   string
	NodeName      string
	Labels        map[string]string
	DSN           string
	Files         map[string]string
	TDP           *models.DelimiterPair
	TLSSkipVerify bool
}

// CheckResult contains the output from the check file and other information.
type CheckResult struct {
	CheckName string
	Silenced  bool
	AlertID   string
	Interval  check.Interval
	Target    Target
	Result    check.Result
}

// CheckResultSummary contains the summary of failed checks for a service.
type CheckResultSummary struct {
	ServiceName    string
	ServiceID      string
	EmergencyCount uint32
	AlertCount     uint32
	CriticalCount  uint32
	ErrorCount     uint32
	WarningCount   uint32
	NoticeCount    uint32
	InfoCount      uint32
	DebugCount     uint32
}

// FilterParams provides fields needed to filter alerts from AlertManager.
type FilterParams struct {
	// IsIA specifies if only Integrated Alerts should be matched.
	IsIA bool
	// IsCheck specifies if only Checks/Advisors alerts should be matched.
	IsCheck bool
	// AlertID is the ID of alert to be matched (if any).
	AlertID string
	// ServiceID is the ID of service to be matched (if any).
	ServiceID string
}
