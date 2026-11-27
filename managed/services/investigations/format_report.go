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
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

const formatReportTimeout = 120 * time.Second

// FormattedReport is the parsed output from the format step.
type FormattedReport struct {
	Summary             string          `json:"summary"`
	SummaryDetailed     string          `json:"summary_detailed"`
	RootCauseSummary    string          `json:"root_cause_summary"`
	ResolutionSummary   string          `json:"resolution_summary"`
	Confidence          string          `json:"confidence"`
	ConfidenceScore     int             `json:"confidence_score"`
	ConfidenceRationale string          `json:"confidence_rationale"`
	Evidence            []EvidenceEntry `json:"evidence"`
	TimelineEvents      []TimelineEvent `json:"timeline_events"`
	Sections            []Section       `json:"sections"`
}

// EvidenceEntry maps a claim to concrete source evidence.
type EvidenceEntry struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Claim        string `json:"claim"`
	SourceTool   string `json:"source_tool"`
	SourceRef    string `json:"source_ref"`
	Excerpt      string `json:"excerpt"`
	TimeRange    string `json:"time_range"`
	Verification string `json:"verification"`
}

// TimelineEvent is a chronological event extracted from the report.
type TimelineEvent struct {
	EventTime   string `json:"event_time"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Section is a single section within the formatted report.
type Section struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// FormatInvestigationReport calls Holmes Chat to convert raw markdown into structured JSON.
// Metadata is the Holmes response metadata for usage tracking (may be nil).
func FormatInvestigationReport(ctx context.Context, client *adre.Client, settings *models.Settings, rawMarkdown string) ([]byte, json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, formatReportTimeout)
	defer cancel()

	ask := "Convert the following investigation report into JSON:\n\n```\n" + rawMarkdown + "\n```"
	req := &adre.ChatRequest{
		Ask:                    ask,
		AdditionalSystemPrompt: adre.InvestigationFormatPrompt,
		BehaviorControls:       adre.ResolveBehaviorControlsForFormatReport(settings),
		Stream:                 false,
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if resp.Analysis == "" {
		return nil, resp.Metadata, errEmptyResponse
	}

	jsonBytes := []byte(strings.TrimSpace(resp.Analysis))
	// Strip markdown code fence if present
	if strings.HasPrefix(string(jsonBytes), "```") {
		jsonBytes = stripCodeFence(jsonBytes)
	}
	return jsonBytes, resp.Metadata, nil
}

var codeFenceRe = regexp.MustCompile("(?s)^\\s*" + "```" + "(?:json)?\\s*\\n(.*)\\n" + "```" + "\\s*$")

func stripCodeFence(b []byte) []byte {
	sub := codeFenceRe.FindSubmatch(b)
	if len(sub) >= 2 { //nolint:mnd
		return sub[1]
	}
	// Fallback: remove leading ```json or ``` and trailing ```
	s := string(b)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return []byte(strings.TrimSpace(s))
}

var errEmptyResponse = &parseError{msg: "empty response from format step"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

// ParseFormattedReport unmarshals JSON into FormattedReport and validates required fields.
func ParseFormattedReport(jsonBytes []byte) (*FormattedReport, error) {
	var fr FormattedReport
	err := json.Unmarshal(jsonBytes, &fr)
	if err != nil {
		return nil, err
	}
	if fr.ConfidenceScore < 0 || fr.ConfidenceScore > 100 {
		fr.ConfidenceScore = 0
	}
	if fr.Confidence == "" {
		fr.Confidence = "medium"
	}
	if fr.Evidence == nil {
		fr.Evidence = []EvidenceEntry{}
	}
	fr.Confidence, fr.ConfidenceScore, fr.ConfidenceRationale = ComputeConfidence(fr)
	return &fr, nil
}

// normalizeInvestigationImageURL stores relative PMM render paths so UI works regardless of public address host.
func normalizeInvestigationImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/v1/grafana/render/") || strings.HasPrefix(raw, "/graph/render/") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if strings.HasPrefix(u.Path, "/v1/grafana/render/") || strings.HasPrefix(u.Path, "/graph/render/") {
		return u.Path
	}
	return raw
}

