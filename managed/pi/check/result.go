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

package check

import (
	"net/url"

	"github.com/pkg/errors"

	"github.com/percona/pmm/managed/pi/common"
)

// Result represents a single check script result that is used to generate alert.
type Result struct {
	Summary     string            `json:"summary"`       // required
	Description string            `json:"description"`   // optional
	ReadMoreURL string            `json:"read_more_url"` //nolint:tagliatelle // optional
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

	// All severity levels (Emergency, Alert, Critical, Error, Warning, Notice, Info, Debug) are supported
	return nil
}
