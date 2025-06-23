package services

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// DatabaseService handles database operations for chat sessions
type DatabaseService struct {
	db       *sql.DB
	reformDB *reform.DB
	dbConfig *config.DatabaseConfig
}

// NewDatabaseService creates a new database service
func NewDatabaseService(db *sql.DB) *DatabaseService {
	log.Printf("🗄️  Database: Creating new database service")

	// Test the connection immediately
	if db != nil {
		if err := db.Ping(); err != nil {
			log.Printf("❌ Database: Connection test failed during service creation: %v", err)
		} else {
			log.Printf("✅ Database: Connection test successful during service creation")
		}
	} else {
		log.Printf("❌ Database: Received nil database connection")
	}

	// Create Reform DB instance
	var reformDB *reform.DB
	if db != nil {
		reformDB = reform.NewDB(db, postgresql.Dialect, reform.NewPrintfLogger(log.Printf))
	}

	return &DatabaseService{
		db:       db,
		reformDB: reformDB,
		dbConfig: config.GetDatabaseConfig(), // Store config for reconnection
	}
}

// ensureConnection ensures the database connection is active, reconnecting if necessary
func (s *DatabaseService) ensureConnection() error {
	if s.db == nil {
		log.Printf("🗄️  Database: Database connection is nil, attempting to establish new connection")
		return s.reconnect()
	}

	if err := s.db.Ping(); err != nil {
		log.Printf("⚠️  Database: Connection ping failed (%v), attempting to reconnect", err)
		return s.reconnect()
	}

	return nil
}

// reconnect establishes a new database connection
func (s *DatabaseService) reconnect() error {
	log.Printf("🗄️  Database: Reconnecting to database...")

	if s.db != nil {
		s.db.Close() // Close existing connection
	}

	newDB, err := s.dbConfig.OpenDatabase()
	if err != nil {
		log.Printf("❌ Database: Failed to reconnect: %v", err)
		return err
	}

	s.db = newDB
	s.reformDB = reform.NewDB(newDB, postgresql.Dialect, reform.NewPrintfLogger(log.Printf))
	log.Printf("✅ Database: Successfully reconnected to database")
	return nil
}

// CreateSession creates a new chat session for a user
func (s *DatabaseService) CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error) {
	log.Printf("🗄️  Database: Creating session for user %s with title '%s'", userID, title)

	// Test database connection before proceeding
	if err := s.ensureConnection(); err != nil {
		log.Printf("❌ Database: Connection failed in CreateSession: %v", err)
		return nil, errors.Wrap(err, "database connection failed")
	}
	log.Printf("✅ Database: Connection successful in CreateSession")

	sessionID := uuid.New().String()
	log.Printf("🗄️  Database: Generated session ID: %s", sessionID)

	session := &models.ChatSession{
		ID:     sessionID,
		UserID: userID,
		Title:  title,
	}

	log.Printf("🗄️  Database: Inserting session with ID=%s, userID=%s, title=%s", sessionID, userID, title)

	err := s.reformDB.WithContext(ctx).Insert(session)
	if err != nil {
		log.Printf("❌ Database: Failed to create chat session: %v", err)
		return nil, errors.Wrap(err, "failed to create chat session")
	}

	log.Printf("✅ Database: Successfully created session: %s", session.ID)
	return session, nil
}

// GetSession retrieves a chat session by ID and user ID
func (s *DatabaseService) GetSession(ctx context.Context, sessionID, userID string) (*models.ChatSession, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		log.Printf("❌ Database: Connection failed in GetSession: %v", err)
		return nil, errors.Wrap(err, "database connection failed")
	}

	var session models.ChatSession
	err := s.reformDB.WithContext(ctx).FindByPrimaryKeyTo(&session, sessionID)
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, errors.New("session not found")
		}
		return nil, errors.Wrap(err, "failed to get chat session")
	}

	// Verify ownership
	if session.UserID != userID {
		return nil, errors.New("session not found")
	}

	return &session, nil
}

// GetUserSessions retrieves all sessions for a user
func (s *DatabaseService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*models.ChatSession, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return nil, errors.Wrap(err, "database connection failed")
	}

	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatSessionTable, "WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3", userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user sessions")
	}

	sessions := make([]*models.ChatSession, len(structs))
	for i, str := range structs {
		sessions[i] = str.(*models.ChatSession)
	}

	return sessions, nil
}

// UpdateSession updates a chat session's title and updated_at timestamp
func (s *DatabaseService) UpdateSession(ctx context.Context, sessionID, userID, title string) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return errors.Wrap(err, "database connection failed")
	}

	// First get the session to verify ownership
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	// Update the session
	session.Title = title
	err = s.reformDB.WithContext(ctx).Update(session)
	if err != nil {
		return errors.Wrap(err, "failed to update chat session")
	}

	return nil
}