// buildBlockDataJSON produces data_json for markdown, finding, or remediation_steps blocks.
func buildBlockDataJSON(blockType, content string) []byte {
	if blockType == BlockTypeRemediationSteps {
		steps := parseRemediationSteps(content)
		if len(steps) > 0 {
			b, _ := json.Marshal(map[string]any{"steps": steps}) //nolint:errchkjson // []string is always marshalable
			return b
		}
	}
	if blockType == BlockTypeImage {
		url := normalizeInvestigationImageURL(content)
		if url != "" {
			b, _ := json.Marshal(map[string]string{"url": url}) //nolint:errchkjson // map[string]string is always marshalable
			return b
		}
	}
	// markdown / finding: {"content": "..."}
	b, _ := json.Marshal(map[string]string{"content": content}) //nolint:errchkjson,goconst // map[string]string is always marshalable
	return b
}

// parseRemediationSteps splits content into steps (numbered list or newline-separated).
func parseRemediationSteps(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	var steps []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip blanks and Markdown code-fence markers (```/```bash). When Holmes wraps commands in a
		// fenced block, splitting line-by-line would otherwise turn the fence markers into empty
		// "steps" that render as empty code boxes in the UI.
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		// Strip leading "1. ", "2)", "- ", "* ", "• ", etc.
		line = numberPrefixRe.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line != "" {
			steps = append(steps, line)
		}
	}
	return steps
}

var numberPrefixRe = regexp.MustCompile(`^\s*\d+[.)]\s*|^\s*[-*•]\s*`)

// ComputeConfidence calculates deterministic confidence from report content.
func ComputeConfidence(fr FormattedReport) (band string, score int, rationale string) { //nolint:nonamedreturns
	score = 50

	// Evidence quality (+0..25)
	evidenceN := min(len(fr.Evidence), 4) //nolint:mnd
	score += evidenceN * 5                //nolint:mnd
	if hasAtLeastTwoEvidenceKinds(fr.Evidence) {
		score += 5
	}

	// Coverage/completeness (+0..15)
	if strings.TrimSpace(fr.RootCauseSummary) != "" &&
		strings.TrimSpace(fr.ResolutionSummary) != "" &&
		strings.TrimSpace(fr.Summary) != "" {
		score += 10
	}
	if len(fr.TimelineEvents) >= 2 && hasTwoValidTimelineEvents(fr.TimelineEvents) {
		score += 5
	}

	// Uncertainty penalties (-0..40)
	text := strings.ToLower(fr.Summary + "\n" + fr.SummaryDetailed + "\n" + fr.RootCauseSummary)
	if strings.Contains(text, "inconclusive") || strings.Contains(text, "unable to determine") {
		score -= 15
	}
	if strings.Contains(text, "possible causes") || strings.Contains(text, "multiple causes") {
		score -= 10
	}
	if len(fr.Evidence) == 0 {
		score -= 10
	}
	if strings.Contains(text, "might be") || strings.Contains(text, "could be") {
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	if score > 100 { //nolint:mnd
		score = 100
	}

	switch {
	case score >= 75: //nolint:mnd
		band = "high"
	case score >= 45: //nolint:mnd
		band = "medium"
	default:
		band = "low"
	}
	rationale = "Computed from evidence count/diversity, report completeness, and uncertainty signals."
	return band, score, rationale
}

func hasAtLeastTwoEvidenceKinds(e []EvidenceEntry) bool {
	kinds := map[string]struct{}{}
	for _, it := range e {
		k := strings.TrimSpace(it.Kind)
		if k == "" {
			continue
		}
		kinds[k] = struct{}{}
		if len(kinds) >= 2 { //nolint:mnd
			return true
		}
	}
	return false
}

func hasTwoValidTimelineEvents(events []TimelineEvent) bool {
	valid := 0
	for _, ev := range events {
		if strings.TrimSpace(ev.EventTime) == "" {
			continue
		}
		_, err := time.Parse(time.RFC3339, ev.EventTime)
		if err != nil {
			continue
		}
		valid++
		if valid >= 2 { //nolint:mnd
			return true
		}
	}
	return false
}
