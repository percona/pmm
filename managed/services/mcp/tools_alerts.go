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
	"cmp"
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/percona/pmm/managed/services/alerting"
)

const (
	// The look-back of the service list that pmm_alerts keeps alerts of (ADR-17).
	alertListLookback = 7 * day

	alertsDefaultLimit = 50
	alertsMaxLimit     = 200

	// The note of every pmm_alerts result, the first line of its text.
	alertsNote = "Currently firing alerts only; resolved alerts are not shown."

	// What Grafana writes for a label that a template rule reads from a series that lacks it; it counts as absent.
	noValue = "[no value]"

	// The Alertmanager label that carries an alert's rule UID.
	ruleUIDLabel = "__alert_rule_uid__"
)

// pmmRuleUIDs are the UIDs of PMM's rule catalog.
var pmmRuleUIDs = alerting.CatalogUIDs()

// severityOrder lists the severities that come ahead of every other one, in order.
var severityOrder = []string{"critical", "error", "warning"}

// isPMMRule reports whether a rule is one PMM provisions: file provenance and a UID of PMM's catalog, never a label.
func isPMMRule(provenance, uid string) bool {
	return provenance == "file" && slices.Contains(pmmRuleUIDs, uid)
}

// labelValue returns a label's value, empty when the label is absent or [no value].
func labelValue(labels map[string]string, name string) string {
	v := labels[name]
	if v == noValue {
		return ""
	}
	return v
}

// serviceSet is the caller's service list as the four sets that alert narrowing checks labels against.
type serviceSet struct {
	serviceIDs, serviceNames, nodeIDs, nodeNames map[string]bool
}

// newServiceSet builds the sets from the services and nodes of listServicesAndNodes.
func newServiceSet(services []serviceInfo, nodes []nodeInfo) serviceSet {
	l := serviceSet{serviceIDs: map[string]bool{}, serviceNames: map[string]bool{}, nodeIDs: map[string]bool{}, nodeNames: map[string]bool{}}
	add := func(set map[string]bool, v string) {
		if v != "" {
			set[v] = true
		}
	}
	for _, svc := range services {
		add(l.serviceIDs, svc.ServiceID)
		add(l.serviceNames, svc.ServiceName)
	}
	for _, node := range nodes {
		add(l.nodeIDs, node.NodeID)
		add(l.nodeNames, node.NodeName)
	}
	return l
}

// keepAlert applies ADR-17's narrowing: PMM's own rules, else each service label in its set, else each node label in its set, else kept.
func (l serviceSet) keepAlert(a grafanaAlert) bool {
	if a.PMMRule {
		return true
	}
	serviceID, serviceName := labelValue(a.Labels, "service_id"), labelValue(a.Labels, "service_name")
	if serviceID != "" || serviceName != "" {
		return inSet(l.serviceIDs, serviceID) && inSet(l.serviceNames, serviceName)
	}
	nodeID, nodeName := labelValue(a.Labels, "node_id"), labelValue(a.Labels, "node_name")
	if nodeID != "" || nodeName != "" {
		return inSet(l.nodeIDs, nodeID) && inSet(l.nodeNames, nodeName)
	}
	return true
}

// covers reports whether each service and node filter of the call names a service or node of the list.
func (l serviceSet) covers(in alertsInput) bool {
	return inSet(l.serviceIDs, in.ServiceID) && inSet(l.serviceNames, in.ServiceName) && inSet(l.nodeNames, in.NodeName)
}

// inSet reports whether v is empty, an absent label or filter, or in the set.
func inSet(set map[string]bool, v string) bool {
	return v == "" || set[v]
}

type alertsInput struct {
	ServiceID   string `json:"service_id,omitempty" jsonschema:"Keep the alerts whose service_id label is this service id"`
	ServiceName string `json:"service_name,omitempty" jsonschema:"Keep the alerts whose service_name label is this service name"`
	NodeName    string `json:"node_name,omitempty" jsonschema:"Keep the alerts whose node_name label is this node name"`
	Severity    string `json:"severity,omitempty" jsonschema:"Keep the alerts of this severity, such as critical, error or warning; case-insensitive"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum alerts to return; default 50, maximum 200"`
	Offset      int    `json:"offset,omitempty" jsonschema:"Number of alerts to skip, for the next page"`
}

// matches reports whether an alert passes the call's filters; an alert without a filtered label never does.
func (in alertsInput) matches(a grafanaAlert) bool {
	return (in.ServiceID == "" || labelValue(a.Labels, "service_id") == in.ServiceID) &&
		(in.ServiceName == "" || labelValue(a.Labels, "service_name") == in.ServiceName) &&
		(in.NodeName == "" || labelValue(a.Labels, "node_name") == in.NodeName) &&
		(in.Severity == "" || strings.EqualFold(a.Labels["severity"], in.Severity))
}

