// Copyright (C) 2025 Percona LLC
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

package investigations

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

const formatReportTimeout = 120 * time.Second

// FormattedReport is the parsed output from the format step.
type FormattedReport struct {
	Summary           string          `json:"summary"`
	SummaryDetailed   string          `json:"summary_detailed"`
	RootCauseSummary  string          `json:"root_cause_summary"`
	ResolutionSummary string          `json:"resolution_summary"`
	TimelineEvents    []TimelineEvent `json:"timeline_events"`
	Sections          []Section       `json:"sections"`
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
func FormatInvestigationReport(ctx context.Context, client *adre.Client, settings *models.Settings, rawMarkdown string) ([]byte, error) {
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
		return nil, err
	}
	if resp.Analysis == "" {
		return nil, errEmptyResponse
	}

	jsonBytes := []byte(strings.TrimSpace(resp.Analysis))
	// Strip markdown code fence if present
	if strings.HasPrefix(string(jsonBytes), "```") {
		jsonBytes = stripCodeFence(jsonBytes)
	}
	return jsonBytes, nil
}

var codeFenceRe = regexp.MustCompile("(?s)^\\s*" + "```" + "(?:json)?\\s*\\n(.*)\\n" + "```" + "\\s*$")

func stripCodeFence(b []byte) []byte {
	sub := codeFenceRe.FindSubmatch(b)
	if len(sub) >= 2 {
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
	if err := json.Unmarshal(jsonBytes, &fr); err != nil {
		return nil, err
	}
	// Allow empty summary/root_cause/resolution; sections may be empty
	return &fr, nil
}

// buildBlockDataJSON produces data_json for markdown, finding, or remediation_steps blocks.
func buildBlockDataJSON(blockType, content string) []byte {
	if blockType == BlockTypeRemediationSteps {
		steps := parseRemediationSteps(content)
		if len(steps) > 0 {
			b, _ := json.Marshal(map[string]interface{}{"steps": steps})
			return b
		}
	}
	// markdown / finding: {"content": "..."}
	b, _ := json.Marshal(map[string]string{"content": content})
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
		if line == "" {
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