// DeleteSession deletes a chat session and all its messages
func (s *DatabaseService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return errors.Wrap(err, "database connection failed")
	}

	// Start transaction
	tx, err := s.reformDB.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Verify ownership first
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	// Delete attachments first
	_, err = tx.Exec("DELETE FROM chat_attachments WHERE message_id IN (SELECT id FROM chat_messages WHERE session_id = $1)", sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat attachments")
	}

	// Delete messages
	_, err = tx.Exec("DELETE FROM chat_messages WHERE session_id = $1", sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat messages")
	}

	// Delete session
	err = tx.Delete(session)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat session")
	}

	return tx.Commit()
}

// SaveMessage saves a chat message to the database
func (s *DatabaseService) SaveMessage(ctx context.Context, sessionID string, msg *models.Message) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return errors.Wrap(err, "database connection failed")
	}

	// Start transaction
	tx, err := s.reformDB.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Convert models.Message to models.ChatMessage
	chatMessage := &models.ChatMessage{
		ID:        msg.ID,
		SessionID: sessionID,
		Role:      msg.Role,
		Content:   msg.Content,
	}

	// Convert tool calls
	if len(msg.ToolCalls) > 0 {
		toolCalls := make(models.ToolCallsField, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			toolCalls[i] = models.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		chatMessage.ToolCalls = &toolCalls
	}

	// Convert tool executions
	if len(msg.ToolExecutions) > 0 {
		toolExecutions := make(models.ToolExecutionsField, len(msg.ToolExecutions))
		for i, te := range msg.ToolExecutions {
			toolExecutions[i] = models.ToolExecution{
				ID:        te.ID,
				ToolName:  te.ToolName,
				Arguments: te.Arguments,
				Result:    te.Result,
				Error:     te.Error,
				StartTime: te.StartTime,
				EndTime:   te.EndTime,
				Duration:  te.Duration,
			}
		}
		chatMessage.ToolExecutions = &toolExecutions
	}

	// Convert approval request
	if msg.ApprovalRequest != nil {
		toolCalls := make([]models.ToolCall, len(msg.ApprovalRequest.ToolCalls))
		for i, tc := range msg.ApprovalRequest.ToolCalls {
			toolCalls[i] = models.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		approvalRequest := models.ApprovalRequestField{
			RequestID: msg.ApprovalRequest.RequestID,
			ToolCalls: toolCalls,
			Processed: msg.ApprovalRequest.Processed,
		}
		chatMessage.ApprovalRequest = &approvalRequest
	}

	// Insert message
	err = tx.Insert(chatMessage)
	if err != nil {
		return errors.Wrap(err, "failed to save chat message")
	}

	// Save attachments if present
	if len(msg.Attachments) > 0 {
		for _, attachment := range msg.Attachments {
			chatAttachment := &models.ChatAttachment{
				ID:        attachment.ID,
				MessageID: msg.ID,
				Filename:  attachment.Filename,
				MimeType:  attachment.MimeType,
				Size:      attachment.Size,
				Content:   attachment.Content,
			}

			err = tx.Insert(chatAttachment)
			if err != nil {
				return errors.Wrap(err, "failed to save chat attachment")
			}
		}
	}

	// Update session timestamp
	_, err = tx.Exec("UPDATE chat_sessions SET updated_at = $1 WHERE id = $2", time.Now(), sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to update session timestamp")
	}

	return tx.Commit()
}

// GetSessionMessages retrieves all messages for a session
func (s *DatabaseService) GetSessionMessages(ctx context.Context, sessionID, userID string, limit, offset int) ([]*models.Message, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return nil, errors.Wrap(err, "database connection failed")
	}

	// Verify session ownership
	_, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatMessageTable, "WHERE session_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3", sessionID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session messages")
	}

	messages := make([]*models.Message, len(structs))
	for i, str := range structs {
		chatMsg := str.(*models.ChatMessage)

		// Convert models.ChatMessage to models.Message
		msg := &models.Message{
			ID:        chatMsg.ID,
			Role:      chatMsg.Role,
			Content:   chatMsg.Content,
			Timestamp: chatMsg.CreatedAt,
		}

		// Convert tool calls
		if chatMsg.ToolCalls != nil {
			msg.ToolCalls = make([]models.ToolCall, len(*chatMsg.ToolCalls))
			for j, tc := range *chatMsg.ToolCalls {
				msg.ToolCalls[j] = models.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		// Convert tool executions
		if chatMsg.ToolExecutions != nil {
			msg.ToolExecutions = make([]models.ToolExecution, len(*chatMsg.ToolExecutions))
			for j, te := range *chatMsg.ToolExecutions {
				msg.ToolExecutions[j] = models.ToolExecution{
					ID:        te.ID,
					ToolName:  te.ToolName,
					Arguments: te.Arguments,
					Result:    te.Result,
					Error:     te.Error,
					StartTime: te.StartTime,
					EndTime:   te.EndTime,
					Duration:  te.Duration,
				}
			}
		}

		// Convert approval request
		if chatMsg.ApprovalRequest != nil {
			toolCalls := make([]models.ToolCall, len(chatMsg.ApprovalRequest.ToolCalls))
			for j, tc := range chatMsg.ApprovalRequest.ToolCalls {
				toolCalls[j] = models.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			msg.ApprovalRequest = &struct {
				RequestID string            `json:"request_id"`
				ToolCalls []models.ToolCall `json:"tool_calls"`
				Processed bool              `json:"processed,omitempty"`
			}{
				RequestID: chatMsg.ApprovalRequest.RequestID,
				ToolCalls: toolCalls,
				Processed: chatMsg.ApprovalRequest.Processed,
			}
		}

		// Load attachments
		attachments, err := s.getMessageAttachments(ctx, msg.ID)
		if err != nil {
			return nil, err
		}
		msg.Attachments = attachments

		messages[i] = msg
	}

	return messages, nil
}

// getMessageAttachments retrieves attachments for a message
func (s *DatabaseService) getMessageAttachments(ctx context.Context, messageID string) ([]models.Attachment, error) {
	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatAttachmentTable, "WHERE message_id = $1 ORDER BY created_at ASC", messageID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get message attachments")
	}

	attachments := make([]models.Attachment, len(structs))
	for i, str := range structs {
		chatAttachment := str.(*models.ChatAttachment)
		attachments[i] = models.Attachment{
			ID:       chatAttachment.ID,
			Filename: chatAttachment.Filename,
			MimeType: chatAttachment.MimeType,
			Size:     chatAttachment.Size,
			Content:  chatAttachment.Content,
		}
	}

	return attachments, nil
}

// ClearSessionMessages deletes all messages for a session
func (s *DatabaseService) ClearSessionMessages(ctx context.Context, sessionID, userID string) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return errors.Wrap(err, "database connection failed")
	}

	// Verify session ownership
	_, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.reformDB.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Delete attachments first
	_, err = tx.Exec("DELETE FROM chat_attachments WHERE message_id IN (SELECT id FROM chat_messages WHERE session_id = $1)", sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat attachments")
	}

	// Delete messages
	_, err = tx.Exec("DELETE FROM chat_messages WHERE session_id = $1", sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat messages")
	}

	// Update session timestamp
	_, err = tx.Exec("UPDATE chat_sessions SET updated_at = $1 WHERE id = $2", time.Now(), sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to update session timestamp")
	}

	return tx.Commit()
}

