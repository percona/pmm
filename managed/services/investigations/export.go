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
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"

	"github.com/percona/pmm/managed/models"
)

// GetInvestigationExportPDF returns an HTML report for the investigation so the client can print to PDF.
func (h *Handlers) GetInvestigationExportPDF(w http.ResponseWriter, _ *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationByID: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load investigation")
		return
	}
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	blocks, err := models.GetInvestigationBlocks(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationBlocks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load blocks")
		return
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].Position < blocks[j].Position })
	timelineEvents, err := models.GetInvestigationTimelineEvents(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationTimelineEvents: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load timeline")
		return
	}
	htmlBytes, err := buildExportHTML(inv, blocks, timelineEvents)
	if err != nil {
		h.l.Errorf("buildExportHTML: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to build export")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=investigation-%s.html", id))
	_, _ = w.Write(htmlBytes)
}

func buildExportHTML(inv *models.Investigation, blocks []*models.InvestigationBlock, timelineEvents []*models.InvestigationTimelineEvent) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>")
	b.WriteString(html.EscapeString(inv.Title))
	b.WriteString("</title><style>")
	b.WriteString("body{font-family:system-ui,-apple-system,sans-serif;font-size:14px;line-height:1.55;max-width:820px;margin:0 auto;padding:0 1.5em;color:#1a1a1a;background:#fff}") //nolint:lll
	b.WriteString(".report-header{background:linear-gradient(135deg,#1e3a5f 0%,#2c5282 100%);color:#fff;padding:1.25em 1.5em;margin:0 -1.5em 1.5em;border-radius:0 0 8px 8px}")       //nolint:lll
	b.WriteString(".report-header h1{font-size:1.4em;margin:0 0 0.25em;font-weight:600}")
	b.WriteString(".report-header .subtitle{font-size:0.85em;opacity:0.9}")
	b.WriteString("h2{font-size:1.15em;margin:1.75em 0 0.5em;color:#1e3a5f;font-weight:600}")
	b.WriteString(".meta-block{background:#f0f4f8;border:1px solid #e2e8f0;border-radius:6px;padding:0.75em 1em;margin-bottom:1.5em;font-size:0.9em;color:#475569}")
	b.WriteString(".meta-block span+span::before{content:\" · \";color:#94a3b8}")
	b.WriteString(".summary-block{background:#f8fafc;border-left:4px solid #2c5282;border-radius:0 6px 6px 0;padding:1em 1.25em;margin:1em 0 1.5em}")
	b.WriteString(".summary-block p{margin:0;white-space:pre-wrap}")
	b.WriteString(".block{margin:1.25em 0;padding:1em 1.25em;border:1px solid #e2e8f0;border-radius:6px;border-left-width:4px;background:#fff}")
	b.WriteString(".block-markdown{border-left-color:#64748b}")
	b.WriteString(".block-finding{border-left-color:#2563eb;background:#f8fafc}")
	b.WriteString(".block-remediation_steps{border-left-color:#16a34a;background:#f0fdf4}")
	b.WriteString(".block-query_result{border-left-color:#64748b;background:#f8fafc}")
	b.WriteString(".block h3{font-size:1em;margin:0 0 0.5em;color:#1e293b;font-weight:600}")
	b.WriteString(".block pre{white-space:pre-wrap;background:#f1f5f9;padding:0.75em;overflow:auto;border-radius:4px;margin:0.5em 0;font-size:0.9em;border:1px solid #e2e8f0}") //nolint:lll
	b.WriteString(".timeline{margin:0.5em 0;padding-left:1.25em}")
	b.WriteString(".timeline-event{margin:0.4em 0;font-size:0.95em;color:#475569}")
	b.WriteString(".report-footer{margin-top:2em;padding-top:1em;border-top:1px solid #e2e8f0;font-size:0.8em;color:#94a3b8;text-align:center}")
	b.WriteString("@media print{.report-header{box-shadow:none;-webkit-print-color-adjust:exact;print-color-adjust:exact}.block{break-inside:avoid;box-shadow:none}.meta-block,.summary-block{box-shadow:none}}") //nolint:lll
	b.WriteString("</style></head><body>")
	b.WriteString("<div class=\"report-header\"><h1>")
	b.WriteString(html.EscapeString(inv.Title))
	b.WriteString("</h1><div class=\"subtitle\">Investigation Report</div></div>")

	// Metadata block
	nodeName, serviceName, clusterName := "", "", ""
	if len(inv.Config) > 0 {
		var cfg map[string]string
		err := json.Unmarshal(inv.Config, &cfg)
		if err == nil {
			nodeName = cfg["node_name"]
			serviceName = cfg["service_name"]
			clusterName = cfg["cluster_name"]
		}
	}
	timeRange := formatTime(inv.TimeFrom) + " — " + formatTime(inv.TimeTo)
	source := inv.SourceType
	if source == "" {
		source = "—"
	}
	b.WriteString("<div class=\"meta-block\"><span>Time range: " + html.EscapeString(timeRange) + "</span>")
	b.WriteString("<span>Source: " + html.EscapeString(source) + "</span>")
	if nodeName != "" {
		b.WriteString("<span>Node: " + html.EscapeString(nodeName) + "</span>")
	}
	if serviceName != "" {
		b.WriteString("<span>Service: " + html.EscapeString(serviceName) + "</span>")
	}
	if clusterName != "" {
		b.WriteString("<span>Cluster: " + html.EscapeString(clusterName) + "</span>")
	}
	b.WriteString("<span>Status: " + html.EscapeString(inv.Status) + "</span>")
	b.WriteString("<span>Created: " + html.EscapeString(formatTime(inv.CreatedAt)) + "</span>")
	b.WriteString("</div>")

	// Summary
	if inv.Summary != "" {
		b.WriteString("<h2>Summary</h2><div class=\"summary-block\"><p>")
		b.WriteString(html.EscapeString(inv.Summary))
		b.WriteString("</p></div>")
	}

	// Timeline section
	if len(timelineEvents) > 0 {
		b.WriteString("<h2>Timeline</h2><ol class=\"timeline\">")
		for _, te := range timelineEvents {
			dtStr := te.EventTime.Format("2006-01-02 15:04:05") + " UTC"
			label := dtStr
			if te.Title != "" {
				label += " - " + te.Title
			}
			if te.Description != "" {
				label += " - " + te.Description
			}
			b.WriteString("<li class=\"timeline-event\">")
			b.WriteString(html.EscapeString(label))
			b.WriteString("</li>")
		}
		b.WriteString("</ol>")
	}

	// Report blocks
	for _, blk := range blocks {
		blockClass := "block block-" + blk.Type
		b.WriteString("<div class=\"" + html.EscapeString(blockClass) + "\">")
		b.WriteString("<h3>")
		b.WriteString(html.EscapeString(blk.Type))
		if blk.Title != "" {
			b.WriteString(": ")
			b.WriteString(html.EscapeString(blk.Title))
		}
		b.WriteString("</h3>")
		content, err := blockExportContent(blk)
		if err != nil {
			return nil, err
		}
		b.WriteString(content)
		b.WriteString("</div>")
	}

	// Root cause / Resolution
	if inv.RootCauseSummary != "" {
		b.WriteString("<h2>Root cause</h2><div class=\"summary-block\"><p>")
		b.WriteString(html.EscapeString(inv.RootCauseSummary))
		b.WriteString("</p></div>")
	}
	if inv.ResolutionSummary != "" {
		b.WriteString("<h2>Resolution</h2><div class=\"summary-block\"><p>")
		b.WriteString(html.EscapeString(inv.ResolutionSummary))
		b.WriteString("</p></div>")
	}

	b.WriteString("<div class=\"report-footer\">Generated by Percona Monitoring and Management · ")
	b.WriteString(html.EscapeString(formatTime(inv.CreatedAt)))
	b.WriteString("</div>")
	b.WriteString("<script>window.onload=function(){window.print()}</script></body></html>")
	return b.Bytes(), nil
}

