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

	"github.com/asaskevich/govalidator"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/pkg/errors"
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

// SaaS contains settings related to the SaaS platform.
type SaaS struct {
	// Advisor checks disabled, false by default.
	STTDisabled bool `json:"stt_disabled"`
	// List of disabled STT checks
	DisabledSTTChecks []string `json:"disabled_stt_checks"`
	// STT check intervals
	STTCheckIntervals STTCheckIntervals `json:"stt_check_intervals"`
}

// Alerting contains settings related to Alerting.
type Alerting struct {
	Disabled              bool                   `json:"disabled"`
	EmailAlertingSettings *EmailAlertingSettings `json:"email_settings"`
	SlackAlertingSettings *SlackAlertingSettings `json:"slack_settings"`
}

// Settings contains PMM Server settings.
type Settings struct {
	PMMPublicAddress string `json:"pmm_public_address"`

	Updates struct {
		Disabled bool `json:"disabled"`
	} `json:"updates"`

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

	VictoriaMetrics struct {
		CacheEnabled bool `json:"cache_enabled"`
	} `json:"victoria_metrics"`

	SaaS SaaS `json:"sass"` // sic :(

	// DBaaS config options
	DBaaS struct {
		Enabled bool `json:"enabled"`
	} `json:"dbaas"`

	Alerting Alerting `json:"alerting"`

	Azurediscover struct {
		Enabled bool `json:"enabled"`
	} `json:"azure"`

	BackupManagement struct {
		Disabled bool `json:"disabled"`
	} `json:"backup_management"`

	// PMMServerID is generated on the first start of PMM server.
	PMMServerID string `json:"pmmServerID"`

	// DefaultRoleID defines a default role to be assigned to new users.
	DefaultRoleID int `json:"default_role_id"`

	// AccessControl holds information about access control.
	AccessControl struct {
		// Enabled is true if access control is enabled.
		Enabled bool `json:"enabled"`
	} `json:"access_control"`
}

// EmailAlertingSettings represents email settings for Integrated Alerting.
type EmailAlertingSettings struct {
	From       string `json:"from"`
	Smarthost  string `json:"smarthost"`
	Hello      string `json:"hello"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Identity   string `json:"identity"`
	Secret     string `json:"secret"`
	RequireTLS bool   `json:"require_tls"`
}

// Validate validates structure's fields.
func (e *EmailAlertingSettings) Validate() error {
	if !govalidator.IsEmail(e.From) {
		return errors.Errorf("invalid \"from\" email %q", e.From)
	}

	if !govalidator.IsDialString(e.Smarthost) {
		return errors.New("invalid server address, expected format host:port")
	}

	if e.Hello != "" {
		if !govalidator.IsHost(e.Hello) {
			return errors.New("invalid hello field, expected valid host")
		}
	}

	return nil
}

// SlackAlertingSettings represents Slack settings for Integrated Alerting.
type SlackAlertingSettings struct {
	URL string `json:"url"`
}

// STTCheckIntervals represents intervals between STT checks.
type STTCheckIntervals struct {
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

	if s.SaaS.STTCheckIntervals.RareInterval == 0 {
		s.SaaS.STTCheckIntervals.RareInterval = 78 * time.Hour
	}

	if s.SaaS.STTCheckIntervals.StandardInterval == 0 {
		s.SaaS.STTCheckIntervals.StandardInterval = 24 * time.Hour
	}

	if s.SaaS.STTCheckIntervals.FrequentInterval == 0 {
		s.SaaS.STTCheckIntervals.FrequentInterval = 4 * time.Hour
	}

	// AWSInstanceChecked is false by default
	// SSHKey is empty by default
	// AlertManagerURL is empty by default
	// SaaS.STTDisabled is false by default
	// DBaaS.Enabled is false by default
	// Alerting.Disabled is false by default
	// VictoriaMetrics CacheEnable is false by default
	// PMMPublicAddress is empty by default
	// Azurediscover.Enabled is false by default
}
