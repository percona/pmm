package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

// ChatService handles chat conversations and coordinates LLM and MCP services
type ChatService struct {
	llm              *LLMService
	mcp              *MCPService
	sessions         map[string]*ChatSession
	systemPrompt     string
	pendingApprovals map[string]*models.ToolApprovalRequest // requestID -> approval request
	mu               sync.RWMutex
}

// ChatSession represents an active chat session
type ChatSession struct {
	ID       string
	Messages []*models.Message
	Created  time.Time
	Updated  time.Time
}

// NewChatService creates a new chat service
func NewChatService(llmService *LLMService, mcpService *MCPService, systemPrompt string) *ChatService {
	return &ChatService{
		llm:              llmService,
		mcp:              mcpService,
		sessions:         make(map[string]*ChatSession),
		systemPrompt:     systemPrompt,
		pendingApprovals: make(map[string]*models.ToolApprovalRequest),
	}
}

// ProcessMessage processes a user message and generates a response
func (s *ChatService) ProcessMessage(ctx context.Context, sessionID, userMessage string) (*models.ChatResponse, error) {
	return s.ProcessMessageWithAttachments(ctx, sessionID, userMessage, nil)
}

// ProcessStreamMessage processes a user message and returns a streaming response
func (s *ChatService) ProcessStreamMessage(ctx context.Context, sessionID, userMessage string) (<-chan *models.StreamMessage, error) {
	log.Printf("🚀 Chat: Starting stream processing for session %s, message: %q", sessionID, userMessage)

	// Check if this is a tool approval message
	if strings.HasPrefix(userMessage, "[APPROVE_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		// Extract request ID
		requestID := strings.TrimSuffix(strings.TrimPrefix(userMessage, "[APPROVE_TOOLS:"), "]")
		log.Printf("🔧 Chat: Detected tool approval message for request: %s", requestID)

		// Process tool approval directly in streaming
		return s.processToolApprovalInStream(ctx, sessionID, requestID, true)
	}

	if strings.HasPrefix(userMessage, "[DENY_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		// Extract request ID
		requestID := strings.TrimSuffix(strings.TrimPrefix(userMessage, "[DENY_TOOLS:"), "]")
		log.Printf("🔧 Chat: Detected tool denial message for request: %s", requestID)

		// Process tool denial directly in streaming
		return s.processToolApprovalInStream(ctx, sessionID, requestID, false)
	}

	s.mu.Lock()
	session := s.getOrCreateSession(sessionID)
	s.mu.Unlock()

	// Add user message to session
	userMsg := &models.Message{
		ID:        fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	session.Messages = append(session.Messages, userMsg)
	session.Updated = time.Now()
	s.mu.Unlock()

	log.Printf("📝 Chat: Added user message to session, total messages: %d", len(session.Messages))

	// Get available tools
	tools := s.mcp.GetTools()
	log.Printf("🔧 Chat: Retrieved %d available tools for streaming", len(tools))

	// Prepare messages with system prompt
	preparedMessages := s.prepareMessagesWithSystemPrompt(session.Messages)

	// Generate streaming response from LLM
	streamChan, err := s.llm.GenerateStreamResponse(ctx, preparedMessages, tools)
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

			s.mu.Lock()
			session.Messages = append(session.Messages, assistantMsg)
			session.Updated = time.Now()
			totalMessages := len(session.Messages)
			s.mu.Unlock()

			log.Printf("💾 Chat: Saved assistant message to session, total messages: %d", totalMessages)

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

				s.mu.Lock()
				session.Messages = append(session.Messages, approvalMsg)
				session.Updated = time.Now()
				s.mu.Unlock()

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

// GetHistory returns the chat history for a session
func (s *ChatService) GetHistory(sessionID string) *models.ChatHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return &models.ChatHistory{
			SessionID: sessionID,
			Messages:  []*models.Message{},
		}
	}

	return &models.ChatHistory{
		SessionID: sessionID,
		Messages:  session.Messages,
	}
}

