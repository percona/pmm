package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrUnauthorized    = errors.New("session not found or not owned by user")
)

// logrusReformLogger implements reform.Logger for logrus
// Logs all SQL statements at Debug level, no-ops for Before/After
// Usage: reform.NewDB(db, dialect, &logrusReformLogger{l})
type logrusReformLogger struct {
	l *logrus.Entry
}

func (l *logrusReformLogger) Printf(format string, args ...interface{}) {
	l.l.Debugf(format, args...)
}
func (l *logrusReformLogger) Before(_ string, _ []interface{})                          {}
func (l *logrusReformLogger) After(_ string, _ []interface{}, _ time.Duration, _ error) {}

// DatabaseService handles database operations for chat sessions
// Now supports reconnection using the stored DatabaseConfig.
type DatabaseService struct {
	db       *sql.DB
	reformDB *reform.DB
	dbConfig *config.DatabaseConfig
	l        *logrus.Entry
}

// NewDatabaseService creates a new database service
func NewDatabaseService(dbConfig *config.DatabaseConfig) (*DatabaseService, error) {
	l := logrus.WithField("component", "database-service")
	l.Info("Creating new database service")

	db, err := dbConfig.OpenDatabase()
	if err != nil {
		l.WithError(err).Error("Failed to open database connection")
		return nil, err
	}

	if err := db.Ping(); err != nil {
		l.WithError(err).Error("Connection test failed during service creation")
		return nil, err
	}
	l.Info("Connection test successful during service creation")

	reformDB := reform.NewDB(db, postgresql.Dialect, &logrusReformLogger{l})

	return &DatabaseService{
		db:       db,
		reformDB: reformDB,
		dbConfig: dbConfig,
		l:        l,
	}, nil
}

// ensureConnection ensures the database connection is active, reconnecting if necessary
func (s *DatabaseService) ensureConnection() error {
	if s.db == nil {
		s.l.Warn("Database connection is nil, attempting to establish new connection")
		return s.reconnect()
	}

	if err := s.db.Ping(); err != nil {
		s.l.WithError(err).Warn("Connection ping failed, attempting to reconnect")
		return s.reconnect()
	}

	return nil
}

// reconnect establishes a new database connection using the stored DatabaseConfig
func (s *DatabaseService) reconnect() error {
	s.l.Info("Attempting to reconnect using stored DatabaseConfig...")

	if s.db != nil {
		s.db.Close()
	}

	db, err := s.dbConfig.OpenDatabase()
	if err != nil {
		s.l.WithError(err).Error("Failed to open new connection during reconnection")
		return fmt.Errorf("failed to open new database connection during reconnection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		s.l.WithError(err).Error("Ping failed on new connection during reconnection")
		return fmt.Errorf("failed to ping new database connection during reconnection: %w", err)
	}

	s.db = db
	s.reformDB = reform.NewDB(db, postgresql.Dialect, &logrusReformLogger{s.l})
	s.l.Info("Successfully reconnected to database")
	return nil
}

// CreateSession creates a new chat session for a user
func (s *DatabaseService) CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error) {
	s.l.WithFields(logrus.Fields{"user_id": userID, "title": title}).Info("Creating session for user")

	// Test database connection before proceeding
	if err := s.ensureConnection(); err != nil {
		s.l.WithError(err).Error("Connection failed in CreateSession")
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	s.l.Info("Connection successful in CreateSession")

	sessionID := uuid.New().String()
	s.l.WithField("session_id", sessionID).Debug("Generated session ID")

	session := &models.ChatSession{
		ID:     sessionID,
		UserID: userID,
		Title:  title,
	}

	s.l.WithFields(logrus.Fields{"session_id": sessionID, "user_id": userID, "title": title}).Debug("Inserting session")

	err := s.reformDB.WithContext(ctx).Insert(session)
	if err != nil {
		s.l.WithError(err).Error("Failed to create chat session")
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	s.l.WithField("session_id", session.ID).Info("Successfully created session")
	return session, nil
}

// GetSession retrieves a chat session by ID and user ID
func (s *DatabaseService) GetSession(ctx context.Context, sessionID, userID string) (*models.ChatSession, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		s.l.WithError(err).Error("Connection failed in GetSession")
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	var session models.ChatSession
	err := s.reformDB.WithContext(ctx).FindByPrimaryKeyTo(&session, sessionID)
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get chat session: %w", err)
	}

	// Verify ownership
	if session.UserID != userID {
		return nil, ErrUnauthorized
	}

	return &session, nil
}

// GetUserSessions retrieves all sessions for a user
func (s *DatabaseService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*models.ChatSession, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatSessionTable, "WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3", userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
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
		return fmt.Errorf("database connection failed: %w", err)
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
		return fmt.Errorf("failed to update chat session: %w", err)
	}

	return nil
}

