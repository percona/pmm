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
	"net/url"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/validators"
)

// ErrTxRequired is returned when a transaction is required.
var ErrTxRequired = errors.New("TxRequired")

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
	DisableUpdates bool
	EnableUpdates  bool

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
	// List of STT checks to disable
	DisableSTTChecks []string
	// List of STT checks to enable
	EnableSTTChecks []string
	// STT check intervals
	STTCheckIntervals STTCheckIntervals

	// Enable DBaaS features.
	EnableDBaaS bool
	// Disable DBaaS features.
	DisableDBaaS bool

	// Enable Azure Discover features.
	EnableAzurediscover bool
	// Disable Azure Discover features.
	DisableAzurediscover bool

	// Enable Integrated Alerting features.
	EnableAlerting bool
	// Disable Integrated Alerting features.
	DisableAlerting bool

	// Enable Access Control features.
	EnableAccessControl bool
	// Disable Access Control features.
	DisableAccessControl bool

	// Email config for Integrated Alerting.
	EmailAlertingSettings *EmailAlertingSettings
	// If true removes email alerting settings.
	RemoveEmailAlertingSettings bool

	// Slack config for Integrated Alerting.
	SlackAlertingSettings *SlackAlertingSettings
	// If true removes Slack alerting settings.
	RemoveSlackAlertingSettings bool

	// EnableVMCache enables caching for vmdb search queries
	EnableVMCache bool
	// DisableVMCache disables caching for vmdb search queries
	DisableVMCache bool

	// PMM Server public address.
	PMMPublicAddress       string
	RemovePMMPublicAddress bool

	// Enable Backup Management features.
	EnableBackupManagement bool
	// Disable Backup Management features.
	DisableBackupManagement bool

	// DefaultRoleID sets a default role to be assigned to new users.
	DefaultRoleID int
}

// SetPMMServerID should be run on start up to generate unique PMM Server ID.
func SetPMMServerID(q reform.DBTX) error {
	settings, err := GetSettings(q)
	if err != nil {
		return err
	}
	if settings.PMMServerID != "" {
		return nil
	}
	settings.PMMServerID = uuid.NewString()
	return SaveSettings(q, settings)
}

