package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"realtime_chat_platform/internal/database"
	"realtime_chat_platform/internal/models"

	"github.com/gin-gonic/gin"
)

// UploadAvatarHandler handles avatar file upload
func UploadAvatarHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file type
	if !isValidImageType(file) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only JPG, PNG, and GIF are allowed"})
		return
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum size is 5MB"})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("avatar_%d_%d%s", userID, time.Now().Unix(), ext)

	// Create uploads directory if it doesn't exist
	uploadDir := "web/static/uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Save file
	filepath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Update user's avatar URL in database
	avatarURL := fmt.Sprintf("/static/uploads/avatars/%s", filename)
	fmt.Printf("Updating avatar URL for user %d: %s\n", userID, avatarURL)

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).UpdateColumn("avatar", avatarURL).Error; err != nil {
		// If database update fails, delete the uploaded file
		os.Remove(filepath)
		fmt.Printf("Failed to update avatar in database: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	fmt.Printf("Successfully updated avatar in database\n")

	// Get updated user profile
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Avatar uploaded successfully",
		"avatar_url": avatarURL,
		"profile": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"avatar":      user.Avatar,
			"bio":         user.Bio,
			"last_active": user.LastActive,
			"created_at":  user.CreatedAt,
		},
	})
}

// isValidImageType checks if the uploaded file is a valid image
func isValidImageType(file *multipart.FileHeader) bool {
	contentType := file.Header.Get("Content-Type")
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, validType := range validTypes {
		if strings.Contains(contentType, validType) {
			return true
		}
	}

	// Also check file extension as fallback
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

	for _, validExt := range validExtensions {
		if ext == validExt {
			return true
		}
	}

	return false
}