// ClearHistory clears the chat history for a session
func (s *ChatService) ClearHistory(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		session.Messages = []*models.Message{}
		session.Updated = time.Now()
	}

	return nil
}

// GetAvailableTools returns all available MCP tools
func (s *ChatService) GetAvailableTools() []models.MCPTool {
	return s.mcp.GetTools()
}

// GetMCPServerStatus returns the status of all MCP servers
func (s *ChatService) GetMCPServerStatus() interface{} {
	return s.mcp.GetServerStatus()
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
		session := s.getOrCreateSession(approval.SessionID)
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
		followUpChan, err := s.llm.GenerateStreamResponse(ctx, preparedMessages, tools)
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

// processToolApprovalInStream processes tool approval within the streaming flow
func (s *ChatService) processToolApprovalInStream(ctx context.Context, sessionID, requestID string, approved bool) (<-chan *models.StreamMessage, error) {
	log.Printf("🔧 Chat: Processing tool approval in stream for request %s, approved: %t", requestID, approved)

	// Create the approval response structure and delegate to ProcessToolApproval
	approval := &models.ToolApprovalResponse{
		SessionID: sessionID,
		RequestID: requestID,
		Approved:  approved,
	}

	return s.ProcessToolApproval(ctx, approval)
}

// getOrCreateSession gets or creates a chat session (caller must hold lock)
func (s *ChatService) getOrCreateSession(sessionID string) *ChatSession {
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		session = &ChatSession{
			ID:       sessionID,
			Messages: []*models.Message{},
			Created:  time.Now(),
			Updated:  time.Now(),
		}
		s.sessions[sessionID] = session
	}

	return session
}

// CleanupOldSessions removes sessions older than the specified duration
func (s *ChatService) CleanupOldSessions(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for sessionID, session := range s.sessions {
		if session.Updated.Before(cutoff) {
			delete(s.sessions, sessionID)
		}
	}
}

