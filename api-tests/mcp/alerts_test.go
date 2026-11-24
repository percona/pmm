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
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	alertingClient "github.com/percona/pmm/api/alerting/v1/json/client"
	alerting "github.com/percona/pmm/api/alerting/v1/json/client/alerting_service"
)

const (
	// The note of every pmm_alerts result, as managed/services/mcp/tools_alerts.go words it.
	alertsNote     = "Firing, recovering and pending alerts only; resolved alerts are not shown. A rule fires only after its pending period, so a problem that began minutes ago may be pending or not listed yet."
	alertsMaxLimit = 200

	// How long TestAlertsUnderAccessRoles waits for its rules to fire and for its silence to mark an alert.
	alertsWait = 3 * time.Minute
)

// firingAlert is an alert of pmm_alerts' structured result.
type firingAlert struct {
	Name          string            `json:"name"`
	Severity      string            `json:"severity"`
	State         string            `json:"state"`
	Labels        map[string]string `json:"labels"`
	StartsAt      time.Time         `json:"starts_at"`
	Silenced      bool              `json:"silenced"`
	SilencedUntil *time.Time        `json:"silenced_until"`
	PMMURL        string            `json:"pmm_url"`
}

type alertsOutput struct {
	Alerts []firingAlert `json:"alerts"`
	Page   struct {
		Total     int  `json:"total"`
		Truncated bool `json:"truncated"`
	} `json:"page"`
	Note string `json:"note"`
}

// ofRule returns the alerts of the rule with the title.
func (o alertsOutput) ofRule(title string) []firingAlert {
	return slices.DeleteFunc(slices.Clone(o.Alerts), func(a firingAlert) bool { return a.Name != title })
}

// labelValues returns the sorted distinct values of a label of the alerts that carry it.
func labelValues(alerts []firingAlert, name string) []string {
	var values []string
	for _, a := range alerts {
		if v := a.Labels[name]; v != "" {
			values = append(values, v)
		}
	}
	return sortedUnique(values)
}

func TestAlerts(t *testing.T) {
	t.Parallel()

	session := connect(t)

	t.Run("Note", func(t *testing.T) {
		t.Parallel()

		var out alertsOutput
		text := callResult(t, session, "pmm_alerts", map[string]any{"limit": alertsMaxLimit}, &out)
		assert.Equal(t, alertsNote, out.Note)
		assert.True(t, strings.HasPrefix(text, alertsNote+"\n"), text)
		assert.Len(t, out.Alerts, min(out.Page.Total, alertsMaxLimit))
		if len(out.Alerts) == 0 {
			t.Log("no alert is firing: the checks of each alert's name, start, pmm_url and labels were skipped")
		}
		for _, a := range out.Alerts {
			assert.NotEmpty(t, a.Name)
			assert.Contains(t, []string{"firing", "recovering", "pending"}, a.State, a.Name)
			assert.False(t, a.StartsAt.IsZero(), a.Name)
			assert.Contains(t, a.PMMURL, "/graph/alerting/", a.Name)
			for name := range a.Labels {
				assert.False(t, strings.HasPrefix(name, "__"), "alert %s carries Grafana's internal label %s", a.Name, name)
			}
		}
	})

	t.Run("FilterMatchingNoAlert", func(t *testing.T) {
		t.Parallel()

		var out alertsOutput
		res := (&caller{session: session}).call(t, "pmm_alerts", map[string]any{"service_name": "mcp-alerts-no-such-service"})
		res.requireOK(t)
		res.decode(t, &out)
		assert.Empty(t, out.Alerts)
		assert.Zero(t, out.Page.Total)
		assert.Equal(t, alertsNote+"\nNo firing, recovering or pending alerts match the service_name filter: "+
			"check its value, or call pmm_alerts without it.", res.text)
	})

	t.Run("LimitOverMaximum", func(t *testing.T) {
		t.Parallel()

		res := (&caller{session: session}).call(t, "pmm_alerts", map[string]any{"limit": alertsMaxLimit + 1})
		res.requireError(t, "invalid_input")
		assert.Contains(t, res.text, fmt.Sprintf("limit must be between 1 and %d; got %d", alertsMaxLimit, alertsMaxLimit+1))
	})
}

