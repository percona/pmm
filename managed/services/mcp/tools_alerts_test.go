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

package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	rulesPath    = "/graph/api/prometheus/grafana/api/v1/rules"
	amAlertsPath = "/graph/api/alertmanager/grafana/api/v2/alerts"
	silencesPath = "/graph/api/alertmanager/grafana/api/v2/silences"
)

// alertsRoutes serves the alert fixtures.
func alertsRoutes() map[string]fixture {
	return map[string]fixture{
		"GET " + rulesPath:    {file: "alerts_rules.json"},
		"GET " + amAlertsPath: {file: "alerts_am_alerts.json"},
		"GET " + silencesPath: {file: "alerts_silences.json"},
	}
}

// callAlerts calls pmm_alerts and returns its structured result and text.
func callAlerts(t *testing.T, fake *fakePMM, args map[string]any) (alertsResult, string) {
	t.Helper()

	res := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", args)
	return structured[alertsResult](t, res), textOf(t, res)
}

// alertKeys names each row as "<name>@<service_name>", "<name>@<node_name>" or "<name>@-", "[no value]" counting as absent.
func alertKeys(rows []alertRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		target := "-"
		for _, label := range []string{"service_name", "node_name"} {
			v := r.Labels[label]
			if v != "" && v != "[no value]" {
				target = v
				break
			}
		}
		out = append(out, r.Name+"@"+target)
	}
	return out
}

// firingAlert is one alert of rulesBody; pending or recovering puts it in that state.
type firingAlert struct {
	rule       string
	labels     map[string]string
	activeAt   time.Time
	pending    bool
	recovering bool
}

// rulesBody is a rules API response in the shape of alerts_rules.json, with one rule per alert.
func rulesBody(t *testing.T, alerts ...firingAlert) []byte {
	t.Helper()

	rules := make([]map[string]any, 0, len(alerts))
	for i, a := range alerts {
		labels := map[string]string{"alertname": a.rule, "grafana_folder": "Alerts"}
		maps.Copy(labels, a.labels)
		ruleState, alertState := "firing", "Alerting"
		switch {
		case a.pending:
			ruleState, alertState = "pending", "Pending"
		case a.recovering:
			ruleState, alertState = "recovering", "Recovering"
		}
		rules = append(rules, map[string]any{
			"state":      ruleState,
			"name":       a.rule,
			"folderUid":  "alerts",
			"uid":        fmt.Sprintf("rule-%03d", i),
			"health":     "ok",
			"type":       "alerting",
			"provenance": nil,
			"alerts": []map[string]any{{
				"labels":      labels,
				"annotations": map[string]string{"summary": a.rule + " fired"},
				"state":       alertState,
				"activeAt":    a.activeAt.UTC().Format(time.RFC3339),
			}},
		})
	}
	return mustJSON(t, map[string]any{
		"status": "success",
		"data": map[string]any{
			"groups": []map[string]any{{"name": "Generated", "file": "Alerts", "folderUid": "alerts", "rules": rules}},
		},
	})
}

