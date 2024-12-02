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

package checks

import (
	"testing"

	"github.com/percona/saas/pkg/check"
	"github.com/percona/saas/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/services"
)

func TestRegistry(t *testing.T) {
	t.Run("create and collect Alerts", func(t *testing.T) {
		r := newRegistry()
		checkResults := []services.CheckResult{
			{
				CheckName: "name",
				Interval:  check.Standard,
				Target: services.Target{
					AgentID:   "123",
					ServiceID: "123",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Result: check.Result{
					Summary:     "check summary",
					Description: "check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels: map[string]string{
						"baz": "qux",
					},
				},
			},
			{
				CheckName: "name2",
				Target: services.Target{
					AgentID:   "321",
					ServiceID: "321",
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Result: check.Result{
					Summary:     "check summary 2",
					Description: "check description 2",
					ReadMoreURL: "https://www.example2.com",
					Severity:    common.Notice,
					Labels: map[string]string{
						"qux": "baz",
					},
				},
			},
		}

		r.set(checkResults)

		// Empty interval means standard
		checkResults[1].Interval = check.Standard

		collectedAlerts := r.getCheckResults("")
		assert.ElementsMatch(t, checkResults, collectedAlerts)
	})

	t.Run("delete check results by interval", func(t *testing.T) {
		r := newRegistry()
		checkResults := []services.CheckResult{
			{
				CheckName: "name",
				Interval:  check.Standard,
				Target: services.Target{
					AgentID:   "123",
					ServiceID: "123",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Result: check.Result{
					Summary:     "check summary",
					Description: "check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels: map[string]string{
						"baz": "qux",
					},
				},
			},
			{
				CheckName: "name2",
				Interval:  check.Frequent,
				Target: services.Target{
					AgentID:   "321",
					ServiceID: "321",
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Result: check.Result{
					Summary:     "check summary 2",
					Description: "check description 2",
					ReadMoreURL: "https://www.example2.com",
					Severity:    common.Notice,
					Labels: map[string]string{
						"qux": "baz",
					},
				},
			},
		}

		r.set(checkResults)
		r.deleteByInterval(check.Standard)

		collectedAlerts := r.getCheckResults("")
		require.Len(t, collectedAlerts, 1)
		assert.Equal(t, checkResults[1], collectedAlerts[0])
	})

	t.Run("delete check result by name", func(t *testing.T) {
		r := newRegistry()
		checkResults := []services.CheckResult{
			{
				CheckName: "name1",
				Interval:  check.Standard,
				Target: services.Target{
					AgentID:   "123",
					ServiceID: "123",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Result: check.Result{
					Summary:     "check summary 1",
					Description: "check description 1",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels: map[string]string{
						"baz": "qux",
					},
				},
			},
			{
				CheckName: "name2",
				Interval:  check.Standard,
				Target: services.Target{
					AgentID:   "321",
					ServiceID: "321",
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Result: check.Result{
					Summary:     "check summary 2",
					Description: "check description 2",
					ReadMoreURL: "https://www.example2.com",
					Severity:    common.Notice,
					Labels: map[string]string{
						"qux": "baz",
					},
				},
			},
		}

		r.set(checkResults)
		r.deleteByName([]string{"name1"})

		collectedAlerts := r.getCheckResults("")
		require.Len(t, collectedAlerts, 1)
		assert.Equal(t, checkResults[1], collectedAlerts[0])
	})

	t.Run("empty interval recognized as standard", func(t *testing.T) {
		r := newRegistry()
		checkResults := []services.CheckResult{
			{
				CheckName: "name",
				Interval:  check.Standard,
				Target: services.Target{
					AgentID:   "123",
					ServiceID: "123",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Result: check.Result{
					Summary:     "check summary",
					Description: "check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Warning,
					Labels: map[string]string{
						"baz": "qux",
					},
				},
			},
			{
				CheckName: "name2",
				Target: services.Target{
					AgentID:   "321",
					ServiceID: "321",
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Result: check.Result{
					Summary:     "check summary 2",
					Description: "check description 2",
					ReadMoreURL: "https://www.example2.com",
					Severity:    common.Notice,
					Labels: map[string]string{
						"qux": "baz",
					},
				},
			},
		}

		r.set(checkResults)
		r.deleteByInterval(check.Standard)

		collectedAlerts := r.getCheckResults("")
		assert.Empty(t, collectedAlerts)
	})
}
