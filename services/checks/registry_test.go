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

package checks

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("create and collect Alerts", func(t *testing.T) {
		alertTTL := resolveTimeoutFactor * defaultResendInterval
		r := newRegistry(alertTTL)

		nowValue := time.Now().UTC().Round(0) // strip a monotonic clock reading
		r.nowF = func() time.Time { return nowValue }
		checkResults := []sttCheckResult{
			{
				checkName: "name",
				interval:  check.Standard,
				target: target{
					agentID:   "/agent_id/123",
					serviceID: "/service_id/123",
					labels: map[string]string{
						"foo": "bar",
					},
				},
				result: check.Result{
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
				checkName: "name2",
				target: target{
					agentID:   "/agent_id/321",
					serviceID: "/service_id/321",
					labels: map[string]string{
						"bar": "foo",
					},
				},
				result: check.Result{
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

		expectedAlerts := []*ammodels.PostableAlert{
			{
				Annotations: map[string]string{
					"summary":       "check summary",
					"description":   "check description",
					"read_more_url": "https://www.example.com",
				},
				EndsAt: strfmt.DateTime(nowValue.Add(alertTTL)),
				Alert: ammodels.Alert{
					Labels: map[string]string{
						"alert_id":  "/stt/e7b471407fe9734eac5b6adb178ee0ef08ef45f2",
						"alertname": "name",
						"baz":       "qux",
						"foo":       "bar",
						"severity":  "warning",
						"stt_check": "1",
					},
				},
			},
			{
				Annotations: map[string]string{
					"summary":       "check summary 2",
					"description":   "check description 2",
					"read_more_url": "https://www.example2.com",
				},
				EndsAt: strfmt.DateTime(nowValue.Add(alertTTL)),
				Alert: ammodels.Alert{
					Labels: map[string]string{
						"alert_id":  "/stt/8fa5695dc34160333eeeb05f00bf1ddbd98be59c",
						"alertname": "name2",
						"qux":       "baz",
						"bar":       "foo",
						"severity":  "notice",
						"stt_check": "1",
					},
				},
			},
		}

		collectedAlerts := r.collect()
		assert.ElementsMatch(t, expectedAlerts, collectedAlerts)
	})

	t.Run("delete check results by interval", func(t *testing.T) {
		alertTTL := resolveTimeoutFactor * defaultResendInterval
		r := newRegistry(alertTTL)

		nowValue := time.Now().UTC().Round(0) // strip a monotonic clock reading
		r.nowF = func() time.Time { return nowValue }
		checkResults := []sttCheckResult{
			{
				checkName: "name",
				interval:  check.Standard,
				target: target{
					agentID:   "/agent_id/123",
					serviceID: "/service_id/123",
					labels: map[string]string{
						"foo": "bar",
					},
				},
				result: check.Result{
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
				checkName: "name2",
				interval:  check.Frequent,
				target: target{
					agentID:   "/agent_id/321",
					serviceID: "/service_id/321",
					labels: map[string]string{
						"bar": "foo",
					},
				},
				result: check.Result{
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

		expectedAlert := &ammodels.PostableAlert{
			Annotations: map[string]string{
				"summary":       "check summary 2",
				"description":   "check description 2",
				"read_more_url": "https://www.example2.com",
			},
			EndsAt: strfmt.DateTime(nowValue.Add(alertTTL)),
			Alert: ammodels.Alert{
				Labels: map[string]string{
					"alert_id":  "/stt/8fa5695dc34160333eeeb05f00bf1ddbd98be59c",
					"alertname": "name2",
					"qux":       "baz",
					"bar":       "foo",
					"severity":  "notice",
					"stt_check": "1",
				},
			},
		}

		collectedAlerts := r.collect()
		require.Len(t, collectedAlerts, 1)
		assert.Equal(t, expectedAlert, collectedAlerts[0])
	})

	t.Run("delete check result by name", func(t *testing.T) {
		alertTTL := resolveTimeoutFactor * defaultResendInterval
		r := newRegistry(alertTTL)

		nowValue := time.Now().UTC().Round(0) // strip a monotonic clock reading
		r.nowF = func() time.Time { return nowValue }
		checkResults := []sttCheckResult{
			{
				checkName: "name1",
				target: target{
					agentID:   "/agent_id/123",
					serviceID: "/service_id/123",
					labels: map[string]string{
						"foo": "bar",
					},
				},
				result: check.Result{
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
				checkName: "name2",
				target: target{
					agentID:   "/agent_id/321",
					serviceID: "/service_id/321",
					labels: map[string]string{
						"bar": "foo",
					},
				},
				result: check.Result{
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

		expectedAlert := &ammodels.PostableAlert{
			Annotations: map[string]string{
				"summary":       "check summary 2",
				"description":   "check description 2",
				"read_more_url": "https://www.example2.com",
			},
			EndsAt: strfmt.DateTime(nowValue.Add(alertTTL)),
			Alert: ammodels.Alert{
				Labels: map[string]string{
					"alert_id":  "/stt/8fa5695dc34160333eeeb05f00bf1ddbd98be59c",
					"alertname": "name2",
					"qux":       "baz",
					"bar":       "foo",
					"severity":  "notice",
					"stt_check": "1",
				},
			},
		}

		collectedAlerts := r.collect()
		require.Len(t, collectedAlerts, 1)
		assert.Equal(t, expectedAlert, collectedAlerts[0])
	})

	t.Run("empty interval recognized as standard", func(t *testing.T) {
		alertTTL := resolveTimeoutFactor * defaultResendInterval
		r := newRegistry(alertTTL)

		nowValue := time.Now().UTC().Round(0) // strip a monotonic clock reading
		r.nowF = func() time.Time { return nowValue }
		checkResults := []sttCheckResult{
			{
				checkName: "name",
				interval:  check.Standard,
				target: target{
					agentID:   "/agent_id/123",
					serviceID: "/service_id/123",
					labels: map[string]string{
						"foo": "bar",
					},
				},
				result: check.Result{
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
				checkName: "name2",
				target: target{
					agentID:   "/agent_id/321",
					serviceID: "/service_id/321",
					labels: map[string]string{
						"bar": "foo",
					},
				},
				result: check.Result{
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

		collectedAlerts := r.collect()
		assert.Empty(t, collectedAlerts)
	})
}