func TestAlertsTool(t *testing.T) {
	t.Parallel()

	note := "Firing, recovering and pending alerts only; resolved alerts are not shown. A rule fires only after its pending period, so a problem that began minutes ago may be pending or not listed yet."
	at := func(hour, minute int) time.Time { return time.Date(2026, 9, 14, hour, minute, 0, 0, time.UTC) }
	ptr := func(v time.Time) *time.Time { return &v }

	t.Run("EveryAlertOfTheRules", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes())
		got, text := callAlerts(t, fake, nil)

		assert.Equal(t, []string{
			"PMM Grafana is down@pmm-server",
			"Fake leader@shop-b",
			"PostgreSQL down@shop-d",
			"MySQL down@shop-b",
			"MySQL down@shop-a",
			"MySQL down@shop-rds",
			"MySQL down@shop-gone",
			"MySQL down@shop-quiet",
			"ProxySQL down@proxy-1",
			"Node high CPU@host-a",
			"Node high CPU@host-shared",
			"Node high CPU@host-b",
			"Too many connections@shop-a",
			"Too many connections@shop-c",
			"Node high CPU@bare-host",
			"Vector one@-",
			"Too many connections@shop-quiet",
		}, alertKeys(got.Alerts), "the pending alert comes after the firing ones")
		assert.Equal(t, page{Total: 17, Offset: 0, Limit: 50}, got.Page)
		assert.Equal(t, note, got.Note)
		assert.True(t, strings.HasPrefix(text, note+"\n- critical | PMM Grafana is down | node=pmm-server | since 2026-09-14T11:45:00Z | "), text)
		fake.assertForwardedAuth(t)

		assert.Equal(t, alertRow{
			Name:     "MySQL down",
			Severity: "critical",
			State:    "firing",
			Labels: map[string]string{
				"alertname": "MySQL down", "grafana_folder": "Alerts", "node_name": "host-a", "percona_alerting": "1",
				"service_id": "svc-a", "service_name": "shop-a", "severity": "critical", "template_name": "pmm_mysql_down",
			},
			Summary:     "MySQL down (shop-a)",
			Description: "MySQL shop-a on host-a is down.",
			StartsAt:    at(11, 0),
			PMMURL:      "https://pmm.example.com/graph/alerting/grafana/mysql-down/view",
		}, got.Alerts[4], "an Alertmanager alert with one more label is not this alert's")

		rules := fake.requestsTo(rulesPath)
		require.Len(t, rules, 1)
		assert.Equal(t, []string{"firing", "recovering", "pending"}, queryOf(t, rules[0])["state"])
		am := fake.requestsTo(amAlertsPath)
		require.Len(t, am, 1)
		assert.Equal(t, "false", queryOf(t, am[0]).Get("active"))
		assert.Len(t, fake.requestsTo(silencesPath), 1)
	})

	t.Run("SilenceJoinedByRuleUIDAndLabelSet", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes())
		got, text := callAlerts(t, fake, nil)
		require.Len(t, got.Alerts, 17)
		byKey := map[string]alertRow{}
		for i, key := range alertKeys(got.Alerts) {
			byKey[key] = got.Alerts[i]
		}

		silenced := byKey["Too many connections@shop-a"]
		assert.True(t, silenced.Silenced)
		assert.Equal(t, ptr(time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)), silenced.SilencedUntil, "the latest end of its two silences")

		other := byKey["Too many connections@shop-c"]
		assert.False(t, other.Silenced, "the same rule's other alert; a silenced alert of another rule UID has its labels")
		assert.Nil(t, other.SilencedUntil)

		inhibited := byKey["Node high CPU@host-a"]
		assert.False(t, inhibited.Silenced, "inhibited, not silenced")
		assert.Nil(t, inhibited.SilencedUntil)

		internal := byKey["Vector one@-"]
		assert.True(t, internal.Silenced, "matched with Grafana's __ labels on both sides")

		pending := byKey["Too many connections@shop-quiet"]
		assert.Equal(t, "pending", pending.State)
		assert.False(t, pending.Silenced, "the Alertmanager has no pending alert")
		assert.Equal(t, ptr(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC)), internal.SilencedUntil)
		assert.NotContains(t, internal.Labels, "__grafana_receiver__")
		assert.NotContains(t, internal.Labels, "__grafana_autogenerated__")
		assert.Equal(t, "[no value]", internal.Labels["service_name"])

		assert.NotContains(t, alertKeys(got.Alerts), "Hidden rule@shop-a", "an Alertmanager alert never adds an alert")
		assert.NotContains(t, text, "Hidden rule")
		assert.Equal(t, 2, strings.Count(text, "silenced until"), text)

		// The rules API keeps a label whose template expands to an empty string; Grafana's Alertmanager drops it.
		fake = newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath: {body: rulesBody(t, firingAlert{rule: "empty", labels: map[string]string{"severity": "warning", "instance": ""}, activeAt: at(9, 0)})},
			"GET " + amAlertsPath: {body: mustJSON(t, []map[string]any{{
				"labels": map[string]string{"alertname": "empty", "grafana_folder": "Alerts", "severity": "warning", ruleUIDLabel: "rule-000"},
				"status": map[string]any{"inhibitedBy": []string{}, "silencedBy": []string{"5f1c2b7e-0000-4000-8000-000000000001"}, "state": "suppressed"},
			}})},
		}))
		got, _ = callAlerts(t, fake, nil)
		require.Len(t, got.Alerts, 1)
		assert.True(t, got.Alerts[0].Silenced, "a label with an empty value is left out of the join")
		assert.Equal(t, ptr(time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)), got.Alerts[0].SilencedUntil)
	})

	t.Run("PendingAlertIsNeverSilenced", func(t *testing.T) {
		t.Parallel()

		// An Alertmanager alert still held from an earlier firing, with the pending alert's labels and rule UID.
		var am []map[string]any
		require.NoError(t, json.Unmarshal(readFixture(t, "alerts_am_alerts.json"), &am))
		earlier := am[0]
		labels, ok := earlier["labels"].(map[string]any)
		require.True(t, ok)
		labels["service_id"], labels["service_name"] = "svc-quiet", "shop-quiet"
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + amAlertsPath: {body: mustJSON(t, []map[string]any{earlier})},
		}))

		got, _ := callAlerts(t, fake, map[string]any{"service_name": "shop-quiet"})
		require.Contains(t, alertKeys(got.Alerts), "Too many connections@shop-quiet")
		for _, a := range got.Alerts {
			if a.State == "pending" {
				assert.False(t, a.Silenced)
				assert.Nil(t, a.SilencedUntil)
			}
		}
	})

	t.Run("SilenceMissingFromSilences", func(t *testing.T) {
		t.Parallel()

		// The conn alert of shop-a, silenced only by a silence that /silences does not return.
		var am []map[string]any
		require.NoError(t, json.Unmarshal(readFixture(t, "alerts_am_alerts.json"), &am))
		conn := am[0]
		require.Equal(t, "conn", conn["labels"].(map[string]any)[ruleUIDLabel])
		conn["status"] = map[string]any{"inhibitedBy": []string{}, "silencedBy": []string{"5f1c2b7e-0000-4000-8000-0000000000ff"}, "state": "suppressed"}
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + amAlertsPath: {body: mustJSON(t, []map[string]any{conn})},
		}))

		got, text := callAlerts(t, fake, map[string]any{"service_name": "shop-a"})
		require.Equal(t, []string{"MySQL down@shop-a", "Too many connections@shop-a"}, alertKeys(got.Alerts))
		assert.True(t, got.Alerts[1].Silenced)
		assert.Nil(t, got.Alerts[1].SilencedUntil, "no end is known")
		assert.False(t, got.Alerts[0].Silenced)
		assert.Contains(t, text, "| since 2026-09-14T10:00:00Z | silenced | [View in PMM", text)
		assert.NotContains(t, text, "silenced until", text)
		assert.Len(t, fake.requestsTo(silencesPath), 1)
	})

	t.Run("Text", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes())
		_, text := callAlerts(t, fake, map[string]any{"service_name": "shop-a"})
		assert.Equal(t, strings.Join([]string{
			note,
			"- critical | MySQL down | service=shop-a | node=host-a | since 2026-09-14T11:00:00Z | " +
				"[View in PMM ↗](https://pmm.example.com/graph/alerting/grafana/mysql-down/view)",
			"  MySQL down (shop-a)",
			"- warning | Too many connections | service=shop-a | node=host-a | since 2026-09-14T10:00:00Z | silenced until 2026-09-15T08:00:00Z | " +
				"[View in PMM ↗](https://pmm.example.com/graph/alerting/grafana/conn/view)",
			"  MySQL too many connections (shop-a)",
		}, "\n"), text)

		_, text = callAlerts(t, fake, map[string]any{"severity": "info"})
		assert.Equal(t, strings.Join([]string{
			note,
			"- info | Vector one | since 2026-09-14T08:00:00Z | silenced until 2026-09-14T18:00:00Z | " +
				"[View in PMM ↗](https://pmm.example.com/graph/alerting/grafana/vector-one/view)",
			"  Always firing",
		}, "\n"), text, "[no value] labels are absent")
	})

	t.Run("Filters", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes())
		for _, tc := range []struct {
			args map[string]any
			want []string
		}{
			{map[string]any{"service_name": "shop-a"}, []string{"MySQL down@shop-a", "Too many connections@shop-a"}},
			{map[string]any{"service_id": "svc-c"}, []string{"Too many connections@shop-c"}},
			{
				map[string]any{"node_name": "host-a"},
				[]string{"MySQL down@shop-a", "MySQL down@shop-quiet", "Node high CPU@host-a", "Too many connections@shop-a", "Too many connections@shop-quiet"},
			},
			{map[string]any{"node_name": "host-shared"}, []string{"PostgreSQL down@shop-d", "Node high CPU@host-shared", "Too many connections@shop-c"}},
			{
				map[string]any{"severity": "WARNING"},
				[]string{
					"Node high CPU@host-a", "Node high CPU@host-shared", "Node high CPU@host-b", "Too many connections@shop-a", "Too many connections@shop-c",
					"Node high CPU@bare-host", "Too many connections@shop-quiet",
				},
			},
			{
				map[string]any{"severity": "critical"},
				[]string{
					"PMM Grafana is down@pmm-server", "Fake leader@shop-b", "PostgreSQL down@shop-d", "MySQL down@shop-b", "MySQL down@shop-a",
					"MySQL down@shop-rds", "MySQL down@shop-gone", "MySQL down@shop-quiet", "ProxySQL down@proxy-1",
				},
			},
			{map[string]any{"service_name": "shop-a", "severity": "warning"}, []string{"Too many connections@shop-a"}},
			{map[string]any{"service_id": "svc-c", "severity": "critical"}, []string{}},
			{map[string]any{"severity": "crit"}, []string{}},
		} {
			got, _ := callAlerts(t, fake, tc.args)
			assert.Equal(t, tc.want, alertKeys(got.Alerts), "%v", tc.args)
			assert.Equal(t, len(tc.want), got.Page.Total, "%v", tc.args)
		}
	})

	t.Run("NotNarrowedToTheServiceList", func(t *testing.T) {
		t.Parallel()

		// The caller's up series name shop-a, shop-quiet and shop-c only.
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET /graph/api/datasources/name/Metrics": {file: "datasource_metrics.json"},
			serviceListKey: {file: "alerts_up.json"},
		}))
		got, _ := callAlerts(t, fake, map[string]any{"service_name": "shop-b"})
		assert.Equal(t, []string{"Fake leader@shop-b", "MySQL down@shop-b"}, alertKeys(got.Alerts))
		got, _ = callAlerts(t, fake, map[string]any{"node_name": "host-b"})
		assert.Equal(t, []string{"MySQL down@shop-b", "Node high CPU@host-b"}, alertKeys(got.Alerts))

		alerting := len(fake.requestsTo(rulesPath)) + len(fake.requestsTo(amAlertsPath)) + len(fake.requestsTo(silencesPath))
		assert.Equal(t, alerting, fake.requestCount(), "only Grafana's alerting APIs are read")
		assert.Zero(t, listQueries(fake), "no service list is read")
	})

	t.Run("FilterMatchingNoAlert", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			args map[string]any
			want string
		}{
			{map[string]any{"service_id": "svc-none"}, "No firing, recovering or pending alerts match the service_id filter: check its value, or call pmm_alerts without it."},
			{map[string]any{"service_name": "shop-none"}, "No firing, recovering or pending alerts match the service_name filter: check its value, or call pmm_alerts without it."},
			{map[string]any{"node_name": "host-none"}, "No firing, recovering or pending alerts match the node_name filter: check its value, or call pmm_alerts without it."},
			{
				map[string]any{"service_id": "svc-a", "service_name": "shop-b"},
				"No firing, recovering or pending alerts match the service_id and service_name filters: check their values, or call pmm_alerts without them.",
			},
			{
				map[string]any{"service_name": "shop-b", "node_name": "host-a", "severity": "warning"},
				"No firing, recovering or pending alerts match the service_name, node_name and severity filters: check their values, or call pmm_alerts without them.",
			},
		} {
			got, text := callAlerts(t, newFakePMM(t, alertsRoutes()), tc.args)
			assert.Empty(t, got.Alerts, "%v", tc.args)
			assert.Zero(t, got.Page.Total, "%v", tc.args)
			assert.Equal(t, note+"\n"+tc.want, text, "%v", tc.args)
		}
	})

	t.Run("Order", func(t *testing.T) {
		t.Parallel()

		server := func(name, severity string, activeAt time.Time) firingAlert {
			labels := map[string]string{}
			if severity != "" {
				labels["severity"] = severity
			}
			return firingAlert{rule: name, labels: labels, activeAt: activeAt}
		}
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath: {body: rulesBody(t,
				server("b-warning-old", "warning", at(9, 0)),
				server("a-critical-old", "Critical", at(9, 0)),
				server("c-error", "error", at(10, 0)),
				server("d-info", "info", at(11, 0)),
				server("e-high", "high", at(11, 0)),
				server("f-none", "", at(11, 30)),
				server("g-critical-new", "critical", at(11, 0)),
				server("h-warning-new", "warning", at(11, 0)),
				server("a-warning-new", "warning", at(11, 0)),
				server("i-alert", "Alert", at(9, 0)),
				server("j-emergency", "emergency", at(8, 0)),
			)},
		}))
		got, _ := callAlerts(t, fake, nil)
		assert.Equal(t, []string{
			"j-emergency@-", "i-alert@-",
			"g-critical-new@-", "a-critical-old@-",
			"c-error@-",
			"a-warning-new@-", "h-warning-new@-", "b-warning-old@-",
			"e-high@-", "d-info@-",
			"f-none@-",
		}, alertKeys(got.Alerts))
	})

	t.Run("Pending", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath: {body: rulesBody(t,
				firingAlert{rule: "b-critical-pending", labels: map[string]string{"severity": "critical"}, activeAt: at(11, 30), pending: true},
				firingAlert{rule: "a-warning-firing", labels: map[string]string{"severity": "warning"}, activeAt: at(9, 0)},
				firingAlert{rule: "c-emergency-pending", labels: map[string]string{"severity": "emergency"}, activeAt: at(11, 0), pending: true},
			)},
		}))
		got, text := callAlerts(t, fake, nil)
		assert.Equal(t, []string{"a-warning-firing@-", "c-emergency-pending@-", "b-critical-pending@-"}, alertKeys(got.Alerts),
			"firing alerts come first, then pending ones, each by severity")
		assert.Equal(t, []string{"firing", "pending", "pending"}, []string{got.Alerts[0].State, got.Alerts[1].State, got.Alerts[2].State})
		assert.Equal(t, at(11, 0), got.Alerts[1].StartsAt, "a pending alert's start is when it went pending")
		assert.False(t, got.Alerts[1].Silenced)
		assert.Contains(t, text, "\n- emergency | c-emergency-pending | pending | since 2026-09-14T11:00:00Z | ")
		assert.Contains(t, text, "\n- warning | a-warning-firing | since 2026-09-14T09:00:00Z | ")
	})

	t.Run("Recovering", func(t *testing.T) {
		t.Parallel()

		// The Alertmanager alert of the recovering alert, rule-001, silenced until 2026-09-15T08:00:00Z.
		am := []map[string]any{{
			"labels": map[string]string{"alertname": "c-recovering", "grafana_folder": "Alerts", "severity": "warning", ruleUIDLabel: "rule-001"},
			"status": map[string]any{"inhibitedBy": []string{}, "silencedBy": []string{"5f1c2b7e-0000-4000-8000-000000000001"}, "state": "suppressed"},
		}}
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath: {body: rulesBody(t,
				firingAlert{rule: "b-critical-pending", labels: map[string]string{"severity": "critical"}, activeAt: at(11, 30), pending: true},
				firingAlert{rule: "c-recovering", labels: map[string]string{"severity": "warning"}, activeAt: at(11, 50), recovering: true},
				firingAlert{rule: "a-warning-firing", labels: map[string]string{"severity": "warning"}, activeAt: at(9, 0)},
			)},
			"GET " + amAlertsPath: {body: mustJSON(t, am)},
		}))
		got, text := callAlerts(t, fake, nil)
		require.Equal(t, []string{"a-warning-firing@-", "c-recovering@-", "b-critical-pending@-"}, alertKeys(got.Alerts),
			"firing alerts come first, then recovering, then pending ones")
		assert.Equal(t, []string{"firing", "recovering", "pending"}, []string{got.Alerts[0].State, got.Alerts[1].State, got.Alerts[2].State})
		assert.True(t, got.Alerts[1].Silenced, "a recovering alert still fires in the Alertmanager")
		assert.Equal(t, ptr(time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)), got.Alerts[1].SilencedUntil)
		assert.Contains(t, text, "\n- warning | c-recovering | recovering | since 2026-09-14T11:50:00Z | silenced until 2026-09-15T08:00:00Z | ")
		assert.Contains(t, text, "\n- warning | a-warning-firing | since 2026-09-14T09:00:00Z | ")
	})

	t.Run("TiesByLabelSet", func(t *testing.T) {
		t.Parallel()

		// One rule's alerts that started firing together, out of order as Grafana's state cache returns them.
		alerts := make([]map[string]any, 0, 4)
		for _, instance := range []string{"db-3", "db-1", "db-4", "db-2"} {
			alerts = append(alerts, map[string]any{
				"labels":   map[string]string{"alertname": "Outage", "severity": "critical", "instance": instance},
				"state":    "Alerting",
				"activeAt": at(11, 0).Format(time.RFC3339),
			})
		}
		body := mustJSON(t, map[string]any{"status": "success", "data": map[string]any{"groups": []map[string]any{{
			"name": "Outage", "rules": []map[string]any{{"uid": "outage", "name": "Outage", "provenance": nil, "alerts": alerts}},
		}}}})
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath:    {body: body},
			"GET " + amAlertsPath: {body: []byte("[]")},
		}))

		for range 3 {
			got, _ := callAlerts(t, fake, nil)
			instances := make([]string, 0, len(got.Alerts))
			for _, a := range got.Alerts {
				instances = append(instances, a.Labels["instance"])
			}
			assert.Equal(t, []string{"db-1", "db-2", "db-3", "db-4"}, instances)
		}
	})

	t.Run("Paging", func(t *testing.T) {
		t.Parallel()

		many := make([]firingAlert, 0, 120)
		for i := range 120 {
			many = append(many, firingAlert{
				rule:     fmt.Sprintf("Alert %03d", i),
				labels:   map[string]string{"severity": "warning"},
				activeAt: testNow.Add(-time.Duration(i) * time.Minute),
			})
		}
		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath:    {body: rulesBody(t, many...)},
			"GET " + amAlertsPath: {body: []byte("[]")},
		}))

		got, text := callAlerts(t, fake, map[string]any{"limit": 50})
		assert.Equal(t, page{Total: 120, Offset: 0, Limit: 50, Truncated: true}, got.Page)
		require.Len(t, got.Alerts, 50)
		assert.Equal(t, "Alert 000", got.Alerts[0].Name, "newest first")
		assert.True(t, strings.HasSuffix(text, "\nshowing 1–50 of 120; pass offset=50 for more"), text)
		assert.True(t, strings.HasPrefix(text, note+"\n- warning | Alert 000 | since 2026-09-14T12:00:00Z | "), text)

		got, _ = callAlerts(t, fake, nil)
		assert.Equal(t, page{Total: 120, Offset: 0, Limit: 50, Truncated: true}, got.Page, "the default limit is 50")

		got, text = callAlerts(t, fake, map[string]any{"limit": 50, "offset": 100})
		assert.Equal(t, page{Total: 120, Offset: 100, Limit: 50}, got.Page)
		require.Len(t, got.Alerts, 20)
		assert.Equal(t, "Alert 100", got.Alerts[0].Name)
		assert.NotContains(t, text, "showing ")

		got, text = callAlerts(t, fake, map[string]any{"limit": 50, "offset": 150})
		assert.Empty(t, got.Alerts)
		assert.Equal(t, note+"\nNo rows at offset 150; the alert list has 120.", text)

		got, _ = callAlerts(t, fake, map[string]any{"limit": 200})
		assert.Len(t, got.Alerts, 120)
	})

	t.Run("InvalidPagingBeforeAnyCall", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, nil)
		session := connect(t, newQANService(t, fake, false))
		for want, args := range map[string]map[string]any{
			"limit must be between 1 and 200; got 201": {"limit": 201},
			"offset must not be negative; got -1":      {"offset": -1},
		} {
			text, isError := callText(t, session, "pmm_alerts", args)
			assert.True(t, isError, "%v", args)
			assert.Equal(t, "error: invalid_input\n"+want, text)
		}
		assert.Zero(t, fake.requestCount())
	})

	t.Run("CallsInFlight", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes())
		s := newQANService(t, fake, false)
		session := connect(t, s)
		key := keyOf(callerAuth{authorization: testToken})

		releases := make([]func(), 0, maxMetricCallsPerCaller)
		for range maxMetricCallsPerCaller {
			release, hit := s.metrics.acquire(key)
			require.Equal(t, limitNone, hit)
			releases = append(releases, release)
		}
		text, isError := callText(t, session, "pmm_alerts", nil)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Contains(t, text, "at most 4 metric calls can be in flight for these credentials")
		assert.Zero(t, fake.requestCount())

		// Invalid paging is reported before the limit.
		text, _ = callText(t, session, "pmm_alerts", map[string]any{"limit": alertsMaxLimit + 1})
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)

		for _, release := range releases {
			release()
		}
		for i := range maxMetricCallsPerReplica {
			release, hit := s.metrics.acquire(fmt.Sprintf("other-%d", i))
			require.Equal(t, limitNone, hit)
			t.Cleanup(release)
		}
		text, isError = callText(t, session, "pmm_alerts", nil)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Contains(t, text, "this replica already has 8 metric calls")
		assert.Zero(t, fake.requestCount())
	})

	t.Run("CallRate", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes())
		s := newQANService(t, fake, false)
		session := connect(t, s)

		// More calls than the per-caller slots, one at a time: each call releases its slot.
		for range metricCallsBurst {
			text, isError := callText(t, session, "pmm_alerts", nil)
			require.False(t, isError, text)
		}
		before := fake.requestCount()
		for _, name := range []string{"pmm_alerts", "pmm_list_metrics"} {
			res := callTool(t, session, name, nil)
			text := textOf(t, res)
			require.True(t, res.IsError, text)
			assert.Equal(t, "error: rate_limited\ntoo many calls for these credentials; retry in 1 s", text, "%s: the metric tools share one bucket", name)
			meta, ok := res.Meta["com.percona/error"].(map[string]any)
			require.True(t, ok, "%v", res.Meta)
			assert.InDelta(t, 1, meta["retry_after"], 0, name)
		}
		assert.Equal(t, before, fake.requestCount(), "a refused call reaches no backend")

		text, isError := callText(t, connectWithHeaders(t, s, http.Header{"Authorization": {"Bearer glsa_other_token"}}), "pmm_alerts", nil)
		assert.False(t, isError, text)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()

		unauthorized := fixture{file: "error_401.json", status: http.StatusUnauthorized}
		for path, want := range map[string]string{
			rulesPath:    "error: unauthorized\nPMM returned 401 on GET " + rulesPath + ": Unauthorized",
			amAlertsPath: "error: unauthorized\nPMM returned 401 on GET " + amAlertsPath + ": Unauthorized",
			silencesPath: "error: unauthorized\nPMM returned 401 on GET " + silencesPath + ": Unauthorized",
		} {
			fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{"GET " + path: unauthorized}))
			text, isError := callText(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", nil)
			assert.True(t, isError, path)
			assert.Equal(t, want, text, path)
		}
	})

	t.Run("EmptyResultCarriesTheNote", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + rulesPath: {body: []byte(`{"status":"success","data":{"groups":[],"totals":{}}}`)},
		}))
		res := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", nil)
		got := structured[alertsResult](t, res)
		assert.Empty(t, got.Alerts)
		assert.Equal(t, page{Total: 0, Offset: 0, Limit: 50}, got.Page)
		assert.Equal(t, note, got.Note)
		assert.Equal(t, note+"\nNo firing, recovering or pending alerts found.", textOf(t, res))
		assert.Contains(t, string(mustJSON(t, res.StructuredContent)), `"alerts":[]`)
		assert.Empty(t, fake.requestsTo(amAlertsPath), "no silence state to read")

		got, text := callAlerts(t, newFakePMM(t, alertsRoutes()), map[string]any{"severity": "none"})
		assert.Empty(t, got.Alerts)
		assert.Equal(t, note, got.Note)
		assert.Equal(t, note+"\nNo firing, recovering or pending alerts match the severity filter: check its value, or call pmm_alerts without it.", text)
	})

	t.Run("NoSilencesReadWithoutASilencedAlert", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
			"GET " + amAlertsPath: {body: []byte("[]")},
		}))
		got, _ := callAlerts(t, fake, nil)
		require.Len(t, got.Alerts, 17)
		for _, row := range got.Alerts {
			assert.False(t, row.Silenced, row.Name)
		}
		assert.Empty(t, fake.requestsTo(silencesPath))
	})

	t.Run("Description", func(t *testing.T) {
		t.Parallel()

		tool := listedTools(t)["pmm_alerts"]
		require.NotNil(t, tool)
		assert.Contains(t, tool.Description, note)
		assert.Contains(t, tool.Description, "Access Roles don't narrow them")
		assert.NotContains(t, strings.ToLower(tool.Description), "restrict")
	})
}

func TestAlertRuleURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://pmm.example.com/graph/alerting/grafana/mysql-down/view", alertRuleURL("https://pmm.example.com/", "mysql-down"))
	assert.Equal(t, "graph/alerting/grafana/mysql-down/view", alertRuleURL("", "mysql-down"))
	assert.Equal(t, "https://pmm.example.com/graph/alerting/list", alertRuleURL("https://pmm.example.com/", ""))
}

func TestAlertsTextOneLinePerAlert(t *testing.T) {
	t.Parallel()

	note := "Firing, recovering and pending alerts only; resolved alerts are not shown. A rule fires only after its pending period, so a problem that began minutes ago may be pending or not listed yet."
	r := alertsResult{
		Note: note,
		Page: page{Total: 1, Limit: 50},
		Alerts: []alertRow{{
			Name:     "Disk\n- critical | Fake alert",
			Severity: "warning\r\n",
			Labels:   map[string]string{"service_name": "shop-a\n- x", "node_name": "host\ta"},
			Summary:  "line one\nline two",
			StartsAt: time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC),
			PMMURL:   "https://pmm.example.com/graph/alerting/grafana/disk/view",
		}},
	}
	assert.Equal(t, strings.Join([]string{
		note,
		"- warning | Disk - critical | Fake alert | service=shop-a - x | node=host a | since 2026-09-14T10:00:00Z | " +
			"[View in PMM ↗](https://pmm.example.com/graph/alerting/grafana/disk/view)",
		"  line one line two",
	}, "\n"), r.render())
}