func blockExportContent(blk *models.InvestigationBlock) (string, error) { //nolint:cyclop,gocognit
	switch blk.Type {
	case "remediation_steps":
		var data map[string]any
		if len(blk.DataJSON) > 0 {
			err := json.Unmarshal(blk.DataJSON, &data)
			if err != nil {
				return "", fmt.Errorf("data_json: %w", err)
			}
		}
		steps, _ := data["steps"].([]any)
		if len(steps) == 0 {
			return "<p>(no steps)</p>", nil
		}
		var b bytes.Buffer
		b.WriteString("<ul>")
		for _, s := range steps {
			text := fmt.Sprint(s)
			b.WriteString("<li>")
			b.WriteString(html.EscapeString(text))
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
		return b.String(), nil
	case "summary", "markdown", "finding":
		var data map[string]any
		if len(blk.DataJSON) > 0 {
			err := json.Unmarshal(blk.DataJSON, &data)
			if err != nil {
				return "", fmt.Errorf("data_json: %w", err)
			}
		}
		text := ""
		if c, ok := data["content"].(string); ok {
			text = c
		}
		if text == "" && blk.Title != "" {
			text = blk.Title
		}
		if text == "" {
			return "<p>(no content)</p>", nil
		}
		return "<pre>" + html.EscapeString(text) + "</pre>", nil
	case "query_result":
		var data map[string]any
		if len(blk.DataJSON) > 0 {
			err := json.Unmarshal(blk.DataJSON, &data)
			if err != nil {
				return "", fmt.Errorf("data_json: %w", err)
			}
		}
		result, _ := data["result"].(string)
		if result == "" && data["query"] != nil {
			result = fmt.Sprint(data["query"])
		}
		if result == "" {
			result = "(no result)"
		}
		return "<pre>" + html.EscapeString(result) + "</pre>", nil
	case "image":
		var cfg map[string]any
		if len(blk.ConfigJSON) > 0 {
			err := json.Unmarshal(blk.ConfigJSON, &cfg)
			if err != nil {
				return "", fmt.Errorf("config_json: %w", err)
			}
		}
		var data map[string]any
		if len(blk.DataJSON) > 0 {
			err := json.Unmarshal(blk.DataJSON, &data)
			if err != nil {
				return "", fmt.Errorf("data_json: %w", err)
			}
		}
		src := strings.TrimSpace(fmt.Sprint(cfg["url"]))
		if src == "" || src == "<nil>" { //nolint:goconst
			src = strings.TrimSpace(fmt.Sprint(cfg["image_url"]))
		}
		if src == "" || src == "<nil>" {
			src = strings.TrimSpace(fmt.Sprint(data["url"]))
		}
		if src == "" || src == "<nil>" {
			src = strings.TrimSpace(fmt.Sprint(data["image_url"]))
		}
		if src == "" || src == "<nil>" {
			src = strings.TrimSpace(fmt.Sprint(data["content"]))
		}
		if src == "" || src == "<nil>" {
			return "<p>(no image url)</p>", nil
		}
		alt := strings.TrimSpace(fmt.Sprint(cfg["alt"]))
		if alt == "" || alt == "<nil>" {
			alt = strings.TrimSpace(fmt.Sprint(data["alt"]))
		}
		if alt == "" || alt == "<nil>" {
			alt = blk.Title
		}
		caption := strings.TrimSpace(fmt.Sprint(cfg["caption"]))
		if caption == "" || caption == "<nil>" {
			caption = strings.TrimSpace(fmt.Sprint(data["caption"]))
		}
		var b bytes.Buffer
		b.WriteString("<figure style=\"margin:0\">")
		b.WriteString("<img src=\"")
		b.WriteString(html.EscapeString(src))
		b.WriteString("\" alt=\"")
		b.WriteString(html.EscapeString(alt))
		b.WriteString("\" style=\"max-width:100%;height:auto;border-radius:6px;border:1px solid #e2e8f0\" />")
		if caption != "" && caption != "<nil>" {
			b.WriteString("<figcaption style=\"font-size:12px;color:#64748b;margin-top:6px\">")
			b.WriteString(html.EscapeString(caption))
			b.WriteString("</figcaption>")
		}
		b.WriteString("</figure>")
		return b.String(), nil
	default:
		// Generic: show data_json as formatted JSON; fall back to raw bytes.
		if len(blk.DataJSON) == 0 {
			return "<p>(no data)</p>", nil
		}
		rawHTML := "<pre>" + html.EscapeString(string(blk.DataJSON)) + "</pre>"
		var v any
		err := json.Unmarshal(blk.DataJSON, &v)
		if err != nil {
			return rawHTML, nil //nolint:nilerr // fallback: show raw bytes when unmarshal fails
		}
		content, mErr := json.MarshalIndent(v, "", "  ")
		if mErr != nil {
			return rawHTML, nil //nolint:nilerr // defensive: re-marshal of already-unmarshalled JSON should not fail
		}
		return "<pre>" + html.EscapeString(string(content)) + "</pre>", nil
	}
}
