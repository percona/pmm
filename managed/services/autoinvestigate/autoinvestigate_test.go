// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package autoinvestigate

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
)

func settingsWith(minSev string, matchers []string) *models.Settings {
	s := &models.Settings{}
	s.Adre.AutoInvestigateMinSeverity = minSev
	s.Adre.AutoInvestigateLabelMatchers = matchers
	return s
}

func TestPassesSeverityFloor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		floor, sev string
		want       bool
	}{
		{"", "info", true}, // no floor ⇒ all pass
		{"critical", "critical", true},
		{"critical", "warning", false},
		{"warning", "critical", true},
		{"warning", "info", false},
		{"critical", "", true},      // no severity label ⇒ not filtered by the floor
		{"critical", "bogus", true}, // unrankable severity ⇒ not filtered by the floor
	}
	for _, c := range cases {
		if got := passesSeverityFloor(c.floor, c.sev); got != c.want {
			t.Errorf("passesSeverityFloor(%q,%q)=%v want %v", c.floor, c.sev, got, c.want)
		}
	}
}

func TestPassesLabelMatchers(t *testing.T) {
	t.Parallel()
	labels := map[string]string{"severity": "critical", "service_type": "mysql"}
	if !passesLabelMatchers([]string{"service_type=mysql"}, labels) {
		t.Error("matching label should pass")
	}
	if passesLabelMatchers([]string{"service_type=postgresql"}, labels) {
		t.Error("non-matching label should fail")
	}
	if !passesLabelMatchers([]string{"severity=critical", "service_type=mysql"}, labels) {
		t.Error("all matching labels should pass")
	}
	if passesLabelMatchers([]string{"severity=critical", "service_type=postgresql"}, labels) {
		t.Error("one non-matching label should fail (AND semantics)")
	}
}

func TestPassesSelection(t *testing.T) {
	t.Parallel()
	a := Alert{Labels: map[string]string{"severity": "warning", "team": "db"}}
	if passesSelection(settingsWith("critical", nil), a) {
		t.Error("warning should be filtered by a critical floor")
	}
	if !passesSelection(settingsWith("warning", []string{"team=db"}), a) {
		t.Error("warning with matching label should pass a warning floor")
	}
}

func TestParseAlertmanagerAlerts(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"fingerprint":"fp1","labels":{"alertname":"X","severity":"critical"},"annotations":{"summary":"s"}},{"labels":{}}]`)
	got := ParseAlertmanagerAlerts(raw)
	if len(got) != 1 {
		t.Fatalf("expected 1 alert (the one without a fingerprint is dropped), got %d", len(got))
	}
	if got[0].Fingerprint != "fp1" || !got[0].firing() {
		t.Errorf("unexpected alert: %+v", got[0])
	}
}

func TestParseGrafanaWebhook(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp1","labels":{"severity":"critical"}},{"status":"resolved","fingerprint":"fp2"}]}`)
	got := ParseGrafanaWebhook(raw)
	if len(got) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(got))
	}
	if !got[0].firing() || !got[1].resolved() {
		t.Errorf("status not preserved: %+v", got)
	}
}

func TestBuildAlertInvestigation(t *testing.T) {
	t.Parallel()
	a := Alert{
		Fingerprint: "fp1",
		Status:      "firing",
		Labels:      map[string]string{"alertname": "HighCPU", "severity": "critical", "service_name": "mysql-1"},
	}
	inv := buildAlertInvestigation(a)
	if inv.AlertFingerprint != "fp1" || inv.SourceType != "alert" || inv.SourceRef != "fp1" {
		t.Errorf("unexpected investigation: fp=%q source=%q ref=%q", inv.AlertFingerprint, inv.SourceType, inv.SourceRef)
	}
	if inv.Title != "Alert: HighCPU" {
		t.Errorf("unexpected title %q", inv.Title)
	}
	if len(inv.Config) == 0 {
		t.Error("config should carry the alert snapshot")
	}
}

func TestMarkEpisodeBounded(t *testing.T) {
	t.Parallel()
	s := New(nil, nil, nil, logrus.WithField("test", "autoinvestigate"))
	for i := range maxEpisodes + 100 {
		s.markEpisode("fp-" + string(rune(i)))
	}
	s.mu.Lock()
	n := len(s.episodes)
	s.mu.Unlock()
	// Once full, each insert evicts exactly one entry, so the map stays pinned at the bound (a bug
	// that cleared the whole map would leave n far below maxEpisodes).
	if n != maxEpisodes {
		t.Fatalf("episode map size = %d, want exactly %d", n, maxEpisodes)
	}
}