// DeleteSession deletes a chat session and all its messages
func (s *DatabaseService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Start transaction
	tx, err := s.reformDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
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
		return fmt.Errorf("failed to delete chat attachments: %w", err)
	}

	// Delete messages
	_, err = tx.Exec("DELETE FROM chat_messages WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete chat messages: %w", err)
	}

	// Delete session
	err = tx.Delete(session)
	if err != nil {
		return fmt.Errorf("failed to delete chat session: %w", err)
	}

	return tx.Commit()
}

// SaveMessage saves a chat message to the database
func (s *DatabaseService) SaveMessage(ctx context.Context, sessionID string, msg *models.Message) error {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Start transaction
	tx, err := s.reformDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
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
		return fmt.Errorf("failed to save chat message: %w", err)
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
				return fmt.Errorf("failed to save chat attachment: %w", err)
			}
		}
	}

	// Update session timestamp using Reform
	session := &models.ChatSession{ID: sessionID}
	err = tx.Reload(session)
	if err != nil {
		return fmt.Errorf("failed to reload session for timestamp update: %w", err)
	}

	err = tx.Update(session)
	if err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	return tx.Commit()
}

// GetSessionMessages retrieves all messages for a session
func (s *DatabaseService) GetSessionMessages(ctx context.Context, sessionID, userID string, limit, offset int) ([]*models.Message, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Verify session ownership
	_, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatMessageTable, "WHERE session_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3", sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
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
			msg.ApprovalRequest = &models.ApprovalRequest{
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
		return nil, fmt.Errorf("failed to get message attachments: %w", err)
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
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Verify session ownership
	_, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.reformDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete attachments first using Reform bulk delete
	_, err = tx.DeleteFrom(models.ChatAttachmentTable,
		"WHERE message_id IN (SELECT id FROM chat_messages WHERE session_id = $1)",
		sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete chat attachments: %w", err)
	}

	// Delete messages using Reform bulk delete
	_, err = tx.DeleteFrom(models.ChatMessageTable, "WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete chat messages: %w", err)
	}

	// Update session timestamp by fetching and updating the session record
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session for update: %w", err)
	}

	err = tx.Update(session)
	if err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	return tx.Commit()
}

// GetPendingApprovals retrieves all unprocessed approval requests for a session
func (s *DatabaseService) GetPendingApprovals(ctx context.Context, sessionID string) ([]*models.ToolApprovalRequest, error) {
	// Ensure database connection is active
	if err := s.ensureConnection(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Query messages with unprocessed approval requests using Reform SelectAllFrom
	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatMessageTable,
		"WHERE session_id = $1 AND approval_request IS NOT NULL AND (approval_request->>'processed')::boolean = false",
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
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
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Find the message with the approval request using Reform
	structs, err := s.reformDB.WithContext(ctx).SelectAllFrom(models.ChatMessageTable,
		"WHERE session_id = $1 AND approval_request->>'request_id' = $2",
		sessionID, requestID)
	if err != nil {
		return fmt.Errorf("failed to find approval request: %w", err)
	}

	if len(structs) == 0 {
		return errors.New("approval request not found")
	}

	// Update the approval request using Reform
	chatMsg := structs[0].(*models.ChatMessage)
	if chatMsg.ApprovalRequest != nil {
		chatMsg.ApprovalRequest.Processed = true
		err = s.reformDB.WithContext(ctx).Update(chatMsg)
		if err != nil {
			return fmt.Errorf("failed to mark approval as processed: %w", err)
		}
	}

	s.l.WithField("request_id", requestID).Info("Marked approval request as processed")
	return nil
}

// DB returns the underlying *sql.DB instance.
func (s *DatabaseService) DB() *sql.DB {
	return s.db
}
