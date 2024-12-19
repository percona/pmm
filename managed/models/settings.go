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

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

// Default values for settings. These values are used when settings are not set.
const (
	AdvisorsEnabledDefault             = true
	AlertingEnabledDefault             = true
	TelemetryEnabledDefault            = true
	UpdatesEnabledDefault              = true
	BackupManagementEnabledDefault     = true
	VictoriaMetricsCacheEnabledDefault = false
	AzureDiscoverEnabledDefault        = false
	AccessControlEnabledDefault        = false
)

// MetricsResolutions contains standard VictoriaMetrics metrics resolutions.
type MetricsResolutions struct {
	HR time.Duration `json:"hr"`
	MR time.Duration `json:"mr"`
	LR time.Duration `json:"lr"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (r MetricsResolutions) Value() (driver.Value, error) { return jsonValue(r) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (r *MetricsResolutions) Scan(src interface{}) error { return jsonScan(r, src) }

// Advisors contains settings related to the Portal Advisors.
type Advisors struct {
	// Advisor checks disabled, false by default.
	Enabled *bool `json:"enabled"`
	// List of disabled advisors
	DisabledAdvisors []string `json:"disabled_advisors"`
	// Advisor run intervals
	AdvisorRunIntervals AdvisorsRunIntervals `json:"advisor_run_intervals"`
}

// Settings contains PMM Server settings.
type Settings struct {
	PMMPublicAddress string `json:"pmm_public_address"`

	Updates struct {
		Enabled *bool `json:"enabled"`
	} `json:"updates"`

	Telemetry struct {
		Enabled *bool  `json:"enabled"`
		UUID    string `json:"uuid"`
	} `json:"telemetry"`

	MetricsResolutions MetricsResolutions `json:"metrics_resolutions"`

	DataRetention time.Duration `json:"data_retention"`

	AWSPartitions []string `json:"aws_partitions"`

	AWSInstanceChecked bool `json:"aws_instance_checked"`

	SSHKey string `json:"ssh_key"`

	VictoriaMetrics struct {
		CacheEnabled *bool `json:"cache_enabled"`
	} `json:"victoria_metrics"`

	SaaS Advisors `json:"sass"` // sic :(

	Nomad struct {
		Enabled *bool `json:"enabled"`
	}

	Alerting struct {
		Enabled *bool `json:"enabled"`
	} `json:"alerting"`

	Azurediscover struct {
		Enabled *bool `json:"enabled"`
	} `json:"azure"`

	BackupManagement struct {
		Enabled *bool `json:"enabled"`
	} `json:"backup_management"`

	// PMMServerID is generated on the first start of PMM server.
	PMMServerID string `json:"pmmServerID"`

	// DefaultRoleID defines a default role to be assigned to new users.
	DefaultRoleID int `json:"default_role_id"`

	// AccessControl holds information about access control.
	AccessControl struct {
		// Enabled is true if access control is enabled.
		Enabled *bool `json:"enabled"`
	} `json:"access_control"`

	// Contains all encrypted tables in format 'db.table.column'.
	EncryptedItems []string `json:"encrypted_items"`
}

// IsAlertingEnabled returns true if alerting is enabled.
func (s *Settings) IsAlertingEnabled() bool {
	if s.Alerting.Enabled != nil {
		return *s.Alerting.Enabled
	}
	return AlertingEnabledDefault
}

// IsTelemetryEnabled returns true if telemetry is enabled.
func (s *Settings) IsTelemetryEnabled() bool {
	if s.Telemetry.Enabled != nil {
		return *s.Telemetry.Enabled
	}
	return TelemetryEnabledDefault
}

// IsUpdatesEnabled returns true if updates are enabled.
func (s *Settings) IsUpdatesEnabled() bool {
	if s.Updates.Enabled != nil {
		return *s.Updates.Enabled
	}
	return UpdatesEnabledDefault
}

// IsBackupManagementEnabled returns true if backup management is enabled.
func (s *Settings) IsBackupManagementEnabled() bool {
	if s.BackupManagement.Enabled != nil {
		return *s.BackupManagement.Enabled
	}
	return BackupManagementEnabledDefault
}

// IsAdvisorsEnabled returns true if advisors are enabled.
func (s *Settings) IsAdvisorsEnabled() bool {
	if s.SaaS.Enabled != nil {
		return *s.SaaS.Enabled
	}

	return AdvisorsEnabledDefault
}

// IsNomadEnabled returns true if Nomad is enabled.
func (s *Settings) IsNomadEnabled() bool {
	if s.Nomad.Enabled != nil {
		return *s.Nomad.Enabled && s.PMMPublicAddress != ""
	}
	return false
}

// IsAzureDiscoverEnabled returns true if Azure discovery is enabled.
func (s *Settings) IsAzureDiscoverEnabled() bool {
	if s.Azurediscover.Enabled != nil {
		return *s.Azurediscover.Enabled
	}
	return AzureDiscoverEnabledDefault
}

// IsAccessControlEnabled returns true if access control is enabled.
func (s *Settings) IsAccessControlEnabled() bool {
	if s.AccessControl.Enabled != nil {
		return *s.AccessControl.Enabled
	}
	return AccessControlEnabledDefault
}

// IsVictoriaMetricsCacheEnabled returns true if VictoriaMetrics cache is enabled.
func (s *Settings) IsVictoriaMetricsCacheEnabled() bool {
	if s.VictoriaMetrics.CacheEnabled != nil {
		return *s.VictoriaMetrics.CacheEnabled
	}
	return VictoriaMetricsCacheEnabledDefault
}

// AdvisorsRunIntervals represents intervals between Advisors checks.
type AdvisorsRunIntervals struct {
	StandardInterval time.Duration `json:"standard_interval"`
	RareInterval     time.Duration `json:"rare_interval"`
	FrequentInterval time.Duration `json:"frequent_interval"`
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

	if s.SaaS.AdvisorRunIntervals.RareInterval == 0 {
		s.SaaS.AdvisorRunIntervals.RareInterval = 78 * time.Hour
	}

	if s.SaaS.AdvisorRunIntervals.StandardInterval == 0 {
		s.SaaS.AdvisorRunIntervals.StandardInterval = 24 * time.Hour
	}

	if s.SaaS.AdvisorRunIntervals.FrequentInterval == 0 {
		s.SaaS.AdvisorRunIntervals.FrequentInterval = 4 * time.Hour
	}
}