// alertRules are the titles of the rules TestAlertsUnderAccessRoles creates.
type alertRules struct {
	// services fires once per MySQL service.
	services string
	// node fires once per node, with a node_name and no service label.
	node string
	// server fires once, with no service or node label.
	server string
	// hidden is as server, in a folder without Viewer access.
	hidden string
}

// createAlertRules creates two folders, one without Viewer access, a template on vector(1) and the rules, and deletes them when the test ends.
func createAlertRules(t *testing.T, suffix int64) alertRules {
	t.Helper()

	template := createServerTemplate(t, suffix)
	visible := createFolder(t, fmt.Sprintf("mcp-visible-%d", suffix), "MCP api-tests, visible")
	hidden := createFolder(t, fmt.Sprintf("mcp-hidden-%d", suffix), "MCP api-tests, no Viewer access")
	requireJSON(t, adminAuthorization(), http.MethodPost, "graph/api/access-control/folders/"+hidden+"/builtInRoles/Viewer", nil, map[string]string{"permission": ""}, nil)

	rules := alertRules{
		services: fmt.Sprintf("MCP alerts services %d", suffix),
		node:     fmt.Sprintf("MCP alerts node %d", suffix),
		server:   fmt.Sprintf("MCP alerts server %d", suffix),
		hidden:   fmt.Sprintf("MCP alerts hidden %d", suffix),
	}
	// Threshold 0 makes the templates fire for every series they read.
	createRule(t, visible, rules.services, "pmm_mysql_too_many_connections", true)
	createRule(t, visible, rules.node, "pmm_node_high_cpu_load", true)
	createRule(t, visible, rules.server, template, false)
	createRule(t, hidden, rules.hidden, template, false)
	return rules
}

// matrixAlerts are the titles of the rules TestAccessRoleMatrix creates.
type matrixAlerts struct {
	// services fires once per MySQL service, mysql-b's included.
	services string
	// server fires once, with no service or node label.
	server string
}

// createMatrixAlerts creates a folder, a template on vector(1) and the rules, deletes them when the test ends, and waits for the rules
// to fire for mysql-a, mysql-b and the server.
func createMatrixAlerts(t *testing.T, suffix int64) matrixAlerts {
	t.Helper()

	template := createServerTemplate(t, suffix)
	folder := createFolder(t, fmt.Sprintf("mcp-matrix-%d", suffix), "MCP api-tests, Access Role matrix")
	rules := matrixAlerts{
		services: fmt.Sprintf("MCP matrix services %d", suffix),
		server:   fmt.Sprintf("MCP matrix server %d", suffix),
	}
	// Threshold 0 makes the template fire for every MySQL service.
	createRule(t, folder, rules.services, "pmm_mysql_too_many_connections", true)
	createRule(t, folder, rules.server, template, false)
	waitFor(t, "the matrix's rules to fire", func() bool {
		current := firingRules(t)
		return slices.Equal(intersection(current[rules.services].values("service_name"), "mysql-a", "mysql-b"), []string{"mysql-a", "mysql-b"}) &&
			len(current[rules.server].alerts) == 1
	})
	return rules
}

