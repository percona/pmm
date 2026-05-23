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
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/reform.v1"
)

// DefaultAdreChatTitle is the server default conversation title before the first user message.
const DefaultAdreChatTitle = "New chat"

// AdreTitleMaxRunes is the maximum title length in Unicode code points.
const AdreTitleMaxRunes = 50

// TruncateAdreTitle trims s to AdreTitleMaxRunes runes for titles and first-line auto-titles.
func TruncateAdreTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= AdreTitleMaxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:AdreTitleMaxRunes])
}

// GetAdreConversationOwned loads a conversation by id only if created_by matches.
// Returns (nil, nil) when the row is not found, matching the convention used
// throughout managed/models for nullable lookups.
func GetAdreConversationOwned(q *reform.DB, id int64, createdBy string) (*AdreConversation, error) {
	var c AdreConversation
	err := q.SelectOneTo(&c, "WHERE id = $1 AND created_by = $2", id, createdBy)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil //nolint:nilnil // "not found" sentinel matching managed/models convention
		}
		return nil, err
	}
	return &c, nil
}

// CreateAdreConversation inserts a new row; identity id is filled by reform/DB.
func CreateAdreConversation(q *reform.DB, c *AdreConversation) error {
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	c.LastMessageAt = now
	if c.MetadataJSON == nil {
		c.MetadataJSON = []byte("{}")
	}
	if strings.TrimSpace(c.Title) == "" {
		c.Title = DefaultAdreChatTitle
	}
	return q.Save(c)
}

// UpdateAdreConversation updates title and updated_at.
func UpdateAdreConversation(q *reform.DB, c *AdreConversation) error {
	c.UpdatedAt = time.Now().UTC()
	return q.Save(c)
}

// TouchAdreConversationLastMessage updates last_message_at and updated_at.
func TouchAdreConversationLastMessage(q reform.DBTX, id int64, at time.Time) error {
	_, err := q.Exec(
		`UPDATE adre_conversations SET last_message_at = $1, updated_at = $2 WHERE id = $3`,
		at.UTC(), time.Now().UTC(), id,
	)
	return err
}

// DeleteAdreConversation deletes a conversation owned by user (cascade messages).
func DeleteAdreConversation(q reform.DBTX, id int64, createdBy string) (bool, error) {
	res, err := q.Exec(`DELETE FROM adre_conversations WHERE id = $1 AND created_by = $2`, id, createdBy)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// AdreConversationListRow is one row for GET /v1/adre/conversations.
type AdreConversationListRow struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastMessageAt time.Time `json:"last_message_at"`
}

