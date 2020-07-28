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

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

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

	AWSPartitions []string `json:"aws_partitions"`

	AWSInstanceChecked bool `json:"aws_instance_checked"`

	SSHKey string `json:"ssh_key"`

	// not url.URL to keep username and password
	AlertManagerURL string `json:"alert_manager_url"`

	// Saas config options
	SaaS struct {
		// Percona Platform user email
		Email string `json:"email"`
		// Percona Platform session Id
		SessionID string `json:"session_id"`
		// Security Threat Tool enabled
		STTEnabled bool `json:"stt_enabled"`
	} `json:"sass"`
}

// fillDefaults sets zero values to their default values.
// Used for migrating settings to the newer version.
func (s *Settings) fillDefaults() {
	// no default for Telemetry UUID - it set by telemetry service

	if s.MetricsResolutions.HR == 0 {
		s.MetricsResolutions.HR = 5 * time.Second
	}
	if s.MetricsResolutions.MR == 0 {
		s.MetricsResolutions.MR = 10 * time.Second
	}
	if s.MetricsResolutions.LR == 0 {
		s.MetricsResolutions.LR = 60 * time.Second
	}

	if s.DataRetention == 0 {
		s.DataRetention = 30 * 24 * time.Hour
	}

	if len(s.AWSPartitions) == 0 {
		s.AWSPartitions = []string{endpoints.AwsPartitionID}
	}

	// AWSInstanceChecked is false by default
	// SSHKey is empty by default
	// AlertManagerURL is empty by default
	// SaaS.STTEnabled is false by default
}
