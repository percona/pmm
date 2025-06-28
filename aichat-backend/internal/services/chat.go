package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

// ChatService manages chat sessions and coordinates with LLM and MCP services
type ChatService struct {
	llmProvider      LLMProvider
	mcp              *MCPService
	database         *DatabaseService
	streamManager    *StreamManager
	userSessions     map[string]map[string]*ChatSession // userID -> sessionID -> session
	systemPrompt     string
	pendingApprovals map[string]*models.ToolApprovalRequest // requestID -> approval request
	sessionsMu       sync.RWMutex                           // Protects userSessions
	approvalsMu      sync.RWMutex                           // Protects pendingApprovals
	promptMu         sync.RWMutex                           // Protects systemPrompt
	l                *logrus.Entry                          // Logger with component field
}

// ChatSession represents a chat session for a user
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
		streamManager:    NewStreamManager(),
		userSessions:     make(map[string]map[string]*ChatSession),
		systemPrompt:     "",
		pendingApprovals: make(map[string]*models.ToolApprovalRequest),
		l:                logrus.WithField("component", "chat-service"),
	}
}

// SetSystemPrompt sets the system prompt for the chat service
func (s *ChatService) SetSystemPrompt(prompt string) {
	s.promptMu.Lock()
	defer s.promptMu.Unlock()
	s.systemPrompt = prompt
}

// InitiateStreamForUser initiates a streaming conversation and returns a stream ID
func (s *ChatService) InitiateStreamForUser(ctx context.Context, userID, sessionID, userMessage string) (string, error) {
	s.l.WithFields(logrus.Fields{
		"user_id":    userID,
		"session_id": sessionID,
	}).Debug("Initiating stream for user")

	// Create a unique stream ID
	streamID := fmt.Sprintf("stream_%s_%d", userID, time.Now().UnixNano())

	// Start the stream processing in the background
	streamChan, err := s.ProcessStreamMessageForUser(ctx, userID, sessionID, userMessage)
	if err != nil {
		return "", err
	}

	// Store the stream channel for later retrieval using StreamManager
	s.streamManager.StoreStream(streamID, userID, streamChan)

	s.l.WithField("stream_id", streamID).Debug("Stream initiated successfully")
	return streamID, nil
}

// ConnectToStream connects to an existing stream by ID
func (s *ChatService) ConnectToStream(ctx context.Context, userID, streamID string) (<-chan *models.StreamMessage, error) {
	s.l.WithFields(logrus.Fields{
		"user_id":   userID,
		"stream_id": streamID,
	}).Debug("Connecting to stream")

	// Use StreamManager to connect to the stream
	return s.streamManager.ConnectToStream(ctx, userID, streamID)
}

