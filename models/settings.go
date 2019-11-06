// pmm-managed
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

package models

import (
	"time"
)

// MaxTableCount is the default maximum table count when mysqld_exporter should not make heavy impact on performance.
// FIXME Make it configurable. https://jira.percona.com/browse/PMM-4535
const MaxTableCount = 1000

// MetricsResolutions contains standard Prometheus metrics resolutions.
type MetricsResolutions struct {
	HR time.Duration `json:"hr"`
	MR time.Duration `json:"mr"`
	LR time.Duration `json:"lr"`
}

// Settings contains PMM Server settings.
type Settings struct {
	Telemetry struct {
		Disabled bool   `json:"disabled"`
		UUID     string `json:"uuid"`
	} `json:"telemetry"`

	MetricsResolutions MetricsResolutions `json:"metrics_resolutions"`

	DataRetention time.Duration `json:"data_retention"`
}

// fillDefaults sets zero values to their default values.
// Used for migrating settings to the newer version.
func (s *Settings) fillDefaults() {
	// no default for Telemetry UUID - it set by telemetry service

	if s.MetricsResolutions.HR == 0 {
		s.MetricsResolutions.HR = 5 * time.Second
	}
	if s.MetricsResolutions.MR == 0 {
		s.MetricsResolutions.MR = 5 * time.Second
	}
	if s.MetricsResolutions.LR == 0 {
		s.MetricsResolutions.LR = 60 * time.Second
	}

	if s.DataRetention == 0 {
		s.DataRetention = 30 * 24 * time.Hour
	}
}