// createServerTemplate creates a template on vector(1), whose alert carries no service or node label, and deletes it when the test ends.
func createServerTemplate(t *testing.T, suffix int64) string {
	t.Helper()

	template := fmt.Sprintf("mcp_alerts_server_level_%d", suffix)
	_, err := alertingClient.Default.AlertingService.CreateTemplate(&alerting.CreateTemplateParams{
		Body: alerting.CreateTemplateBody{Yaml: fmt.Sprintf(`---
templates:
  - name: %s
    version: 1
    summary: MCP api-tests, no service or node label
    expr: vector(1)
    for: 10s
    severity: info
    annotations:
      summary: Server-level test alert
      description: A rule on vector(1); its alert carries no service or node label.
`, template)},
		Context: t.Context(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := alertingClient.Default.AlertingService.DeleteTemplate(&alerting.DeleteTemplateParams{Name: template, Context: pmmapitests.Context})
		if err != nil {
			t.Errorf("failed to delete alert template %s: %s", template, err)
		}
	})
	return template
}

// createFolder creates a Grafana folder and deletes it with its rules when the test ends.
func createFolder(t *testing.T, uid, title string) string {
	t.Helper()

	requireJSON(t, adminAuthorization(), http.MethodPost, "graph/api/folders", nil, map[string]string{"uid": uid, "title": title}, nil)
	t.Cleanup(func() {
		code, body, err := request(pmmapitests.Context, http.MethodDelete, "graph/api/folders/"+uid, url.Values{"forceDeleteRules": {"true"}},
			http.Header{"Authorization": {adminAuthorization()}}, nil)
		require.NoError(t, err)
		if code != http.StatusOK {
			t.Errorf("failed to delete folder %s: %d %s", uid, code, body)
		}
	})
	return uid
}

// createRule creates a rule from a template in group mcp-alerts of the folder, evaluated every 10 s; zeroThreshold sets its threshold to 0.
func createRule(t *testing.T, folderUID, title, template string, zeroThreshold bool) {
	t.Helper()

	body := alerting.CreateRuleBody{
		TemplateName: template,
		Name:         title,
		Group:        "mcp-alerts",
		FolderUID:    folderUID,
		For:          "10s",
		Interval:     "10s",
		Severity:     new("SEVERITY_WARNING"),
	}
	if zeroThreshold {
		body.Params = []*alerting.CreateRuleParamsBodyParamsItems0{{Name: "threshold", Type: new("PARAM_TYPE_FLOAT"), Float: 0}}
	}
	_, err := alertingClient.Default.AlertingService.CreateRule(&alerting.CreateRuleParams{Body: body, Context: t.Context()})
	require.NoError(t, err)
}

// firingRule is a rule of Grafana's rules API with the labels of its firing alerts.
type firingRule struct {
	uid    string
	alerts []map[string]string
}

// values returns the sorted distinct values of a label of the rule's firing alerts.
func (r firingRule) values(name string) []string {
	var values []string
	for _, labels := range r.alerts {
		if v := labels[name]; v != "" {
			values = append(values, v)
		}
	}
	return sortedUnique(values)
}

// firingRules returns the rules that Grafana's rules API returns to the package's admin, by title.
func firingRules(t *testing.T) map[string]firingRule {
	t.Helper()

	var resp struct {
		Data struct {
			Groups []struct {
				Rules []struct {
					UID    string `json:"uid"`
					Name   string `json:"name"`
					Alerts []struct {
						Labels map[string]string `json:"labels"`
						State  string            `json:"state"`
					} `json:"alerts"`
				} `json:"rules"`
			} `json:"groups"`
		} `json:"data"`
	}
	requireJSON(t, adminAuthorization(), http.MethodGet, "graph/api/prometheus/grafana/api/v1/rules", nil, nil, &resp)
	rules := map[string]firingRule{}
	for _, group := range resp.Data.Groups {
		for _, rule := range group.Rules {
			r := firingRule{uid: rule.UID}
			for _, a := range rule.Alerts {
				if strings.HasPrefix(a.State, "Alerting") {
					r.alerts = append(r.alerts, a.Labels)
				}
			}
			rules[rule.Name] = r
		}
	}
	return rules
}

// waitFor polls cond every 5 s until it holds, and fails the test after alertsWait.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(alertsWait)
	for !cond() {
		if time.Now().After(deadline) {
			require.FailNow(t, "timed out waiting for "+what)
		}
		select {
		case <-pmmapitests.Context.Done():
			require.FailNow(t, "canceled while waiting for "+what)
		case <-time.After(5 * time.Second):
		}
	}
}