// UpdateSettings updates only non-zero, non-empty values.
func UpdateSettings(q reform.DBTX, params *ChangeSettingsParams) (*Settings, error) { //nolint:cyclop,maintidx
	err := ValidateSettings(params)
	if err != nil {
		return nil, NewInvalidArgumentError(err.Error())
	}

	if params.DefaultRoleID != 0 {
		tx, ok := q.(*reform.TX)
		if !ok {
			return nil, fmt.Errorf("%w: changing Role ID requires a *reform.TX", ErrTxRequired)
		}

		if err := lockRoleForChange(tx, params.DefaultRoleID); err != nil {
			return nil, err
		}
	}

	settings, err := GetSettings(q)
	if err != nil {
		return nil, err
	}

	if params.DisableUpdates {
		settings.Updates.Disabled = true
	}
	if params.EnableUpdates {
		settings.Updates.Disabled = false
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
		settings.AWSPartitions = deduplicateStrings(params.AWSPartitions)
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
		settings.SaaS.STTDisabled = true
	}
	if params.EnableSTT {
		settings.SaaS.STTDisabled = false
	}

	if params.STTCheckIntervals.RareInterval != 0 {
		settings.SaaS.STTCheckIntervals.RareInterval = params.STTCheckIntervals.RareInterval
	}
	if params.STTCheckIntervals.StandardInterval != 0 {
		settings.SaaS.STTCheckIntervals.StandardInterval = params.STTCheckIntervals.StandardInterval
	}
	if params.STTCheckIntervals.FrequentInterval != 0 {
		settings.SaaS.STTCheckIntervals.FrequentInterval = params.STTCheckIntervals.FrequentInterval
	}

	if len(params.DisableSTTChecks) != 0 {
		settings.SaaS.DisabledSTTChecks = deduplicateStrings(append(settings.SaaS.DisabledSTTChecks, params.DisableSTTChecks...))
	}

	if len(params.EnableSTTChecks) != 0 {
		m := make(map[string]struct{}, len(params.EnableSTTChecks))
		for _, p := range params.EnableSTTChecks {
			m[p] = struct{}{}
		}

		var res []string
		for _, c := range settings.SaaS.DisabledSTTChecks {
			if _, ok := m[c]; !ok {
				res = append(res, c)
			}
		}
		settings.SaaS.DisabledSTTChecks = res
	}

	if params.EnableDBaaS {
		settings.DBaaS.Enabled = true
	}

	if params.DisableDBaaS {
		settings.DBaaS.Enabled = false
	}

	if params.DisableVMCache {
		settings.VictoriaMetrics.CacheEnabled = false
	}

	if params.EnableVMCache {
		settings.VictoriaMetrics.CacheEnabled = true
	}

	if params.PMMPublicAddress != "" {
		settings.PMMPublicAddress = params.PMMPublicAddress
	}

	if params.RemovePMMPublicAddress {
		settings.PMMPublicAddress = ""
	}

	if params.DisableAzurediscover {
		settings.Azurediscover.Enabled = false
	}
	if params.EnableAzurediscover {
		settings.Azurediscover.Enabled = true
	}

	if params.DisableAlerting {
		settings.Alerting.Disabled = true
	}

	if params.EnableAlerting {
		settings.Alerting.Disabled = false
	}

	if params.DisableAccessControl {
		settings.AccessControl.Enabled = false
	}
	if params.EnableAccessControl {
		settings.AccessControl.Enabled = true
	}

	if params.RemoveEmailAlertingSettings {
		settings.Alerting.EmailAlertingSettings = nil
	}

	if params.RemoveSlackAlertingSettings {
		settings.Alerting.SlackAlertingSettings = nil
	}

	if params.EmailAlertingSettings != nil {
		settings.Alerting.EmailAlertingSettings = params.EmailAlertingSettings
	}
	if params.SlackAlertingSettings != nil {
		settings.Alerting.SlackAlertingSettings = params.SlackAlertingSettings
	}

	if params.DisableBackupManagement {
		settings.BackupManagement.Disabled = true
	}

	if params.EnableBackupManagement {
		settings.BackupManagement.Disabled = false
	}

	if params.DefaultRoleID != 0 {
		settings.DefaultRoleID = params.DefaultRoleID
	}

	err = SaveSettings(q, settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func lockRoleForChange(tx *reform.TX, roleID int) error {
	var r Role
	if err := FindAndLockRole(tx, roleID, &r); err != nil { //nolint:revive
		return err
	}

	return nil
}

func validateSlackAlertingSettings(params *ChangeSettingsParams) error {
	if params.SlackAlertingSettings != nil && params.RemoveSlackAlertingSettings {
		return errors.New("both slack_alerting_settings and remove_slack_alerting_settings are present")
	}

	if params.SlackAlertingSettings == nil {
		return nil
	}

	if !govalidator.IsURL(params.SlackAlertingSettings.URL) {
		return errors.New("invalid url value")
	}

	return nil
}

func validateEmailAlertingSettings(params *ChangeSettingsParams) error {
	if params.EmailAlertingSettings != nil && params.RemoveEmailAlertingSettings {
		return errors.New("both email_alerting_settings and remove_email_alerting_settings are present")
	}

	if params.EmailAlertingSettings == nil {
		return nil
	}

	return params.EmailAlertingSettings.Validate()
}

// ValidateSettings validates settings changes.
func ValidateSettings(params *ChangeSettingsParams) error { //nolint:cyclop
	if params.EnableUpdates && params.DisableUpdates {
		return errors.New("both enable_updates and disable_updates are present")
	}
	if params.EnableTelemetry && params.DisableTelemetry {
		return errors.New("both enable_telemetry and disable_telemetry are present")
	}
	if params.EnableSTT && params.DisableSTT {
		return errors.New("both enable_stt and disable_stt are present")
	}
	if params.EnableVMCache && params.DisableVMCache {
		return errors.New("both enable_vm_cache and disable_vm_cache are present")
	}
	if params.EnableAlerting && params.DisableAlerting {
		return errors.New("both enable_alerting and disable_alerting are present")
	}
	if err := validateEmailAlertingSettings(params); err != nil {
		return err
	}
	if err := validateSlackAlertingSettings(params); err != nil {
		return err
	}
	if params.EnableBackupManagement && params.DisableBackupManagement {
		return errors.New("both enable_backup_management and disable_backup_management are present")
	}
	// TODO: consider refactoring this and the validation for STT check intervals
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
			switch err.(type) { //nolint:errorlint
			case validators.DurationNotAllowedError:
				return errors.Errorf("%s: should be a natural number of seconds", v.fieldName)
			case validators.MinDurationError:
				return errors.Errorf("%s: minimal resolution is 1s", v.fieldName)
			default:
				return errors.Errorf("%s: unknown error for", v.fieldName)
			}
		}
	}

	checkCases = []struct {
		dur       time.Duration
		fieldName string
	}{
		{params.STTCheckIntervals.RareInterval, "rare_interval"},
		{params.STTCheckIntervals.StandardInterval, "standard_interval"},
		{params.STTCheckIntervals.FrequentInterval, "frequent_interval"},
	}
	for _, v := range checkCases {
		if v.dur == 0 {
			continue
		}

		if _, err := validators.ValidateSTTCheckInterval(v.dur); err != nil {
			switch err.(type) { //nolint:errorlint
			case validators.DurationNotAllowedError:
				return errors.Errorf("%s: should be a natural number of seconds", v.fieldName)
			case validators.MinDurationError:
				return errors.Errorf("%s: minimal resolution is 1s", v.fieldName)
			default:
				return errors.Errorf("%s: unknown error for", v.fieldName)
			}
		}
	}

	if params.DataRetention != 0 {
		if _, err := validators.ValidateDataRetention(params.DataRetention); err != nil {
			switch err.(type) { //nolint:errorlint
			case validators.DurationNotAllowedError:
				return errors.New("data_retention: should be a natural number of days")
			case validators.MinDurationError:
				return errors.New("data_retention: minimal resolution is 24h")
			default:
				return errors.New("data_retention: unknown error")
			}
		}
	}

	if err := validators.ValidateAWSPartitions(params.AWSPartitions); err != nil {
		return err
	}

	if params.AlertManagerURL != "" {
		if params.RemoveAlertManagerURL {
			return errors.New("both alert_manager_url and remove_alert_manager_url are present")
		}

		// custom validation for typical error that is not handled well by url.Parse
		if !strings.Contains(params.AlertManagerURL, "//") {
			return errors.Errorf("invalid alert_manager_url: %s - missing protocol scheme", params.AlertManagerURL)
		}
		u, err := url.Parse(params.AlertManagerURL)
		if err != nil {
			return errors.Errorf("invalid alert_manager_url: %s", err)
		}
		if u.Scheme == "" {
			return errors.Errorf("invalid alert_manager_url: %s - missing protocol scheme", params.AlertManagerURL)
		}
		if u.Host == "" {
			return errors.Errorf("invalid alert_manager_url: %s - missing host", params.AlertManagerURL)
		}
	}

	if params.PMMPublicAddress != "" && params.RemovePMMPublicAddress {
		return errors.New("both pmm_public_address and remove_pmm_public_address are present")
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
