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
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/utils/validators"
)

// GetSettings returns current PMM Server settings.
func GetSettings(q reform.DBTX) (*Settings, error) {
	var b []byte
	if err := q.QueryRow("SELECT settings FROM settings").Scan(&b); err != nil {
		return nil, errors.Wrap(err, "failed to select settings")
	}

	var s Settings

	if err := json.Unmarshal(b, &s); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal settings")
	}

	s.fillDefaults()
	return &s, nil
}

// ChangeSettingsParams contains values to change data in settings table.
type ChangeSettingsParams struct {
	// We don't save it to db
	DisableUpdates bool

	DisableTelemetry bool
	EnableTelemetry  bool

	MetricsResolutions MetricsResolutions

	DataRetention time.Duration

	AWSPartitions []string

	SSHKey string

	// not url.URL to keep username and password
	AlertManagerURL       string
	RemoveAlertManagerURL bool

	// Enable Security Threat Tool
	EnableSTT bool
	// Disable Security Threat Tool
	DisableSTT bool

	// Percona Platform user email
	Email string
	// Percona Platform session Id
	SessionID string
	// LogOut user from Percona Platform, i.e. remove user email and session id
	LogOut bool
}

// UpdateSettings updates only non-zero, non-empty values.
func UpdateSettings(q reform.DBTX, params *ChangeSettingsParams) (*Settings, error) {
	err := ValidateSettings(params)
	if err != nil {
		return nil, err
	}

	settings, err := GetSettings(q)
	if err != nil {
		return nil, err
	}

	if err := validateSettingsConflicts(params, settings); err != nil {
		return nil, err
	}

	if params.DisableTelemetry {
		settings.Telemetry.Disabled = true
		settings.Telemetry.UUID = ""
	}
	if params.EnableTelemetry {
		settings.Telemetry.Disabled = false
	}
	if params.MetricsResolutions.LR != 0 {
		settings.MetricsResolutions.LR = params.MetricsResolutions.LR
	}
	if params.MetricsResolutions.MR != 0 {
		settings.MetricsResolutions.MR = params.MetricsResolutions.MR
	}
	if params.MetricsResolutions.HR != 0 {
		settings.MetricsResolutions.HR = params.MetricsResolutions.HR
	}
	if params.DataRetention != 0 {
		settings.DataRetention = params.DataRetention
	}

	if len(params.AWSPartitions) != 0 {
		settings.AWSPartitions = deduplicateAWSPartitions(params.AWSPartitions)
	}
	if params.SSHKey != "" {
		settings.SSHKey = params.SSHKey
	}
	if params.AlertManagerURL != "" {
		settings.AlertManagerURL = params.AlertManagerURL
	}
	if params.RemoveAlertManagerURL {
		settings.AlertManagerURL = ""
	}

	if params.DisableSTT {
		settings.SaaS.STTEnabled = false
	}
	if params.EnableSTT {
		settings.SaaS.STTEnabled = true
	}
	if params.LogOut {
		settings.SaaS.SessionID = ""
		settings.SaaS.Email = ""
	}

	if params.SessionID != "" {
		settings.SaaS.SessionID = params.SessionID
	}

	if params.Email != "" {
		settings.SaaS.Email = params.Email
	}

	err = SaveSettings(q, settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// ValidateSettings validates settings changes.
func ValidateSettings(params *ChangeSettingsParams) error {
	if params.EnableTelemetry && params.DisableTelemetry {
		return fmt.Errorf("Both enable_telemetry and disable_telemetry are present.") //nolint:golint,stylecheck
	}
	if params.EnableSTT && params.DisableSTT {
		return fmt.Errorf("Both enable_stt and disable_stt are present.") //nolint:golint,stylecheck
	}

	checkCases := []struct {
		dur       time.Duration
		fieldName string
	}{
		{params.MetricsResolutions.HR, "hr"},
		{params.MetricsResolutions.MR, "mr"},
		{params.MetricsResolutions.LR, "lr"},
	}
	for _, v := range checkCases {
		if v.dur == 0 {
			continue
		}

		if _, err := validators.ValidateMetricResolution(v.dur); err != nil {
			switch err.(type) {
			case validators.DurationNotAllowedError:
				return fmt.Errorf("%s: should be a natural number of seconds", v.fieldName)
			case validators.MinDurationError:
				return fmt.Errorf("%s: minimal resolution is 1s", v.fieldName)
			default:
				return fmt.Errorf("%s: unknown error for", v.fieldName)
			}
		}
	}

	if params.DataRetention != 0 {
		if _, err := validators.ValidateDataRetention(params.DataRetention); err != nil {
			switch err.(type) {
			case validators.DurationNotAllowedError:
				return fmt.Errorf("data_retention: should be a natural number of days")
			case validators.MinDurationError:
				return fmt.Errorf("data_retention: minimal resolution is 24h")
			default:
				return fmt.Errorf("data_retention: unknown error")
			}
		}
	}

	var err error
	if err = validators.ValidateAWSPartitions(params.AWSPartitions); err != nil {
		return err
	}

	if params.AlertManagerURL != "" {
		if params.RemoveAlertManagerURL {
			return fmt.Errorf("Both alert_manager_url and remove_alert_manager_url are present.") //nolint:golint,stylecheck
		}

		// custom validation for typical error that is not handled well by url.Parse
		if !strings.Contains(params.AlertManagerURL, "//") {
			return fmt.Errorf("Invalid alert_manager_url: %s - missing protocol scheme.", params.AlertManagerURL) //nolint:golint,stylecheck
		}
		u, err := url.Parse(params.AlertManagerURL)
		if err != nil {
			return fmt.Errorf("Invalid alert_manager_url: %s.", err) //nolint:golint,stylecheck
		}
		if u.Scheme == "" {
			return fmt.Errorf("Invalid alert_manager_url: %s - missing protocol scheme.", params.AlertManagerURL) //nolint:golint,stylecheck
		}
		if u.Host == "" {
			return fmt.Errorf("Invalid alert_manager_url: %s - missing host.", params.AlertManagerURL) //nolint:golint,stylecheck
		}
	}

	return nil
}

func validateSettingsConflicts(params *ChangeSettingsParams, settings *Settings) error {
	if params.EnableSTT && !params.EnableTelemetry && settings.Telemetry.Disabled {
		return fmt.Errorf("Cannot enable STT while telemetry is disabled.") //nolint:golint,stylecheck
	}
	if params.EnableSTT && params.DisableTelemetry {
		return fmt.Errorf("Cannot enable STT while disabling telemetry.") //nolint:golint,stylecheck
	}
	if params.DisableTelemetry && !params.DisableSTT && settings.SaaS.STTEnabled {
		return fmt.Errorf("Cannot disable telemetry while STT is enabled.") //nolint:golint,stylecheck
	}

	if params.LogOut && (params.Email != "" || params.SessionID != "") {
		return fmt.Errorf("Cannot loguot while updating Percona Platform user data.") //nolint:golint,stylecheck
	}

	return nil
}

// SaveSettings saves PMM Server settings.
// It may modify passed settings to fill defaults.
func SaveSettings(q reform.DBTX, s *Settings) error {
	s.fillDefaults()

	b, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal settings")
	}

	_, err = q.Exec("UPDATE settings SET settings = $1", b)
	if err != nil {
		return errors.Wrap(err, "failed to update settings")
	}

	return nil
}

// deduplicateAWSPartitions deduplicates AWS partitions list.
func deduplicateAWSPartitions(partitions []string) []string {
	set := make(map[string]struct{})
	for _, p := range partitions {
		set[p] = struct{}{}
	}

	slice := make([]string, 0, len(set))
	for partition := range set {
		slice = append(slice, partition)
	}
	sort.Strings(slice)

	return slice
}