// silenceAlert silences the alert of the rule with the service_name for an hour, expires the silence when the test ends, and returns its end.
func silenceAlert(t *testing.T, ruleUID, serviceName string) time.Time {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	end := now.Add(time.Hour)
	var created struct {
		SilenceID string `json:"silenceID"`
	}
	requireJSON(t, adminAuthorization(), http.MethodPost, "graph/api/alertmanager/grafana/api/v2/silences", nil, map[string]any{
		"matchers": []map[string]any{
			{"name": "__alert_rule_uid__", "value": ruleUID, "isRegex": false, "isEqual": true},
			{"name": "service_name", "value": serviceName, "isRegex": false, "isEqual": true},
		},
		"startsAt":  now.Format(time.RFC3339),
		"endsAt":    end.Format(time.RFC3339),
		"createdBy": "pmm-api-tests",
		"comment":   "PMM-15790 MCP api-tests",
	}, &created)
	require.NotEmpty(t, created.SilenceID)
	t.Cleanup(func() {
		code, body, err := request(pmmapitests.Context, http.MethodDelete, "graph/api/alertmanager/grafana/api/v2/silence/"+created.SilenceID, nil,
			http.Header{"Authorization": {adminAuthorization()}}, nil)
		require.NoError(t, err)
		if code != http.StatusOK {
			t.Errorf("failed to expire silence %s: %d %s", created.SilenceID, code, body)
		}
	})
	return end
}

// restrictedViewer creates a Viewer with one Access Role of the filter and returns its Basic Authorization value.
func restrictedViewer(t *testing.T, login, filter string) string {
	t.Helper()

	userID := pmmapitests.CreateUser(t, login, matrixPassword, "Viewer")
	pmmapitests.AssignRoles(t, userID, pmmapitests.CreateRole(t, login, filter))
	return basicAuthorization(login, matrixPassword)
}

// alertsOf calls pmm_alerts with the largest page and returns its result.
func alertsOf(t *testing.T, c *caller) alertsOutput {
	t.Helper()

	var out alertsOutput
	res := c.call(t, "pmm_alerts", map[string]any{"limit": alertsMaxLimit})
	res.requireOK(t)
	res.decode(t, &out)
	require.False(t, out.Page.Truncated, "the alerts do not fit on one page")
	assert.Equal(t, alertsNote, out.Note)
	return out
}

// waitForSilence calls pmm_alerts as c until the alert of the rule for the service is silenced, and returns that result.
func waitForSilence(t *testing.T, c *caller, rule, serviceName string) alertsOutput {
	t.Helper()

	var out alertsOutput
	waitFor(t, "the silence to mark "+serviceName+"'s alert", func() bool {
		out = alertsOf(t, c)
		return slices.ContainsFunc(out.ofRule(rule), func(alert firingAlert) bool {
			return alert.Labels["service_name"] == serviceName && alert.Silenced
		})
	})
	return out
}

