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

// SendMessage handles POST /v1/chat/send
func (h *ChatHandler) SendMessage(c *gin.Context) {
	var request models.ChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	response, err := h.chatService.ProcessMessage(ctx, request.SessionID, request.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process message: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetHistory handles GET /v1/chat/history
func (h *ChatHandler) GetHistory(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "session_id parameter is required",
		})
		return
	}

	history := h.chatService.GetHistory(sessionID)
	c.JSON(http.StatusOK, history)
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

	h.chatService.ClearHistory(sessionID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Chat history cleared successfully",
	})
}

// StreamChat handles GET /v1/chat/stream (Server-Sent Events)
func (h *ChatHandler) StreamChat(c *gin.Context) {
	sessionID := c.Query("session_id")
	message := c.Query("message")

	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "message parameter is required",
		})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Process streaming message
	streamChan, err := h.chatService.ProcessStreamMessage(ctx, sessionID, message)
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

// GetMCPServerStatus handles GET /v1/chat/mcp/servers/status
func (h *ChatHandler) GetMCPServerStatus(c *gin.Context) {
	// This requires access to the MCP service directly
	// We'll need to pass it through the chat service or create a separate handler
	// For now, we'll create a method in the chat service to expose this
	status := h.chatService.GetMCPServerStatus()

	c.JSON(http.StatusOK, gin.H{
		"servers":   status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	response, err := h.chatService.ProcessMessageWithAttachments(ctx, request.SessionID, request.Message, request.Attachments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process message: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