// GetPendingApprovals retrieves all unprocessed approval requests for a session
func (s *DatabaseService) GetPendingApprovals(ctx context.Context, sessionID string) ([]*models.ToolApprovalRequest, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return nil, errors.Wrap(err, "database connection failed")
	}

	// Query messages with unprocessed approval requests
	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatMessageTable,
		"WHERE session_id = $1 AND approval_request IS NOT NULL AND approval_request->>'processed' = 'false'",
		sessionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pending approvals")
	}

	var approvals []*models.ToolApprovalRequest
	for _, str := range structs {
		chatMsg := str.(*models.ChatMessage)
		if chatMsg.ApprovalRequest != nil && !chatMsg.ApprovalRequest.Processed {
			approval := &models.ToolApprovalRequest{
				SessionID: sessionID,
				RequestID: chatMsg.ApprovalRequest.RequestID,
				ToolCalls: chatMsg.ApprovalRequest.ToolCalls,
			}
			approvals = append(approvals, approval)
		}
	}

	return approvals, nil
}

// MarkApprovalProcessed marks an approval request as processed in the database
func (s *DatabaseService) MarkApprovalProcessed(ctx context.Context, sessionID, requestID string) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return errors.Wrap(err, "database connection failed")
	}

	// Update the approval_request field to mark as processed
	_, err := s.reformDB.Exec(
		"UPDATE chat_messages SET approval_request = jsonb_set(approval_request, '{processed}', 'true') WHERE session_id = $1 AND approval_request->>'request_id' = $2",
		sessionID, requestID)
	if err != nil {
		return errors.Wrap(err, "failed to mark approval as processed")
	}

	log.Printf("💾 Database: Marked approval request %s as processed", requestID)
	return nil
}