func TestAlertFilters(t *testing.T) {
	t.Parallel()

	bare := grafanaAlert{Labels: map[string]string{"alertname": "No labels"}}
	allNoValue := grafanaAlert{Labels: map[string]string{"alertname": "Server", "service_name": "[no value]", "node_name": "[no value]"}}
	for _, in := range []alertsInput{
		{ServiceID: "svc-a"},
		{ServiceName: "shop-a"},
		{NodeName: "host-a"},
		{Severity: "warning"},
		{ServiceName: "[no value]"},
		{NodeName: "[no value]"},
	} {
		assert.False(t, in.matches(bare), "%+v: an alert without the label", in)
		assert.False(t, in.matches(allNoValue), "%+v: an alert whose label is [no value]", in)
	}
	assert.True(t, alertsInput{}.matches(bare))

	alert := grafanaAlert{Labels: map[string]string{"service_id": "svc-a", "service_name": "shop-a", "node_name": "host-a", "severity": "Critical"}}
	assert.True(t, alertsInput{ServiceID: "svc-a", ServiceName: "shop-a", NodeName: "host-a", Severity: "CRITICAL"}.matches(alert))
	assert.False(t, alertsInput{ServiceName: "Shop-A"}.matches(alert), "names compare exactly")
	assert.False(t, alertsInput{Severity: "crit"}.matches(alert))
}

