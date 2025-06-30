package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/percona/pmm/aichat-backend/internal/models"
	"github.com/percona/pmm/aichat-backend/internal/services"
)

// ChatHandler handles HTTP requests for chat functionality
type ChatHandler struct {
	chatService *services.ChatService
}

// NewChatHandler creates a new chat handler
func NewChatHandler(chatService *services.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// collectStreamResponse collects stream messages into a single ChatResponse
func (h *ChatHandler) collectStreamResponse(streamChan <-chan *models.StreamMessage, sessionID string) *models.ChatResponse {
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

	message := &models.Message{
		ID:        generateID("collected"),
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

// SendMessage handles POST /v1/chat/send
func (h *ChatHandler) SendMessage(c *gin.Context) {
	var request models.ChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Check if streaming is requested
	streaming := c.Query("streaming") == "true"

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	if streaming {
		// For streaming requests, use background context with timeout so streams don't live forever
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) //nolint:lostcancel

		// For streaming requests, initiate the stream and return stream info
		streamID, err := h.chatService.InitiateStreamForUser(ctx, userID, request.SessionID, request.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to initiate stream: %v", err),
			})
			cancel()
			return
		}

		go func() {
			<-ctx.Done()
			cancel()
			h.chatService.GetStreamManager().CloseStream(streamID)
		}()

		c.JSON(http.StatusOK, gin.H{
			"stream_id":  streamID,
			"session_id": request.SessionID,
			"stream_url": fmt.Sprintf("/v1/chat/stream/%s", streamID),
		})
	} else {
		// Non-streaming request - use request context so it cancels if client disconnects
		ctx := c.Request.Context()

		// Non-streaming request - get stream channel and collect all results
		streamChan, err := h.chatService.ProcessStreamMessageForUser(ctx, userID, request.SessionID, request.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to process message: %v", err),
			})
			return
		}

		// Collect all stream messages into a complete response
		response := h.collectStreamResponse(streamChan, request.SessionID)
		c.JSON(http.StatusOK, response)
	}
}

// ClearHistory handles DELETE /v1/chat/clear
func (h *ChatHandler) ClearHistory(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "session_id parameter is required",
		})
		return
	}

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	h.chatService.ClearHistoryForUser(userID, sessionID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Chat history cleared successfully",
	})
}

// StreamChat handles GET /v1/chat/stream (Server-Sent Events) - Legacy endpoint
// Note: For new implementations, use POST /v1/chat/send?streaming=true + GET /v1/chat/stream/{streamId}
func (h *ChatHandler) StreamChat(c *gin.Context) {
	sessionID := c.Query("session_id")
	message := c.Query("message")

	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "message parameter is required",
		})
		return
	}

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Use background context with timeout for streaming - don't tie to HTTP request lifecycle but prevent infinite streams
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Process streaming message
	streamChan, err := h.chatService.ProcessStreamMessageForUser(ctx, userID, sessionID, message)
	if err != nil {
		// Send error as SSE
		c.SSEvent("error", gin.H{
			"error": "Failed to process message: " + err.Error(),
		})
		return
	}

	// Stream responses
	for streamMsg := range streamChan {
		data, _ := json.Marshal(streamMsg)
		c.SSEvent("message", string(data))
		c.Writer.Flush()

		// Check if client disconnected
		if streamMsg.Type == "done" || streamMsg.Type == "error" {
			break
		}
	}
}

// StreamByID handles GET /v1/chat/stream/{streamId} (Server-Sent Events)
func (h *ChatHandler) StreamByID(c *gin.Context) {
	streamID := c.Param("streamId")
	if streamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "stream ID is required",
		})
		return
	}

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Use background context with timeout for connecting to stream
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Connect to existing stream
	streamChan, err := h.chatService.ConnectToStream(ctx, userID, streamID)
	cancel() // Cancel immediately after connecting, the stream itself has its own timeout
	if err != nil {
		// Send error as SSE
		c.SSEvent("error", gin.H{
			"error": "Failed to connect to stream: " + err.Error(),
		})
		return
	}

	// Stream responses
	for streamMsg := range streamChan {
		data, _ := json.Marshal(streamMsg)
		c.SSEvent("message", string(data))
		c.Writer.Flush()

		// Check if client disconnected
		if streamMsg.Type == "done" || streamMsg.Type == "error" {
			break
		}
	}
}

