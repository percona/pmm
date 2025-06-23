package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

//go:generate ../../../bin/reform

// ToolCallsField represents JSONB field for tool calls
type ToolCallsField []ToolCall

// Value implements database/sql/driver.Valuer interface.
func (t ToolCallsField) Value() (driver.Value, error) {
	if len(t) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tool calls")
	}
	return b, nil
}

// Scan implements database/sql.Scanner interface.
func (t *ToolCallsField) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.Errorf("expected []byte or string, got %T", src)
	}

	if err := json.Unmarshal(b, t); err != nil {
		return errors.Wrap(err, "failed to unmarshal tool calls")
	}
	return nil
}

// ToolExecutionsField represents JSONB field for tool executions
type ToolExecutionsField []ToolExecution

// Value implements database/sql/driver.Valuer interface.
func (t ToolExecutionsField) Value() (driver.Value, error) {
	if len(t) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tool executions")
	}
	return b, nil
}

// Scan implements database/sql.Scanner interface.
func (t *ToolExecutionsField) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.Errorf("expected []byte or string, got %T", src)
	}

	if err := json.Unmarshal(b, t); err != nil {
		return errors.Wrap(err, "failed to unmarshal tool executions")
	}
	return nil
}

// ApprovalRequestField represents JSONB field for approval requests
type ApprovalRequestField struct {
	RequestID string     `json:"request_id"`
	ToolCalls []ToolCall `json:"tool_calls"`
	Processed bool       `json:"processed,omitempty"`
}

// Value implements database/sql/driver.Valuer interface.
func (a ApprovalRequestField) Value() (driver.Value, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal approval request")
	}
	return b, nil
}

// Scan implements database/sql.Scanner interface.
func (a *ApprovalRequestField) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.Errorf("expected []byte or string, got %T", src)
	}

	if err := json.Unmarshal(b, a); err != nil {
		return errors.Wrap(err, "failed to unmarshal approval request")
	}
	return nil
}

// ChatSession represents a chat session.
//
//reform:chat_sessions
type ChatSession struct {
	ID        string    `reform:"id,pk" json:"id"`
	UserID    string    `reform:"user_id" json:"user_id"`
	Title     string    `reform:"title" json:"title"`
	CreatedAt time.Time `reform:"created_at" json:"created_at"`
	UpdatedAt time.Time `reform:"updated_at" json:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *ChatSession) BeforeInsert() error {
	now := time.Now().UTC()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *ChatSession) BeforeUpdate() error {
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *ChatSession) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	return nil
}

// ChatMessage represents a chat message.
//
//reform:chat_messages
type ChatMessage struct {
	ID              string                `reform:"id,pk" json:"id"`
	SessionID       string                `reform:"session_id" json:"session_id"`
	Role            string                `reform:"role" json:"role"`
	Content         string                `reform:"content" json:"content"`
	ToolCalls       *ToolCallsField       `reform:"tool_calls" json:"tool_calls,omitempty"`
	ToolExecutions  *ToolExecutionsField  `reform:"tool_executions" json:"tool_executions,omitempty"`
	ApprovalRequest *ApprovalRequestField `reform:"approval_request" json:"approval_request,omitempty"`
	CreatedAt       time.Time             `reform:"created_at" json:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (m *ChatMessage) BeforeInsert() error {
	m.CreatedAt = time.Now().UTC()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (m *ChatMessage) AfterFind() error {
	m.CreatedAt = m.CreatedAt.UTC()
	return nil
}

// ChatAttachment represents a file attachment.
//
//reform:chat_attachments
type ChatAttachment struct {
	ID        string    `reform:"id,pk" json:"id"`
	MessageID string    `reform:"message_id" json:"message_id"`
	Filename  string    `reform:"filename" json:"filename"`
	MimeType  string    `reform:"mime_type" json:"mime_type"`
	Size      int64     `reform:"size" json:"size"`
	Content   string    `reform:"content" json:"content"`
	CreatedAt time.Time `reform:"created_at" json:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (a *ChatAttachment) BeforeInsert() error {
	a.CreatedAt = time.Now().UTC()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (a *ChatAttachment) AfterFind() error {
	a.CreatedAt = a.CreatedAt.UTC()
	return nil
}

// Check interfaces
var (
	_ reform.BeforeInserter = (*ChatSession)(nil)
	_ reform.BeforeUpdater  = (*ChatSession)(nil)
	_ reform.AfterFinder    = (*ChatSession)(nil)
	_ reform.BeforeInserter = (*ChatMessage)(nil)
	_ reform.AfterFinder    = (*ChatMessage)(nil)
	_ reform.BeforeInserter = (*ChatAttachment)(nil)
	_ reform.AfterFinder    = (*ChatAttachment)(nil)
	_ driver.Valuer         = (*ToolCallsField)(nil)
	_ driver.Valuer         = (*ToolExecutionsField)(nil)
	_ driver.Valuer         = (*ApprovalRequestField)(nil)
)