// ProcessStreamMessageForUser processes a user message and returns a streaming response for a specific user
func (s *ChatService) ProcessStreamMessageForUser(ctx context.Context, userID, sessionID, userMessage string) (<-chan *models.StreamMessage, error) {
	s.l.WithFields(logrus.Fields{
		"user_id":     userID,
		"session_id":  sessionID,
		"message_len": len(userMessage),
	}).Debug("Starting stream processing for user")

	// Check if this is a tool approval message
	if strings.HasPrefix(userMessage, "[APPROVE_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		// Extract request ID
		requestID := strings.TrimSuffix(strings.TrimPrefix(userMessage, "[APPROVE_TOOLS:"), "]")
		s.l.WithField("request_id", requestID).Debug("Detected tool approval message")

		// Process tool approval directly in streaming
		return s.processToolApprovalInStreamForUser(ctx, userID, sessionID, requestID, true)
	}

	if strings.HasPrefix(userMessage, "[DENY_TOOLS:") && strings.HasSuffix(userMessage, "]") {
		// Extract request ID
		requestID := strings.TrimSuffix(strings.TrimPrefix(userMessage, "[DENY_TOOLS:"), "]")
		s.l.WithField("request_id", requestID).Debug("Detected tool denial message")

		// Process tool denial directly in streaming
		return s.processToolApprovalInStreamForUser(ctx, userID, sessionID, requestID, false)
	}

	s.sessionsMu.Lock()
	session := s.getOrCreateSessionForUser(userID, sessionID)
	actualSessionID := session.ID // Get the actual session ID (might be different if generated)
	s.sessionsMu.Unlock()

	s.l.WithFields(logrus.Fields{
		"actual_session_id": actualSessionID,
		"original_session":  sessionID,
	}).Debug("Using session for processing")

	// Add user message to session
	userMsg := &models.Message{
		ID:        fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}

	s.addMessageToSession(ctx, actualSessionID, session, userMsg)

	s.l.WithFields(logrus.Fields{
		"session_id":     actualSessionID,
		"total_messages": len(session.Messages),
	}).Debug("Added user message to session")

	// Get available tools
	tools := s.mcp.GetTools()
	s.l.WithField("tool_count", len(tools)).Debug("Retrieved available tools for streaming")

	// Prepare messages with system prompt
	preparedMessages := s.prepareMessagesWithSystemPrompt(session.Messages)

	// Generate streaming response from LLM
	streamChan, err := s.llmProvider.GenerateStreamResponse(ctx, preparedMessages, tools)
	if err != nil {
		s.l.WithError(err).Error("Failed to start LLM streaming")
		return nil, err
	}

	s.l.Debug("LLM streaming started successfully")

	// Create output channel
	outputChan := make(chan *models.StreamMessage, 10)

	// Process stream
	go func() {
		defer close(outputChan)
		defer s.l.WithField("session_id", actualSessionID).Debug("Stream processing completed")

		var fullContent string
		var messageCount int

		s.l.Debug("Starting stream message processing loop")

		s.handleStreamMessage(ctx, actualSessionID, session, streamChan, outputChan, &fullContent, tools, &messageCount)

		s.l.Warn("Stream channel closed without 'done' message")
	}()

	return outputChan, nil
}

