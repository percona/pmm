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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
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
	alertsNote     = "Currently firing alerts only; resolved alerts are not shown."
	alertsMaxLimit = 200
	// The look-back of pmm_alerts' service list.
	alertListLookback = 7 * 24 * time.Hour

	// How long TestAlertsNarrowing waits for its rules to fire and for its silence to mark an alert.
	alertsWait = 3 * time.Minute
)

// firingAlert is an alert of pmm_alerts' structured result.
type firingAlert struct {
	Name          string            `json:"name"`
	Severity      string            `json:"severity"`
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
		for _, a := range out.Alerts {
			assert.NotEmpty(t, a.Name)
			assert.False(t, a.StartsAt.IsZero(), a.Name)
			assert.Contains(t, a.PMMURL, "/graph/alerting/", a.Name)
			for name := range a.Labels {
				assert.False(t, strings.HasPrefix(name, "__"), "alert %s carries Grafana's internal label %s", a.Name, name)
			}
		}
	})

	t.Run("ServiceOutsideList", func(t *testing.T) {
		t.Parallel()

		const name = "mcp-alerts-no-such-service"
		res := (&caller{session: session}).call(t, "pmm_alerts", map[string]any{"service_name": name})
		res.requireNotInList(t, "", alertListLookback)
		assert.NotContains(t, res.text, name)
	})

	t.Run("LimitOverMaximum", func(t *testing.T) {
		t.Parallel()

		res := (&caller{session: session}).call(t, "pmm_alerts", map[string]any{"limit": alertsMaxLimit + 1})
		res.requireError(t, "invalid_input")
		assert.Contains(t, res.text, fmt.Sprintf("limit must be between 1 and %d; got %d", alertsMaxLimit, alertsMaxLimit+1))
	})
}

// adminRequest sends a JSON request to the server under test as the package's admin and returns the status and body.
func adminRequest(t *testing.T, method, path string, query url.Values, payload any) (int, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(b)
	}
	u := pmmapitests.BaseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	u.User = nil
	req, err := http.NewRequestWithContext(pmmapitests.Context, method, u.String(), body)
	require.NoError(t, err)
	req.Header.Set("Authorization", adminAuthorization())
	req.Header.Set("Content-Type", "application/json")
	res, err := httpClient(false).Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, b
}

// requireAdminRequest is adminRequest that fails the test unless the status is 2xx (Grafana answers 202 to a new silence), and
// decodes the body into out when out is not nil.
func requireAdminRequest(t *testing.T, method, path string, payload, out any) {
	t.Helper()

	code, body := adminRequest(t, method, path, nil, payload)
	require.True(t, code >= 200 && code < 300, "%s %s: %d %s", method, path, code, body)
	if out != nil {
		require.NoError(t, json.Unmarshal(body, out))
	}
}

// alertRules are the titles of the rules TestAlertsNarrowing creates.
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

	visible := createFolder(t, fmt.Sprintf("mcp-visible-%d", suffix), "MCP api-tests, visible")
	hidden := createFolder(t, fmt.Sprintf("mcp-hidden-%d", suffix), "MCP api-tests, no Viewer access")
	requireAdminRequest(t, http.MethodPost, "graph/api/access-control/folders/"+hidden+"/builtInRoles/Viewer", map[string]string{"permission": ""}, nil)

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

