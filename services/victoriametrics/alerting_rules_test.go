// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package victoriametrics

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/tests"
)

func TestAlertingRules(t *testing.T) {
	t.Run("ValidateRules", func(t *testing.T) {
		s := NewAlertingRules()

		t.Run("Valid", func(t *testing.T) {
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
			err := s.ValidateRules(context.Background(), rules)
			assert.NoError(t, err)
		})

		t.Run("FormerZero", func(t *testing.T) {
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
			err := s.ValidateRules(context.Background(), rules)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Invalid alerting rules."), err)
		})

		t.Run("Invalid", func(t *testing.T) {
			rules := strings.TrimSpace(`
groups:
- name: example
  rules:
  - alert: HighRequestLatency
			`) + "\n"
			err := s.ValidateRules(context.Background(), rules)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Invalid alerting rules."), err)
		})
	})
}