// GetMCPTools handles GET /v1/chat/mcp/tools
func (h *ChatHandler) GetMCPTools(c *gin.Context) {
	// Check for force refresh parameter
	forceRefresh := c.Query("force_refresh") == "true"

	if forceRefresh {
		// Force refresh tools from MCP servers
		if err := h.chatService.RefreshTools(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to refresh tools: %v", err),
			})
			return
		}
	}

	// Get tools (either cached or freshly refreshed)
	tools := h.chatService.GetAvailableTools()

	response := models.MCPToolsResponse{
		Tools:        tools,
		ForceRefresh: forceRefresh,
	}

	c.JSON(http.StatusOK, response)
}

// SendMessageWithFiles handles POST /v1/chat/send-with-files (multipart form)
func (h *ChatHandler) SendMessageWithFiles(c *gin.Context) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(64 << 20) // 64 MB max memory
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse multipart form: " + err.Error(),
		})
		return
	}

	// Get message and session_id from form
	message := c.PostForm("message")
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "message field is required",
		})
		return
	}

	sessionID := c.PostForm("session_id")

	// Process file attachments
	var attachments []models.Attachment
	form := c.Request.MultipartForm
	if form.File != nil {
		for fieldName, files := range form.File {
			if !strings.HasPrefix(fieldName, "file") {
				continue
			}

			for _, fileHeader := range files {
				// Read file content
				file, err := fileHeader.Open()
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "Failed to open file: " + err.Error(),
					})
					return
				}
				defer file.Close()

				// Read file content
				content, err := io.ReadAll(file)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "Failed to read file: " + err.Error(),
					})
					return
				}

				// Create attachment
				attachment := models.Attachment{
					ID:       uuid.New().String(),
					Filename: fileHeader.Filename,
					MimeType: fileHeader.Header.Get("Content-Type"),
					Size:     fileHeader.Size,
				}

				// For small files (< 10MB), embed content as base64
				// For larger files, we could save to disk and store path
				if len(content) < 10*1024*1024 { // 10MB
					attachment.Content = base64.StdEncoding.EncodeToString(content)
				} else {
					// For larger files, you might want to save to disk
					// and store the path, or use cloud storage
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "File too large. Maximum size is 10MB per file.",
					})
					return
				}

				attachments = append(attachments, attachment)
			}
		}
	}

	// Create chat request
	request := models.ChatRequest{
		Message:     message,
		SessionID:   sessionID,
		Attachments: attachments,
	}

	// Check if streaming is requested
	streaming := c.Query("streaming") == "true"

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	// Convert attachments to pointer slice for the service method
	var attachmentPtrs []*models.Attachment
	for i := range attachments {
		attachmentPtrs = append(attachmentPtrs, &attachments[i])
	}

	if streaming {
		// For streaming requests, use background context with timeout so streams don't live forever
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// For streaming requests with files, initiate the stream and return stream info
		// Note: We could extend InitiateStreamForUser to support attachments, but for now use a different approach
		streamChan, err := h.chatService.ProcessStreamMessageWithAttachmentsForUser(ctx, userID, request.SessionID, request.Message, attachmentPtrs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to initiate stream: %v", err),
			})
			return
		}

		// For file uploads with streaming, we'll store and return a stream ID
		streamID := fmt.Sprintf("file_stream_%s_%s", userID, uuid.New().String())
		h.chatService.GetStreamManager().StoreStream(streamID, userID, streamChan)

		c.JSON(http.StatusOK, gin.H{
			"stream_id":  streamID,
			"session_id": request.SessionID,
			"stream_url": fmt.Sprintf("/v1/chat/stream/%s", streamID),
		})
	} else {
		// Non-streaming request - use request context so it cancels if client disconnects
		ctx := c.Request.Context()

		// Handle tool approval/denial messages - these need special handling
		if strings.HasPrefix(request.Message, "[APPROVE_TOOLS:") || strings.HasPrefix(request.Message, "[DENY_TOOLS:") {
			// For tool approval messages, use the regular stream processing
			streamChan, err := h.chatService.ProcessStreamMessageForUser(ctx, userID, request.SessionID, request.Message)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Failed to process message: %v", err),
				})
				return
			}
			response := h.collectStreamResponse(streamChan, request.SessionID)
			c.JSON(http.StatusOK, response)
			return
		}

		// Non-streaming request - get stream channel and collect all results
		streamChan, err := h.chatService.ProcessStreamMessageWithAttachmentsForUser(ctx, userID, request.SessionID, request.Message, attachmentPtrs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to process message: %v", err),
			})
			return
		}

		// Collect all stream messages into a complete response
		response := h.collectStreamResponse(streamChan, request.SessionID)
		c.JSON(http.StatusOK, response)
	}
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String())
}