// ListAdreConversations returns conversations for a user, newest activity first, keyset pagination.
// TitleContains filters by case-insensitive substring on title when non-empty.
func ListAdreConversations(q reform.DBTX, createdBy, titleContains string, limit int, afterLastMsgAt *time.Time, afterID *int64) ([]AdreConversationListRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 { //nolint:mnd
		limit = 100
	}
	titleContains = strings.TrimSpace(titleContains)
	var rows *sql.Rows
	var err error
	if afterLastMsgAt == nil || afterID == nil {
		if titleContains == "" {
			rows, err = q.Query(`
				SELECT id, title, created_at, updated_at, last_message_at
				FROM adre_conversations
				WHERE created_by = $1
				ORDER BY last_message_at DESC, id DESC
				LIMIT $2`, createdBy, limit)
		} else {
			rows, err = q.Query(`
				SELECT id, title, created_at, updated_at, last_message_at
				FROM adre_conversations
				WHERE created_by = $1 AND title ILIKE '%' || $3 || '%'
				ORDER BY last_message_at DESC, id DESC
				LIMIT $2`, createdBy, limit, titleContains)
		}
	} else {
		if titleContains == "" {
			rows, err = q.Query(`
				SELECT id, title, created_at, updated_at, last_message_at
				FROM adre_conversations
				WHERE created_by = $1
				  AND (last_message_at, id) < ($2::timestamptz, $3::bigint)
				ORDER BY last_message_at DESC, id DESC
				LIMIT $4`, createdBy, *afterLastMsgAt, *afterID, limit)
		} else {
			rows, err = q.Query(`
				SELECT id, title, created_at, updated_at, last_message_at
				FROM adre_conversations
				WHERE created_by = $1 AND title ILIKE '%' || $4 || '%'
				  AND (last_message_at, id) < ($2::timestamptz, $3::bigint)
				ORDER BY last_message_at DESC, id DESC
				LIMIT $5`, createdBy, *afterLastMsgAt, *afterID, titleContains, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AdreConversationListRow
	for rows.Next() {
		var r AdreConversationListRow
		err := rows.Scan(&r.ID, &r.Title, &r.CreatedAt, &r.UpdatedAt, &r.LastMessageAt)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// EncodeAdreConversationCursor encodes keyset position for pagination.
func EncodeAdreConversationCursor(lastMsgAt time.Time, id int64) string {
	b, err := json.Marshal(map[string]any{
		"t":  lastMsgAt.UTC().Format(time.RFC3339Nano),
		"id": id,
	})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeAdreConversationCursor decodes cursor from EncodeAdreConversationCursor.
func DecodeAdreConversationCursor(s string) (*time.Time, *int64, error) {
	if s == "" {
		return nil, nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, nil, err
	}
	var m struct {
		T  string `json:"t"`
		ID int64  `json:"id"`
	}
	if err := json.Unmarshal(raw, &m); err != nil { //nolint:noinlineerr
		return nil, nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, m.T)
	if err != nil {
		return nil, nil, err
	}
	return &t, &m.ID, nil
}

// ListAdreMessages returns messages for a conversation in created_at order (oldest first).
func ListAdreMessages(q reform.DBTX, conversationID int64, beforeID *int64, afterID *int64, limit int) ([]AdreMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 { //nolint:mnd
		limit = 100
	}
	var rows *sql.Rows
	var err error
	switch {
	case beforeID != nil:
		rows, err = q.Query(`
			SELECT id, conversation_id, role, content, tool_name, tool_result_json, model,
			       prompt_tokens, completion_tokens, total_tokens, cached_tokens, total_cost, usage_event_id, created_at
			FROM adre_messages
			WHERE conversation_id = $1
			  AND (created_at, id) < (
				SELECT created_at, id FROM adre_messages WHERE id = $2 AND conversation_id = $1
			  )
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, conversationID, *beforeID, limit)
	case afterID != nil:
		rows, err = q.Query(`
			SELECT id, conversation_id, role, content, tool_name, tool_result_json, model,
			       prompt_tokens, completion_tokens, total_tokens, cached_tokens, total_cost, usage_event_id, created_at
			FROM adre_messages
			WHERE conversation_id = $1
			  AND (created_at, id) > (
				SELECT created_at, id FROM adre_messages WHERE id = $2 AND conversation_id = $1
			  )
			ORDER BY created_at ASC, id ASC
			LIMIT $3`, conversationID, *afterID, limit)
	default:
		rows, err = q.Query(`
			SELECT id, conversation_id, role, content, tool_name, tool_result_json, model,
			       prompt_tokens, completion_tokens, total_tokens, cached_tokens, total_cost, usage_event_id, created_at
			FROM adre_messages
			WHERE conversation_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`, conversationID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var list []AdreMessage
	for rows.Next() {
		var m AdreMessage
		err := rows.Scan(
			&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.ToolName, &m.ToolResultJSON, &m.Model,
			&m.PromptTokens, &m.CompletionTokens, &m.TotalTokens, &m.CachedTokens, &m.TotalCost, &m.UsageEventID, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil { //nolint:noinlineerr
		return nil, err
	}
	// Default branch returns newest-first; reverse to oldest-first for API consistency.
	if beforeID == nil && afterID == nil {
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
	}
	return list, nil
}

// LoadAdreMessagesForHolmesHistory returns messages with id strictly less than excludeFromID (exclusive), oldest first.
func LoadAdreMessagesForHolmesHistory(q reform.DBTX, conversationID int64, excludeFromID int64) ([]AdreMessage, error) {
	rows, err := q.Query(`
		SELECT id, conversation_id, role, content, tool_name, tool_result_json, model,
		       prompt_tokens, completion_tokens, total_tokens, cached_tokens, total_cost, usage_event_id, created_at
		FROM adre_messages
		WHERE conversation_id = $1 AND id < $2
		ORDER BY created_at ASC, id ASC`, conversationID, excludeFromID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var list []AdreMessage
	for rows.Next() {
		var m AdreMessage
		err := rows.Scan(
			&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.ToolName, &m.ToolResultJSON, &m.Model,
			&m.PromptTokens, &m.CompletionTokens, &m.TotalTokens, &m.CachedTokens, &m.TotalCost, &m.UsageEventID, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// PurgeAdreConversationsOlderThan deletes conversations whose last_message_at is before cutoff (retention job).
func PurgeAdreConversationsOlderThan(q reform.DBTX, cutoff time.Time) (int64, error) {
	res, err := q.Exec(`DELETE FROM adre_conversations WHERE last_message_at < $1`, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return n, err
}

// CreateAdreMessage inserts a message row (identity id assigned by DB).
func CreateAdreMessage(q *reform.DB, m *AdreMessage) error {
	m.CreatedAt = time.Now().UTC()
	return q.Save(m)
}

// CountAdreUserMessages returns how many user-role messages exist in a conversation.
func CountAdreUserMessages(q reform.DBTX, conversationID int64) (int, error) {
	var n int
	err := q.QueryRow(
		`SELECT COUNT(*) FROM adre_messages WHERE conversation_id = $1 AND role = 'user'`,
		conversationID,
	).Scan(&n)
	return n, err
}

// AdreSearchHit is one full-text search result row.
type AdreSearchHit struct {
	MessageID      int64     `json:"message_id"`
	ConversationID int64     `json:"conversation_id"`
	Role           string    `json:"role"`
	Snippet        string    `json:"snippet"`
	CreatedAt      time.Time `json:"created_at"`
}

// SearchAdreMessagesFTS runs FTS scoped to created_by (q must be non-empty after trim).
func SearchAdreMessagesFTS(q reform.DBTX, createdBy, qtext string, limit int) ([]AdreSearchHit, error) {
	qtext = strings.TrimSpace(qtext)
	if qtext == "" {
		return nil, errors.New("empty search query")
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 { //nolint:mnd
		limit = 100
	}
	rows, err := q.Query(`
		SELECT m.id, m.conversation_id, m.role, m.created_at,
			ts_headline('simple',
				coalesce(m.content,'') || ' ' || coalesce(m.tool_result_json::text,''),
				plainto_tsquery('simple', $3),
				'StartSel=<<, StopSel=>>, MaxFragments=3, MinWords=5, MaxWords=25'
			) AS headline
		FROM adre_messages m
		INNER JOIN adre_conversations c ON c.id = m.conversation_id
		WHERE c.created_by = $1
		  AND m.content_tsv @@ plainto_tsquery('simple', $3)
		ORDER BY ts_rank_cd(m.content_tsv, plainto_tsquery('simple', $3)) DESC, m.created_at DESC
		LIMIT $2`, createdBy, limit, qtext)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AdreSearchHit
	for rows.Next() {
		var h AdreSearchHit
		err := rows.Scan(&h.MessageID, &h.ConversationID, &h.Role, &h.CreatedAt, &h.Snippet)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
