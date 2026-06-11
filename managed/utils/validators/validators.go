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
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode"
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
		return errors.New("aws_partitions: list is too long")
	}

	for _, p := range partitions {
		if !slices.Contains(AWSPartitions(), p) {
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

var (
	// ErrInvalidPasswordLen is returned when a password does not meet complexity requirements.
	ErrInvalidPasswordLen = func(minLen int) error {
		return fmt.Errorf("password must be at least %d characters long", minLen)
	}
	// ErrInvalidPasswordLetter is returned when a password does not contain at least one letter.
	ErrInvalidPasswordLetter = errors.New("password must contain at least one letter")
	// ErrInvalidPasswordDigit is returned when a password does not contain at least one digit.
	ErrInvalidPasswordDigit = errors.New("password must contain at least one digit")
	// ErrInvalidPasswordSpecial is returned when a password does not contain at least one special character.
	ErrInvalidPasswordSpecial = errors.New("password must contain at least one special character")
)

// ValidatePassword checks if a password meets complexity requirements:
// - At least minLen characters long
// - At least one uppercase or lowercase letter
// - At least one numeric digit
// - At least one special character (punctuation or symbol).
func ValidatePassword(password string, minLen int) error {
	var (
		hasLetter  = false
		hasNumber  = false
		hasSpecial = false
	)

	if len(password) < minLen {
		return ErrInvalidPasswordLen(minLen)
	}

	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsNumber(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
		// If all conditions are met, we can stop checking further characters.
		if hasLetter && hasNumber && hasSpecial {
			break
		}
	}

	if !hasLetter {
		return ErrInvalidPasswordLetter
	}
	if !hasNumber {
		return ErrInvalidPasswordDigit
	}
	if !hasSpecial {
		return ErrInvalidPasswordSpecial
	}

	return nil
}
