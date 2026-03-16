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
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func formatTime(t time.Time) string {
	return t.Format(timeFormat)
}

type httpError struct {
	status int
	msg    string
}

func getBlockAndCheckInvestigation(db *reform.DB, investigationID, blockID string) (*models.InvestigationBlock, *httpError) {
	inv, err := models.GetInvestigationByID(db, investigationID)
	if err != nil {
		return nil, &httpError{http.StatusInternalServerError, "Failed to get investigation"}
	}
	if inv == nil {
		return nil, &httpError{http.StatusNotFound, "Investigation not found"}
	}
	var block models.InvestigationBlock
	if err := db.FindByPrimaryKeyTo(&block, blockID); err != nil {
		if errors.As(err, &reform.ErrNoRows) {
			return nil, &httpError{http.StatusNotFound, "Block not found"}
		}
		return nil, &httpError{http.StatusInternalServerError, "Failed to get block"}
	}
	if block.InvestigationID != investigationID {
		return nil, &httpError{http.StatusNotFound, "Block not found"}
	}
	return &block, nil
}

// Response DTOs and conversion helpers.

type investigationResponse struct {
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	Status            string          `json:"status"`
	Severity          string          `json:"severity"`
	CreatedAt         string          `json:"created_at"`
	UpdatedAt         string          `json:"updated_at"`
	CreatedBy         string          `json:"created_by"`
	TimeFrom          string          `json:"time_from"`
	TimeTo            string          `json:"time_to"`
	Summary           string          `json:"summary"`
	SummaryDetailed   string          `json:"summary_detailed"`
	RootCauseSummary  string          `json:"root_cause_summary"`
	ResolutionSummary string          `json:"resolution_summary"`
	SourceType        string          `json:"source_type"`
	SourceRef         string          `json:"source_ref"`
	NodeName          string          `json:"node_name,omitempty"`
	ServiceName       string          `json:"service_name,omitempty"`
	ClusterName       string          `json:"cluster_name,omitempty"`
	Blocks            []blockResponse `json:"blocks,omitempty"`
}

func investigationToResponse(inv *models.Investigation) investigationResponse {
	resp := investigationResponse{
		ID:                inv.ID,
		Title:             inv.Title,
		Status:            inv.Status,
		Severity:          inv.Severity,
		CreatedAt:         formatTime(inv.CreatedAt),
		UpdatedAt:         formatTime(inv.UpdatedAt),
		CreatedBy:         inv.CreatedBy,
		TimeFrom:          formatTime(inv.TimeFrom),
		TimeTo:            formatTime(inv.TimeTo),
		Summary:           inv.Summary,
		SummaryDetailed:   inv.SummaryDetailed,
		RootCauseSummary:  inv.RootCauseSummary,
		ResolutionSummary: inv.ResolutionSummary,
		SourceType:        inv.SourceType,
		SourceRef:         inv.SourceRef,
	}
	if len(inv.Config) > 0 {
		var cfg map[string]string
		if err := json.Unmarshal(inv.Config, &cfg); err == nil {
			if v := cfg["node_name"]; v != "" {
				resp.NodeName = v
			}
			if v := cfg["service_name"]; v != "" {
				resp.ServiceName = v
			}
			if v := cfg["cluster_name"]; v != "" {
				resp.ClusterName = v
			}
		}
	}
	return resp
}