// TestAlertsUnderAccessRoles creates rules that fire for mysql-a and mysql-b, for every node and for no service or node, one in a
// folder without Viewer access, and a silence on mysql-a's alert, and checks that pmm_alerts returns the alerts of every folder
// the caller can read, whatever its Access Roles.
func TestAlertsUnderAccessRoles(t *testing.T) {
	if !pmmapitests.RunMCPAccessRoleTests {
		t.Skip("Needs TestAccessRoleMatrix's environment, of which it uses mysql-a and mysql-b, each on its own node. It creates " +
			"alert rules, folders, a silence, Access Roles and users, and turns Access Control on, so it changes server-wide " +
			"settings. Set -pmm.run-mcp-access-role-tests [PMM_RUN_MCP_ACCESS_ROLE_TESTS].")
	}

	pmmapitests.SetAccessControl(t, true)

	admin := &caller{session: connectWith(t, adminAuthorization())}
	var inventory inventoryOutput
	res := admin.call(t, "pmm_inventory", map[string]any{"limit": 200})
	res.requireOK(t)
	res.decode(t, &inventory)
	services := map[string]inventoryRow{}
	all := make([]string, 0, len(inventory.Services))
	var mysql []string
	for _, row := range inventory.Services {
		services[row.ServiceName] = row
		all = append(all, row.ServiceName)
		if row.Engine == "mysql" {
			mysql = append(mysql, row.ServiceName)
		}
	}
	mysql = sortedUnique(mysql)
	a, b := services["mysql-a"], services["mysql-b"]
	require.NotEmpty(t, a.ServiceID, "pmm_inventory of the package's admin lacks mysql-a")
	require.NotEmpty(t, b.ServiceID, "pmm_inventory of the package's admin lacks mysql-b")
	require.NotEqual(t, a.NodeName, b.NodeName, "mysql-a and mysql-b must run on different nodes")

	suffix := time.Now().UnixNano()
	rules := createAlertRules(t, suffix)

	var current map[string]firingRule
	waitFor(t, "the rules to fire", func() bool {
		current = firingRules(t)
		return slices.Equal(intersection(current[rules.services].values("service_name"), mysql...), mysql) &&
			len(intersection(current[rules.node].values("node_name"), a.NodeName, b.NodeName)) == 2 &&
			len(current[rules.server].alerts) == 1 &&
			len(current[rules.hidden].alerts) == 1
	})
	silenceEnd := silenceAlert(t, current[rules.services].uid, "mysql-a")

	// Every node whose series the rule reads.
	allNodes := current[rules.node].values("node_name")

	t.Run("Admin", func(t *testing.T) {
		out := waitForSilence(t, admin, rules.services, "mysql-a")

		ofServices := out.ofRule(rules.services)
		assert.Equal(t, []string{"mysql-a", "mysql-b"}, intersection(labelValues(ofServices, "service_name"), "mysql-a", "mysql-b"))
		for _, alert := range ofServices {
			switch alert.Labels["service_name"] {
			case "mysql-a":
				assert.True(t, alert.Silenced)
				require.NotNil(t, alert.SilencedUntil)
				assert.WithinDuration(t, silenceEnd, *alert.SilencedUntil, time.Second)
			case "mysql-b":
				assert.False(t, alert.Silenced, "the silence matches mysql-a's alert of the rule only")
				assert.Nil(t, alert.SilencedUntil)
			}
		}
		assert.Equal(t, allNodes, labelValues(out.ofRule(rules.node), "node_name"))
		assert.Len(t, out.ofRule(rules.server), 1)
		assert.Len(t, out.ofRule(rules.hidden), 1, "the package's admin reads every folder")
	})

	for _, tc := range []struct {
		login, filter string
		list          []string
	}{
		{login: "mcp-alerts-a", filter: `{service_name="mysql-a"}`, list: []string{"mysql-a"}},
		{login: "mcp-alerts-not-b", filter: `{service_name!="mysql-b"}`, list: without(all, "mysql-b")},
	} {
		t.Run(tc.login, func(t *testing.T) {
			authorization := restrictedViewer(t, fmt.Sprintf("%s-%d", tc.login, suffix), tc.filter)
			// The Access Role hides mysql-b from the other tools; pmm_alerts shows its alerts all the same.
			waitForServiceList(t, authorization, tc.list)
			viewer := &caller{session: connectWith(t, authorization)}

			out := waitForSilence(t, viewer, rules.services, "mysql-a")
			ofServices := out.ofRule(rules.services)
			assert.Equal(t, mysql, intersection(labelValues(ofServices, "service_name"), mysql...), "every MySQL service, mysql-b included")
			for _, alert := range ofServices {
				assert.Equal(t, alert.Labels["service_name"] == "mysql-a", alert.Silenced, "alert of %s", alert.Labels["service_name"])
			}
			assert.Equal(t, allNodes, labelValues(out.ofRule(rules.node), "node_name"), "every node")
			assert.Len(t, out.ofRule(rules.server), 1, "an alert that names no service or node is shown")
			assert.Empty(t, out.ofRule(rules.hidden), "the rule's folder gives Viewers no access")

			var filtered alertsOutput
			res := viewer.call(t, "pmm_alerts", map[string]any{"service_name": "mysql-b"})
			res.requireOK(t)
			res.decode(t, &filtered)
			ofB := filtered.ofRule(rules.services)
			require.Len(t, ofB, 1, "the alert of mysql-b")
			assert.Equal(t, b.ServiceID, ofB[0].Labels["service_id"])
		})
	}
}
