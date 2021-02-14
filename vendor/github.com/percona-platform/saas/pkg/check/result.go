package check

import (
	"net/url"

	"github.com/pkg/errors"

	"github.com/percona-platform/saas/pkg/common"
)

// Result represents a single check script result that is used to generate alert.
type Result struct {
	Summary     string            `json:"summary"`       // required
	Description string            `json:"description"`   // optional
	ReadMoreURL string            `json:"read_more_url"` // optional
	Severity    common.Severity   `json:"severity"`      // required
	Labels      map[string]string `json:"labels"`        // optional
}

// Validate checks result fields for emptiness/correctness.
func (r *Result) Validate() error {
	if r.Summary == "" {
		return errors.New("summary is empty")
	}

	if r.ReadMoreURL != "" {
		_, err := url.ParseRequestURI(r.ReadMoreURL)
		if err != nil {
			return errors.Errorf("read_more_url: %s is invalid", r.ReadMoreURL)
		}
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
