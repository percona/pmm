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
)

//go:generate go tool reform

// HolmesUsageEvent is one Holmes /api/chat completion recorded for usage tracking.
//
//reform:holmes_usage_events
type HolmesUsageEvent struct {
	ID                 int64     `reform:"id,pk"`
	CreatedAt          time.Time `reform:"created_at"`
	Feature            string    `reform:"feature"`
	FeatureRef         string    `reform:"feature_ref"`
	AdreConversationID *int64    `reform:"adre_conversation_id"`
	InvestigationID    string    `reform:"investigation_id"`
	Model              string    `reform:"model"`
	PromptTokens       *int32    `reform:"prompt_tokens"`
	CompletionTokens   *int32    `reform:"completion_tokens"`
	TotalTokens        *int32    `reform:"total_tokens"`
	CachedTokens       *int32    `reform:"cached_tokens"`
	TotalCost          *float64  `reform:"total_cost"`
	CostPrompt         *float64  `reform:"cost_prompt"`
	CostCompletion     *float64  `reform:"cost_completion"`
	CostCached         *float64  `reform:"cost_cached"`
	LatencyMs          *int32    `reform:"latency_ms"`
	TriggeredBy        string    `reform:"triggered_by"`
	Stream             bool      `reform:"stream"`
	MetadataJSON       []byte    `reform:"metadata_json"`
}
