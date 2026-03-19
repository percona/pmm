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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/percona/pmm/managed/models"
)

type serviceNowCreateRequest struct {
	ClientToken      string `json:"client_token"`
	ShortDescription string `json:"short_description"`
	Description      string `json:"description"`
	TicketType       string `json:"ticket_type"`
}

type serviceNowCreateResponse struct {
	Result struct {
		Success      bool   `json:"success"`
		TicketID     string `json:"ticket_id"`
		TableName    string `json:"table_name"`
		Message      string `json:"message"`
		ErrorMessage string `json:"error_message"`
	} `json:"result"`
}

type serviceNowDetailsRequest struct {
	ClientToken string `json:"client_token"`
	TicketID    string `json:"ticket_id"`
}

type serviceNowDetailsResponse struct {
	Result struct {
		Success       bool `json:"success"`
		TicketDetails struct {
			Number string `json:"number"`
			State  string `json:"state"`
		} `json:"ticket_details"`
		ErrorMessage string `json:"error_message"`
	} `json:"result"`
}

// deriveServiceNowInstanceURL extracts the base instance URL from the API URL
// (e.g. "https://perconadev.service-now.com/api/pellc/percona_connector/create" -> "https://perconadev.service-now.com").
func deriveServiceNowInstanceURL(apiURL string) string {
	u, err := url.Parse(apiURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// deriveTicketDetailsURL replaces "/create" with "/ticket_details" in the API URL.
func deriveTicketDetailsURL(createURL string) string {
	if i := strings.LastIndex(createURL, "/create"); i >= 0 {
		return createURL[:i] + "/ticket_details"
	}
	return ""
}

// fetchTicketNumber calls the ServiceNow /ticket_details endpoint to get the human-readable ticket number.
func fetchTicketNumber(ctx context.Context, detailsURL, apiKey, clientToken, ticketID string) (string, error) {
	payload, err := json.Marshal(serviceNowDetailsRequest{
		ClientToken: clientToken,
		TicketID:    ticketID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal details request: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, detailsURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build details request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-sn-apikey", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("details request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read details response: %w", err)
	}

	var detailsResp serviceNowDetailsResponse
	if err := json.Unmarshal(body, &detailsResp); err != nil {
		return "", fmt.Errorf("unmarshal details response: %w", err)
	}

	if !detailsResp.Result.Success {
		return "", fmt.Errorf("details error: %s", detailsResp.Result.ErrorMessage)
	}

	return detailsResp.Result.TicketDetails.Number, nil
}

var (
	mdCodeFenceRe  = regexp.MustCompile("(?s)```(?:[a-zA-Z]*)\n?(.*?)```")
	mdHeaderRe     = regexp.MustCompile(`(?m)^(#{1,4})\s+(.+)$`)
	mdInlineCodeRe = regexp.MustCompile("`([^`]+)`")
)

// markdownToServiceNowHTML converts markdown formatting to HTML suitable for ServiceNow description fields.
func markdownToServiceNowHTML(md string) string {
	s := md
	s = mdCodeFenceRe.ReplaceAllString(s, "<pre><code>$1</code></pre>")
	s = mdHeaderRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := mdHeaderRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		level := len(parts[1])
		if level < 1 {
			level = 1
		} else if level > 4 {
			level = 4
		}
		return fmt.Sprintf("<h%d>%s</h%d>", level, parts[2], level)
	})
	s = mdInlineCodeRe.ReplaceAllString(s, "<code>$1</code>")
	return s
}

// buildDescription assembles a markdown description from the investigation and its blocks.
func buildDescription(inv *models.Investigation, blocks []*models.InvestigationBlock) string {
	var sb strings.Builder

	if inv.Summary != "" {
		sb.WriteString("## Summary\n")
		sb.WriteString(inv.Summary)
		sb.WriteString("\n\n")
	}
	if inv.RootCauseSummary != "" {
		sb.WriteString("## Root Cause\n")
		sb.WriteString(inv.RootCauseSummary)
		sb.WriteString("\n\n")
	}
	if inv.ResolutionSummary != "" {
		sb.WriteString("## Resolution\n")
		sb.WriteString(inv.ResolutionSummary)
		sb.WriteString("\n\n")
	}
	if inv.SummaryDetailed != "" {
		sb.WriteString("## Detailed Summary\n")
		sb.WriteString(inv.SummaryDetailed)
		sb.WriteString("\n\n")
	}

	for _, b := range blocks {
		if b.Title != "" {
			sb.WriteString("## ")
			sb.WriteString(b.Title)
			sb.WriteString("\n")
		}
		if len(b.DataJSON) > 0 {
			var data map[string]interface{}
			if err := json.Unmarshal(b.DataJSON, &data); err == nil {
				if content, ok := data["content"].(string); ok && content != "" {
					sb.WriteString(content)
					sb.WriteString("\n\n")
					continue
				}
			}
			sb.WriteString(string(b.DataJSON))
			sb.WriteString("\n\n")
		}
	}

	return markdownToServiceNowHTML(strings.TrimSpace(sb.String()))
}

// PostServiceNowTicket handles POST /v1/investigations/:id/servicenow.
func (h *Handlers) PostServiceNowTicket(w http.ResponseWriter, r *http.Request, id string) {
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

	if inv.ServiceNowTicketID != "" {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("ServiceNow ticket already exists: %s", inv.ServiceNowTicketID))
		return
	}

	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to read settings")
		return
	}

	if settings.Adre.ServiceNowURL == "" || settings.Adre.ServiceNowAPIKey == "" || settings.Adre.ServiceNowClientToken == "" {
		writeJSONError(w, http.StatusBadRequest, "ServiceNow is not configured. Set URL, API key, and client token in AI Assistant settings.")
		return
	}

	blocks, err := models.GetInvestigationBlocks(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationBlocks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load investigation blocks")
		return
	}

	description := buildDescription(inv, blocks)
	shortDescription := inv.Title
	if shortDescription == "" {
		shortDescription = "PMM Investigation " + inv.ID
	}

	payload := serviceNowCreateRequest{
		ClientToken:      settings.Adre.ServiceNowClientToken,
		ShortDescription: shortDescription,
		Description:      description,
		TicketType:       "incident",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		h.l.Errorf("Marshal ServiceNow request: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to build request")
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, settings.Adre.ServiceNowURL, bytes.NewReader(body))
	if err != nil {
		h.l.Errorf("NewRequest: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to build HTTP request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-sn-apikey", settings.Adre.ServiceNowAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		h.l.Errorf("ServiceNow request failed: %v", err)
		writeJSONError(w, http.StatusBadGateway, "ServiceNow request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.l.Errorf("Read ServiceNow response: %v", err)
		writeJSONError(w, http.StatusBadGateway, "Failed to read ServiceNow response")
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		h.l.Errorf("ServiceNow returned %d: %s", resp.StatusCode, string(respBody))
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("ServiceNow returned HTTP %d", resp.StatusCode))
		return
	}

	var snResp serviceNowCreateResponse
	if err := json.Unmarshal(respBody, &snResp); err != nil {
		h.l.Errorf("Unmarshal ServiceNow response: %v", err)
		writeJSONError(w, http.StatusBadGateway, "Invalid ServiceNow response")
		return
	}

	if !snResp.Result.Success {
		errMsg := snResp.Result.ErrorMessage
		if errMsg == "" {
			errMsg = snResp.Result.Message
		}
		h.l.Errorf("ServiceNow error: %s", errMsg)
		writeJSONError(w, http.StatusBadGateway, "ServiceNow error: "+errMsg)
		return
	}

	inv.ServiceNowTicketID = snResp.Result.TicketID

	// Fetch ticket details to get the human-readable number (e.g. INC0289676)
	detailsURL := deriveTicketDetailsURL(settings.Adre.ServiceNowURL)
	if detailsURL != "" {
		number, err := fetchTicketNumber(r.Context(), detailsURL, settings.Adre.ServiceNowAPIKey, settings.Adre.ServiceNowClientToken, snResp.Result.TicketID)
		if err != nil {
			h.l.Warnf("Failed to fetch ticket number (ticket created OK): %v", err)
		} else if number != "" {
			inv.ServiceNowTicketNumber = number
		}
	}

	if err := models.UpdateInvestigation(h.db, inv); err != nil {
		h.l.Errorf("UpdateInvestigation (ticket ID): %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Ticket created but failed to save ticket ID")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"ticket_id":     snResp.Result.TicketID,
		"ticket_number": inv.ServiceNowTicketNumber,
		"message":       snResp.Result.Message,
	})
}
