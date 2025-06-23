package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

// ChatService handles chat conversations and coordinates LLM and MCP services
type ChatService struct {
	llmProvider      LLMProvider
	mcp              *MCPService
	database         *DatabaseService
	userSessions     map[string]map[string]*ChatSession // userID -> sessionID -> session
	systemPrompt     string
	pendingApprovals map[string]*models.ToolApprovalRequest // requestID -> approval request
	mu               sync.RWMutex
}

// ChatSession represents an active chat session
type ChatSession struct {
	ID       string
	UserID   string
	Messages []*models.Message
	Created  time.Time
	Updated  time.Time
}

// NewChatService creates a new chat service
func NewChatService(llmProvider LLMProvider, mcp *MCPService, database *DatabaseService) *ChatService {
	return &ChatService{
		llmProvider:      llmProvider,
		mcp:              mcp,
		database:         database,
		userSessions:     make(map[string]map[string]*ChatSession),
		systemPrompt:     "",
		pendingApprovals: make(map[string]*models.ToolApprovalRequest),
	}
}

// SetSystemPrompt sets the system prompt for the chat service
func (s *ChatService) SetSystemPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemPrompt = prompt
}

// ProcessMessageWithUser processes a user message and generates a response for a specific user
func (s *ChatService) ProcessMessageWithUser(ctx context.Context, userID, sessionID, userMessage string) (*models.ChatResponse, error) {
	return s.ProcessMessageWithAttachmentsForUser(ctx, userID, sessionID, userMessage, nil)
}

// ProcessStreamMessageForUser processes a user message and returns a streaming response for a specific user
func (s *ChatService) ProcessStreamMessageForUser(ctx context.Context, userID, sessionID, userMessage string) (<-chan *models.StreamMessage, error) {
	log.Printf("🚀 Chat: Starting stream processing for user %s, session %s, message: %q", userID, sessionID, userMessage)

	// Check if this is a tool approval message
	if strings.HasPrefix(userMessage, "[APPROVE_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		// Extract request ID
		requestID := strings.TrimSuffix(strings.TrimPrefix(userMessage, "[APPROVE_TOOLS:"), "]")
		log.Printf("🔧 Chat: Detected tool approval message for request: %s", requestID)

		// Process tool approval directly in streaming
		return s.processToolApprovalInStreamForUser(ctx, userID, sessionID, requestID, true)
	}

	if strings.HasPrefix(userMessage, "[DENY_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		// Extract request ID
		requestID := strings.TrimSuffix(strings.TrimPrefix(userMessage, "[DENY_TOOLS:"), "]")
		log.Printf("🔧 Chat: Detected tool denial message for request: %s", requestID)

		// Process tool denial directly in streaming
		return s.processToolApprovalInStreamForUser(ctx, userID, sessionID, requestID, false)
	}

	s.mu.Lock()
	session := s.getOrCreateSessionForUser(userID, sessionID)
	s.mu.Unlock()

	// Add user message to session
	userMsg := &models.Message{
		ID:        fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}

	s.addMessageToSession(ctx, sessionID, session, userMsg)

	log.Printf("📝 Chat: Added user message to session, total messages: %d", len(session.Messages))

	// Get available tools
	tools := s.mcp.GetTools()
	log.Printf("🔧 Chat: Retrieved %d available tools for streaming", len(tools))

	// Prepare messages with system prompt
	preparedMessages := s.prepareMessagesWithSystemPrompt(session.Messages)

	// Generate streaming response from LLM
	streamChan, err := s.llmProvider.GenerateStreamResponse(ctx, preparedMessages, tools)
	if err != nil {
		log.Printf("❌ Chat: Failed to start LLM streaming: %v", err)
		return nil, err
	}

	log.Printf("✅ Chat: LLM streaming started successfully")

	// Create output channel
	outputChan := make(chan *models.StreamMessage, 10)

	// Process stream
	go func() {
		defer close(outputChan)
		defer log.Printf("🏁 Chat: Stream processing completed for session %s", sessionID)

		var fullContent string
		var messageCount int

		log.Printf("🔄 Chat: Starting stream message processing loop")

		s.handleStreamMessage(ctx, sessionID, session, streamChan, outputChan, &fullContent, tools, &messageCount)

		log.Printf("⚠️  Chat: Stream channel closed without 'done' message")
	}()

	return outputChan, nil
}

