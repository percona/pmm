package common

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:generate ../../bin/stringer -type=Severity -linecomment

// Severity represents alert severity level.
type Severity int

// Supported severity levels.
const (
	Unknown   Severity = iota // unknown
	Emergency                 // emergency
	Alert                     // alert
	Critical                  // critical
	Error                     // error
	Warning                   // warning
	Notice                    // notice
	Info                      // info
	Debug                     // debug
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
		return errors.Errorf("unknown severity level: %s", s)
	}

	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (s Severity) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (s *Severity) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	*s = ParseSeverity(str)

	return nil
}

// Check interfaces.
var (
	_ yaml.Marshaler = (*Severity)(nil)
	// _ yaml.Unmarshaler = (*Severity)(nil) // TODO migrate to yaml.v3
)
