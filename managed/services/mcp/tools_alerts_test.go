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

// alertsRoutes serves the alert fixtures, with up as the service list.
func alertsRoutes(up string) map[string]fixture {
	return map[string]fixture{
		"GET /graph/api/datasources/name/Metrics": {file: "datasource_metrics.json"},
		serviceListKey:        {file: up},
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

// firingAlert is one alert of rulesBody.
type firingAlert struct {
	rule     string
	labels   map[string]string
	activeAt time.Time
}

// rulesBody is a rules API response in the shape of alerts_rules.json, with one rule per alert.
func rulesBody(t *testing.T, alerts ...firingAlert) []byte {
	t.Helper()

	rules := make([]map[string]any, 0, len(alerts))
	for i, a := range alerts {
		labels := map[string]string{"alertname": a.rule, "grafana_folder": "Alerts"}
		maps.Copy(labels, a.labels)
		rules = append(rules, map[string]any{
			"state":      "firing",
			"name":       a.rule,
			"folderUid":  "alerts",
			"uid":        fmt.Sprintf("rule-%03d", i),
			"health":     "ok",
			"type":       "alerting",
			"provenance": nil,
			"alerts": []map[string]any{{
				"labels":      labels,
				"annotations": map[string]string{"summary": a.rule + " fired"},
				"state":       "Alerting",
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

func TestKeepAlert(t *testing.T) {
	t.Parallel()

	// A restricted caller's list: one service and its node, and a host whose node_exporter series it can read.
	list := newServiceSet(
		[]serviceInfo{{ServiceID: "svc-a", ServiceName: "shop-a", Engine: engineMySQL, NodeID: "node-a", NodeName: "host-a"}},
		[]nodeInfo{{NodeID: "node-a", NodeName: "host-a"}, {NodeID: "node-bare", NodeName: "bare-host"}},
	)
	alert := func(labels ...string) grafanaAlert {
		a := grafanaAlert{RuleUID: "user-rule", Labels: map[string]string{"alertname": "Some alert", "severity": "warning"}}
		for i := 0; i < len(labels); i += 2 {
			a.Labels[labels[i]] = labels[i+1]
		}
		return a
	}
	pmmRule := func(uid string, labels ...string) grafanaAlert {
		a := alert(labels...)
		a.RuleUID, a.PMMRule = uid, true
		return a
	}
	withUID := func(a grafanaAlert, uid string) grafanaAlert {
		a.RuleUID = uid
		return a
	}
	annotated := alert("service_id", "svc-a", "service_name", "shop-a")
	annotated.Annotations = map[string]string{"summary": "shop-b is down", "description": "svc-b on host-b does not answer"}
	hiddenInAnnotation := alert("instance", "x")
	hiddenInAnnotation.Annotations = map[string]string{"summary": "shop-b is down"}

	for _, tc := range []struct {
		name  string
		alert grafanaAlert
		keep  bool
	}{
		{name: "a service in the list", alert: alert("service_id", "svc-a", "service_name", "shop-a", "node_id", "node-a", "node_name", "host-a"), keep: true},
		{name: "a node in the list", alert: alert("node_id", "node-bare", "node_name", "bare-host"), keep: true},
		{name: "a service outside the list", alert: alert("service_id", "svc-b", "service_name", "shop-b"), keep: false},
		{name: "only a service_name outside the list", alert: alert("service_name", "shop-b"), keep: false},
		{
			name:  "a service in the list on a node outside it, as an RDS service's down alert names pmm-server",
			alert: alert("service_id", "svc-a", "service_name", "shop-a", "node_name", "pmm-server"),
			keep:  true,
		},
		{name: "a listed service_id with a service_name outside the list", alert: alert("service_id", "svc-a", "service_name", "shop-b"), keep: false},
		{name: "a node outside the list", alert: alert("node_name", "host-b"), keep: false},
		{name: "a listed node_id with a node_name outside the list", alert: alert("node_id", "node-a", "node_name", "host-b"), keep: false},
		{name: "PMM's HA rule on a replica's PMM_HA_NODE_ID", alert: pmmRule("pmm-ha-node-unreachable", "node_id", "pmm-server-1"), keep: true},
		{name: "PMM's ClickHouse rule on HA, with service_name clickhouse", alert: pmmRule("pmm-clickhouse-down", "service_name", "clickhouse"), keep: true},
		{
			name:  "PMM's Grafana rule on a standalone server",
			alert: pmmRule("pmm-grafana-down", "node_name", "pmm-server", "instance", "pmm-server", "job", "grafana", "pmm_provisioned", "1"),
			keep:  true,
		},
		{
			name:  "a user's rule labelled pmm_provisioned on a service outside the list",
			alert: alert("pmm_provisioned", "1", "service_id", "svc-b", "service_name", "shop-b"),
			keep:  false,
		},
		{
			name:  "an Editor's rule with a catalog UID and no provenance on a service outside the list",
			alert: withUID(alert("pmm_provisioned", "1", "service_id", "svc-b", "service_name", "shop-b"), "pmm-ha-no-leader"),
			keep:  false,
		},
		{name: "no service or node label", alert: alert("instance", "x", "job", "custom"), keep: true},
		{name: "a template rule's node-level alert on a listed node", alert: alert("service_name", "[no value]", "node_name", "host-a"), keep: true},
		{name: "a template rule's node-level alert on a node outside the list", alert: alert("service_name", "[no value]", "node_name", "host-b"), keep: false},
		{name: "a template rule's server-level alert", alert: alert("service_name", "[no value]", "node_name", "[no value]"), keep: true},
		{name: "every service and node label [no value]", alert: alert("service_id", "[no value]", "service_name", "[no value]", "node_id", "[no value]", "node_name", "[no value]"), keep: true},
		{name: "an annotation naming a hidden service on a listed service's alert", alert: annotated, keep: true},
		{name: "an annotation naming a hidden service on a server-level alert", alert: hiddenInAnnotation, keep: true},
		{name: "a node_name equal to a listed service's name", alert: alert("node_name", "shop-a"), keep: false},
		{name: "a node_id equal to a listed service's id", alert: alert("node_id", "svc-a"), keep: false},
		{name: "a service_name equal to a listed node's name", alert: alert("service_name", "host-a"), keep: false},
		{name: "a service_id equal to a listed node's id", alert: alert("service_id", "node-a"), keep: false},
	} {
		assert.Equal(t, tc.keep, list.keepAlert(tc.alert), tc.name)
	}
}

func TestIsPMMRule(t *testing.T) {
	t.Parallel()

	assert.True(t, isPMMRule("file", "pmm-grafana-down"))
	assert.True(t, isPMMRule("file", "pmm-ha-node-unreachable"))
	assert.False(t, isPMMRule("", "pmm-ha-no-leader"), "a catalog UID without provenance")
	assert.False(t, isPMMRule("api", "pmm-grafana-down"), "a catalog UID with API provenance")
	assert.False(t, isPMMRule("file", "custom-file-rule"), "a file-provisioned rule outside the catalog")
}

func TestAlertsTool(t *testing.T) {
	t.Parallel()

	note := "Currently firing alerts only; resolved alerts are not shown."
	at := func(hour, minute int) time.Time { return time.Date(2026, 9, 14, hour, minute, 0, 0, time.UTC) }
	ptr := func(v time.Time) *time.Time { return &v }

	t.Run("NarrowedToTheServiceList", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
		got, text := callAlerts(t, fake, nil)

		assert.Equal(t, []string{
			"PMM Grafana is down@pmm-server",
			"MySQL down@shop-a",
			"MySQL down@shop-quiet",
			"Node high CPU@host-a",
			"Node high CPU@host-shared",
			"Too many connections@shop-a",
			"Too many connections@shop-c",
			"Vector one@-",
		}, alertKeys(got.Alerts))
		assert.Equal(t, page{Total: 8, Offset: 0, Limit: 50}, got.Page)
		assert.Equal(t, note, got.Note)
		assert.True(t, strings.HasPrefix(text, note+"\n- critical | PMM Grafana is down | node=pmm-server | since 2026-09-14T11:45:00Z | "), text)
		fake.assertForwardedAuth(t)

		assert.Equal(t, alertRow{
			Name:     "MySQL down",
			Severity: "critical",
			Labels: map[string]string{
				"alertname": "MySQL down", "grafana_folder": "Alerts", "node_name": "host-a", "percona_alerting": "1",
				"service_id": "svc-a", "service_name": "shop-a", "severity": "critical", "template_name": "pmm_mysql_down",
			},
			Summary:     "MySQL down (shop-a)",
			Description: "MySQL shop-a on host-a is down.",
			StartsAt:    at(11, 0),
			PMMURL:      "https://pmm.example.com/graph/alerting/grafana/mysql-down/view",
		}, got.Alerts[1], "an Alertmanager alert with one more label is not this alert's")

		rules := fake.requestsTo(rulesPath)
		require.Len(t, rules, 1)
		assert.Equal(t, "firing", queryOf(t, rules[0]).Get("state"))
		am := fake.requestsTo(amAlertsPath)
		require.Len(t, am, 1)
		assert.Equal(t, "false", queryOf(t, am[0]).Get("active"))
		assert.Len(t, fake.requestsTo(silencesPath), 1)
	})

	t.Run("SilenceJoinedByRuleUIDAndLabelSet", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
		got, text := callAlerts(t, fake, nil)
		require.Len(t, got.Alerts, 8)
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
		assert.Equal(t, ptr(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC)), internal.SilencedUntil)
		assert.NotContains(t, internal.Labels, "__grafana_receiver__")
		assert.NotContains(t, internal.Labels, "__grafana_autogenerated__")
		assert.Equal(t, "[no value]", internal.Labels["service_name"])

		assert.NotContains(t, alertKeys(got.Alerts), "Hidden rule@shop-a", "an Alertmanager alert never adds an alert")
		assert.NotContains(t, text, "Hidden rule")
		assert.Equal(t, 2, strings.Count(text, "silenced until"), text)
	})

	t.Run("Text", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
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

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
		for _, tc := range []struct {
			args map[string]any
			want []string
		}{
			{map[string]any{"service_name": "shop-a"}, []string{"MySQL down@shop-a", "Too many connections@shop-a"}},
			{map[string]any{"service_id": "svc-c"}, []string{"Too many connections@shop-c"}},
			{
				map[string]any{"node_name": "host-a"},
				[]string{"MySQL down@shop-a", "MySQL down@shop-quiet", "Node high CPU@host-a", "Too many connections@shop-a"},
			},
			{map[string]any{"node_name": "host-shared"}, []string{"Node high CPU@host-shared", "Too many connections@shop-c"}},
			{
				map[string]any{"severity": "WARNING"},
				[]string{"Node high CPU@host-a", "Node high CPU@host-shared", "Too many connections@shop-a", "Too many connections@shop-c"},
			},
			{
				map[string]any{"severity": "critical"},
				[]string{"PMM Grafana is down@pmm-server", "MySQL down@shop-a", "MySQL down@shop-quiet"},
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

	t.Run("FilterOutsideTheList", func(t *testing.T) {
		t.Parallel()

		want := "error: not_found\n" + errNotInList(window{From: testNow.Add(-7 * day), To: testNow}, testNow, "").message
		assert.Contains(t, want, "searched 2026-09-07T12:00:00Z – 2026-09-14T12:00:00Z")
		for _, args := range []map[string]any{
			{"service_id": "svc-b"},
			{"service_name": "shop-b"},
			{"node_name": "host-b"},
			{"service_id": "svc-gone"},
			{"service_id": "svc-a", "service_name": "shop-b"},
		} {
			fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
			text, isError := callText(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", args)
			assert.True(t, isError, "%v", args)
			assert.Equal(t, want, text, "%v", args)
			for _, part := range []string{"svc-", "shop-", "host-"} {
				assert.NotContains(t, text, part, "%v", args)
			}
			assert.Empty(t, fake.requestsTo(rulesPath), "%v: the filter is refused before any alert is read", args)
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
		fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{
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
			)},
		}))
		got, _ := callAlerts(t, fake, nil)
		assert.Equal(t, []string{
			"g-critical-new@-", "a-critical-old@-",
			"c-error@-",
			"a-warning-new@-", "h-warning-new@-", "b-warning-old@-",
			"e-high@-", "d-info@-",
			"f-none@-",
		}, alertKeys(got.Alerts))
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
		fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{
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
		fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{
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

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
		s := newQANService(t, fake, false)
		session := connect(t, s)
		key := callerKey(callerAuth{authorization: testToken})

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
			_, hit := s.metrics.acquire(fmt.Sprintf("other-%d", i))
			require.Equal(t, limitNone, hit)
		}
		text, isError = callText(t, session, "pmm_alerts", nil)
		require.True(t, isError, text)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Contains(t, text, "this replica already has 8 metric calls")
		assert.Zero(t, fake.requestCount())
	})

	t.Run("CallRate", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
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
			fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{"GET " + path: unauthorized}))
			text, isError := callText(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", nil)
			assert.True(t, isError, path)
			assert.Equal(t, want, text, path)
		}
	})

	t.Run("ServiceListFails", func(t *testing.T) {
		t.Parallel()

		for want, fx := range map[errorCode]fixture{
			codePMMUnavailable: failed,
			codeUnauthorized:   {file: "error_401.json", status: http.StatusUnauthorized},
		} {
			fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{serviceListKey: fx}))
			text, isError := callText(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", nil)
			assert.True(t, isError, text)
			assert.True(t, strings.HasPrefix(text, "error: "+string(want)+"\n"), text)
			assert.Empty(t, fake.requestsTo(rulesPath), "no alert is read without the list")
			assert.Empty(t, fake.requestsTo(amAlertsPath))
			assert.Empty(t, fake.requestsTo(silencesPath))
		}
	})

	t.Run("EmptyResultCarriesTheNote", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{
			"GET " + rulesPath: {body: []byte(`{"status":"success","data":{"groups":[],"totals":{}}}`)},
		}))
		res := callTool(t, connect(t, newQANService(t, fake, false)), "pmm_alerts", nil)
		got := structured[alertsResult](t, res)
		assert.Empty(t, got.Alerts)
		assert.Equal(t, page{Total: 0, Offset: 0, Limit: 50}, got.Page)
		assert.Equal(t, note, got.Note)
		assert.Equal(t, note+"\nNo firing alerts found.", textOf(t, res))
		assert.Contains(t, string(mustJSON(t, res.StructuredContent)), `"alerts":[]`)
		assert.Empty(t, fake.requestsTo(amAlertsPath), "no silence state to read")

		got, text := callAlerts(t, newFakePMM(t, alertsRoutes("alerts_up.json")), map[string]any{"severity": "none"})
		assert.Empty(t, got.Alerts)
		assert.Equal(t, note, got.Note)
		assert.Equal(t, note+"\nNo firing alerts found.", text)
	})

	t.Run("NoSilencesReadWithoutASilencedAlert", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(alertsRoutes("alerts_up.json"), map[string]fixture{
			"GET " + amAlertsPath: {body: []byte("[]")},
		}))
		got, _ := callAlerts(t, fake, nil)
		require.Len(t, got.Alerts, 8)
		for _, row := range got.Alerts {
			assert.False(t, row.Silenced, row.Name)
		}
		assert.Empty(t, fake.requestsTo(silencesPath))
	})

	t.Run("AQuietServiceStaysForSevenDays", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, alertsRoutes("alerts_up.json"))
		got, _ := callAlerts(t, fake, nil)
		keys := alertKeys(got.Alerts)
		assert.Contains(t, keys, "MySQL down@shop-quiet", "the service's last up sample is 3 hours old")
		assert.NotContains(t, keys, "MySQL down@shop-gone", "the service's last up sample is 8 days old")

		lists := queriesWith(fake, "tlast_over_time(up[")
		require.Len(t, lists, 1)
		q := queryOf(t, lists[0])
		assert.Equal(t, fmt.Sprintf(serviceListQuery, int64(alertListLookback/time.Second)), q.Get("query"))
		assert.Contains(t, q.Get("query"), "up[604800s]")
		assert.Equal(t, "1789387200", q.Get("time"))
	})

	t.Run("TwoServicesOnOneNode", func(t *testing.T) {
		t.Parallel()

		got, _ := callAlerts(t, newFakePMM(t, alertsRoutes("alerts_up.json")), nil)
		keys := alertKeys(got.Alerts)
		assert.Contains(t, keys, "Too many connections@shop-c")
		assert.Contains(t, keys, "Node high CPU@host-shared", "the node's own alert")
		assert.NotContains(t, keys, "PostgreSQL down@shop-d", "the other service on the node")
	})

	t.Run("PMMRulesByProvenanceAndCatalogUID", func(t *testing.T) {
		t.Parallel()

		got, _ := callAlerts(t, newFakePMM(t, alertsRoutes("alerts_up.json")), nil)
		require.NotEmpty(t, got.Alerts)
		assert.Equal(t, "PMM Grafana is down@pmm-server", alertKeys(got.Alerts)[0])
		assert.Equal(t, "https://pmm.example.com/graph/alerting/grafana/pmm-grafana-down/view", got.Alerts[0].PMMURL)
		assert.NotContains(t, alertKeys(got.Alerts), "Fake leader@shop-b", "a catalog UID without file provenance")
	})

	t.Run("NegativeAccessRoleFilter", func(t *testing.T) {
		t.Parallel()

		// The list as VictoriaMetrics returns it for {service_name!="shop-b"}: every service but B, and every node_exporter series.
		got, _ := callAlerts(t, newFakePMM(t, alertsRoutes("alerts_up_negative.json")), nil)
		keys := alertKeys(got.Alerts)
		assert.NotContains(t, keys, "MySQL down@shop-b")
		assert.NotContains(t, keys, "Fake leader@shop-b")
		assert.Contains(t, keys, "Node high CPU@host-b", "the node-only alert of B's host")
		assert.Contains(t, keys, "PostgreSQL down@shop-d")
		assert.Contains(t, keys, "MySQL down@shop-a")
	})

	t.Run("ServiceWhoseAgentRunsOnAnotherNode", func(t *testing.T) {
		t.Parallel()

		// The list as VictoriaMetrics returns it for {environment="prod"}; the mysql_down alert names the pmm-agent's node and no environment.
		got, _ := callAlerts(t, newFakePMM(t, alertsRoutes("alerts_up_prod.json")), nil)
		assert.Equal(t, []string{"PMM Grafana is down@pmm-server", "MySQL down@shop-rds", "Vector one@-"}, alertKeys(got.Alerts))
		assert.Equal(t, "pmm-server", got.Alerts[1].Labels["node_name"])
	})

	t.Run("AccessControlOff", func(t *testing.T) {
		t.Parallel()

		got, _ := callAlerts(t, newFakePMM(t, alertsRoutes("alerts_up_all.json")), nil)
		assert.Equal(t, []string{"PMM Grafana is down@pmm-server", "ProxySQL down@proxy-1", "Node high CPU@bare-host", "Vector one@-"}, alertKeys(got.Alerts))
	})

	t.Run("Description", func(t *testing.T) {
		t.Parallel()

		tool := listedTools(t)["pmm_alerts"]
		require.NotNil(t, tool)
		assert.Contains(t, tool.Description, note)
		assert.NotContains(t, strings.ToLower(tool.Description), "restrict")
	})
}

func TestAlertRuleURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://pmm.example.com/graph/alerting/grafana/mysql-down/view", alertRuleURL("https://pmm.example.com/", "mysql-down"))
	assert.Equal(t, "graph/alerting/grafana/mysql-down/view", alertRuleURL("", "mysql-down"))
	assert.Equal(t, "https://pmm.example.com/graph/alerting/list", alertRuleURL("https://pmm.example.com/", ""))
}

func TestAlertFilters(t *testing.T) {
	t.Parallel()

	bare := grafanaAlert{Labels: map[string]string{"alertname": "No labels"}}
	noValue := grafanaAlert{Labels: map[string]string{"alertname": "Server", "service_name": "[no value]", "node_name": "[no value]"}}
	for _, in := range []alertsInput{
		{ServiceID: "svc-a"},
		{ServiceName: "shop-a"},
		{NodeName: "host-a"},
		{Severity: "warning"},
		{ServiceName: "[no value]"},
		{NodeName: "[no value]"},
	} {
		assert.False(t, in.matches(bare), "%+v: an alert without the label", in)
		assert.False(t, in.matches(noValue), "%+v: an alert whose label is [no value]", in)
	}
	assert.True(t, alertsInput{}.matches(bare))

	alert := grafanaAlert{Labels: map[string]string{"service_id": "svc-a", "service_name": "shop-a", "node_name": "host-a", "severity": "Critical"}}
	assert.True(t, alertsInput{ServiceID: "svc-a", ServiceName: "shop-a", NodeName: "host-a", Severity: "CRITICAL"}.matches(alert))
	assert.False(t, alertsInput{ServiceName: "Shop-A"}.matches(alert), "names compare exactly")
	assert.False(t, alertsInput{Severity: "crit"}.matches(alert))
}
