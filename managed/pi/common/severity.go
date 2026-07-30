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

// Package common provides shared types and utilities used across Percona Intelligence (PI) services.
package common

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:generate go tool stringer -type=Severity -linecomment

// Severity represents alert severity level as present in Advisors.
type Severity int //nolint:recvcheck

// Supported severity levels.
const (
	// Use the same values as PMM API: https://github.com/percona/pmm/blob/main/api/managementpb/severity.proto
	// That allows direct conversions without custom conversion function.

	// Inline comments are for the go:generate stringer above.

	Unknown   Severity = 0 // unknown
	Emergency Severity = 1 // emergency
	Alert     Severity = 2 // alert
	Critical  Severity = 3 // critical
	Error     Severity = 4 // error
	Warning   Severity = 5 // warning
	Notice    Severity = 6 // notice
	Info      Severity = 7 // info
	Debug     Severity = 8 // debug
)

// ParseSeverity casts string to Severity.
func ParseSeverity(s string) Severity {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "emergency":
		return Emergency
	case "alert":
		return Alert
	case "critical":
		return Critical
	case "error":
		return Error
	case "warning":
		return Warning
	case "notice":
		return Notice
	case "info":
		return Info
	case "debug":
		return Debug
	default:
		return Unknown
	}
}

// Validate returns error in case of invalid severity value.
func (s Severity) Validate() error {
	if s < Emergency || s > Debug {
		return fmt.Errorf("unknown severity level: %s", s)
	}

	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (s Severity) MarshalYAML() (any, error) {
	return s.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (s *Severity) UnmarshalYAML(unmarshal func(any) error) error {
	var str string

	err := unmarshal(&str)
	if err != nil {
		return err
	}

	*s = ParseSeverity(str)

	return nil
}

// Check interfaces.
var (
	_ yaml.Marshaler = (*Severity)(nil)
)