func (s *ChatService) handleStreamMessage(ctx context.Context, sessionID string, session *ChatSession, streamChan <-chan *models.StreamMessage, outputChan chan<- *models.StreamMessage, fullContent *string, tools []models.MCPTool, messageCount *int) {
	var streamToolCalls []models.ToolCall

	for streamMsg := range streamChan {
		(*messageCount)++
		streamMsg.SessionID = sessionID

		log.Printf("📦 Chat: Processing stream message %d, type: %s", *messageCount, streamMsg.Type)

		if streamMsg.Type == "message" {
			*fullContent += streamMsg.Content
			log.Printf("📝 Chat: Accumulated content length: %d", len(*fullContent))
			outputChan <- streamMsg
		} else if streamMsg.Type == "tool_call" {
			// Handle tool calls in streaming - collect them for execution after stream completes
			log.Printf("🔧 Chat: Streaming tool call detected: %s", streamMsg.Content)

			// Parse tool call from content (format: "Function call: toolname(args)")
			if strings.HasPrefix(streamMsg.Content, "Function call: ") {
				funcCallStr := strings.TrimPrefix(streamMsg.Content, "Function call: ")

				// Parse function name and arguments
				parenIndex := strings.Index(funcCallStr, "(")
				if parenIndex > 0 {
					funcName := funcCallStr[:parenIndex]
					argsStr := strings.TrimSuffix(funcCallStr[parenIndex+1:], ")")

					log.Printf("🔧 Chat: Parsing streaming tool call - function: %s, args: %s", funcName, argsStr)

					// Create tool call object for later execution
					toolCall := models.ToolCall{
						ID:   fmt.Sprintf("stream_call_%d", len(streamToolCalls)),
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      funcName,
							Arguments: argsStr,
						},
					}

					streamToolCalls = append(streamToolCalls, toolCall)
					log.Printf("🔧 Chat: Collected tool call for later execution: %s (total: %d)", funcName, len(streamToolCalls))

					// Send notification to user that tool will be executed
					outputChan <- &models.StreamMessage{
						Type:      "message",
						Content:   fmt.Sprintf("🔧 Executing %s...\n", funcName),
						SessionID: sessionID,
					}
				} else {
					log.Printf("❌ Chat: Invalid tool call format: %s", funcCallStr)
				}
			} else {
				log.Printf("❌ Chat: Unexpected tool call format: %s", streamMsg.Content)
			}
		} else if streamMsg.Type == "error" {
			log.Printf("❌ Chat: Streaming error: %s", streamMsg.Error)
			outputChan <- streamMsg
			return
		} else if streamMsg.Type == "done" {
			// Check if the done message includes tool calls (Gemini style)
			if len(streamMsg.ToolCalls) > 0 {
				log.Printf("🔧 Chat: Done message includes %d tool calls from LLM", len(streamMsg.ToolCalls))
				streamToolCalls = append(streamToolCalls, streamMsg.ToolCalls...)
			}

			log.Printf("✅ Chat: Stream completed. Total messages: %d, content length: %d, tool calls: %d",
				*messageCount, len(*fullContent), len(streamToolCalls))

			// Save complete message to session
			assistantMsg := &models.Message{
				ID:        fmt.Sprintf("assistant_%d", time.Now().UnixNano()),
				Role:      "assistant",
				Content:   *fullContent,
				Timestamp: time.Now(),
				ToolCalls: streamToolCalls,
			}

			s.addMessageToSession(ctx, sessionID, session, assistantMsg)

			log.Printf("💾 Chat: Saved assistant message to session, total messages: %d", len(session.Messages))

			// Handle tool calls if present (request approval instead of executing immediately)
			if len(streamToolCalls) > 0 {
				log.Printf("🔧 Chat: Requesting approval for %d tool call(s) from streaming response", len(streamToolCalls))

				// Generate approval request ID
				requestID := fmt.Sprintf("approval_%d", time.Now().UnixNano())

				// Create approval request
				approvalRequest := &models.ToolApprovalRequest{
					SessionID: sessionID,
					ToolCalls: streamToolCalls,
					RequestID: requestID,
				}

				// Store pending approval
				s.mu.Lock()
				s.pendingApprovals[requestID] = approvalRequest
				s.mu.Unlock()

				// Create tool approval message in chat history
				approvalMsg := &models.Message{
					ID:        fmt.Sprintf("approval_%d", time.Now().UnixNano()),
					Role:      "tool_approval",
					Content:   fmt.Sprintf("🔧 The assistant wants to execute %d tool(s). Please approve or deny the request.", len(streamToolCalls)),
					Timestamp: time.Now(),
					ToolCalls: streamToolCalls,
				}

				s.addMessageToSession(ctx, sessionID, session, approvalMsg)

				// Send approval request to frontend via stream
				outputChan <- &models.StreamMessage{
					Type:      "tool_approval_request",
					Content:   fmt.Sprintf("🔧 The assistant wants to execute %d tool(s). Do you approve?", len(streamToolCalls)),
					SessionID: sessionID,
					ToolCalls: streamToolCalls,
					RequestID: requestID,
				}

				// Send done message - user needs to approve before continuing
				outputChan <- &models.StreamMessage{
					Type:      "done",
					SessionID: sessionID,
				}
				return
			} else {
				// Check if the response content suggests tool usage without proper tool calls
				if s.detectToolCallAttempts(*fullContent) {
					log.Printf("⚠️  Chat: Streaming response suggests tool usage but no tool calls were detected. Content: %s", *fullContent)
				}

				// Send done message for non-tool responses
				outputChan <- &models.StreamMessage{
					Type:      "done",
					SessionID: sessionID,
				}
			}
		} else {
			log.Printf("🤔 Chat: Unknown stream message type: %s", streamMsg.Type)
		}
	}
}