// alertRow is one alert in the result of pmm_alerts.
type alertRow struct {
	Name          string            `json:"name" jsonschema:"Untrusted. The alert rule's title."`
	Severity      string            `json:"severity,omitempty" jsonschema:"Untrusted. The alert's severity label, left out when it has none."`
	Labels        map[string]string `json:"labels,omitempty" jsonschema:"Untrusted. The alert's labels, without Grafana's internal labels that start with __."`
	Summary       string            `json:"summary,omitempty" jsonschema:"Untrusted. The alert's summary annotation."`
	Description   string            `json:"description,omitempty" jsonschema:"Untrusted. The alert's description annotation."`
	StartsAt      time.Time         `json:"starts_at" jsonschema:"When the alert started firing."`
	Silenced      bool              `json:"silenced" jsonschema:"True when a silence mutes the alert's notifications."`
	SilencedUntil *time.Time        `json:"silenced_until,omitempty" jsonschema:"When the latest of the silences that mute the alert ends."`
	PMMURL        string            `json:"pmm_url" jsonschema:"Link to the alert's rule in PMM."`
}

// alertsResult is the result of pmm_alerts.
type alertsResult struct {
	Alerts []alertRow `json:"alerts" jsonschema:"Firing alerts on this page."`
	Page   page       `json:"page" jsonschema:"Paging of the full list."`
	Note   string     `json:"note" jsonschema:"What the list covers."`
}

func (r alertsResult) render() string {
	lines := []string{r.Note}
	if r.Page.Total == 0 {
		return strings.Join(append(lines, "No firing alerts found."), "\n")
	}
	if len(r.Alerts) == 0 {
		return strings.Join(append(lines, r.Page.pastEnd("alert list")), "\n")
	}
	for _, a := range r.Alerts {
		var parts []string
		if a.Severity != "" {
			parts = append(parts, echo(a.Severity))
		}
		parts = append(parts, echo(a.Name))
		if svc := cmp.Or(labelValue(a.Labels, "service_name"), labelValue(a.Labels, "service_id")); svc != "" {
			parts = append(parts, "service="+echo(svc))
		}
		if node := cmp.Or(labelValue(a.Labels, "node_name"), labelValue(a.Labels, "node_id")); node != "" {
			parts = append(parts, "node="+echo(node))
		}
		parts = append(parts, "since "+a.StartsAt.UTC().Format(time.RFC3339))
		switch {
		case a.SilencedUntil != nil:
			parts = append(parts, "silenced until "+a.SilencedUntil.UTC().Format(time.RFC3339))
		case a.Silenced:
			parts = append(parts, "silenced")
		}
		parts = append(parts, link("View in PMM", a.PMMURL))
		lines = append(lines, "- "+strings.Join(parts, " | "))
		if summary := strings.Join(strings.Fields(a.Summary), " "); summary != "" {
			lines = append(lines, "  "+echo(summary))
		}
	}
	if m := r.Page.marker(); m != "" {
		lines = append(lines, m)
	}
	return strings.Join(lines, "\n")
}

func (s *Service) registerAlertsTool(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_alerts",
		Title: "Firing alerts",
		Description: "List the alerts firing now in PMM's alerting, from the alert rules your Grafana folder permissions let you read. " +
			alertsNote + " The list holds the alerts of the services and nodes in your service list (those whose up series you can " +
			"read and that sent metrics in the last 7 days), the alerts that name no service or node, and PMM Server's own alerts. " +
			"Each alert has its severity, labels, summary, description, start time, whether a silence mutes it and until when, and " +
			"a link to its rule in PMM; the most severe and newest come first. Filter by service_id, service_name, node_name or " +
			"severity. Read only: it does not silence or acknowledge alerts.",
		Annotations: readOnly("Firing alerts"),
	}, s.alerts)
}

