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

package investigations

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
)

const (
	// How many recent investigations are compared; enough for a team's history, cheap to score in memory.
	similarScanLimit  = 500
	similarMaxResults = 5
	// A shared node alone qualifies: that host had trouble before.
	similarMinScore = 2.0
	// A share of distinct words from which two descriptions count as similar.
	similarTextMin = 0.15
	// Text adds up to this many points, ten per unit of word overlap.
	similarTextMax    = 4.0
	similarTextWeight = 10.0
	// The closest past incidents an investigation run is told about, and how much of each.
	similarContextMax   = 3
	similarContextRunes = 300
)

// Signals of a recurring incident, strongest first.
var similarityWeights = []struct {
	reason string
	weight float64
	field  func(incidentKey) string
}{
	{"Same alert", 4, func(k incidentKey) string { return k.alertname }},
	{"Same service", 3, func(k incidentKey) string { return k.service }},
	{"Same node", 2, func(k incidentKey) string { return k.node }},
	{"Same cluster", 1, func(k incidentKey) string { return k.cluster }},
}

var wordRe = regexp.MustCompile(`[a-z][a-z0-9_]{3,}`)

// Words too common in incident reports to tell two apart.
var stopWords = map[string]bool{
	"alert": true, "chat": true, "about": true, "after": true, "also": true, "because": true, "been": true,
	"before": true, "both": true, "call": true, "does": true, "each": true, "from": true, "have": true, "into": true,
	"many": true, "more": true, "most": true, "only": true, "other": true, "over": true, "same": true, "since": true,
	"some": true, "than": true, "that": true, "then": true, "there": true, "these": true, "they": true, "this": true,
	"time": true, "very": true, "were": true, "what": true, "when": true, "which": true, "while": true, "will": true,
	"with": true, "would": true, "your": true, "investigation": true,
}

// incidentKey is what makes two incidents look alike.
type incidentKey struct {
	alertname, service, node, cluster string
	words                             map[string]bool
}

func normalized(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// words are a text's distinct significant words, with a plural's s dropped so "deadlocks" meets "deadlock".
func words(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range wordRe.FindAllString(strings.ToLower(text), -1) {
		if stopWords[w] {
			continue
		}
		if len(w) > 4 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
			w = w[:len(w)-1]
		}
		out[w] = true
	}
	return out
}

func incident(alertname, service, node, cluster, text string) incidentKey {
	return incidentKey{
		alertname: normalized(alertname),
		service:   normalized(service),
		node:      normalized(node),
		cluster:   normalized(cluster),
		words:     words(text),
	}
}

// incidentOf reads an investigation's alert labels and inventory names from its config, and its words from what it says.
func incidentOf(inv *models.Investigation) incidentKey {
	cfg := investigationConfigMap(inv)
	str := func(key string) string {
		v, _ := cfg[key].(string)
		return v
	}
	var labels map[string]string
	var alerts []alertSnapshotEntry
	if json.Unmarshal([]byte(str("alert_snapshot")), &alerts) == nil && len(alerts) > 0 {
		labels = alerts[0].Labels
	}
	first := func(values ...string) string {
		for _, v := range values {
			if strings.TrimSpace(v) != "" {
				return v
			}
		}
		return ""
	}
	title := strings.TrimPrefix(strings.TrimPrefix(inv.Title, "Alert: "), "Chat: ")
	return incident(
		labels["alertname"],
		first(str("service_name"), labels["service_name"]),
		first(str("node_name"), labels["node_name"]),
		first(str("cluster_name"), labels["cluster"], labels["cluster_name"]),
		strings.Join([]string{title, inv.Summary, inv.RootCauseSummary, userRequestFromInvestigation(inv)}, "\n"),
	)
}

func (k incidentKey) empty() bool {
	return k.alertname == "" && k.service == "" && k.node == "" && k.cluster == "" && len(k.words) == 0
}

// similarity scores how alike two incidents are and says why.
func similarity(a, b incidentKey) (float64, []string) {
	var score float64
	var reasons []string
	for _, s := range similarityWeights {
		if v := s.field(a); v != "" && v == s.field(b) {
			score += s.weight
			reasons = append(reasons, s.reason)
		}
	}
	shared := 0
	for w := range a.words {
		if b.words[w] {
			shared++
		}
	}
	union := len(a.words) + len(b.words) - shared
	if union > 0 {
		jaccard := float64(shared) / float64(union)
		if jaccard >= similarTextMin {
			score += math.Min(similarTextMax, similarTextWeight*jaccard)
			reasons = append(reasons, "Similar description")
		}
	}
	return score, reasons
}