// TestAlertsPendingPageWithoutStart pins that a page of pending alerts reads no silence state, so a failing
// Alertmanager doesn't fail it, and that an alert without Grafana's activeAt shows no start time.
func TestAlertsPendingPageWithoutStart(t *testing.T) {
	t.Parallel()

	rules := `{"status":"success","data":{"groups":[{"name":"MySQL","file":"Alerts","rules":[{"name":"MySQL down","uid":"mysql-down",` +
		`"alerts":[{"labels":{"alertname":"MySQL down","service_id":"svc-a","service_name":"shop-a","node_name":"host-a"},"state":"Pending"}]}]}]}}`
	fake := newFakePMM(t, withRoutes(alertsRoutes(), map[string]fixture{
		"GET " + rulesPath:    {body: []byte(rules)},
		"GET " + amAlertsPath: {status: http.StatusInternalServerError, body: []byte(`{"message":"boom"}`)},
	}))
	res := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", nil)
	got := structured[alertsResult](t, res)
	require.Len(t, got.Alerts, 1)
	assert.Equal(t, alertPending, got.Alerts[0].State)
	assert.True(t, got.Alerts[0].StartsAt.IsZero())
	assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), "starts_at")
	assert.NotContains(t, textOf(t, res), "since")
	assert.Empty(t, fake.requestsTo(amAlertsPath))
}
