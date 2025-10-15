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

	"github.com/AlekSi/pointer"
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
	EnableUpdates *bool

	EnableTelemetry *bool

	MetricsResolutions MetricsResolutions

	DataRetention time.Duration

	// List of AWS partitions to use. If empty - default partitions will be used. If nil - no changes will be made.
	AWSPartitions []string

	SSHKey *string

	// Enable Advisors
	EnableAdvisors *bool

	EnableNomad *bool

	// List of Advisor checks to disable
	DisableAdvisorChecks []string
	// List of Advisor checks to enable
	EnableAdvisorChecks []string
	// Advisors run intervals
	AdvisorsRunInterval AdvisorsRunIntervals

	// Enable Azure Discover features.
	EnableAzurediscover *bool
	// Enable Percona Alerting features.
	EnableAlerting *bool

	// Enable Access Control features.
	EnableAccessControl *bool

	// EnableVMCache enables caching for vmdb search queries
	EnableVMCache *bool

	// PMM Server public address.
	PMMPublicAddress *string

	// Enable Backup Management features.
	EnableBackupManagement *bool

	// DefaultRoleID sets a default role to be assigned to new users.
	DefaultRoleID *int

	// List of items in format 'db.table.column' to be encrypted.
	EncryptedItems []string
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
func UpdateSettings(q reform.DBTX, params *ChangeSettingsParams) (*Settings, error) { //nolint:cyclop
	err := ValidateSettings(params)
	if err != nil {
		return nil, NewInvalidArgumentError("%s", err.Error())
	}

	if params.DefaultRoleID != nil {
		tx, ok := q.(*reform.TX)
		if !ok {
			return nil, fmt.Errorf("%w: changing Role ID requires a *reform.TX", ErrTxRequired)
		}

		var r Role
		if err := findRole(tx, *params.DefaultRoleID, &r); err != nil {
			return nil, err
		}
	}

	settings, err := GetSettings(q)
	if err != nil {
		return nil, err
	}

	if params.EnableUpdates != nil {
		settings.Updates.Enabled = params.EnableUpdates
	}

	if params.EnableTelemetry != nil {
		settings.Telemetry.Enabled = params.EnableTelemetry

		if !*settings.Telemetry.Enabled {
			settings.Telemetry.UUID = ""
		}
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

	if params.AWSPartitions != nil {
		settings.AWSPartitions = deduplicateStrings(params.AWSPartitions)
	}

	if params.SSHKey != nil {
		settings.SSHKey = pointer.GetString(params.SSHKey)
	}

	if params.EnableAdvisors != nil {
		settings.SaaS.Enabled = params.EnableAdvisors
	}

	if params.EnableNomad != nil {
		settings.Nomad.Enabled = params.EnableNomad
	}

	if params.AdvisorsRunInterval.RareInterval != 0 {
		settings.SaaS.AdvisorRunIntervals.RareInterval = params.AdvisorsRunInterval.RareInterval
	}
	if params.AdvisorsRunInterval.StandardInterval != 0 {
		settings.SaaS.AdvisorRunIntervals.StandardInterval = params.AdvisorsRunInterval.StandardInterval
	}
	if params.AdvisorsRunInterval.FrequentInterval != 0 {
		settings.SaaS.AdvisorRunIntervals.FrequentInterval = params.AdvisorsRunInterval.FrequentInterval
	}

	if len(params.DisableAdvisorChecks) != 0 {
		settings.SaaS.DisabledAdvisors = deduplicateStrings(append(settings.SaaS.DisabledAdvisors, params.DisableAdvisorChecks...))
	}

	if len(params.EnableAdvisorChecks) != 0 {
		m := make(map[string]struct{}, len(params.EnableAdvisorChecks))
		for _, p := range params.EnableAdvisorChecks {
			m[p] = struct{}{}
		}

		var res []string
		for _, c := range settings.SaaS.DisabledAdvisors {
			if _, ok := m[c]; !ok {
				res = append(res, c)
			}
		}
		settings.SaaS.DisabledAdvisors = res
	}

	if params.EnableVMCache != nil {
		settings.VictoriaMetrics.CacheEnabled = params.EnableVMCache
	}

	if params.PMMPublicAddress != nil {
		settings.PMMPublicAddress = pointer.GetString(params.PMMPublicAddress)
	}

	if params.EnableAzurediscover != nil {
		settings.Azurediscover.Enabled = params.EnableAzurediscover
	}

	if params.EnableAlerting != nil {
		settings.Alerting.Enabled = params.EnableAlerting
	}

	if params.EnableAccessControl != nil {
		settings.AccessControl.Enabled = params.EnableAccessControl
	}

	if params.EnableBackupManagement != nil {
		settings.BackupManagement.Enabled = params.EnableBackupManagement
	}

	if params.DefaultRoleID != nil {
		settings.DefaultRoleID = *params.DefaultRoleID
	}

	if params.EncryptedItems != nil {
		settings.EncryptedItems = params.EncryptedItems
	}

	err = SaveSettings(q, settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// ValidateSettings validates settings changes.
func ValidateSettings(params *ChangeSettingsParams) error {
	// TODO: consider refactoring this and the validation for Advisors run intervals
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
		{params.AdvisorsRunInterval.RareInterval, "rare_interval"},
		{params.AdvisorsRunInterval.StandardInterval, "standard_interval"},
		{params.AdvisorsRunInterval.FrequentInterval, "frequent_interval"},
	}
	for _, v := range checkCases {
		if v.dur == 0 {
			continue
		}

		if _, err := validators.ValidateAdvisorRunInterval(v.dur); err != nil {
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