// ClearHistoryForUser clears chat history for a specific user's session
func (s *ChatService) ClearHistoryForUser(userID, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userSessions, exists := s.userSessions[userID]
	if !exists {
		return
	}

	delete(userSessions, sessionID)

	// If user has no more sessions, remove user entry
	if len(userSessions) == 0 {
		delete(s.userSessions, userID)
	}
}

// getOrCreateSessionForUser gets or creates a session for a specific user
func (s *ChatService) getOrCreateSessionForUser(userID, sessionID string) *ChatSession {
	// Ensure user sessions map exists
	if s.userSessions[userID] == nil {
		s.userSessions[userID] = make(map[string]*ChatSession)
	}

	session, exists := s.userSessions[userID][sessionID]
	if !exists {
		session = &ChatSession{
			ID:       sessionID,
			UserID:   userID,
			Messages: []*models.Message{},
			Created:  time.Now(),
			Updated:  time.Now(),
		}
		s.userSessions[userID][sessionID] = session
	}
	return session
}

// isTextFile checks if the MIME type represents a text file
func (s *ChatService) isTextFile(mimeType string) bool {
	textTypes := []string{
		"text/plain",
		"text/html",
		"text/css",
		"text/javascript",
		"text/xml",
		"application/json",
		"application/xml",
		"application/yaml",
		"application/x-yaml",
		"text/yaml",
		"text/x-yaml",
		"text/markdown",
		"text/x-markdown",
	}

	for _, textType := range textTypes {
		if strings.Contains(mimeType, textType) {
			return true
		}
	}

	// Also check for common programming language extensions
	if strings.Contains(mimeType, "text/") ||
		strings.Contains(mimeType, "application/javascript") ||
		strings.Contains(mimeType, "application/typescript") {
		return true
	}

	return false
}