type blockResponse struct {
	ID              string          `json:"id"`
	InvestigationID string          `json:"investigation_id"`
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	Position        int             `json:"position"`
	ConfigJSON      json.RawMessage `json:"config_json,omitempty"`
	DataJSON        json.RawMessage `json:"data_json,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

func blockToResponse(b *models.InvestigationBlock) blockResponse {
	return blockResponse{
		ID:              b.ID,
		InvestigationID: b.InvestigationID,
		Type:            b.Type,
		Title:           b.Title,
		Position:        b.Position,
		ConfigJSON:      b.ConfigJSON,
		DataJSON:        b.DataJSON,
		CreatedAt:       formatTime(b.CreatedAt),
		UpdatedAt:       formatTime(b.UpdatedAt),
	}
}

func blocksToResponse(blocks []*models.InvestigationBlock) []blockResponse {
	out := make([]blockResponse, len(blocks))
	for i, b := range blocks {
		out[i] = blockToResponse(b)
	}
	return out
}

type timelineEventResponse struct {
	ID              string          `json:"id"`
	InvestigationID string          `json:"investigation_id"`
	EventTime       string          `json:"event_time"`
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Source          string          `json:"source"`
	MetadataJSON    json.RawMessage `json:"metadata_json,omitempty"`
}

func timelineEventToResponse(e *models.InvestigationTimelineEvent) timelineEventResponse {
	return timelineEventResponse{
		ID:              e.ID,
		InvestigationID: e.InvestigationID,
		EventTime:       formatTime(e.EventTime),
		Type:            e.Type,
		Title:           e.Title,
		Description:     e.Description,
		Source:          e.Source,
		MetadataJSON:    e.MetadataJSON,
	}
}

func timelineToResponse(events []*models.InvestigationTimelineEvent) []timelineEventResponse {
	out := make([]timelineEventResponse, len(events))
	for i, e := range events {
		out[i] = timelineEventToResponse(e)
	}
	return out
}

type artifactResponse struct {
	ID              string          `json:"id"`
	InvestigationID string          `json:"investigation_id"`
	Type            string          `json:"type"`
	URIOrBlobRef    string          `json:"uri_or_blob_ref"`
	Source          string          `json:"source"`
	MetadataJSON    json.RawMessage `json:"metadata_json,omitempty"`
	CreatedAt       string          `json:"created_at"`
}

func artifactToResponse(a *models.InvestigationArtifact) artifactResponse {
	return artifactResponse{
		ID:              a.ID,
		InvestigationID: a.InvestigationID,
		Type:            a.Type,
		URIOrBlobRef:    a.URIOrBlobRef,
		Source:          a.Source,
		MetadataJSON:    a.MetadataJSON,
		CreatedAt:       formatTime(a.CreatedAt),
	}
}

func artifactsToResponse(artifacts []*models.InvestigationArtifact) []artifactResponse {
	out := make([]artifactResponse, len(artifacts))
	for i, a := range artifacts {
		out[i] = artifactToResponse(a)
	}
	return out
}

type commentResponse struct {
	ID              string          `json:"id"`
	InvestigationID string          `json:"investigation_id"`
	BlockID         *string         `json:"block_id,omitempty"`
	AnchorJSON      json.RawMessage `json:"anchor_json,omitempty"`
	Author          string          `json:"author"`
	Content         string          `json:"content"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

func commentToResponse(c *models.InvestigationComment) commentResponse {
	return commentResponse{
		ID:              c.ID,
		InvestigationID: c.InvestigationID,
		BlockID:         c.BlockID,
		AnchorJSON:      c.AnchorJSON,
		Author:          c.Author,
		Content:         c.Content,
		CreatedAt:       formatTime(c.CreatedAt),
		UpdatedAt:       formatTime(c.UpdatedAt),
	}
}

func commentsToResponse(comments []*models.InvestigationComment) []commentResponse {
	out := make([]commentResponse, len(comments))
	for i, c := range comments {
		out[i] = commentToResponse(c)
	}
	return out
}

type messageResponse struct {
	ID              string          `json:"id"`
	InvestigationID string          `json:"investigation_id"`
	Role            string          `json:"role"`
	Content         string          `json:"content"`
	ToolName        string          `json:"tool_name,omitempty"`
	ToolResultJSON  json.RawMessage `json:"tool_result_json,omitempty"`
	CreatedAt       string          `json:"created_at"`
}

func messageToResponse(m *models.InvestigationMessage) messageResponse {
	return messageResponse{
		ID:              m.ID,
		InvestigationID: m.InvestigationID,
		Role:            m.Role,
		Content:         m.Content,
		ToolName:        m.ToolName,
		ToolResultJSON:  m.ToolResultJSON,
		CreatedAt:       formatTime(m.CreatedAt),
	}
}

func messagesToResponse(messages []*models.InvestigationMessage) []messageResponse {
	out := make([]messageResponse, len(messages))
	for i, m := range messages {
		out[i] = messageToResponse(m)
	}
	return out
}
