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

// AdreConversation is a persisted ADRE chat thread (PMM shell).
//
//reform:adre_conversations
type AdreConversation struct {
	ID            int64     `reform:"id,pk"`
	Title         string    `reform:"title"`
	CreatedBy     string    `reform:"created_by"`
	CreatedAt     time.Time `reform:"created_at"`
	UpdatedAt     time.Time `reform:"updated_at"`
	LastMessageAt time.Time `reform:"last_message_at"`
	MetadataJSON  []byte    `reform:"metadata_json"`
}

// AdreMessage is one row in an ADRE conversation (user, assistant, system, or tool).
// The content_tsv column is GENERATED STORED and is not mapped here (search uses raw SQL).
//
//reform:adre_messages
type AdreMessage struct {
	ID               int64     `reform:"id,pk"`
	ConversationID   int64     `reform:"conversation_id"`
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
	CreatedAt        time.Time `reform:"created_at"`
}