// isImageFile checks if the MIME type represents an image file
func (s *ChatService) isImageFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// prepareMessagesWithSystemPrompt prepares messages for LLM with system prompt
func (s *ChatService) prepareMessagesWithSystemPrompt(messages []*models.Message) []*models.Message {
	if s.systemPrompt == "" {
		return messages
	}

	// Check if the first message is already a system message
	if len(messages) > 0 && messages[0].Role == "system" {
		return messages
	}

	// Enhance system prompt with current time context
	currentTime := time.Now()
	timeContext := fmt.Sprintf("\n\nCURRENT CONTEXT:\n- Current time: %s (UTC)\n- Current time (local): %s\n- For QAN queries, use a 12-hour period by default (from %s to %s UTC) unless the user specifies otherwise\n- When using QAN tools, format timestamps in RFC3339 format (e.g., %s)",
		currentTime.UTC().Format("2006-01-02 15:04:05"),
		currentTime.Format("2006-01-02 15:04:05 MST"),
		currentTime.Add(-12*time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		currentTime.UTC().Format(time.RFC3339))

	enhancedSystemPrompt := s.systemPrompt + timeContext

	// Create system message
	systemMsg := &models.Message{
		ID:        "system_prompt",
		Role:      "system",
		Content:   enhancedSystemPrompt,
		Timestamp: time.Now(),
	}

	// Prepend system message to the conversation
	preparedMessages := make([]*models.Message, 0, len(messages)+1)
	preparedMessages = append(preparedMessages, systemMsg)
	preparedMessages = append(preparedMessages, messages...)

	return preparedMessages
}

// executeToolCalls executes a list of tool calls and returns tool executions and result messages
func (s *ChatService) executeToolCalls(ctx context.Context, toolCalls []models.ToolCall, session *ChatSession) ([]models.ToolExecution, []*models.Message) {
	var toolExecutions []models.ToolExecution
	var toolMessages []*models.Message

	for i, toolCall := range toolCalls {
		log.Printf("🔧 Tool call %d: ID=%s, Type=%s, Function=%s, Args=%s",
			i+1, toolCall.ID, toolCall.Type, toolCall.Function.Name, toolCall.Function.Arguments)

		// Track tool execution
		startTime := time.Now()
		toolExecution := models.ToolExecution{
			ID:        toolCall.ID,
			ToolName:  toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
			StartTime: startTime,
		}

		// Execute tool
		toolResult, err := s.mcp.ExecuteTool(ctx, toolCall)
		endTime := time.Now()
		toolExecution.EndTime = endTime
		toolExecution.Duration = endTime.Sub(startTime).Milliseconds()

		if err != nil {
			log.Printf("❌ Tool execution failed for %s: %v", toolCall.Function.Name, err)
			toolExecution.Error = err.Error()
			toolResult = fmt.Sprintf("Error executing tool: %v", err)
		} else {
			log.Printf("✅ Tool execution successful for %s, result length: %d", toolCall.Function.Name, len(toolResult))
		}

		toolExecution.Result = toolResult
		toolExecutions = append(toolExecutions, toolExecution)

		// Add tool result as a message
		toolMsg := &models.Message{
			ID:        fmt.Sprintf("tool_%s", toolCall.ID),
			Role:      "tool",
			Content:   toolResult,
			Timestamp: time.Now(),
		}

		toolMessages = append(toolMessages, toolMsg)

		// Add to session and save to database
		s.addMessageToSession(ctx, session.ID, session, toolMsg)
	}

	return toolExecutions, toolMessages
}

// ProcessToolApproval processes user's approval/denial of tool execution
func (s *ChatService) ProcessToolApproval(ctx context.Context, approval *models.ToolApprovalResponse) (<-chan *models.StreamMessage, error) {
	log.Printf("🔧 Chat: Processing tool approval for request %s, approved: %t", approval.RequestID, approval.Approved)

	s.mu.Lock()
	approvalRequest, exists := s.pendingApprovals[approval.RequestID]
	if exists {
		delete(s.pendingApprovals, approval.RequestID)
	}
	s.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("approval request %s not found or expired", approval.RequestID)
	}

	// Create output channel
	outputChan := make(chan *models.StreamMessage, 10)

	// Process approval
	go func() {
		defer close(outputChan)

		if !approval.Approved {
			log.Printf("❌ Chat: Tool execution denied by user for request %s", approval.RequestID)
			outputChan <- &models.StreamMessage{
				Type:      "message",
				Content:   "❌ Tool execution was denied by user.",
				SessionID: approval.SessionID,
			}
			outputChan <- &models.StreamMessage{
				Type:      "done",
				SessionID: approval.SessionID,
			}
			return
		}

		log.Printf("✅ Chat: Tool execution approved by user for request %s", approval.RequestID)

		// Get session for context
		s.mu.Lock()
		session := s.getOrCreateSessionForUser("default-user", approval.SessionID)
		s.mu.Unlock()

		// Determine which tools to execute (all or selective)
		toolsToExecute := approvalRequest.ToolCalls
		if len(approval.ApprovedIDs) > 0 {
			// Selective approval - only execute approved tools
			toolsToExecute = []models.ToolCall{}
			for _, toolCall := range approvalRequest.ToolCalls {
				for _, approvedID := range approval.ApprovedIDs {
					if toolCall.ID == approvedID {
						toolsToExecute = append(toolsToExecute, toolCall)
						break
					}
				}
			}
		}

		log.Printf("🔧 Chat: Executing %d approved tool(s)", len(toolsToExecute))

		// Execute approved tools using common method
		toolExecutions, toolMessages := s.executeToolCalls(ctx, toolsToExecute, session)

		// Add tool executions to the last assistant message
		s.mu.Lock()
		if len(session.Messages) > 0 {
			lastMsg := session.Messages[len(session.Messages)-1]
			if lastMsg.Role == "assistant" {
				lastMsg.ToolExecutions = toolExecutions
			}
		}

		// Tool result messages are already added to session by executeToolCalls
		// Get updated messages for LLM
		allMessages := make([]*models.Message, len(session.Messages))
		copy(allMessages, session.Messages)
		s.mu.Unlock()

		// Send tool execution information to the frontend
		outputChan <- &models.StreamMessage{
			Type:           "tool_execution",
			Content:        fmt.Sprintf("🔧 Executed %d tool(s)", len(toolExecutions)),
			SessionID:      approval.SessionID,
			ToolExecutions: toolExecutions,
		}

		log.Printf("🔧 Chat: Getting LLM follow-up response with %d tool results", len(toolMessages))

		// Send notification that LLM is processing results
		outputChan <- &models.StreamMessage{
			Type:      "message",
			Content:   "\n🤖 Processing results...\n\n",
			SessionID: approval.SessionID,
		}

		// Get available tools for follow-up
		tools := s.mcp.GetTools()

		// Get follow-up response from LLM with fresh context
		preparedMessages := s.prepareMessagesWithSystemPrompt(allMessages)
		followUpChan, err := s.llmProvider.GenerateStreamResponse(ctx, preparedMessages, tools)
		if err != nil {
			log.Printf("❌ Chat: Failed to get follow-up response: %v", err)
			outputChan <- &models.StreamMessage{
				Type:      "error",
				Error:     fmt.Sprintf("Failed to get follow-up response: %v", err),
				SessionID: approval.SessionID,
			}
			outputChan <- &models.StreamMessage{
				Type:      "done",
				SessionID: approval.SessionID,
			}
			return
		}

		// Process follow-up streaming response
		var fullContent string
		var messageCount int
		s.handleStreamMessage(ctx, approval.SessionID, session, followUpChan, outputChan, &fullContent, tools, &messageCount)
	}()

	return outputChan, nil
}

