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
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const prefix = "/v1/investigations"

// Handlers provides HTTP handlers for the Investigations API.
type Handlers struct {
	db *reform.DB
	l  *logrus.Entry
}

// NewHandlers creates new Investigations HTTP handlers.
func NewHandlers(db *reform.DB) *Handlers {
	return &Handlers{db: db, l: logrus.WithField("component", "investigations-handlers")}
}

// ServeHTTP dispatches all /v1/investigations/* routes.
func (h *Handlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	path = strings.Trim(path, "/")
	segments := strings.Split(path, "/")

	switch {
	case path == "":
		switch r.Method {
		case http.MethodGet:
			h.ListInvestigations(w, r)
		case http.MethodPost:
			h.CreateInvestigation(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	case len(segments) >= 1 && segments[0] != "":
		id := segments[0]
		switch {
		case len(segments) == 1:
			switch r.Method {
			case http.MethodGet:
				h.GetInvestigation(w, r, id)
			case http.MethodPatch:
				h.PatchInvestigation(w, r, id)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		case len(segments) == 2 && segments[1] == "blocks":
			switch r.Method {
			case http.MethodGet:
				h.GetInvestigationBlocks(w, r, id)
			case http.MethodPost:
				h.PostInvestigationBlock(w, r, id)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		case len(segments) == 3 && segments[1] == "blocks":
			blockID := segments[2]
			switch r.Method {
			case http.MethodPatch:
				h.PatchInvestigationBlock(w, r, id, blockID)
			case http.MethodDelete:
				h.DeleteInvestigationBlock(w, r, id, blockID)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		case len(segments) == 2 && segments[1] == "timeline":
			switch r.Method {
			case http.MethodGet:
				h.GetInvestigationTimeline(w, r, id)
			case http.MethodPost:
				h.PostInvestigationTimeline(w, r, id)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		case len(segments) == 2 && segments[1] == "artifacts":
			switch r.Method {
			case http.MethodGet:
				h.GetInvestigationArtifacts(w, r, id)
			case http.MethodPost:
				h.PostInvestigationArtifact(w, r, id)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		case len(segments) == 2 && segments[1] == "comments":
			switch r.Method {
			case http.MethodGet:
				h.GetInvestigationComments(w, r, id)
			case http.MethodPost:
				h.PostInvestigationComment(w, r, id)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
			return
		case len(segments) == 2 && segments[1] == "messages":
			if r.Method == http.MethodGet {
				h.GetInvestigationMessages(w, r, id)
				return
			}
		case len(segments) == 2 && segments[1] == "chat":
			if r.Method == http.MethodPost {
				h.PostInvestigationChat(w, r, id)
				return
			}
		case len(segments) == 2 && segments[1] == "run":
			if r.Method == http.MethodPost {
				h.PostInvestigationRun(w, r, id)
				return
			}
		}
	}
	writeJSONError(w, http.StatusNotFound, "Not Found")
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *Handlers) ListInvestigations(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	list, err := models.ListInvestigations(h.db, status, limit, offset)
	if err != nil {
		h.l.Errorf("ListInvestigations: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to list investigations")
		return
	}
	type item struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		TimeFrom  string `json:"time_from,omitempty"`
		TimeTo    string `json:"time_to,omitempty"`
	}
	out := make([]item, len(list))
	for i, inv := range list {
		out[i] = item{
			ID:        inv.ID,
			Title:     inv.Title,
			Status:    inv.Status,
			CreatedAt: inv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: inv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			TimeFrom:  inv.TimeFrom.Format("2006-01-02T15:04:05Z07:00"),
			TimeTo:    inv.TimeTo.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handlers) CreateInvestigation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title      string `json:"title"`
		TimeFrom   string `json:"time_from"`
		TimeTo     string `json:"time_to"`
		SourceType string `json:"source_type"`
		SourceRef  string `json:"source_ref"`
		Summary    string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	now := time.Now().UTC()
	timeFrom := now
	timeTo := now
	if body.TimeFrom != "" {
		t, err := parseTime(body.TimeFrom)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "time_from: "+err.Error())
			return
		}
		timeFrom = t
	}
	if body.TimeTo != "" {
		t, err := parseTime(body.TimeTo)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "time_to: "+err.Error())
			return
		}
		timeTo = t
	}
	inv := &models.Investigation{
		ID:         models.NewInvestigationID(),
		Title:      body.Title,
		Status:     "open",
		TimeFrom:   timeFrom,
		TimeTo:     timeTo,
		Summary:    body.Summary,
		SourceType: body.SourceType,
		SourceRef:  body.SourceRef,
	}
	if inv.SourceType == "" {
		inv.SourceType = "manual"
	}
	if err := models.CreateInvestigation(h.db, inv); err != nil {
		h.l.Errorf("CreateInvestigation: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create investigation")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(investigationToResponse(inv))
}

func (h *Handlers) GetInvestigation(w http.ResponseWriter, r *http.Request, id string) {
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
	blocks, err := models.GetInvestigationBlocks(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationBlocks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get blocks")
		return
	}
	resp := investigationToResponse(inv)
	resp.Blocks = blocksToResponse(blocks)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) PatchInvestigation(w http.ResponseWriter, r *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			writeJSONError(w, http.StatusNotFound, "Investigation not found")
			return
		}
		h.l.Errorf("GetInvestigationByID: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	var body struct {
		Title              *string `json:"title"`
		Status             *string `json:"status"`
		Summary            *string `json:"summary"`
		SummaryDetailed    *string `json:"summary_detailed"`
		RootCauseSummary   *string `json:"root_cause_summary"`
		ResolutionSummary  *string `json:"resolution_summary"`
		Severity           *string `json:"severity"`
		TimeFrom           *string `json:"time_from"`
		TimeTo             *string `json:"time_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Title != nil {
		inv.Title = *body.Title
	}
	if body.Status != nil {
		inv.Status = *body.Status
	}
	if body.Summary != nil {
		inv.Summary = *body.Summary
	}
	if body.SummaryDetailed != nil {
		inv.SummaryDetailed = *body.SummaryDetailed
	}
	if body.RootCauseSummary != nil {
		inv.RootCauseSummary = *body.RootCauseSummary
	}
	if body.ResolutionSummary != nil {
		inv.ResolutionSummary = *body.ResolutionSummary
	}
	if body.Severity != nil {
		inv.Severity = *body.Severity
	}
	if body.TimeFrom != nil {
		if *body.TimeFrom == "" {
			inv.TimeFrom = time.Now().UTC()
		} else {
			t, err := parseTime(*body.TimeFrom)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "time_from: "+err.Error())
				return
			}
			inv.TimeFrom = t
		}
	}
	if body.TimeTo != nil {
		if *body.TimeTo == "" {
			inv.TimeTo = time.Now().UTC()
		} else {
			t, err := parseTime(*body.TimeTo)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "time_to: "+err.Error())
				return
			}
			inv.TimeTo = t
		}
	}
	if err := models.UpdateInvestigation(h.db, inv); err != nil {
		h.l.Errorf("UpdateInvestigation: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to update investigation")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(investigationToResponse(inv))
}

func (h *Handlers) GetInvestigationBlocks(w http.ResponseWriter, r *http.Request, id string) {
	inv, _ := models.GetInvestigationByID(h.db, id)
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	blocks, err := models.GetInvestigationBlocks(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationBlocks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get blocks")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(blocksToResponse(blocks))
}

func (h *Handlers) PostInvestigationBlock(w http.ResponseWriter, r *http.Request, id string) {
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
	var body struct {
		Type       string          `json:"type"`
		Title      string          `json:"title"`
		Position   int             `json:"position"`
		ConfigJSON json.RawMessage `json:"config_json"`
		DataJSON   json.RawMessage `json:"data_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "type is required")
		return
	}
	block := &models.InvestigationBlock{
		ID:              models.NewInvestigationID(),
		InvestigationID:  id,
		Type:            body.Type,
		Title:           body.Title,
		Position:        body.Position,
		ConfigJSON:      body.ConfigJSON,
		DataJSON:        body.DataJSON,
	}
	if err := models.CreateInvestigationBlock(h.db, block); err != nil {
		h.l.Errorf("CreateInvestigationBlock: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create block")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(blockToResponse(block))
}

func (h *Handlers) PatchInvestigationBlock(w http.ResponseWriter, r *http.Request, id, blockID string) {
	block, err := getBlockAndCheckInvestigation(h.db, id, blockID)
	if err != nil {
		writeJSONError(w, err.status, err.msg)
		return
	}
	var body struct {
		Type       *string         `json:"type"`
		Title      *string         `json:"title"`
		Position   *int            `json:"position"`
		ConfigJSON json.RawMessage `json:"config_json"`
		DataJSON   json.RawMessage `json:"data_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Type != nil {
		block.Type = *body.Type
	}
	if body.Title != nil {
		block.Title = *body.Title
	}
	if body.Position != nil {
		block.Position = *body.Position
	}
	if body.ConfigJSON != nil {
		block.ConfigJSON = body.ConfigJSON
	}
	if body.DataJSON != nil {
		block.DataJSON = body.DataJSON
	}
	if err := models.UpdateInvestigationBlock(h.db, block); err != nil {
		h.l.Errorf("UpdateInvestigationBlock: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to update block")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(blockToResponse(block))
}

func (h *Handlers) DeleteInvestigationBlock(w http.ResponseWriter, r *http.Request, id, blockID string) {
	_, err := getBlockAndCheckInvestigation(h.db, id, blockID)
	if err != nil {
		writeJSONError(w, err.status, err.msg)
		return
	}
	if err := models.DeleteInvestigationBlock(h.db, blockID); err != nil {
		h.l.Errorf("DeleteInvestigationBlock: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete block")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetInvestigationTimeline(w http.ResponseWriter, r *http.Request, id string) {
	inv, _ := models.GetInvestigationByID(h.db, id)
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	events, err := models.GetInvestigationTimelineEvents(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationTimelineEvents: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get timeline")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(timelineToResponse(events))
}

func (h *Handlers) PostInvestigationTimeline(w http.ResponseWriter, r *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			writeJSONError(w, http.StatusNotFound, "Investigation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	var body struct {
		EventTime   string          `json:"event_time"`
		Type        string          `json:"type"`
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Source      string          `json:"source"`
		MetadataJSON json.RawMessage `json:"metadata_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.EventTime == "" || body.Type == "" || body.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "event_time, type, and title are required")
		return
	}
	t, err := parseTime(body.EventTime)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "event_time: "+err.Error())
		return
	}
	event := &models.InvestigationTimelineEvent{
		ID:              models.NewInvestigationID(),
		InvestigationID:  id,
		EventTime:       t,
		Type:            body.Type,
		Title:           body.Title,
		Description:     body.Description,
		Source:          body.Source,
		MetadataJSON:    body.MetadataJSON,
	}
	if err := models.CreateInvestigationTimelineEvent(h.db, event); err != nil {
		h.l.Errorf("CreateInvestigationTimelineEvent: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create timeline event")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(timelineEventToResponse(event))
}

func (h *Handlers) GetInvestigationArtifacts(w http.ResponseWriter, r *http.Request, id string) {
	inv, _ := models.GetInvestigationByID(h.db, id)
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	artifacts, err := models.GetInvestigationArtifacts(h.db, id)
	if err != nil {
		h.l.Errorf("GetInvestigationArtifacts: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get artifacts")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(artifactsToResponse(artifacts))
}

func (h *Handlers) PostInvestigationArtifact(w http.ResponseWriter, r *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			writeJSONError(w, http.StatusNotFound, "Investigation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	var body struct {
		Type         string          `json:"type"`
		URIOrBlobRef string          `json:"uri_or_blob_ref"`
		Source       string          `json:"source"`
		MetadataJSON  json.RawMessage `json:"metadata_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Type == "" || body.URIOrBlobRef == "" {
		writeJSONError(w, http.StatusBadRequest, "type and uri_or_blob_ref are required")
		return
	}
	artifact := &models.InvestigationArtifact{
		ID:              models.NewInvestigationID(),
		InvestigationID:  id,
		Type:            body.Type,
		URIOrBlobRef:    body.URIOrBlobRef,
		Source:          body.Source,
		MetadataJSON:    body.MetadataJSON,
	}
	if err := models.CreateInvestigationArtifact(h.db, artifact); err != nil {
		h.l.Errorf("CreateInvestigationArtifact: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create artifact")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(artifactToResponse(artifact))
}

func (h *Handlers) GetInvestigationComments(w http.ResponseWriter, r *http.Request, id string) {
	inv, _ := models.GetInvestigationByID(h.db, id)
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	blockID := r.URL.Query().Get("block_id")
	var filter *string
	if blockID != "" {
		filter = &blockID
	}
	comments, err := models.GetInvestigationComments(h.db, id, filter)
	if err != nil {
		h.l.Errorf("GetInvestigationComments: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get comments")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(commentsToResponse(comments))
}

func (h *Handlers) PostInvestigationComment(w http.ResponseWriter, r *http.Request, id string) {
	inv, err := models.GetInvestigationByID(h.db, id)
	if err != nil || inv == nil {
		if inv == nil {
			writeJSONError(w, http.StatusNotFound, "Investigation not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to get investigation")
		return
	}
	var body struct {
		Content    string          `json:"content"`
		BlockID    *string         `json:"block_id"`
		AnchorJSON json.RawMessage `json:"anchor_json"`
		Author     string          `json:"author"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if body.Content == "" {
		writeJSONError(w, http.StatusBadRequest, "content is required")
		return
	}
	c := &models.InvestigationComment{
		ID:              models.NewInvestigationID(),
		InvestigationID:  id,
		BlockID:         body.BlockID,
		AnchorJSON:      body.AnchorJSON,
		Author:          body.Author,
		Content:         body.Content,
	}
	if err := models.CreateInvestigationComment(h.db, c); err != nil {
		h.l.Errorf("CreateInvestigationComment: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create comment")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(commentToResponse(c))
}

func (h *Handlers) GetInvestigationMessages(w http.ResponseWriter, r *http.Request, id string) {
	inv, _ := models.GetInvestigationByID(h.db, id)
	if inv == nil {
		writeJSONError(w, http.StatusNotFound, "Investigation not found")
		return
	}
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	messages, err := models.GetInvestigationMessages(h.db, id, limit, offset)
	if err != nil {
		h.l.Errorf("GetInvestigationMessages: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get messages")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(messagesToResponse(messages))
}
