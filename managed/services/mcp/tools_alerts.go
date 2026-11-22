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
)

const (
	alertsDefaultLimit = 50
	alertsMaxLimit     = 200

	// The note of every pmm_alerts result, the first line of its text.
	alertsNote = "Firing, recovering and pending alerts only; resolved alerts are not shown. A rule fires only after its pending period, " +
		"so a problem that began minutes ago may be pending or not listed yet."

	// The states of pmm_alerts rows.
	alertFiring     = "firing"
	alertRecovering = "recovering"
	alertPending    = "pending"

	// What Grafana writes for a label that a template rule reads from a series that lacks it; it counts as absent.
	noValue = "[no value]"

	// The Alertmanager label that carries an alert's rule UID.
	ruleUIDLabel = "__alert_rule_uid__"
)

// severityOrder lists the severities that come ahead of every other one, in order.
var severityOrder = []string{"emergency", "alert", "critical", "error", "warning"}

// alertStateOrder lists the states of pmm_alerts rows in the order they are listed.
var alertStateOrder = []string{alertFiring, alertRecovering, alertPending}

// labelValue returns a label's value, empty when the label is absent or [no value].
func labelValue(labels map[string]string, name string) string {
	v := labels[name]
	if v == noValue {
		return ""
	}
	return v
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

// filterNames returns the names of the call's filters, in the order the input lists them.
func (in alertsInput) filterNames() []string {
	var names []string
	for _, f := range []struct{ name, value string }{
		{"service_id", in.ServiceID}, {"service_name", in.ServiceName}, {"node_name", in.NodeName}, {"severity", in.Severity},
	} {
		if f.value != "" {
			names = append(names, f.name)
		}
	}
	return names
}

// alertRow is one alert in the result of pmm_alerts.
type alertRow struct {
	Name          string            `json:"name" jsonschema:"Untrusted. The alert rule's title."`
	Severity      string            `json:"severity,omitempty" jsonschema:"Untrusted. The alert's severity label, left out when it has none."`
	Labels        map[string]string `json:"labels,omitempty" jsonschema:"Untrusted. The alert's labels, without Grafana's internal labels that start with __."`
	Summary       string            `json:"summary,omitempty" jsonschema:"Untrusted. The alert's summary annotation."`
	Description   string            `json:"description,omitempty" jsonschema:"Untrusted. The alert's description annotation."`
	State         string            `json:"state" jsonschema:"firing; recovering when its condition cleared but its rule keeps it firing; or pending before it fires."`
	StartsAt      time.Time         `json:"starts_at,omitzero" jsonschema:"When the alert started firing, recovering or pending; left out when Grafana gives no time."`
	Silenced      bool              `json:"silenced" jsonschema:"True when a silence mutes the alert's notifications; false while it is pending."`
	SilencedUntil *time.Time        `json:"silenced_until,omitempty" jsonschema:"When the latest of the silences that mute the alert ends."`
	PMMURL        string            `json:"pmm_url" jsonschema:"Link to the alert's rule in PMM."`
}

// alertsResult is the result of pmm_alerts.
type alertsResult struct {
	Alerts []alertRow `json:"alerts" jsonschema:"Firing, recovering and pending alerts on this page."`
	Page   page       `json:"page" jsonschema:"Paging of the full list."`
	Note   string     `json:"note" jsonschema:"What the list covers."`

	// filters names the call's filters, which the text of an empty list names.
	filters []string
}

// oneLine collapses s's whitespace, newlines included, and cuts it to the echo length, so an alert field stays on its line.
func oneLine(s string) string {
	return echo(strings.Join(strings.Fields(s), " "))
}

func (r alertsResult) render() string {
	lines := []string{r.Note}
	if r.Page.Total == 0 {
		return strings.Join(append(lines, noAlerts(r.filters)), "\n")
	}
	if len(r.Alerts) == 0 {
		return strings.Join(append(lines, r.Page.pastEnd("alert list")), "\n")
	}
	for _, a := range r.Alerts {
		var parts []string
		if a.Severity != "" {
			parts = append(parts, oneLine(a.Severity))
		}
		parts = append(parts, oneLine(a.Name))
		if a.State == alertRecovering || a.State == alertPending {
			parts = append(parts, a.State)
		}
		if svc := cmp.Or(labelValue(a.Labels, "service_name"), labelValue(a.Labels, "service_id")); svc != "" {
			parts = append(parts, "service="+oneLine(svc))
		}
		if node := cmp.Or(labelValue(a.Labels, "node_name"), labelValue(a.Labels, "node_id")); node != "" {
			parts = append(parts, "node="+oneLine(node))
		}
		if !a.StartsAt.IsZero() {
			parts = append(parts, "since "+a.StartsAt.UTC().Format(time.RFC3339))
		}
		switch {
		case a.SilencedUntil != nil:
			parts = append(parts, "silenced until "+a.SilencedUntil.UTC().Format(time.RFC3339))
		case a.Silenced:
			parts = append(parts, "silenced")
		}
		parts = append(parts, link("View in PMM", a.PMMURL))
		lines = append(lines, "- "+strings.Join(parts, " | "))
		if summary := oneLine(a.Summary); summary != "" {
			lines = append(lines, "  "+summary)
		}
	}
	if m := r.Page.marker(); m != "" {
		lines = append(lines, m)
	}
	return strings.Join(lines, "\n")
}

// noAlerts is the text of an empty list, which names the filters that left nothing, so that a mistyped value is not
// read as a server without alerts.
func noAlerts(filters []string) string {
	switch len(filters) {
	case 0:
		return "No firing, recovering or pending alerts found."
	case 1:
		return "No firing, recovering or pending alerts match the " + filters[0] + " filter: check its value, or call pmm_alerts without it."
	default:
		last := len(filters) - 1
		return "No firing, recovering or pending alerts match the " + strings.Join(filters[:last], ", ") + " and " + filters[last] +
			" filters: check their values, or call pmm_alerts without them."
	}
}

func (s *Service) registerAlertsTool(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_alerts",
		Title: "Firing, recovering and pending alerts",
		Description: "List the alerts firing now in PMM's alerting, the recovering ones whose condition cleared but whose rule keeps " +
			"them firing for a while, and the pending ones whose rule waits out its pending period, " +
			"from the alert rules your Grafana folder permissions let you read. " +
			alertsNote + " It lists the alerts that Grafana's rules API returns to you, as PMM's Alerts page does; Access Roles don't narrow them. " +
			"Each alert has its severity, labels, summary, description, start time, whether a silence mutes it and until when, and " +
			"a link to its rule in PMM; firing alerts come first, then recovering, then pending ones, each the most severe and newest first. " +
			"Filter by service_id, service_name, node_name or " +
			"severity. Read only: it does not silence or acknowledge alerts.",
		Annotations: readOnly("Firing, recovering and pending alerts"),
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

	all, err := s.api.Alerts(ctx, auth)
	if err != nil {
		return alertsResult{}, err
	}
	kept := slices.DeleteFunc(all, func(a grafanaAlert) bool { return !in.matches(a) })
	for i := range kept {
		kept[i].labelKey = labelSetKey(kept[i].Labels)
	}
	slices.SortFunc(kept, compareAlerts)

	paged, pg, err := paginate(kept, in.Offset, in.Limit, alertsDefaultLimit, alertsMaxLimit)
	if err != nil {
		return alertsResult{}, err
	}
	// Only firing and recovering alerts reach the Alertmanager, so a page of pending ones needs no silence state.
	if slices.ContainsFunc(paged, func(a grafanaAlert) bool { return !a.Pending }) {
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
			State:         alertState(a),
			StartsAt:      a.StartsAt,
			Silenced:      a.Silenced,
			SilencedUntil: a.SilencedUntil,
			PMMURL:        alertRuleURL(base, a.RuleUID),
		})
	}
	return alertsResult{Alerts: rows, Page: pg, Note: alertsNote, filters: in.filterNames()}, nil
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
		if a.Pending {
			continue
		}
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

// severityRank ranks emergency, alert, critical, error and warning first, then every other severity, then none.
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

// alertState is an alert's state as pmm_alerts names it.
func alertState(a grafanaAlert) string {
	switch {
	case a.Pending:
		return alertPending
	case a.Recovering:
		return alertRecovering
	default:
		return alertFiring
	}
}

// compareAlerts orders firing alerts first, then recovering, then pending ones, then by severity, case-insensitively and
// other severities alphabetically, then newest first, then by name.
func compareAlerts(a, b grafanaAlert) int {
	sa, sb := strings.ToLower(a.Labels["severity"]), strings.ToLower(b.Labels["severity"])
	c := cmp.Or(
		cmp.Compare(slices.Index(alertStateOrder, alertState(a)), slices.Index(alertStateOrder, alertState(b))),
		cmp.Compare(severityRank(sa), severityRank(sb)),
		strings.Compare(sa, sb),
		b.StartsAt.Compare(a.StartsAt),
		strings.Compare(a.Labels["alertname"], b.Labels["alertname"]),
		strings.Compare(a.RuleUID, b.RuleUID),
	)
	if c != 0 {
		return c
	}
	return strings.Compare(a.labelKey, b.labelKey)
}
