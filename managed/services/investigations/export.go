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
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sort"

	"github.com/pkg/errors"

	"github.com/percona/pmm/managed/models"
)

// GetInvestigationExportPDF returns an HTML report for the investigation so the client can print to PDF.
func (h *Handlers) GetInvestigationExportPDF(w http.ResponseWriter, r *http.Request, id string) {
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
	htmlBytes, err := buildExportHTML(inv, blocks)
	if err != nil {
		h.l.Errorf("buildExportHTML: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to build export")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=investigation-%s.html", id))
	_, _ = w.Write(htmlBytes)
}

func buildExportHTML(inv *models.Investigation, blocks []*models.InvestigationBlock) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>")
	b.WriteString(html.EscapeString(inv.Title))
	b.WriteString("</title><style>body{font-family:sans-serif;max-width:800px;margin:2em auto;padding:0 1em}h1{font-size:1.5em}.meta{color:#666;font-size:0.9em;margin:0.5em 0}.block{margin:1.5em 0;padding:1em;border:1px solid #ddd;border-radius:4px}.block h3{font-size:1em;margin:0 0 0.5em}.block pre{white-space:pre-wrap;background:#f5f5f5;padding:0.5em;overflow:auto}@media print{.block{break-inside:avoid}}</style></head><body>")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(inv.Title))
	b.WriteString("</h1>")
	b.WriteString("<div class=\"meta\">")
	b.WriteString("Status: " + html.EscapeString(inv.Status) + " &middot; ")
	b.WriteString("Severity: " + html.EscapeString(inv.Severity) + " &middot; ")
	b.WriteString("Created: " + html.EscapeString(formatTime(inv.CreatedAt)))
	if inv.Summary != "" {
		b.WriteString("</div><p>")
		b.WriteString(html.EscapeString(inv.Summary))
		b.WriteString("</p>")
	} else {
		b.WriteString("</div>")
	}
	for _, blk := range blocks {
		b.WriteString("<div class=\"block\">")
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
	b.WriteString("<script>window.onload=function(){window.print()}</script></body></html>")
	return b.Bytes(), nil
}

func blockExportContent(blk *models.InvestigationBlock) (string, error) {
	switch blk.Type {
	case "summary", "markdown", "finding":
		var data map[string]interface{}
		if len(blk.DataJSON) > 0 {
			if err := json.Unmarshal(blk.DataJSON, &data); err != nil {
				return "", errors.Wrap(err, "data_json")
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
		var data map[string]interface{}
		if len(blk.DataJSON) > 0 {
			if err := json.Unmarshal(blk.DataJSON, &data); err != nil {
				return "", errors.Wrap(err, "data_json")
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
	default:
		// Generic: show data_json as formatted JSON or title
		if len(blk.DataJSON) > 0 {
			var raw map[string]interface{}
			if err := json.Unmarshal(blk.DataJSON, &raw); err != nil {
				return "<pre>" + html.EscapeString(string(blk.DataJSON)) + "</pre>", nil
			}
			content, _ := json.MarshalIndent(raw, "", "  ")
			return "<pre>" + html.EscapeString(string(content)) + "</pre>", nil
		}
		return "<p>(no data)</p>", nil
	}
}