func (s *ChatService) handleStreamMessage(ctx context.Context, sessionID string, session *ChatSession, streamChan <-chan *models.StreamMessage, outputChan chan<- *models.StreamMessage, fullContent *string, tools []models.MCPTool, messageCount *int) {
	var streamToolCalls []models.ToolCall

	for streamMsg := range streamChan {
		(*messageCount)++
		streamMsg.SessionID = sessionID

		s.l.WithField("stream_message_count", *messageCount).Debug("Processing stream message")

		if streamMsg.Type == "message" {
			*fullContent += streamMsg.Content
			s.l.WithField("accumulated_content_length", len(*fullContent)).Debug("Accumulated content length")
			outputChan <- streamMsg
		} else if streamMsg.Type == "tool_call" {
			// Handle tool calls in streaming - collect them for execution after stream completes
			s.l.WithField("streaming_tool_call", streamMsg.Content).Debug("Streaming tool call detected")

			// Parse tool call from content (format: "Function call: toolname(args)")
			if strings.HasPrefix(streamMsg.Content, "Function call: ") {
				funcCallStr := strings.TrimPrefix(streamMsg.Content, "Function call: ")

				// Parse function name and arguments
				parenIndex := strings.Index(funcCallStr, "(")
				if parenIndex > 0 {
					funcName := funcCallStr[:parenIndex]
					argsStr := strings.TrimSuffix(funcCallStr[parenIndex+1:], ")")

					s.l.WithFields(logrus.Fields{
						"function_name": funcName,
						"function_args": argsStr,
					}).Debug("Parsing streaming tool call")

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
					s.l.WithField("collected_tool_call", funcName).Debug("Collected tool call for later execution")

					// Send notification to user that tool will be executed
					outputChan <- &models.StreamMessage{
						Type:      "message",
						Content:   fmt.Sprintf("🔧 Executing %s...\n", funcName),
						SessionID: sessionID,
					}
				} else {
					s.l.WithField("invalid_tool_call_format", funcCallStr).Error("Invalid tool call format")
				}
			} else {
				s.l.WithField("unexpected_tool_call_format", streamMsg.Content).Error("Unexpected tool call format")
			}
		} else if streamMsg.Type == "error" {
			s.l.WithField("error", streamMsg.Error).Error("Streaming error")
			outputChan <- streamMsg
			return
		} else if streamMsg.Type == "done" {
			// Check if the done message includes tool calls (Gemini style)
			if len(streamMsg.ToolCalls) > 0 {
				s.l.WithField("tool_calls_in_done_message", len(streamMsg.ToolCalls)).Debug("Done message includes tool calls from LLM")
				streamToolCalls = append(streamToolCalls, streamMsg.ToolCalls...)
			}

			s.l.WithFields(logrus.Fields{
				"session_id":     sessionID,
				"total_messages": *messageCount,
				"content_length": len(*fullContent),
				"tool_calls":     len(streamToolCalls),
			}).Debug("Stream completed")

			// Save complete message to session
			assistantMsg := &models.Message{
				ID:        fmt.Sprintf("assistant_%d", time.Now().UnixNano()),
				Role:      "assistant",
				Content:   *fullContent,
				Timestamp: time.Now(),
				ToolCalls: streamToolCalls,
			}

			s.addMessageToSession(ctx, sessionID, session, assistantMsg)

			s.l.WithField("session_id", sessionID).Debug("Saved assistant message to session")

			// Generate and update session title if this is the first exchange (2 messages: user + assistant)
			// Note: We need to count only user/assistant messages, not tool approval messages
			userMessages := 0
			assistantMessages := 0
			for _, msg := range session.Messages {
				if msg.Role == "user" {
					userMessages++
				} else if msg.Role == "assistant" {
					assistantMessages++
				}
			}

			if userMessages == 1 && assistantMessages == 1 {
				// Find the first user message and current assistant message
				var firstUserMessage, firstAssistantMessage string
				for _, msg := range session.Messages {
					if msg.Role == "user" && firstUserMessage == "" {
						firstUserMessage = msg.Content
					} else if msg.Role == "assistant" && firstAssistantMessage == "" {
						firstAssistantMessage = msg.Content
						break
					}
				}

				// Generate title in background to avoid blocking the response
				if firstUserMessage != "" && firstAssistantMessage != "" {
					go func(userMsg, assistantMsg string) {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()

						newTitle := s.generateSessionTitle(ctx, userMsg, assistantMsg)
						// We need to extract userID from session context - for now use a placeholder
						// In real implementation, we should pass userID to this function
						s.updateSessionTitle(ctx, sessionID, session.UserID, newTitle)
					}(firstUserMessage, firstAssistantMessage)
				}
			}

			// Handle tool calls if present (request approval instead of executing immediately)
			if len(streamToolCalls) > 0 {
				s.l.WithField("tool_calls_in_streaming_response", len(streamToolCalls)).Debug("Requesting approval for tool calls from streaming response")

				// Generate approval request ID
				requestID := fmt.Sprintf("approval_%d", time.Now().UnixNano())

				// Create approval request
				approvalRequest := &models.ToolApprovalRequest{
					SessionID: sessionID,
					ToolCalls: streamToolCalls,
					RequestID: requestID,
				}

				// Store pending approval
				s.approvalsMu.Lock()
				s.pendingApprovals[requestID] = approvalRequest
				s.approvalsMu.Unlock()

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
					s.l.WithField("streaming_response_content", *fullContent).Warn("Streaming response suggests tool usage but no tool calls were detected")
				}

				// Send done message for non-tool responses
				outputChan <- &models.StreamMessage{
					Type:      "done",
					SessionID: sessionID,
				}
			}
		} else {
			s.l.WithField("unknown_stream_message_type", streamMsg.Type).Warn("Unknown stream message type")
		}
	}
}

