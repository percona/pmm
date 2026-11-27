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

package models

import (
	"time"

	"github.com/google/uuid"
)

//go:generate go tool reform

// Investigation represents an incident investigation (report) as stored in the database.
//
//reform:investigations
type Investigation struct {
	ID                     string    `reform:"id,pk"`
	Title                  string    `reform:"title"`
	Status                 string    `reform:"status"`
	Severity               string    `reform:"severity"`
	CreatedAt              time.Time `reform:"created_at"`
	UpdatedAt              time.Time `reform:"updated_at"`
	CreatedBy              string    `reform:"created_by"`
	TimeFrom               time.Time `reform:"time_from"`
	TimeTo                 time.Time `reform:"time_to"`
	Summary                string    `reform:"summary"`
	SummaryDetailed        string    `reform:"summary_detailed"`
	RootCauseSummary       string    `reform:"root_cause_summary"`
	ResolutionSummary      string    `reform:"resolution_summary"`
	SourceType             string    `reform:"source_type"`
	SourceRef              string    `reform:"source_ref"`
	AlertFingerprint       string    `reform:"alert_fingerprint"`
	Tags                   []byte    `reform:"tags"`
	Config                 []byte    `reform:"config"`
	ServiceNowTicketID     string    `reform:"servicenow_ticket_id"`
	ServiceNowTicketNumber string    `reform:"servicenow_ticket_number"`
	HolmesTotalTokens      int64     `reform:"holmes_total_tokens"`
	HolmesTotalCost        float64   `reform:"holmes_total_cost"`
	HolmesCallCount        int       `reform:"holmes_call_count"`
}

// InvestigationBlock represents a block (section) within an investigation report.
//
//reform:investigation_blocks
type InvestigationBlock struct {
	ID              string    `reform:"id,pk"`
	InvestigationID string    `reform:"investigation_id"`
	Type            string    `reform:"type"`
	Title           string    `reform:"title"`
	Position        int       `reform:"position"`
	ConfigJSON      []byte    `reform:"config_json"`
	DataJSON        []byte    `reform:"data_json"`
	CreatedAt       time.Time `reform:"created_at"`
	UpdatedAt       time.Time `reform:"updated_at"`
	CreatedBy       string    `reform:"created_by"`
	UpdatedBy       string    `reform:"updated_by"`
}

// InvestigationArtifact represents an artifact (snapshot, log excerpt, etc.) linked to an investigation.
//
//reform:investigation_artifacts
type InvestigationArtifact struct {
	ID              string    `reform:"id,pk"`
	InvestigationID string    `reform:"investigation_id"`
	Type            string    `reform:"type"`
	URIOrBlobRef    string    `reform:"uri_or_blob_ref"`
	Source          string    `reform:"source"`
	MetadataJSON    []byte    `reform:"metadata_json"`
	CreatedAt       time.Time `reform:"created_at"`
}

// InvestigationMessage represents a chat message (user, assistant, or tool) in an investigation.
//
//reform:investigation_messages
type InvestigationMessage struct {
	ID               string    `reform:"id,pk"`
	InvestigationID  string    `reform:"investigation_id"`
	Role             string    `reform:"role"`
	Content          string    `reform:"content"`
	ToolName         string    `reform:"tool_name"`
	ToolResultJSON   []byte    `reform:"tool_result_json"`
	Model            string    `reform:"model"`
	PromptTokens     *int32    `reform:"prompt_tokens"`
	CompletionTokens *int32    `reform:"completion_tokens"`
	TotalTokens      *int32    `reform:"total_tokens"`
	CachedTokens     *int32    `reform:"cached_tokens"`
	TotalCost        *float64  `reform:"total_cost"`
	UsageEventID     *int64    `reform:"usage_event_id"`
	HolmesFeature    string    `reform:"holmes_feature"`
	CreatedAt        time.Time `reform:"created_at"`
}

// InvestigationComment represents a comment on an investigation or a block.
//
//reform:investigation_comments
type InvestigationComment struct {
	ID              string    `reform:"id,pk"`
	InvestigationID string    `reform:"investigation_id"`
	BlockID         *string   `reform:"block_id"`
	AnchorJSON      []byte    `reform:"anchor_json"`
	Author          string    `reform:"author"`
	Content         string    `reform:"content"`
	CreatedAt       time.Time `reform:"created_at"`
	UpdatedAt       time.Time `reform:"updated_at"`
}

// InvestigationTimelineEvent represents a timeline event in an investigation.
//
//reform:investigation_timeline_events
type InvestigationTimelineEvent struct {
	ID              string    `reform:"id,pk"`
	InvestigationID string    `reform:"investigation_id"`
	EventTime       time.Time `reform:"event_time"`
	Type            string    `reform:"type"`
	Title           string    `reform:"title"`
	Description     string    `reform:"description"`
	Source          string    `reform:"source"`
	MetadataJSON    []byte    `reform:"metadata_json"`
}

// NewInvestigationID returns a new UUID string for investigations and related entities.
func NewInvestigationID() string {
	return uuid.New().String()
}
