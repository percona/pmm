package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var authLogger = logrus.WithField("component", "auth-handler")

// getUserID extracts user ID from X-User-ID header set by PMM auth server
// Returns the user ID string if valid, or empty string if invalid/missing
func getUserID(c *gin.Context) string {
	userIDHeader := c.GetHeader("X-User-ID")
	authLogger.WithField("user_id_header", userIDHeader).Debug("Extracting user ID from header")
	if userIDHeader == "" {
		return ""
	}

	// Validate that it's a valid positive integer
	if userID, err := strconv.Atoi(userIDHeader); err == nil && userID > 0 {
		return userIDHeader
	}

	// If invalid, return empty string
	authLogger.WithField("user_id_header", userIDHeader).Warn("Invalid user ID header format")
	return ""
}

// requireUserID extracts and validates user ID, returning error response if invalid
func requireUserID(c *gin.Context) (string, bool) {
	userID := getUserID(c)
	if userID == "" {
		c.JSON(401, gin.H{
			"error": "Authentication required: missing or invalid user ID",
		})
		return "", false
	}
	return userID, true
}