// ClearHistoryForUser clears chat history for a specific user's session
func (s *ChatService) ClearHistoryForUser(userID, sessionID string) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

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
	// If no sessionID provided, generate a new one
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
		s.l.WithField("generated_session_id", sessionID).Debug("Generated new session ID")
	}

	// Ensure user sessions map exists
	if s.userSessions[userID] == nil {
		s.userSessions[userID] = make(map[string]*ChatSession)
	}

	// Check if session exists in memory
	session, exists := s.userSessions[userID][sessionID]
	if exists {
		return session
	}

	// Check if session exists in database
	ctx := context.Background()
	dbSession, err := s.database.GetSession(ctx, sessionID, userID)
	if err == nil {
		// Session exists in database, load it into memory
		s.l.WithField("loaded_session_id", sessionID).Debug("Loaded existing session from database")
		session = &ChatSession{
			ID:       dbSession.ID,
			UserID:   dbSession.UserID,
			Messages: []*models.Message{}, // Messages will be loaded on demand
			Created:  dbSession.CreatedAt,
			Updated:  dbSession.UpdatedAt,
		}
		s.userSessions[userID][sessionID] = session

		// Load pending approvals from database
		s.loadPendingApprovalsFromDB(ctx, sessionID)

		return session
	}

	// Session doesn't exist, create new one
	s.l.WithField("created_session_id", sessionID).Debug("Creating new session in database")

	// Create session in database first with a temporary title
	defaultTitle := fmt.Sprintf("Chat - %s", time.Now().Format("Jan 2, 2006 at 3:04 PM"))
	dbSession, err = s.database.CreateSession(ctx, userID, defaultTitle)
	if err != nil {
		s.l.WithError(err).Error("Failed to create session in database")
		// Fallback to in-memory session
		session = &ChatSession{
			ID:       sessionID,
			UserID:   userID,
			Messages: []*models.Message{},
			Created:  time.Now(),
			Updated:  time.Now(),
		}
	} else {
		// Use the session created in database
		s.l.WithField("created_session_id", dbSession.ID).Debug("Successfully created session in database")
		session = &ChatSession{
			ID:       dbSession.ID,
			UserID:   dbSession.UserID,
			Messages: []*models.Message{},
			Created:  dbSession.CreatedAt,
			Updated:  dbSession.UpdatedAt,
		}
		// Update sessionID to match what was created in database
		sessionID = dbSession.ID
	}

	s.userSessions[userID][sessionID] = session
	return session
}

// prepareMessagesWithSystemPrompt prepares messages for LLM with system prompt
func (s *ChatService) prepareMessagesWithSystemPrompt(messages []*models.Message) []*models.Message {
	s.promptMu.RLock()
	systemPrompt := s.systemPrompt
	s.promptMu.RUnlock()

	if systemPrompt == "" {
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

	enhancedSystemPrompt := systemPrompt + timeContext

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
	if len(toolCalls) == 0 {
		return nil, nil
	}

	s.l.WithField("tool_calls_count", len(toolCalls)).Debug("Executing tool calls in parallel")

	// Create channels for results
	type toolResult struct {
		execution models.ToolExecution
		message   *models.Message
		index     int
	}

	resultChan := make(chan toolResult, len(toolCalls))
	var wg sync.WaitGroup

	// Execute all tools in parallel
	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(index int, tc models.ToolCall) {
			defer wg.Done()

			s.l.WithFields(logrus.Fields{
				"tool_call_index": index + 1,
				"tool_call_id":    tc.ID,
				"tool_call_type":  tc.Type,
				"tool_call_name":  tc.Function.Name,
				"tool_call_args":  tc.Function.Arguments,
			}).Debug("Executing tool call")

			// Track tool execution
			startTime := time.Now()
			toolExecution := models.ToolExecution{
				ID:        tc.ID,
				ToolName:  tc.Function.Name,
				Arguments: tc.Function.Arguments,
				StartTime: startTime,
			}

			// Execute tool
			toolResultStr, err := s.mcp.ExecuteTool(ctx, tc)
			endTime := time.Now()
			toolExecution.EndTime = endTime
			toolExecution.Duration = endTime.Sub(startTime).Milliseconds()

			if err != nil {
				s.l.WithError(err).Error("Tool execution failed")
				toolExecution.Error = err.Error()
				toolResultStr = fmt.Sprintf("Error executing tool: %v", err)
			} else {
				s.l.WithField("tool_execution_result_length", len(toolResultStr)).Debug("Tool execution successful")
			}

			toolExecution.Result = toolResultStr

			// Add tool result as a message
			toolMsg := &models.Message{
				ID:        fmt.Sprintf("tool_%s", tc.ID),
				Role:      "tool",
				Content:   toolResultStr,
				Timestamp: time.Now(),
			}

			// Send result through channel
			resultChan <- toolResult{
				execution: toolExecution,
				message:   toolMsg,
				index:     index,
			}
		}(i, toolCall)
	}

	// Wait for all tools to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in original order
	results := make([]toolResult, len(toolCalls))
	for result := range resultChan {
		results[result.index] = result
	}

	// Extract executions and messages in order
	var toolExecutions []models.ToolExecution
	var toolMessages []*models.Message

	for _, result := range results {
		toolExecutions = append(toolExecutions, result.execution)
		toolMessages = append(toolMessages, result.message)

		// Add to session and save to database
		s.addMessageToSession(ctx, session.ID, session, result.message)
	}

	s.l.WithField("completed_tool_calls_count", len(toolExecutions)).Debug("Completed parallel execution of tools")
	return toolExecutions, toolMessages
}