// Close closes the chat service and cleans up resources
func (s *ChatService) Close() error {
	var errs []error

	// Close LLM service
	if s.llm != nil {
		if err := s.llm.Close(); err != nil {
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
	s.sessions = make(map[string]*ChatSession)
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

// ProcessMessageWithAttachments processes a user message with file attachments and generates a response
func (s *ChatService) ProcessMessageWithAttachments(ctx context.Context, sessionID, userMessage string, attachments []models.Attachment) (*models.ChatResponse, error) {
	s.mu.Lock()
	session := s.getOrCreateSession(sessionID)
	s.mu.Unlock()

	// Add user message to session with original content and attachments (for UI display)
	userMsg := &models.Message{
		ID:          fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Role:        "user",
		Content:     userMessage, // Store original user message for UI
		Timestamp:   time.Now(),
		Attachments: attachments,
	}

	s.mu.Lock()
	session.Messages = append(session.Messages, userMsg)
	session.Updated = time.Now()
	s.mu.Unlock()

	// Create enhanced message for LLM processing (includes attachment descriptions)
	enhancedMessage := userMessage
	if len(attachments) > 0 {
		enhancedMessage = s.processAttachments(userMessage, attachments)
	}

	// Create a temporary enhanced user message for LLM context
	enhancedUserMsg := &models.Message{
		ID:          userMsg.ID,
		Role:        "user",
		Content:     enhancedMessage, // Enhanced content for LLM
		Timestamp:   userMsg.Timestamp,
		Attachments: attachments,
	}

	// Create messages array with enhanced content for LLM
	llmMessages := make([]*models.Message, len(session.Messages)-1) // All messages except the last user message
	copy(llmMessages, session.Messages[:len(session.Messages)-1])
	llmMessages = append(llmMessages, enhancedUserMsg) // Add enhanced user message

	// Get available tools
	tools := s.mcp.GetTools()

	// Prepare messages with system prompt
	preparedLLMMessages := s.prepareMessagesWithSystemPrompt(llmMessages)

	// Generate response from LLM using enhanced messages
	response, err := s.llm.GenerateResponse(ctx, preparedLLMMessages, tools)
	if err != nil {
		return &models.ChatResponse{
			SessionID: sessionID,
			Error:     err.Error(),
		}, nil
	}

	// Set timestamp and update session
	response.Timestamp = time.Now()

	s.mu.Lock()
	session.Messages = append(session.Messages, response)
	session.Updated = time.Now()
	s.mu.Unlock()

	// Handle tool calls if present
	if len(response.ToolCalls) > 0 {
		log.Printf("🔧 LLM attempted to call %d tool(s) in response %s", len(response.ToolCalls), response.ID)

		// Execute tools using common method
		toolExecutions, _ := s.executeToolCalls(ctx, response.ToolCalls, session)

		// Add tool executions to the response message
		response.ToolExecutions = toolExecutions

		// Generate follow-up response with tool results
		log.Printf("🔄 Generating follow-up response after tool execution(s)")

		// Create updated LLM messages with enhanced user message for follow-up
		updatedLLMMessages := make([]*models.Message, 0, len(session.Messages))
		for _, msg := range session.Messages {
			if msg.ID == userMsg.ID && msg.Role == "user" {
				// Replace with enhanced version for LLM
				updatedLLMMessages = append(updatedLLMMessages, enhancedUserMsg)
			} else {
				updatedLLMMessages = append(updatedLLMMessages, msg)
			}
		}

		var noTools []models.MCPTool // Disable tools for follow-up to avoid infinite loops
		preparedUpdatedMessages := s.prepareMessagesWithSystemPrompt(updatedLLMMessages)
		followUpResponse, err := s.llm.GenerateResponse(ctx, preparedUpdatedMessages, noTools)
		if err == nil {
			followUpResponse.Timestamp = time.Now()
			s.mu.Lock()
			session.Messages = append(session.Messages, followUpResponse)
			session.Updated = time.Now()
			s.mu.Unlock()
			response = followUpResponse
			log.Printf("✅ Follow-up response generated successfully")
		} else {
			log.Printf("❌ Failed to generate follow-up response: %v", err)
		}
	} else {
		// Check if the response content suggests tool usage without proper tool calls
		if s.detectToolCallAttempts(response.Content) {
			log.Printf("⚠️  LLM response suggests tool usage but no tool calls were detected. Response: %s", response.Content)
		}
	}

	return &models.ChatResponse{
		Message:   response,
		SessionID: sessionID,
	}, nil
}

// processAttachments processes file attachments and enhances the message content
func (s *ChatService) processAttachments(userMessage string, attachments []models.Attachment) string {
	if len(attachments) == 0 {
		return userMessage
	}

	enhancedMessage := userMessage + "\n\nAttached files:\n"

	for _, attachment := range attachments {
		enhancedMessage += fmt.Sprintf("\n--- File: %s (Size: %d bytes, Type: %s) ---\n",
			attachment.Filename, attachment.Size, attachment.MimeType)

		// For text files, include the content
		if s.isTextFile(attachment.MimeType) && attachment.Content != "" {
			// Decode base64 content
			if content, err := base64.StdEncoding.DecodeString(attachment.Content); err == nil {
				// Limit content size for very large files
				contentStr := string(content)
				if len(contentStr) > 50000 { // Limit to 50KB of text
					contentStr = contentStr[:50000] + "\n... (content truncated)"
				}
				enhancedMessage += contentStr
			}
		} else if s.isImageFile(attachment.MimeType) {
			enhancedMessage += "[Image file - binary content not displayed]"
		} else {
			enhancedMessage += "[Binary file - content not displayed]"
		}

		enhancedMessage += "\n--- End of file ---\n"
	}

	return enhancedMessage
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

	// Create system message
	systemMsg := &models.Message{
		ID:        "system_prompt",
		Role:      "system",
		Content:   s.systemPrompt,
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

		// Add to session
		s.mu.Lock()
		session.Messages = append(session.Messages, toolMsg)
		session.Updated = time.Now()
		s.mu.Unlock()
	}

	return toolExecutions, toolMessages
}
