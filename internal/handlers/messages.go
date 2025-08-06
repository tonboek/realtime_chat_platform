package handlers

import (
	"net/http"
	"strconv"

	"realtime_chat_platform/internal/database"
	"realtime_chat_platform/internal/models"
	"realtime_chat_platform/internal/websocket"

	"github.com/gin-gonic/gin"
)

// GetMessageHistory retrieves recent messages from the database
func GetMessageHistory(c *gin.Context) {
	// Get limit parameter (default 50 messages)
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50 // Default to 50 if invalid
	}

	var messages []models.Message

	// Get recent messages ordered by creation time (newest first)
	if err := database.DB.Order("created_at desc").Limit(limit).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	// Reverse the order to show oldest first in chat
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// GetOnlineUsers returns the list of currently connected users
func GetOnlineUsers(c *gin.Context) {
	onlineUsers := websocket.GlobalHub.GetOnlineUsers()

	c.JSON(http.StatusOK, gin.H{
		"users": onlineUsers,
		"count": len(onlineUsers),
	})
}
