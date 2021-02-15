package check

import (
	"github.com/pkg/errors"

	"github.com/percona-platform/saas/pkg/common"
)

// Result represents a single check script result that is used to generate alert.
type Result struct {
	Summary     string            `json:"summary"`     // required
	Description string            `json:"description"` // optional
	Severity    common.Severity   `json:"severity"`    // required
	Labels      map[string]string `json:"labels"`      // optional
}

// Validate validates check result for minimal correctness.
func (r *Result) Validate() error {
	if r.Summary == "" {
		return errors.New("summary is empty")
	}

	if err := r.Severity.Validate(); err != nil {
		return err
	}

	if r.Severity < common.Error || r.Severity > common.Notice {
		// until UI is ready to support more severities
		return errors.Errorf("unhandled result severity: %s", r.Severity)
	}

	return nil
}
