package check

import (
	"strings"

	"github.com/pkg/errors"
)

//go:generate ../../bin/stringer -type=Severity -linecomment

// Severity represents severity level.
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

// Result represents a single check script result that is used to generate alert.
type Result struct {
	Summary     string            `json:"summary"`     // required
	Description string            `json:"description"` // optional
	Severity    Severity          `json:"severity"`    // required
	Labels      map[string]string `json:"labels"`      // optional
}

// Validate validates check result for minimal correctness.
func (r *Result) Validate() error {
	if r.Summary == "" {
		return errors.New("summary is empty")
	}

	if r.Severity < Emergency || r.Severity > Debug {
		return errors.Errorf("unknown result severity: %s", r.Severity)
	}

	if r.Severity < Error || r.Severity > Notice {
		// until UI is ready to support more severities
		return errors.Errorf("unhandled result severity: %s", r.Severity)
	}

	return nil
}
