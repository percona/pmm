package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/percona/pmm/aichat-backend/internal/models"
	"github.com/percona/pmm/aichat-backend/internal/services"
)

// SessionHandler handles session-related endpoints
type SessionHandler struct {
	dbService *services.DatabaseService
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(dbService *services.DatabaseService) *SessionHandler {
	return &SessionHandler{
		dbService: dbService,
	}
}

// CreateSession handles POST /v1/chat/sessions
func (h *SessionHandler) CreateSession(c *gin.Context) {
	var request struct {
		Title string `json:"title"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	// Default title if empty
	if request.Title == "" {
		request.Title = "New Chat Session"
	}

	session, err := h.dbService.CreateSession(c.Request.Context(), userID, request.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create session: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// ListSessions handles GET /v1/chat/sessions
func (h *SessionHandler) ListSessions(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Calculate offset from page
	offset := (page - 1) * limit

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	sessions, err := h.dbService.GetUserSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get sessions: %v", err),
		})
		return
	}

	// Note: For simplicity, we're not implementing total count query
	// In production, you might want to add a separate count query
	response := gin.H{
		"sessions": sessions,
		"pagination": gin.H{
			"page":   page,
			"limit":  limit,
			"offset": offset,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetSession handles GET /v1/chat/sessions/:id
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	session, err := h.dbService.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		if errors.Is(err, services.ErrSessionNotFound) || errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get session: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSession handles PUT /v1/chat/sessions/:id
func (h *SessionHandler) UpdateSession(c *gin.Context) {
	sessionID := c.Param("id")

	var request struct {
		Title string `json:"title"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if request.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Title is required",
		})
		return
	}

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	err := h.dbService.UpdateSession(c.Request.Context(), sessionID, userID, request.Title)
	if err != nil {
		if errors.Is(err, services.ErrSessionNotFound) || errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update session: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session updated successfully",
	})
}

// DeleteSession handles DELETE /v1/chat/sessions/:id
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	err := h.dbService.DeleteSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		if errors.Is(err, services.ErrSessionNotFound) || errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete session: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session deleted successfully",
	})
}

// GetSessionMessages handles GET /v1/chat/sessions/:id/messages
func (h *SessionHandler) GetSessionMessages(c *gin.Context) {
	sessionID := c.Param("id")

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Calculate offset from page
	offset := (page - 1) * limit

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	// First verify the session belongs to the user
	_, err := h.dbService.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		if errors.Is(err, services.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to verify session: %v", err),
		})
		return
	}

	messages, err := h.dbService.GetSessionMessages(c.Request.Context(), sessionID, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get messages: %v", err),
		})
		return
	}

	// Ensure we return an empty array instead of null if no messages
	if messages == nil {
		messages = []*models.Message{}
	}

	// Note: For simplicity, we're not implementing total count query
	// In production, you might want to add a separate count query
	response := gin.H{
		"messages": messages,
		"pagination": gin.H{
			"page":   page,
			"limit":  limit,
			"offset": offset,
		},
	}

	c.JSON(http.StatusOK, response)
}

// ClearSessionMessages handles DELETE /v1/chat/sessions/:id/messages
func (h *SessionHandler) ClearSessionMessages(c *gin.Context) {
	sessionID := c.Param("id")

	// Get user ID from authentication header
	userID, ok := requireUserID(c)
	if !ok {
		return // Error response already sent by requireUserID
	}

	err := h.dbService.ClearSessionMessages(c.Request.Context(), sessionID, userID)
	if err != nil {
		if errors.Is(err, services.ErrSessionNotFound) || errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to clear session messages: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session messages cleared successfully",
	})
}
