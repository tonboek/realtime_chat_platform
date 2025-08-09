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

	// Create response with user information
	type MessageWithUser struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		Content   string `json:"content"`
		CreatedAt string `json:"created_at"`
		Nickname  string `json:"nickname"`
		Avatar    string `json:"avatar"`
	}

	var messagesWithUser []MessageWithUser
	for _, msg := range messages {
		// Get user information
		var user models.User
		if err := database.DB.Where("username = ?", msg.Username).First(&user).Error; err == nil {
			displayName := user.Username
			if user.Nickname != "" {
				displayName = user.Nickname
			}

			messagesWithUser = append(messagesWithUser, MessageWithUser{
				ID:        msg.ID,
				Username:  displayName,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
				Nickname:  user.Nickname,
				Avatar:    user.Avatar,
			})
		} else {
			// Fallback if user not found
			messagesWithUser = append(messagesWithUser, MessageWithUser{
				ID:        msg.ID,
				Username:  msg.Username,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
				Nickname:  "",
				Avatar:    "",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messagesWithUser,
		"count":    len(messagesWithUser),
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