// createFolder creates a Grafana folder and deletes it with its rules when the test ends.
func createFolder(t *testing.T, uid, title string) string {
	t.Helper()

	requireAdminRequest(t, http.MethodPost, "graph/api/folders", map[string]string{"uid": uid, "title": title}, nil)
	t.Cleanup(func() {
		code, body := adminRequest(t, http.MethodDelete, "graph/api/folders/"+uid, url.Values{"forceDeleteRules": {"true"}}, nil)
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
	getJSON(t, adminAuthorization(), "graph/api/prometheus/grafana/api/v1/rules", nil, &resp)
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
	requireAdminRequest(t, http.MethodPost, "graph/api/alertmanager/grafana/api/v2/silences", map[string]any{
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
		code, body := adminRequest(t, http.MethodDelete, "graph/api/alertmanager/grafana/api/v2/silence/"+created.SilenceID, nil, nil)
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

// hiding returns a caller whose every response is checked for the service's name and IDs.
func hiding(t *testing.T, authorization string, service inventoryRow) *caller {
	t.Helper()

	return &caller{
		session:   connectWith(t, authorization),
		hiddenRE:  regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])` + regexp.QuoteMeta(service.ServiceName) + `(?:$|[^A-Za-z0-9_-])`),
		hiddenIDs: []string{service.ServiceID},
	}
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

// TestAlertsNarrowing creates rules that fire for mysql-a and mysql-b, for every node and for no service or node, one in a
// folder without Viewer access, and a silence on mysql-a's alert, and checks what pmm_alerts returns under Access Roles.
func TestAlertsNarrowing(t *testing.T) {
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
	for _, row := range inventory.Services {
		services[row.ServiceName] = row
		all = append(all, row.ServiceName)
	}
	a, b := services["mysql-a"], services["mysql-b"]
	require.NotEmpty(t, a.ServiceID, "pmm_inventory of the package's admin lacks mysql-a")
	require.NotEmpty(t, b.ServiceID, "pmm_inventory of the package's admin lacks mysql-b")
	require.NotEqual(t, a.NodeName, b.NodeName, "mysql-a and mysql-b must run on different nodes")

	suffix := time.Now().UnixNano()
	rules := createAlertRules(t, suffix)

	var current map[string]firingRule
	waitFor(t, "the rules to fire", func() bool {
		current = firingRules(t)
		return slices.Equal(intersection(current[rules.services].values("service_name"), "mysql-a", "mysql-b"), []string{"mysql-a", "mysql-b"}) &&
			len(intersection(current[rules.node].values("node_name"), a.NodeName, b.NodeName)) == 2 &&
			len(current[rules.server].alerts) == 1 &&
			len(current[rules.hidden].alerts) == 1
	})
	silenceEnd := silenceAlert(t, current[rules.services].uid, "mysql-a")

	// Every node whose series the rule reads; under {service_name!="mysql-b"} the caller's list holds them all.
	allNodes := current[rules.node].values("node_name")

	t.Run("Admin", func(t *testing.T) {
		var out alertsOutput
		waitFor(t, "the silence to mark mysql-a's alert", func() bool {
			out = alertsOf(t, admin)
			i := slices.IndexFunc(out.ofRule(rules.services), func(alert firingAlert) bool { return alert.Labels["service_name"] == "mysql-a" })
			return i >= 0 && out.ofRule(rules.services)[i].Silenced
		})

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

	t.Run("ViewerOfA", func(t *testing.T) {
		authorization := restrictedViewer(t, fmt.Sprintf("mcp-alerts-a-%d", suffix), `{service_name="mysql-a"}`)
		waitForServiceList(t, authorization, []string{"mysql-a"})
		viewer := hiding(t, authorization, b)

		out := alertsOf(t, viewer)
		ofServices := out.ofRule(rules.services)
		require.Len(t, ofServices, 1)
		assert.Equal(t, "mysql-a", ofServices[0].Labels["service_name"])
		assert.Equal(t, a.ServiceID, ofServices[0].Labels["service_id"])
		assert.True(t, ofServices[0].Silenced)
		assert.NotNil(t, ofServices[0].SilencedUntil)
		assert.Equal(t, []string{a.NodeName}, labelValues(out.ofRule(rules.node), "node_name"), "only the node of mysql-a is in the list")
		assert.Len(t, out.ofRule(rules.server), 1, "an alert that names no service or node is shown")
		assert.Empty(t, out.ofRule(rules.hidden), "the rule's folder gives Viewers no access")
		for _, alert := range out.Alerts {
			assert.NotEqual(t, b.NodeName, alert.Labels["node_name"], "alert %s", alert.Name)
		}

		res := viewer.call(t, "pmm_alerts", map[string]any{"service_name": "mysql-b"})
		res.requireNotInList(t, "", alertListLookback)
		res = viewer.call(t, "pmm_alerts", map[string]any{"node_name": b.NodeName})
		res.requireNotInList(t, "", alertListLookback)
		assert.NotContains(t, res.text, b.NodeName)
	})

	t.Run("ViewerNotB", func(t *testing.T) {
		authorization := restrictedViewer(t, fmt.Sprintf("mcp-alerts-not-b-%d", suffix), `{service_name!="mysql-b"}`)
		waitForServiceList(t, authorization, without(all, "mysql-b"))
		viewer := hiding(t, authorization, b)

		out := alertsOf(t, viewer)
		assert.Equal(t, []string{"mysql-a"}, labelValues(out.ofRule(rules.services), "service_name"))
		assert.Equal(t, allNodes, labelValues(out.ofRule(rules.node), "node_name"), "node_exporter's series carry no service_name, so every node is in the list")
		assert.Contains(t, labelValues(out.ofRule(rules.node), "node_name"), b.NodeName)
		assert.Len(t, out.ofRule(rules.server), 1)
		assert.Empty(t, out.ofRule(rules.hidden), "the rule's folder gives Viewers no access")

		res := viewer.call(t, "pmm_alerts", map[string]any{"service_name": "mysql-b"})
		res.requireNotInList(t, "", alertListLookback)
	})
}