// ProcessToolApproval processes user's approval/denial of tool execution
func (s *ChatService) ProcessToolApproval(ctx context.Context, approval *models.ToolApprovalResponse) (<-chan *models.StreamMessage, error) {
	s.l.WithField("request_id", approval.RequestID).Debug("Processing tool approval for request")

	s.approvalsMu.Lock()
	approvalRequest, exists := s.pendingApprovals[approval.RequestID]
	if exists {
		delete(s.pendingApprovals, approval.RequestID)
	}
	s.approvalsMu.Unlock()

	if !exists {
		return nil, fmt.Errorf("approval request %s not found or expired", approval.RequestID)
	}

	// Create output channel
	outputChan := make(chan *models.StreamMessage, 10)

	// Process approval
	go func() {
		defer close(outputChan)

		if !approval.Approved {
			s.l.WithField("request_id", approval.RequestID).Warn("Tool execution denied by user")
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

		s.l.WithField("request_id", approval.RequestID).Debug("Tool execution approved by user")

		s.sessionsMu.Lock()
		session := s.getOrCreateSessionForUser(approval.UserID, approval.SessionID)
		s.sessionsMu.Unlock()

		s.l.WithFields(logrus.Fields{
			"session_id": approval.SessionID,
			"user_id":    approval.UserID,
		}).Debug("Using session for user")

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

		s.l.WithField("tool_calls_to_execute_count", len(toolsToExecute)).Debug("Executing approved tool(s)")

		// Execute approved tools using common method
		toolExecutions, toolMessages := s.executeToolCalls(ctx, toolsToExecute, session)

		// Check if all tool executions were successful
		allSuccessful := true
		var failedTools []string
		for _, execution := range toolExecutions {
			if execution.Error != "" {
				allSuccessful = false
				failedTools = append(failedTools, execution.ToolName)
			}
		}

		// Only mark approval as processed if all tool executions were successful
		if allSuccessful {
			if err := s.database.MarkApprovalProcessed(ctx, approval.SessionID, approval.RequestID); err != nil {
				s.l.WithError(err).Warn("Failed to mark approval as processed in database")
				// Continue processing even if database update fails
			} else {
				s.l.WithField("request_id", approval.RequestID).Debug("Marked approval as processed after successful tool execution")
			}
		} else {
			s.l.WithField("failed_tools", failedTools).Warn("Not marking approval as processed due to failed tool executions")
		}

		// Add tool executions to the last assistant message
		s.sessionsMu.Lock()
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
		s.sessionsMu.Unlock()

		// Send tool execution information to the frontend
		executionStatus := "successful"
		if !allSuccessful {
			executionStatus = "partially failed"
		}
		outputChan <- &models.StreamMessage{
			Type:           "tool_execution",
			Content:        fmt.Sprintf("�� Executed %d tool(s) (%s)", len(toolExecutions), executionStatus),
			SessionID:      approval.SessionID,
			ToolExecutions: toolExecutions,
		}

		s.l.WithField("tool_results_count", len(toolMessages)).Debug("Getting LLM follow-up response with tool results")

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
			s.l.WithError(err).Error("Failed to get follow-up response")
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
	s.l.WithFields(logrus.Fields{
		"user_id":    userID,
		"session_id": sessionID,
		"request_id": requestID,
		"approved":   approved,
	}).Debug("Processing tool approval in stream for user")

	// Create the approval response structure and delegate to ProcessToolApproval
	approval := &models.ToolApprovalResponse{
		SessionID: sessionID,
		RequestID: requestID,
		Approved:  approved,
		UserID:    userID,
	}

	return s.ProcessToolApproval(ctx, approval)
}

// Close closes the chat service and cleans up resources
func (s *ChatService) Close() error {
	var errs []error

	// Close StreamManager
	if s.streamManager != nil {
		if err := s.streamManager.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close StreamManager: %w", err))
		}
	}

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
	s.sessionsMu.Lock()
	s.userSessions = make(map[string]map[string]*ChatSession)
	s.sessionsMu.Unlock()

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

// ProcessStreamMessageWithAttachmentsForUser processes a user message with file attachments and returns a streaming response for a specific user
func (s *ChatService) ProcessStreamMessageWithAttachmentsForUser(ctx context.Context, userID, sessionID, userMessage string, attachments []*models.Attachment) (<-chan *models.StreamMessage, error) {

	s.l.WithFields(logrus.Fields{
		"user_id":     userID,
		"session_id":  sessionID,
		"message_len": len(userMessage),
		"attachments": len(attachments),
	}).Debug("Processing stream message with attachments for user")

	// Handle tool approval/denial messages
	if strings.HasPrefix(userMessage, "[APPROVE_TOOLS:") || strings.HasPrefix(userMessage, "[DENY_TOOLS:") {
		// For tool approval messages, delegate to the existing streaming method
		return s.ProcessStreamMessageForUser(ctx, userID, sessionID, userMessage)
	}

	s.sessionsMu.Lock()
	session := s.getOrCreateSessionForUser(userID, sessionID)
	actualSessionID := session.ID // Get the actual session ID (might be different if generated)
	s.sessionsMu.Unlock()

	s.l.WithFields(logrus.Fields{
		"actual_session_id": actualSessionID,
		"original_session":  sessionID,
	}).Debug("Using session for processing")

	// Create and add user message with attachments
	userMsg := s.createUserMessage(userMessage, attachments)
	s.addMessageToSession(ctx, actualSessionID, session, userMsg)

	s.l.WithFields(logrus.Fields{
		"session_id":     actualSessionID,
		"total_messages": len(session.Messages),
	}).Debug("Added user message with attachments to session")

	// Get available tools
	tools := s.mcp.GetTools()
	s.l.WithField("tool_count", len(tools)).Debug("Retrieved available tools for streaming")

	// Prepare messages with system prompt
	preparedMessages := s.prepareMessagesWithSystemPrompt(session.Messages)

	// Generate streaming response from LLM
	streamChan, err := s.llmProvider.GenerateStreamResponse(ctx, preparedMessages, tools)
	if err != nil {
		s.l.WithError(err).Error("Failed to start LLM streaming")
		return nil, err
	}

	s.l.Debug("LLM streaming started successfully")

	// Create output channel
	outputChan := make(chan *models.StreamMessage, 10)

	// Process stream
	go func() {
		defer close(outputChan)
		defer s.l.WithField("session_id", actualSessionID).Debug("Stream processing completed")

		var fullContent string
		var messageCount int

		s.l.Debug("Starting stream message processing loop")

		s.handleStreamMessage(ctx, actualSessionID, session, streamChan, outputChan, &fullContent, tools, &messageCount)

		s.l.Warn("Stream channel closed without 'done' message")
	}()

	return outputChan, nil
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
	s.sessionsMu.Lock()
	session.Messages = append(session.Messages, message)
	session.Updated = time.Now()
	s.sessionsMu.Unlock()

	// Save message to database
	if err := s.database.SaveMessage(ctx, sessionID, message); err != nil {
		s.l.WithError(err).Warn("Failed to save message to database")
	} else {
		s.l.WithField("session_id", sessionID).Debug("Saved message to database")
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

// GetStreamManager returns the stream manager for direct access
func (s *ChatService) GetStreamManager() *StreamManager {
	return s.streamManager
}

// generateSessionTitle creates a concise title for a session based on the conversation
func (s *ChatService) generateSessionTitle(ctx context.Context, userMessage, assistantResponse string) string {
	// Create a simple prompt for title generation
	titlePrompt := fmt.Sprintf(`Based on this conversation, generate a concise, descriptive title (max 50 characters):

User: %s
Assistant: %s

Respond with ONLY the title, no explanations or quotes. Make it specific and helpful for identifying this conversation later.`,
		userMessage, assistantResponse)

	// Prepare messages for title generation
	titleMessages := []*models.Message{
		{
			ID:        "title_request",
			Role:      "user",
			Content:   titlePrompt,
			Timestamp: time.Now(),
		},
	}

	// Generate title using LLM (without tools)
	titleResponse, err := s.llmProvider.GenerateResponse(ctx, titleMessages, nil)
	if err != nil {
		s.l.WithError(err).Warn("Failed to generate session title")
		// Fallback to extract key words from user message
		return s.generateFallbackTitle(userMessage)
	}

	// Clean up the response
	title := strings.TrimSpace(titleResponse.Content)
	title = strings.Trim(title, "\"'")

	// Limit length
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	// Ensure we have a meaningful title
	if len(title) < 5 || strings.ToLower(title) == "untitled" {
		return s.generateFallbackTitle(userMessage)
	}

	s.l.WithField("generated_session_title", title).Debug("Generated session title")
	return title
}

// generateFallbackTitle creates a fallback title from the user message
func (s *ChatService) generateFallbackTitle(userMessage string) string {
	// Take first few words of user message
	words := strings.Fields(userMessage)
	if len(words) == 0 {
		return "New Chat"
	}

	// Take up to 6 words or 40 characters, whichever comes first
	var titleWords []string
	totalLength := 0
	for i, word := range words {
		if i >= 6 || totalLength+len(word) > 40 {
			break
		}
		titleWords = append(titleWords, word)
		totalLength += len(word) + 1 // +1 for space
	}

	title := strings.Join(titleWords, " ")
	if len(words) > len(titleWords) {
		title += "..."
	}

	return title
}

// updateSessionTitle updates the session title in the database
func (s *ChatService) updateSessionTitle(ctx context.Context, sessionID, userID, newTitle string) {
	if s.database == nil {
		return
	}

	err := s.database.UpdateSession(ctx, sessionID, userID, newTitle)
	if err != nil {
		s.l.WithError(err).Warn("Failed to update session title")
	} else {
		s.l.WithField("updated_session_title", newTitle).Debug("Updated session title")
	}
}

// loadPendingApprovalsFromDB loads pending approvals from the database for a specific session
func (s *ChatService) loadPendingApprovalsFromDB(ctx context.Context, sessionID string) {
	approvals, err := s.database.GetPendingApprovals(ctx, sessionID)
	if err != nil {
		s.l.WithError(err).Warn("Failed to load pending approvals from database")
		return
	}

	s.approvalsMu.Lock()
	defer s.approvalsMu.Unlock()

	for _, approval := range approvals {
		s.pendingApprovals[approval.RequestID] = approval
	}

	s.l.WithField("loaded_approvals_count", len(approvals)).Debug("Loaded pending approvals from database for session")
}
