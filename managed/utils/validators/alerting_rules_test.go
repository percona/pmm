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

package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAlertingRules(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		rules := strings.TrimSpace(`
groups:
- name: example
  rules:
  - alert: HighRequestLatency
    expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
    for: 10m
    labels:
      severity: page
    annotations:
      summary: High request latency
			`) + "\n"
		err := ValidateAlertingRules(ctx, rules)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		rules := strings.TrimSpace(`
groups:
- name: example
  rules:
  - alert: HighRequestLatency
			`) + "\n"
		err := ValidateAlertingRules(ctx, rules)
		assert.Equal(t, &InvalidAlertingRuleError{"Invalid alerting rules."}, err)
	})
}