// processToolApprovalInStreamForUser processes tool approval within the streaming flow for a specific user
func (s *ChatService) processToolApprovalInStreamForUser(ctx context.Context, userID, sessionID, requestID string, approved bool) (<-chan *models.StreamMessage, error) {
	log.Printf("🔧 Chat: Processing tool approval in stream for user %s, session %s, request %s, approved: %t", userID, sessionID, requestID, approved)

	// Create the approval response structure and delegate to ProcessToolApproval
	approval := &models.ToolApprovalResponse{
		SessionID: sessionID,
		RequestID: requestID,
		Approved:  approved,
	}

	return s.ProcessToolApproval(ctx, approval)
}

// Close closes the chat service and cleans up resources
func (s *ChatService) Close() error {
	var errs []error

	// Close LLM service
	if s.llmProvider != nil {
		if err := s.llmProvider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close LLM service: %w", err))
		}
	}

	// Close MCP service
	if s.mcp != nil {
		if err := s.mcp.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MCP service: %w", err))
		}
	}

	// Clear sessions
	s.mu.Lock()
	s.userSessions = make(map[string]map[string]*ChatSession)
	s.mu.Unlock()

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	return nil
}

// detectToolCallAttempts analyzes response content to detect if LLM tried to call tools without proper format
func (s *ChatService) detectToolCallAttempts(content string) bool {
	// Common patterns that suggest tool usage attempts
	toolPatterns := []string{
		"I'll use the",
		"Let me use",
		"I'll execute",
		"Let me execute",
		"Using the",
		"I need to use",
		"I should use",
		"Let me call",
		"I'll call the",
		"calling the",
		"tool:",
		"function:",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range toolPatterns {
		if strings.Contains(lowerContent, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// ProcessMessageWithAttachmentsForUser processes a user message with file attachments and generates a response for a specific user
func (s *ChatService) ProcessMessageWithAttachmentsForUser(ctx context.Context, userID, sessionID, userMessage string, attachments []*models.Attachment) (*models.ChatResponse, error) {
	log.Printf("🚀 Chat: Processing message with attachments for user %s, session %s, message: %q, attachments: %d", userID, sessionID, userMessage, len(attachments))

	// Handle tool approval/denial messages
	if response, handled := s.handleToolApprovalMessage(ctx, sessionID, userMessage); handled {
		return response, nil
	}

	s.mu.Lock()
	session := s.getOrCreateSessionForUser(userID, sessionID)
	s.mu.Unlock()

	// Create and add user message
	userMsg := s.createUserMessage(userMessage, attachments)
	s.addMessageToSession(ctx, sessionID, session, userMsg)

	log.Printf("📝 Chat: Added user message to session, total messages: %d", len(session.Messages))

	// Generate LLM response
	assistantMsg, err := s.generateLLMResponse(ctx, session)
	if err != nil {
		return nil, err
	}

	// Add assistant message to session
	s.addMessageToSession(ctx, sessionID, session, assistantMsg)

	log.Printf("💾 Chat: Saved assistant message to session, total messages: %d", len(session.Messages))

	// Handle tool calls if present
	if len(assistantMsg.ToolCalls) > 0 {
		s.handleToolCallApprovalRequest(sessionID, assistantMsg)
	}

	return &models.ChatResponse{
		Message:   assistantMsg,
		SessionID: sessionID,
	}, nil
}

// handleToolApprovalMessage handles tool approval/denial messages and returns the response if handled
func (s *ChatService) handleToolApprovalMessage(ctx context.Context, sessionID, userMessage string) (*models.ChatResponse, bool) {
	var requestID string
	var approved bool
	var handled bool

	if strings.HasPrefix(userMessage, "[APPROVE_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		requestID = strings.TrimSuffix(strings.TrimPrefix(userMessage, "[APPROVE_TOOLS:"), "]")
		approved = true
		handled = true
		log.Printf("🔧 Chat: Detected tool approval message for request: %s", requestID)
	} else if strings.HasPrefix(userMessage, "[DENY_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		requestID = strings.TrimSuffix(strings.TrimPrefix(userMessage, "[DENY_TOOLS:"), "]")
		approved = false
		handled = true
		log.Printf("🔧 Chat: Detected tool denial message for request: %s", requestID)
	}

	if !handled {
		return nil, false
	}

	// Process tool approval/denial
	approval := &models.ToolApprovalResponse{
		SessionID: sessionID,
		RequestID: requestID,
		Approved:  approved,
	}

	streamChan, err := s.ProcessToolApproval(ctx, approval)
	if err != nil {
		return nil, true // handled but with error
	}

	response := s.collectStreamResponse(streamChan, sessionID, approved)
	return response, true
}

// collectStreamResponse collects stream messages into a single ChatResponse
func (s *ChatService) collectStreamResponse(streamChan <-chan *models.StreamMessage, sessionID string, approved bool) *models.ChatResponse {
	var content strings.Builder
	var toolExecutions []models.ToolExecution

	for streamMsg := range streamChan {
		if streamMsg.Content != "" {
			content.WriteString(streamMsg.Content)
		}
		if len(streamMsg.ToolExecutions) > 0 {
			toolExecutions = append(toolExecutions, streamMsg.ToolExecutions...)
		}
	}

	messageID := fmt.Sprintf("approval_result_%d", time.Now().UnixNano())
	if !approved {
		messageID = fmt.Sprintf("denial_result_%d", time.Now().UnixNano())
	}

	message := &models.Message{
		ID:        messageID,
		Role:      "assistant",
		Content:   content.String(),
		Timestamp: time.Now(),
	}

	if len(toolExecutions) > 0 {
		message.ToolExecutions = toolExecutions
	}

	return &models.ChatResponse{
		Message:   message,
		SessionID: sessionID,
	}
}

// createUserMessage creates a user message with attachments
func (s *ChatService) createUserMessage(userMessage string, attachments []*models.Attachment) *models.Message {
	var msgAttachments []models.Attachment
	if attachments != nil {
		msgAttachments = make([]models.Attachment, len(attachments))
		for i, att := range attachments {
			msgAttachments[i] = *att
		}
	}

	return &models.Message{
		ID:          fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Role:        "user",
		Content:     userMessage,
		Timestamp:   time.Now(),
		Attachments: msgAttachments,
	}
}

// addMessageToSession safely adds a message to the session and saves it to database
func (s *ChatService) addMessageToSession(ctx context.Context, sessionID string, session *ChatSession, message *models.Message) {
	s.mu.Lock()
	session.Messages = append(session.Messages, message)
	session.Updated = time.Now()
	s.mu.Unlock()

	// Save message to database
	if err := s.database.SaveMessage(ctx, sessionID, message); err != nil {
		log.Printf("⚠️  Chat: Failed to save message to database: %v", err)
	} else {
		log.Printf("💾 Chat: Saved message to database")
	}
}

// generateLLMResponse generates an LLM response for the session
func (s *ChatService) generateLLMResponse(ctx context.Context, session *ChatSession) (*models.Message, error) {
	// Get available tools
	tools := s.mcp.GetTools()
	log.Printf("🔧 Chat: Retrieved %d available tools", len(tools))

	// Prepare messages for LLM
	s.mu.RLock()
	allMessages := make([]*models.Message, len(session.Messages))
	copy(allMessages, session.Messages)
	s.mu.RUnlock()

	preparedMessages := s.prepareMessagesWithSystemPrompt(allMessages)

	// Generate response
	response, err := s.llmProvider.GenerateResponse(ctx, preparedMessages, tools)
	if err != nil {
		log.Printf("❌ Chat: LLM generation failed: %v", err)
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	log.Printf("✅ Chat: LLM generated response: %q", response.Content)

	// Create assistant message
	return &models.Message{
		ID:        fmt.Sprintf("assistant_%d", time.Now().UnixNano()),
		Role:      "assistant",
		Content:   response.Content,
		Timestamp: time.Now(),
		ToolCalls: response.ToolCalls,
	}, nil
}

// handleToolCallApprovalRequest handles tool call approval requests
func (s *ChatService) handleToolCallApprovalRequest(sessionID string, assistantMsg *models.Message) {
	log.Printf("🔧 Chat: Requesting approval for %d tool call(s)", len(assistantMsg.ToolCalls))

	// Generate approval request ID
	requestID := fmt.Sprintf("approval_%d", time.Now().UnixNano())

	// Create approval request
	approvalRequest := &models.ToolApprovalRequest{
		SessionID: sessionID,
		ToolCalls: assistantMsg.ToolCalls,
		RequestID: requestID,
	}

	// Store pending approval
	s.mu.Lock()
	s.pendingApprovals[requestID] = approvalRequest
	s.mu.Unlock()

	// Update assistant message with approval request
	assistantMsg.ApprovalRequest = &struct {
		RequestID string            `json:"request_id"`
		ToolCalls []models.ToolCall `json:"tool_calls"`
		Processed bool              `json:"processed,omitempty"`
	}{
		RequestID: requestID,
		ToolCalls: assistantMsg.ToolCalls,
		Processed: false,
	}
}

// GetAvailableTools returns all available MCP tools
func (s *ChatService) GetAvailableTools() []models.MCPTool {
	return s.mcp.GetTools()
}

// RefreshTools forces a refresh of tools from MCP servers
func (s *ChatService) RefreshTools() error {
	return s.mcp.RefreshTools()
}
