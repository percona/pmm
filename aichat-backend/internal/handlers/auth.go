package handlers

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

// getUserID extracts user ID from X-User-ID header set by PMM auth server
// Returns the user ID string if valid, or empty string if invalid/missing
func getUserID(c *gin.Context) string {
	log.Printf("🔍 Auth: Getting user ID from header: %s", c.Request.Header.Get("X-User-ID"))
	userIDHeader := c.GetHeader("X-User-ID")
	log.Printf("🔍 Auth: X-User-ID header: %s", userIDHeader)
	if userIDHeader == "" {
		return ""
	}

	// Validate that it's a valid positive integer
	if userID, err := strconv.Atoi(userIDHeader); err == nil && userID > 0 {
		return userIDHeader
	}

	// If invalid, return empty string
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
