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

// Package validators contains various validators.
package validators

import (
	"fmt"
	"time"
)

const (
	// MetricsResolutionMin is the smallest value metric resolution can accept.
	MetricsResolutionMin = time.Second //nolint:revive
	// MetricsResolutionMultipleOf is value metrics resolution should be multiple of.
	MetricsResolutionMultipleOf = time.Second
	// AdvisorRunIntervalMin is the smallest value Advisors run intervals can accept.
	AdvisorRunIntervalMin = time.Second //nolint:revive
	// AdvisorRunIntervalMultipleOf is value Advisors run intervals should be multiple of.
	AdvisorRunIntervalMultipleOf = time.Second
	// DataRetentionMin is the smallest value data retention can accept.
	DataRetentionMin = 24 * time.Hour //nolint:revive
	// DataRetentionMultipleOf is a value of data retention should be multiple of.
	DataRetentionMultipleOf = 24 * time.Hour
)

// MinDurationError minimum allowed duration error.
type MinDurationError struct {
	Msg string
	Min time.Duration
}

func (e MinDurationError) Error() string { return e.Msg }

// DurationNotAllowedError duration not allowed error.
type DurationNotAllowedError struct {
	Msg string
}

func (e DurationNotAllowedError) Error() string { return e.Msg }

// ValidateDuration validates duration.
func validateDuration(d, min, multipleOf time.Duration) (time.Duration, error) {
	if d < min {
		return d, MinDurationError{"min duration error", min}
	}

	if d.Truncate(multipleOf) != d {
		return d, DurationNotAllowedError{fmt.Sprintf("%v is not multiple of %v", d, multipleOf)}
	}
	return d, nil
}

// ValidateAdvisorRunInterval validates an Advisor run interval.
func ValidateAdvisorRunInterval(value time.Duration) (time.Duration, error) {
	return validateDuration(value, AdvisorRunIntervalMin, AdvisorRunIntervalMultipleOf)
}

// ValidateMetricResolution validate metric resolution.
func ValidateMetricResolution(value time.Duration) (time.Duration, error) {
	return validateDuration(value, MetricsResolutionMin, MetricsResolutionMultipleOf)
}

// ValidateDataRetention validate metric resolution.
func ValidateDataRetention(value time.Duration) (time.Duration, error) {
	return validateDuration(value, DataRetentionMin, DataRetentionMultipleOf)
}

// ValidateAWSPartitions validates AWS partitions list.
func ValidateAWSPartitions(partitions []string) error {
	if len(partitions) > len(AWSPartitions()) {
		return fmt.Errorf("aws_partitions: list is too long")
	}

	for _, p := range partitions {
		var valid bool
		for _, partition := range AWSPartitions() {
			if p == partition {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("aws_partitions: partition %q is invalid", p)
		}
	}

	return nil
}

// AWSPartitions contains standart AWS partitions.
func AWSPartitions() []string {
	return []string{
		"aws",        // Standard commercial AWS
		"aws-cn",     // China regions
		"aws-iso",    // Isolated
		"aws-us-gov", // U.S. GovCloud regions
	}
}