// SimilarInvestigation is a past investigation that looks like the current incident, with how it ended.
type SimilarInvestigation struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	Summary           string    `json:"summary"`
	RootCauseSummary  string    `json:"root_cause_summary"`
	ResolutionSummary string    `json:"resolution_summary"`
	Score             float64   `json:"score"`
	Reasons           []string  `json:"reasons"`
}

// similarInvestigations ranks the candidates opened before an instant that look like the incident; running ones
// have no outcome yet.
func similarInvestigations(key incidentKey, candidates []*models.Investigation, excludeID string, before time.Time) []SimilarInvestigation {
	var out []SimilarInvestigation
	for _, c := range candidates {
		if c.ID == excludeID || c.Status == "running" || !c.CreatedAt.Before(before) {
			continue
		}
		score, reasons := similarity(key, incidentOf(c))
		if score < similarMinScore {
			continue
		}
		out = append(out, SimilarInvestigation{
			ID:                c.ID,
			Title:             c.Title,
			Status:            c.Status,
			CreatedAt:         c.CreatedAt,
			Summary:           c.Summary,
			RootCauseSummary:  c.RootCauseSummary,
			ResolutionSummary: c.ResolutionSummary,
			Score:             score,
			Reasons:           reasons,
		})
	}
	slices.SortStableFunc(out, func(a, b SimilarInvestigation) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	if len(out) > similarMaxResults {
		out = out[:similarMaxResults]
	}
	return out
}

func (h *Handlers) findSimilar(key incidentKey, excludeID string, before time.Time) ([]SimilarInvestigation, error) {
	candidates, err := models.ListInvestigations(h.db, "", "", similarScanLimit, 0, "created_at", "desc")
	if err != nil {
		return nil, err
	}
	return similarInvestigations(key, candidates, excludeID, before), nil
}

// similarContext tells an investigation run how similar past incidents ended, so it can say whether this one recurs.
func similarContext(similar []SimilarInvestigation) string {
	if len(similar) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nSimilar past investigations (say whether this incident is a recurrence of one):\n")
	for _, s := range similar[:min(similarContextMax, len(similar))] {
		fmt.Fprintf(&b, "- %s (%s, %s; %s)\n", s.Title, s.CreatedAt.Format(time.DateOnly), s.Status, strings.Join(s.Reasons, ", "))
		if s.RootCauseSummary != "" {
			b.WriteString("  Root cause: " + truncateRunes(s.RootCauseSummary, similarContextRunes) + "\n")
		}
		if s.ResolutionSummary != "" {
			b.WriteString("  Resolution: " + truncateRunes(s.ResolutionSummary, similarContextRunes) + "\n")
		}
	}
	return b.String()
}

func (h *Handlers) writeSimilar(w http.ResponseWriter, key incidentKey, excludeID string, before time.Time) {
	similar, err := h.findSimilar(key, excludeID, before)
	if err != nil {
		h.l.Errorf("findSimilar: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to find similar investigations")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"similar": similar}) //nolint:errchkjson // response already committed
}

// GetSimilarInvestigations handles GET /v1/investigations/:id/similar: earlier investigations that look like this one.
func (h *Handlers) GetSimilarInvestigations(w http.ResponseWriter, _ *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationByID: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	h.writeSimilar(w, incidentOf(inv), inv.ID, inv.CreatedAt)
}

// FindSimilarInvestigations handles GET /v1/investigations/similar?alertname=&service_name=&node_name=&cluster=&q=:
// past investigations that look like an incident with no investigation yet, such as a firing alert.
func (h *Handlers) FindSimilarInvestigations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key := incident(q.Get("alertname"), q.Get("service_name"), q.Get("node_name"), q.Get("cluster"), q.Get("q"))
	if key.empty() {
		writeJSONError(w, http.StatusBadRequest, "Give an alertname, service_name, node_name, cluster or q")
		return
	}
	h.writeSimilar(w, key, "", time.Now())
}
