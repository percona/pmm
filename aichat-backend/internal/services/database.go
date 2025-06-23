package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

// DatabaseService handles database operations for chat sessions
type DatabaseService struct {
	db *sql.DB
}

// NewDatabaseService creates a new database service
func NewDatabaseService(db *sql.DB) *DatabaseService {
	return &DatabaseService{
		db: db,
	}
}

// CreateSession creates a new chat session for a user
func (s *DatabaseService) CreateSession(ctx context.Context, userID, title string) (*ChatSessionDB, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO chat_sessions (id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, created_at, updated_at
	`

	session := &ChatSessionDB{}
	err := s.db.QueryRowContext(ctx, query, sessionID, userID, title, now, now).Scan(
		&session.ID, &session.UserID, &session.Title, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chat session")
	}

	return session, nil
}

// GetSession retrieves a chat session by ID and user ID
func (s *DatabaseService) GetSession(ctx context.Context, sessionID, userID string) (*ChatSessionDB, error) {
	query := `
		SELECT id, user_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE id = $1 AND user_id = $2
	`

	session := &ChatSessionDB{}
	err := s.db.QueryRowContext(ctx, query, sessionID, userID).Scan(
		&session.ID, &session.UserID, &session.Title, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("session not found")
		}
		return nil, errors.Wrap(err, "failed to get chat session")
	}

	return session, nil
}

// GetUserSessions retrieves all sessions for a user
func (s *DatabaseService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*ChatSessionDB, error) {
	query := `
		SELECT id, user_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user sessions")
	}
	defer rows.Close()

	var sessions []*ChatSessionDB
	for rows.Next() {
		session := &ChatSessionDB{}
		err := rows.Scan(&session.ID, &session.UserID, &session.Title, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan session")
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateSession updates a chat session's title and updated_at timestamp
func (s *DatabaseService) UpdateSession(ctx context.Context, sessionID, userID, title string) error {
	query := `
		UPDATE chat_sessions
		SET title = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`

	result, err := s.db.ExecContext(ctx, query, title, time.Now(), sessionID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to update chat session")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("session not found or not owned by user")
	}

	return nil
}

// DeleteSession deletes a chat session and all its messages
func (s *DatabaseService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Verify ownership first
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM chat_sessions WHERE id = $1 AND user_id = $2", sessionID, userID).Scan(&count)
	if err != nil {
		return errors.Wrap(err, "failed to verify session ownership")
	}
	if count == 0 {
		return errors.New("session not found or not owned by user")
	}

	// Delete attachments first
	_, err = tx.ExecContext(ctx, `
		DELETE FROM chat_attachments 
		WHERE message_id IN (
			SELECT id FROM chat_messages WHERE session_id = $1
		)
	`, sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat attachments")
	}

	// Delete messages
	_, err = tx.ExecContext(ctx, "DELETE FROM chat_messages WHERE session_id = $1", sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat messages")
	}

	// Delete session
	_, err = tx.ExecContext(ctx, "DELETE FROM chat_sessions WHERE id = $1 AND user_id = $2", sessionID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat session")
	}

	return tx.Commit()
}

// SaveMessage saves a chat message to the database
func (s *DatabaseService) SaveMessage(ctx context.Context, sessionID string, msg *models.Message) error {
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Convert tool calls to JSON
	var toolCallsJSON []byte
	if len(msg.ToolCalls) > 0 {
		toolCallsJSON, err = json.Marshal(msg.ToolCalls)
		if err != nil {
			return errors.Wrap(err, "failed to marshal tool calls")
		}
	}

	// Convert tool executions to JSON
	var toolExecutionsJSON []byte
	if len(msg.ToolExecutions) > 0 {
		toolExecutionsJSON, err = json.Marshal(msg.ToolExecutions)
		if err != nil {
			return errors.Wrap(err, "failed to marshal tool executions")
		}
	}

	// Convert approval request to JSON
	var approvalRequestJSON []byte
	if msg.ApprovalRequest != nil {
		approvalRequestJSON, err = json.Marshal(msg.ApprovalRequest)
		if err != nil {
			return errors.Wrap(err, "failed to marshal approval request")
		}
	}

	// Insert message
	query := `
		INSERT INTO chat_messages (id, session_id, role, content, tool_calls, tool_executions, approval_request, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = tx.ExecContext(ctx, query,
		msg.ID, sessionID, msg.Role, msg.Content,
		toolCallsJSON, toolExecutionsJSON, approvalRequestJSON, msg.Timestamp,
	)
	if err != nil {
		return errors.Wrap(err, "failed to save chat message")
	}

	// Save attachments if present
	if len(msg.Attachments) > 0 {
		for _, attachment := range msg.Attachments {
			attachmentQuery := `
				INSERT INTO chat_attachments (id, message_id, filename, mime_type, size, content, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`

			_, err = tx.ExecContext(ctx, attachmentQuery,
				attachment.ID, msg.ID, attachment.Filename, attachment.MimeType,
				attachment.Size, attachment.Content, time.Now(),
			)
			if err != nil {
				return errors.Wrap(err, "failed to save chat attachment")
			}
		}
	}

	// Update session timestamp
	_, err = tx.ExecContext(ctx, "UPDATE chat_sessions SET updated_at = $1 WHERE id = $2", time.Now(), sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to update session timestamp")
	}

	return tx.Commit()
}

// GetSessionMessages retrieves all messages for a session
func (s *DatabaseService) GetSessionMessages(ctx context.Context, sessionID, userID string, limit, offset int) ([]*models.Message, error) {
	// Verify session ownership
	_, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, role, content, tool_calls, tool_executions, approval_request, created_at
		FROM chat_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session messages")
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var toolCallsJSON, toolExecutionsJSON, approvalRequestJSON sql.NullString

		msg := &models.Message{}
		err := rows.Scan(
			&msg.ID, &msg.Role, &msg.Content,
			&toolCallsJSON, &toolExecutionsJSON, &approvalRequestJSON,
			&msg.Timestamp,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan message")
		}

		// Parse tool calls
		if toolCallsJSON.Valid {
			err = json.Unmarshal([]byte(toolCallsJSON.String), &msg.ToolCalls)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal tool calls")
			}
		}

		// Parse tool executions
		if toolExecutionsJSON.Valid {
			err = json.Unmarshal([]byte(toolExecutionsJSON.String), &msg.ToolExecutions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal tool executions")
			}
		}

		// Parse approval request
		if approvalRequestJSON.Valid {
			err = json.Unmarshal([]byte(approvalRequestJSON.String), &msg.ApprovalRequest)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal approval request")
			}
		}

		// Load attachments
		attachments, err := s.getMessageAttachments(ctx, msg.ID)
		if err != nil {
			return nil, err
		}
		msg.Attachments = attachments

		messages = append(messages, msg)
	}

	return messages, nil
}

// getMessageAttachments retrieves attachments for a message
func (s *DatabaseService) getMessageAttachments(ctx context.Context, messageID string) ([]models.Attachment, error) {
	query := `
		SELECT id, filename, mime_type, size, content
		FROM chat_attachments
		WHERE message_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get message attachments")
	}
	defer rows.Close()

	var attachments []models.Attachment
	for rows.Next() {
		var attachment models.Attachment
		err := rows.Scan(&attachment.ID, &attachment.Filename, &attachment.MimeType, &attachment.Size, &attachment.Content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan attachment")
		}
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// ClearSessionMessages deletes all messages for a session
func (s *DatabaseService) ClearSessionMessages(ctx context.Context, sessionID, userID string) error {
	// Verify session ownership
	_, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Delete attachments first
	_, err = tx.ExecContext(ctx, `
		DELETE FROM chat_attachments 
		WHERE message_id IN (
			SELECT id FROM chat_messages WHERE session_id = $1
		)
	`, sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat attachments")
	}

	// Delete messages
	_, err = tx.ExecContext(ctx, "DELETE FROM chat_messages WHERE session_id = $1", sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat messages")
	}

	// Update session timestamp
	_, err = tx.ExecContext(ctx, "UPDATE chat_sessions SET updated_at = $1 WHERE id = $2", time.Now(), sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to update session timestamp")
	}

	return tx.Commit()
}

// ChatSessionDB represents a chat session from the database
type ChatSessionDB struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