func (s *Service) alerts(ctx context.Context, req *mcp.CallToolRequest, in alertsInput) (alertsResult, error) {
	err := checkPage(in.Offset, in.Limit, alertsMaxLimit)
	if err != nil {
		return alertsResult{}, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)
	release, err := s.startMetricCall(auth)
	if err != nil {
		return alertsResult{}, err
	}
	defer release()

	now := s.now()
	w := window{From: now.Add(-alertListLookback), To: now}
	services, nodes, err := s.listServicesAndNodes(ctx, auth, w)
	if err != nil {
		return alertsResult{}, err
	}
	list := newServiceSet(services, nodes)
	if !list.covers(in) {
		return alertsResult{}, errNotInList(w, now, "")
	}

	all, err := s.api.Alerts(ctx, auth)
	if err != nil {
		return alertsResult{}, err
	}
	kept := slices.DeleteFunc(all, func(a grafanaAlert) bool { return !list.keepAlert(a) || !in.matches(a) })
	slices.SortFunc(kept, compareAlerts)

	paged, pg, err := paginate(kept, in.Offset, in.Limit, alertsDefaultLimit, alertsMaxLimit)
	if err != nil {
		return alertsResult{}, err
	}
	if len(paged) > 0 {
		err = s.joinSilences(ctx, auth, paged)
		if err != nil {
			return alertsResult{}, err
		}
	}

	base := s.publicBaseURL(ctx, req.Extra.Header)
	rows := make([]alertRow, 0, len(paged))
	for _, a := range paged {
		labels := maps.Clone(a.Labels)
		maps.DeleteFunc(labels, func(name, _ string) bool { return strings.HasPrefix(name, "__") })
		if len(labels) == 0 {
			labels = nil
		}
		rows = append(rows, alertRow{
			Name:          a.Labels["alertname"],
			Severity:      a.Labels["severity"],
			Labels:        labels,
			Summary:       a.Annotations["summary"],
			Description:   a.Annotations["description"],
			StartsAt:      a.StartsAt,
			Silenced:      a.Silenced,
			SilencedUntil: a.SilencedUntil,
			PMMURL:        alertRuleURL(base, a.RuleUID),
		})
	}
	return alertsResult{Alerts: rows, Page: pg, Note: alertsNote}, nil
}

// joinSilences marks the alerts whose Alertmanager alert, the one with their labels plus rule UID, is silenced, until the latest silence end.
func (s *Service) joinSilences(ctx context.Context, auth callerAuth, alerts []grafanaAlert) error {
	amAlerts, err := s.api.SilencedAlerts(ctx, auth)
	if err != nil {
		return err
	}
	silencedBy := make(map[string][]string, len(amAlerts))
	for _, a := range amAlerts {
		if len(a.Status.SilencedBy) > 0 {
			silencedBy[labelSetKey(a.Labels)] = a.Status.SilencedBy
		}
	}

	ids := make([][]string, len(alerts))
	found := false
	for i, a := range alerts {
		labels := maps.Clone(a.Labels)
		if labels == nil {
			labels = map[string]string{}
		}
		labels[ruleUIDLabel] = a.RuleUID
		ids[i] = silencedBy[labelSetKey(labels)]
		found = found || len(ids[i]) > 0
	}
	if !found {
		return nil
	}

	silences, err := s.api.Silences(ctx, auth)
	if err != nil {
		return err
	}
	ends := make(map[string]time.Time, len(silences))
	for _, silence := range silences {
		ends[silence.ID] = silence.EndsAt
	}
	for i := range alerts {
		if len(ids[i]) == 0 {
			continue
		}
		alerts[i].Silenced = true
		var until time.Time
		for _, id := range ids[i] {
			end, ok := ends[id]
			if ok && end.After(until) {
				until = end
			}
		}
		if !until.IsZero() {
			until = until.UTC()
			alerts[i].SilencedUntil = &until
		}
	}
	return nil
}

// labelSetKey writes a label set as one string, equal for two sets only when the sets are equal.
func labelSetKey(labels map[string]string) string {
	var b strings.Builder
	for _, name := range slices.Sorted(maps.Keys(labels)) {
		b.WriteString(strconv.Quote(name))
		b.WriteByte('=')
		b.WriteString(strconv.Quote(labels[name]))
		b.WriteByte(',')
	}
	return b.String()
}

// severityRank ranks critical, error and warning first, then every other severity, then none.
func severityRank(severity string) int {
	rank := slices.Index(severityOrder, severity)
	switch {
	case rank >= 0:
		return rank
	case severity == "":
		return len(severityOrder) + 1
	default:
		return len(severityOrder)
	}
}

// compareAlerts orders alerts by severity, case-insensitively and other severities alphabetically, then newest first, then by name.
func compareAlerts(a, b grafanaAlert) int {
	sa, sb := strings.ToLower(a.Labels["severity"]), strings.ToLower(b.Labels["severity"])
	c := cmp.Or(
		cmp.Compare(severityRank(sa), severityRank(sb)),
		strings.Compare(sa, sb),
		b.StartsAt.Compare(a.StartsAt),
		strings.Compare(a.Labels["alertname"], b.Labels["alertname"]),
		strings.Compare(a.RuleUID, b.RuleUID),
	)
	if c != 0 {
		return c
	}
	return strings.Compare(labelSetKey(a.Labels), labelSetKey(b.Labels))
}
